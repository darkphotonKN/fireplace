package calendar

import (
	"time"

	"github.com/darkphotonKN/fireplace/internal/constants"
	"github.com/darkphotonKN/fireplace/internal/models"
	"github.com/google/uuid"
)

// Recommendation represents an AI-generated recommendation to schedule.
type Recommendation struct {
	Title       string
	URL         *string
	Description *string
}

// ScheduleInput contains all data needed by the pure scheduling function.
type ScheduleInput struct {
	PlanID          uuid.UUID
	PlanType        string
	DailyItems      []models.ChecklistItem
	LongtermItems   []models.ChecklistItem // expected pre-ranked (by LLM or sequence)
	Recommendations []Recommendation
	PinnedEntries   []CalendarEntry
	StartDate       time.Time
	EndDate         time.Time
}

// dayState tracks available capacity and budgets for a single day.
type dayState struct {
	date   time.Time
	slots  [constants.SlotCap]*CalendarEntry // nil = open
	filled int
}

// budget tracks remaining tier slots for a day.
type budget struct {
	daily, longterm, rec int
}

// ScheduleMonth is a pure deterministic function that distributes items across
// a month range. No DB calls, no side effects.
func ScheduleMonth(input ScheduleInput) []CalendarEntry {
	ratio := constants.GetRatioForPlanType(constants.PlanType(input.PlanType))
	days := initDays(input.StartDate, input.EndDate)

	// 1. Pre-fill pinned entries
	for i := range days {
		for _, pinned := range input.PinnedEntries {
			if sameDay(pinned.ScheduledDate, days[i].date) && pinned.Position >= 1 && pinned.Position <= constants.SlotCap {
				days[i].slots[pinned.Position-1] = &pinned
				days[i].filled++
			}
		}
	}

	// 2. Calculate per-day budgets accounting for pinned entries
	budgets := make([]budget, len(days))
	for i, day := range days {
		capacity := constants.SlotCap - day.filled
		if capacity <= 0 {
			continue
		}
		total := ratio.Daily + ratio.Longterm + ratio.Rec
		d := capacity * ratio.Daily / total
		l := capacity * ratio.Longterm / total
		r := capacity - d - l
		budgets[i] = budget{daily: d, longterm: l, rec: r}
	}

	// 3. Place daily items (scheduled_time items first, then by order)
	scheduledFirst, unscheduled := partitionScheduled(input.DailyItems, input.StartDate, input.EndDate)
	dailyQueue := append(scheduledFirst, unscheduled...)

	for _, item := range dailyQueue {
		placed := false
		// If item has scheduled_time in range, try that day first
		if item.ScheduledTime != nil {
			itemDate := truncateToDate(*item.ScheduledTime)
			startDate := truncateToDate(input.StartDate)
			endDate := truncateToDate(input.EndDate)
			if !itemDate.Before(startDate) && !itemDate.After(endDate) {
				dayIdx := dayIndex(input.StartDate, itemDate)
				if dayIdx >= 0 && dayIdx < len(days) {
					if budgets[dayIdx].daily > 0 {
						if placeEntry(&days[dayIdx], input.PlanID, &item, "daily") {
							budgets[dayIdx].daily--
							placed = true
						}
					} else {
						// Cascade: try longterm budget, then rec budget on same day
						if budgets[dayIdx].longterm > 0 {
							if placeEntry(&days[dayIdx], input.PlanID, &item, "daily") {
								budgets[dayIdx].longterm--
								placed = true
							}
						} else if budgets[dayIdx].rec > 0 {
							if placeEntry(&days[dayIdx], input.PlanID, &item, "daily") {
								budgets[dayIdx].rec--
								placed = true
							}
						}
					}
				}
			}
		}
		if !placed {
			// Find first day with daily budget, then cascade, then overflow
			for i := range days {
				if budgets[i].daily > 0 {
					if placeEntry(&days[i], input.PlanID, &item, "daily") {
						budgets[i].daily--
						placed = true
						break
					}
				}
			}
		}
		if !placed {
			// Cascade: daily overflow goes to latest day with longterm budget
			// (pushes excess toward end of month, keeps early days at ratio)
			bestIdx := findLastDayWithBudget(budgets, func(b budget) int { return b.longterm })
			if bestIdx >= 0 {
				if placeEntry(&days[bestIdx], input.PlanID, &item, "daily") {
					budgets[bestIdx].longterm--
					placed = true
				}
			}
		}
		if !placed {
			bestIdx := findLastDayWithBudget(budgets, func(b budget) int { return b.rec })
			if bestIdx >= 0 {
				if placeEntry(&days[bestIdx], input.PlanID, &item, "daily") {
					budgets[bestIdx].rec--
					placed = true
				}
			}
		}
	}

	// 4. Place longterm items — spread evenly across days
	for _, item := range input.LongtermItems {
		placed := false
		// Find day with highest longterm budget (ties: earliest)
		bestIdx := -1
		bestBudget := 0
		for i := range days {
			if budgets[i].longterm > bestBudget {
				bestBudget = budgets[i].longterm
				bestIdx = i
			}
		}
		if bestIdx >= 0 {
			if placeEntry(&days[bestIdx], input.PlanID, &item, "longterm") {
				budgets[bestIdx].longterm--
				placed = true
			}
		}
		if !placed {
			// Cascade to rec budget
			bestIdx = -1
			bestBudget = 0
			for i := range days {
				if budgets[i].rec > bestBudget {
					bestBudget = budgets[i].rec
					bestIdx = i
				}
			}
			if bestIdx >= 0 {
				if placeEntry(&days[bestIdx], input.PlanID, &item, "longterm") {
					budgets[bestIdx].rec--
					placed = true
				}
			}
		}
		// Final overflow: any open slot
		if !placed {
			for i := range days {
				if days[i].filled < constants.SlotCap {
					if placeEntry(&days[i], input.PlanID, &item, "longterm") {
						break
					}
				}
			}
		}
	}

	// 5. Place recommendations — spread across days with highest rec budget
	for _, rec := range input.Recommendations {
		bestIdx := -1
		bestBudget := 0
		for i := range days {
			if budgets[i].rec > bestBudget {
				bestBudget = budgets[i].rec
				bestIdx = i
			}
		}
		if bestIdx >= 0 {
			placeRec(&days[bestIdx], input.PlanID, &rec)
			budgets[bestIdx].rec--
			continue
		}
		// Overflow: any open slot on any day
		for i := range days {
			if days[i].filled < constants.SlotCap {
				placeRec(&days[i], input.PlanID, &rec)
				break
			}
		}
	}

	// 6. Collect all entries
	var result []CalendarEntry
	for _, day := range days {
		for _, slot := range day.slots {
			if slot != nil {
				result = append(result, *slot)
			}
		}
	}
	return result
}

// findDayWithMostBudget returns the index of the day with the highest value
// from the given budget accessor. Ties broken by fewest filled slots (most
// remaining capacity) to spread items evenly.
func findDayWithMostBudget(days []dayState, budgets []budget, accessor func(budget) int) int {
	bestIdx := -1
	bestVal := 0
	leastFilled := constants.SlotCap + 1
	for i := range days {
		val := accessor(budgets[i])
		if val <= 0 {
			continue
		}
		if val > bestVal || (val == bestVal && days[i].filled < leastFilled) {
			bestIdx = i
			bestVal = val
			leastFilled = days[i].filled
		}
	}
	return bestIdx
}

// findLastDayWithBudget returns the last (latest) day index with budget > 0
// from the accessor. Used for daily overflow to push excess toward end of month.
func findLastDayWithBudget(budgets []budget, accessor func(budget) int) int {
	bestIdx := -1
	for i := range budgets {
		if accessor(budgets[i]) > 0 {
			bestIdx = i
		}
	}
	return bestIdx
}

func initDays(start, end time.Time) []dayState {
	var days []dayState
	for d := start; !d.After(end); d = d.AddDate(0, 0, 1) {
		days = append(days, dayState{date: d})
	}
	return days
}

func sameDay(a, b time.Time) bool {
	return a.Year() == b.Year() && a.Month() == b.Month() && a.Day() == b.Day()
}

func dayIndex(start, target time.Time) int {
	return int(target.Sub(start).Hours() / 24)
}

// placeEntry finds the next open slot in a day and places a checklist item entry.
func placeEntry(day *dayState, planID uuid.UUID, item *models.ChecklistItem, entryType string) bool {
	for pos := 0; pos < constants.SlotCap; pos++ {
		if day.slots[pos] == nil {
			desc := item.Description
			done := item.Done
			itemID := item.ID
			day.slots[pos] = &CalendarEntry{
				PlanID:          planID,
				ChecklistItemID: &itemID,
				EntryType:       entryType,
				ScheduledDate:   day.date,
				Position:        pos + 1,
				Pinned:          false,
				Description:     &desc,
				Done:            &done,
			}
			day.filled++
			return true
		}
	}
	return false
}

// placeRec finds the next open slot and places a recommendation entry.
func placeRec(day *dayState, planID uuid.UUID, rec *Recommendation) bool {
	for pos := 0; pos < constants.SlotCap; pos++ {
		if day.slots[pos] == nil {
			day.slots[pos] = &CalendarEntry{
				PlanID:         planID,
				EntryType:      "recommendation",
				ScheduledDate:  day.date,
				Position:       pos + 1,
				Pinned:         false,
				RecTitle:       &rec.Title,
				RecURL:         rec.URL,
				RecDescription: rec.Description,
			}
			day.filled++
			return true
		}
	}
	return false
}

// partitionScheduled splits items into those with scheduled_time in the date range
// and those without. Compares dates only (ignores time-of-day).
func partitionScheduled(items []models.ChecklistItem, start, end time.Time) (scheduled, unscheduled []models.ChecklistItem) {
	startDate := truncateToDate(start)
	endDate := truncateToDate(end)
	for _, item := range items {
		if item.ScheduledTime != nil {
			itemDate := truncateToDate(*item.ScheduledTime)
			if !itemDate.Before(startDate) && !itemDate.After(endDate) {
				scheduled = append(scheduled, item)
				continue
			}
		}
		unscheduled = append(unscheduled, item)
	}
	return
}

func truncateToDate(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
}

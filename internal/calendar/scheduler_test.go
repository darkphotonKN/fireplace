package calendar

import (
	"testing"
	"time"

	"github.com/darkphotonKN/fireplace/internal/models"
	"github.com/google/uuid"
)

// Helper to create N checklist items with a given scope.
func makeItems(n int, scope string, planID uuid.UUID) []models.ChecklistItem {
	items := make([]models.ChecklistItem, n)
	for i := range items {
		items[i] = models.ChecklistItem{
			Description: scope + " item",
			Done:        false,
			Scope:       scope,
			PlanID:      planID,
		}
		items[i].ID = uuid.New()
	}
	return items
}

// Helper to create N recommendations.
func makeRecs(n int) []Recommendation {
	recs := make([]Recommendation, n)
	for i := range recs {
		title := "Rec video"
		url := "https://example.com/video"
		desc := "A recommendation"
		recs[i] = Recommendation{Title: title, URL: &url, Description: &desc}
	}
	return recs
}

func TestScheduleMonth_DevPlan_BasicRatio(t *testing.T) {
	planID := uuid.New()
	start := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 3, 3, 0, 0, 0, 0, time.UTC) // 3 days

	daily := makeItems(10, "daily", planID)
	longterm := makeItems(5, "longterm", planID)
	recs := makeRecs(3)

	entries := ScheduleMonth(ScheduleInput{
		PlanID:          planID,
		PlanType:        "project",
		DailyItems:      daily,
		LongtermItems:   longterm,
		Recommendations: recs,
		PinnedEntries:   []CalendarEntry{},
		StartDate:       start,
		EndDate:         end,
	})

	// Group entries by date
	byDate := groupByDate(entries)

	day1 := byDate["2026-03-01"]
	dailyCount, longtermCount, recCount := countByType(day1)

	// Dev ratio: 5 daily / 2 longterm / 1 rec
	if dailyCount != 5 {
		t.Errorf("day 1: expected 5 daily, got %d", dailyCount)
	}
	if longtermCount != 2 {
		t.Errorf("day 1: expected 2 longterm, got %d", longtermCount)
	}
	if recCount != 1 {
		t.Errorf("day 1: expected 1 rec, got %d", recCount)
	}
	if len(day1) != 8 {
		t.Errorf("day 1: expected 8 total entries, got %d", len(day1))
	}
}

func TestScheduleMonth_LearningPlan_BasicRatio(t *testing.T) {
	planID := uuid.New()
	start := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 3, 3, 0, 0, 0, 0, time.UTC)

	// Enough items to fill cleanly: 3 days × 3/2/3 = 9 daily, 6 longterm, 9 recs
	daily := makeItems(9, "daily", planID)
	longterm := makeItems(6, "longterm", planID)
	recs := makeRecs(9)

	entries := ScheduleMonth(ScheduleInput{
		PlanID:          planID,
		PlanType:        "learning",
		DailyItems:      daily,
		LongtermItems:   longterm,
		Recommendations: recs,
		PinnedEntries:   []CalendarEntry{},
		StartDate:       start,
		EndDate:         end,
	})

	byDate := groupByDate(entries)
	day1 := byDate["2026-03-01"]
	dailyCount, longtermCount, recCount := countByType(day1)

	// Learning ratio: 3 daily / 2 longterm / 3 rec
	if dailyCount != 3 {
		t.Errorf("day 1: expected 3 daily, got %d", dailyCount)
	}
	if longtermCount != 2 {
		t.Errorf("day 1: expected 2 longterm, got %d", longtermCount)
	}
	if recCount != 3 {
		t.Errorf("day 1: expected 3 rec, got %d", recCount)
	}

	// All 3 days should be full
	if len(entries) != 24 {
		t.Errorf("expected 24 total entries (3 days × 8), got %d", len(entries))
	}
}

func TestScheduleMonth_CascadeRule_DailySlotsOverflowToLongterm(t *testing.T) {
	planID := uuid.New()
	start := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC) // 1 day

	// Dev plan: budget 5/2/1. Only 2 daily items → 3 unused daily slots cascade
	daily := makeItems(2, "daily", planID)
	longterm := makeItems(4, "longterm", planID)
	recs := makeRecs(3)

	entries := ScheduleMonth(ScheduleInput{
		PlanID:          planID,
		PlanType:        "project",
		DailyItems:      daily,
		LongtermItems:   longterm,
		Recommendations: recs,
		PinnedEntries:   []CalendarEntry{},
		StartDate:       start,
		EndDate:         end,
	})

	byDate := groupByDate(entries)
	day1 := byDate["2026-03-01"]
	dailyCount, longtermCount, recCount := countByType(day1)

	// 2 daily placed, 3 unused daily slots cascade down
	// Longterm gets 2 (base) + 3 (cascade) = 5, but only 4 items → 4 placed, 1 cascades to rec
	// Rec gets 1 (base) + 1 (cascade) = 2, and we have 3 recs → 2 placed
	// Total: 2 + 4 + 2 = 8
	if dailyCount != 2 {
		t.Errorf("expected 2 daily, got %d", dailyCount)
	}
	if longtermCount != 4 {
		t.Errorf("expected 4 longterm, got %d", longtermCount)
	}
	if recCount != 2 {
		t.Errorf("expected 2 rec, got %d", recCount)
	}
	if len(day1) != 8 {
		t.Errorf("expected 8 total entries, got %d", len(day1))
	}
}

func TestScheduleMonth_OverflowRule_ExcessPushToNextDay(t *testing.T) {
	planID := uuid.New()
	start := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 3, 2, 0, 0, 0, 0, time.UTC) // 2 days

	// Dev: budget 5/2/1 per day = 10 daily slots across 2 days
	// 12 daily items → 10 fit in daily budget, 2 overflow via cascade
	daily := makeItems(12, "daily", planID)

	entries := ScheduleMonth(ScheduleInput{
		PlanID:          planID,
		PlanType:        "project",
		DailyItems:      daily,
		LongtermItems:   []models.ChecklistItem{},
		Recommendations: []Recommendation{},
		PinnedEntries:   []CalendarEntry{},
		StartDate:       start,
		EndDate:         end,
	})

	byDate := groupByDate(entries)

	// All 12 should be placed across 2 days (cascade fills longterm+rec budgets)
	total := len(byDate["2026-03-01"]) + len(byDate["2026-03-02"])
	if total != 12 {
		t.Errorf("expected 12 total entries, got %d", total)
	}

	// No day should exceed 8
	if len(byDate["2026-03-01"]) > 8 {
		t.Errorf("day 1 exceeded cap: got %d", len(byDate["2026-03-01"]))
	}
	if len(byDate["2026-03-02"]) > 8 {
		t.Errorf("day 2 exceeded cap: got %d", len(byDate["2026-03-02"]))
	}
}

func TestScheduleMonth_PinnedEntries_PreservedAndReduceCapacity(t *testing.T) {
	planID := uuid.New()
	start := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC) // 1 day

	pinnedID := uuid.New()
	pinned := []CalendarEntry{
		{
			ID:            pinnedID,
			PlanID:        planID,
			EntryType:     "longterm",
			ScheduledDate: start,
			Position:      1,
			Pinned:        true,
		},
		{
			ID:            uuid.New(),
			PlanID:        planID,
			EntryType:     "daily",
			ScheduledDate: start,
			Position:      3,
			Pinned:        true,
		},
	}

	// 2 pinned → 6 remaining capacity
	daily := makeItems(10, "daily", planID)
	longterm := makeItems(5, "longterm", planID)
	recs := makeRecs(5)

	entries := ScheduleMonth(ScheduleInput{
		PlanID:          planID,
		PlanType:        "project",
		DailyItems:      daily,
		LongtermItems:   longterm,
		Recommendations: recs,
		PinnedEntries:   pinned,
		StartDate:       start,
		EndDate:         end,
	})

	byDate := groupByDate(entries)
	day1 := byDate["2026-03-01"]

	// Should not exceed 8 total (2 pinned + 6 scheduled)
	if len(day1) != 8 {
		t.Errorf("expected 8 total entries, got %d", len(day1))
	}

	// Pinned entries should be present at their original positions
	foundPinned := 0
	for _, e := range day1 {
		if e.Pinned {
			foundPinned++
		}
	}
	if foundPinned != 2 {
		t.Errorf("expected 2 pinned entries, got %d", foundPinned)
	}

	// Position 1 and 3 should be the pinned entries
	for _, e := range day1 {
		if e.Position == 1 && !e.Pinned {
			t.Error("position 1 should be pinned")
		}
		if e.Position == 3 && !e.Pinned {
			t.Error("position 3 should be pinned")
		}
	}
}

func TestScheduleMonth_AllPinned_ReturnsOnlyPinned(t *testing.T) {
	planID := uuid.New()
	start := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)

	// Fill all 8 slots with pinned entries
	pinned := make([]CalendarEntry, 8)
	for i := range pinned {
		pinned[i] = CalendarEntry{
			ID:            uuid.New(),
			PlanID:        planID,
			EntryType:     "daily",
			ScheduledDate: start,
			Position:      i + 1,
			Pinned:        true,
		}
	}

	entries := ScheduleMonth(ScheduleInput{
		PlanID:          planID,
		PlanType:        "project",
		DailyItems:      makeItems(5, "daily", planID),
		LongtermItems:   makeItems(3, "longterm", planID),
		Recommendations: makeRecs(2),
		PinnedEntries:   pinned,
		StartDate:       start,
		EndDate:         end,
	})

	if len(entries) != 8 {
		t.Errorf("expected 8 entries (all pinned), got %d", len(entries))
	}
	for _, e := range entries {
		if !e.Pinned {
			t.Error("all entries should be pinned")
		}
	}
}

func TestScheduleMonth_NoItems_ReturnsEmpty(t *testing.T) {
	planID := uuid.New()
	start := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 3, 3, 0, 0, 0, 0, time.UTC)

	entries := ScheduleMonth(ScheduleInput{
		PlanID:          planID,
		PlanType:        "project",
		DailyItems:      []models.ChecklistItem{},
		LongtermItems:   []models.ChecklistItem{},
		Recommendations: []Recommendation{},
		PinnedEntries:   []CalendarEntry{},
		StartDate:       start,
		EndDate:         end,
	})

	if len(entries) != 0 {
		t.Errorf("expected 0 entries, got %d", len(entries))
	}
}

func TestScheduleMonth_ScheduledTime_PlacedOnTargetDate(t *testing.T) {
	planID := uuid.New()
	start := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 3, 3, 0, 0, 0, 0, time.UTC)

	scheduledTime := time.Date(2026, 3, 3, 10, 0, 0, 0, time.UTC)
	scheduledItem := models.ChecklistItem{
		Description:   "Scheduled task",
		Done:          false,
		Scope:         "daily",
		PlanID:        planID,
		ScheduledTime: &scheduledTime,
	}
	scheduledItem.ID = uuid.New()

	// Only 1 item with a scheduled time
	entries := ScheduleMonth(ScheduleInput{
		PlanID:          planID,
		PlanType:        "project",
		DailyItems:      []models.ChecklistItem{scheduledItem},
		LongtermItems:   []models.ChecklistItem{},
		Recommendations: []Recommendation{},
		PinnedEntries:   []CalendarEntry{},
		StartDate:       start,
		EndDate:         end,
	})

	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}

	entryDate := entries[0].ScheduledDate.Format("2006-01-02")
	if entryDate != "2026-03-03" {
		t.Errorf("expected entry on 2026-03-03, got %s", entryDate)
	}
}

func TestScheduleMonth_LongtermSpreadEvenly(t *testing.T) {
	planID := uuid.New()
	start := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 3, 3, 0, 0, 0, 0, time.UTC)

	// Dev budget: 2 longterm per day × 3 days = 6 slots. 6 items → 2 per day.
	longterm := makeItems(6, "longterm", planID)

	entries := ScheduleMonth(ScheduleInput{
		PlanID:          planID,
		PlanType:        "project",
		DailyItems:      []models.ChecklistItem{},
		LongtermItems:   longterm,
		Recommendations: []Recommendation{},
		PinnedEntries:   []CalendarEntry{},
		StartDate:       start,
		EndDate:         end,
	})

	byDate := groupByDate(entries)

	for _, date := range []string{"2026-03-01", "2026-03-02", "2026-03-03"} {
		count := len(byDate[date])
		if count != 2 {
			t.Errorf("%s: expected 2 longterm entries, got %d", date, count)
		}
	}
}

// Test helpers
func groupByDate(entries []CalendarEntry) map[string][]CalendarEntry {
	m := make(map[string][]CalendarEntry)
	for _, e := range entries {
		key := e.ScheduledDate.Format("2006-01-02")
		m[key] = append(m[key], e)
	}
	return m
}

func countByType(entries []CalendarEntry) (daily, longterm, rec int) {
	for _, e := range entries {
		switch e.EntryType {
		case "daily":
			daily++
		case "longterm":
			longterm++
		case "recommendation":
			rec++
		}
	}
	return
}

package calendar

import (
	"fmt"
	"sort"
	"time"

	"github.com/google/uuid"
)

type Service struct {
	repo CalendarRepository
}

func NewService(repo CalendarRepository) *Service {
	return &Service{repo: repo}
}

// GetMonth returns a MonthResponse with all days in the given month,
// each populated with their calendar entries.
func (s *Service) GetMonth(planID uuid.UUID, month string) (*MonthResponse, error) {
	startDate, endDate, err := parseMonthRange(month)
	if err != nil {
		return nil, err
	}

	entries, err := s.repo.GetByMonth(planID, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch calendar entries: %w", err)
	}

	days := buildDaySlots(startDate, endDate, entries)

	return &MonthResponse{
		PlanID: planID.String(),
		Month:  month,
		Days:   days,
	}, nil
}

// parseMonthRange parses "YYYY-MM" into the first and last day of that month.
func parseMonthRange(month string) (time.Time, time.Time, error) {
	if month == "" {
		return time.Time{}, time.Time{}, fmt.Errorf("month parameter is required")
	}

	t, err := time.Parse("2006-01", month)
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("invalid month format, expected YYYY-MM: %w", err)
	}

	startDate := t
	endDate := t.AddDate(0, 1, -1) // last day of month

	return startDate, endDate, nil
}

// buildDaySlots creates a DaySlot for every day in the range and
// populates each with matching entries.
func buildDaySlots(startDate, endDate time.Time, entries []CalendarEntry) []DaySlot {
	// Index entries by date string for fast lookup
	entryMap := make(map[string][]CalendarEntry)
	for _, entry := range entries {
		dateKey := entry.ScheduledDate.Format("2006-01-02")
		entryMap[dateKey] = append(entryMap[dateKey], entry)
	}

	var days []DaySlot
	for d := startDate; !d.After(endDate); d = d.AddDate(0, 0, 1) {
		dateKey := d.Format("2006-01-02")
		dayEntries := entryMap[dateKey]
		if dayEntries == nil {
			dayEntries = []CalendarEntry{}
		}

		// Ensure entries are sorted by position
		sort.Slice(dayEntries, func(i, j int) bool {
			return dayEntries[i].Position < dayEntries[j].Position
		})

		days = append(days, DaySlot{
			Date:    dateKey,
			Entries: dayEntries,
		})
	}

	return days
}

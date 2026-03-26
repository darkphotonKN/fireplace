package calendar

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

// mockRepository implements the repository interface for testing.
type mockRepository struct {
	entries []CalendarEntry
	err     error
}

func (m *mockRepository) GetByMonth(planID uuid.UUID, startDate, endDate time.Time) ([]CalendarEntry, error) {
	return m.entries, m.err
}

func TestGetMonth_EmptyMonth_ReturnsAllDays(t *testing.T) {
	repo := &mockRepository{entries: []CalendarEntry{}, err: nil}
	service := NewService(repo)

	planID := uuid.New()
	resp, err := service.GetMonth(planID, "2026-03")

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if resp.PlanID != planID.String() {
		t.Errorf("expected planID %s, got %s", planID.String(), resp.PlanID)
	}

	if resp.Month != "2026-03" {
		t.Errorf("expected month 2026-03, got %s", resp.Month)
	}

	// March 2026 has 31 days
	if len(resp.Days) != 31 {
		t.Errorf("expected 31 days for March, got %d", len(resp.Days))
	}

	// All days should have empty entries
	for _, day := range resp.Days {
		if len(day.Entries) != 0 {
			t.Errorf("expected 0 entries for day %s, got %d", day.Date, len(day.Entries))
		}
	}

	// First day should be 2026-03-01
	if resp.Days[0].Date != "2026-03-01" {
		t.Errorf("expected first day 2026-03-01, got %s", resp.Days[0].Date)
	}

	// Last day should be 2026-03-31
	if resp.Days[30].Date != "2026-03-31" {
		t.Errorf("expected last day 2026-03-31, got %s", resp.Days[30].Date)
	}
}

func TestGetMonth_WithEntries_GroupsByDay(t *testing.T) {
	planID := uuid.New()
	entryID1 := uuid.New()
	entryID2 := uuid.New()
	checklistID := uuid.New()
	desc := "Write tests"
	done := false

	repo := &mockRepository{
		entries: []CalendarEntry{
			{
				ID:              entryID1,
				PlanID:          planID,
				ChecklistItemID: &checklistID,
				EntryType:       "daily",
				ScheduledDate:   time.Date(2026, 3, 5, 0, 0, 0, 0, time.UTC),
				Position:        1,
				Pinned:          false,
				Description:     &desc,
				Done:            &done,
			},
			{
				ID:            entryID2,
				PlanID:        planID,
				EntryType:     "recommendation",
				ScheduledDate: time.Date(2026, 3, 5, 0, 0, 0, 0, time.UTC),
				Position:      2,
				Pinned:        false,
			},
		},
		err: nil,
	}
	service := NewService(repo)

	resp, err := service.GetMonth(planID, "2026-03")

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Day 5 (index 4) should have 2 entries
	day5 := resp.Days[4]
	if day5.Date != "2026-03-05" {
		t.Errorf("expected date 2026-03-05, got %s", day5.Date)
	}
	if len(day5.Entries) != 2 {
		t.Fatalf("expected 2 entries on March 5, got %d", len(day5.Entries))
	}

	if day5.Entries[0].EntryType != "daily" {
		t.Errorf("expected first entry type daily, got %s", day5.Entries[0].EntryType)
	}
	if day5.Entries[1].EntryType != "recommendation" {
		t.Errorf("expected second entry type recommendation, got %s", day5.Entries[1].EntryType)
	}

	// Other days should be empty
	if len(resp.Days[0].Entries) != 0 {
		t.Errorf("expected 0 entries on March 1, got %d", len(resp.Days[0].Entries))
	}
}

func TestGetMonth_February_Returns28Or29Days(t *testing.T) {
	repo := &mockRepository{entries: []CalendarEntry{}, err: nil}
	service := NewService(repo)

	// 2026 is not a leap year
	resp, err := service.GetMonth(uuid.New(), "2026-02")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(resp.Days) != 28 {
		t.Errorf("expected 28 days for Feb 2026, got %d", len(resp.Days))
	}

	// 2024 is a leap year
	resp, err = service.GetMonth(uuid.New(), "2024-02")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(resp.Days) != 29 {
		t.Errorf("expected 29 days for Feb 2024, got %d", len(resp.Days))
	}
}

func TestGetMonth_InvalidMonth_ReturnsError(t *testing.T) {
	repo := &mockRepository{entries: []CalendarEntry{}, err: nil}
	service := NewService(repo)

	_, err := service.GetMonth(uuid.New(), "invalid")
	if err == nil {
		t.Error("expected error for invalid month, got nil")
	}

	_, err = service.GetMonth(uuid.New(), "")
	if err == nil {
		t.Error("expected error for empty month, got nil")
	}
}

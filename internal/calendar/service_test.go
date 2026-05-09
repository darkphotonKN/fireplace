package calendar

import (
	"context"
	"testing"
	"time"

	"github.com/darkphotonKN/fireplace/internal/constants"
	"github.com/darkphotonKN/fireplace/internal/models"
	"github.com/google/uuid"
)

type mockRepository struct {
	items   []*models.ChecklistItem
	itemErr error

	lastPlanID uuid.UUID
	lastStart  time.Time
	lastEnd    time.Time
}

func (m *mockRepository) GetItemsInWindow(ctx context.Context, planID uuid.UUID, windowStart, windowEnd time.Time) ([]*models.ChecklistItem, error) {
	m.lastPlanID = planID
	m.lastStart = windowStart
	m.lastEnd = windowEnd
	return m.items, m.itemErr
}

type mockOwnership struct {
	ownedBy uuid.UUID
	err     error
}

func (m *mockOwnership) AssertPlanOwnership(ctx context.Context, planID, userID uuid.UUID) error {
	if m.err != nil {
		return m.err
	}
	if m.ownedBy != uuid.Nil && m.ownedBy != userID {
		return constants.ErrForbidden
	}
	return nil
}

func dateUTC(y int, mo time.Month, d int) time.Time {
	return time.Date(y, mo, d, 0, 0, 0, 0, time.UTC)
}

func TestGetCalendar_MonthView_ResolvesWindowAsFirstToLast(t *testing.T) {
	repo := &mockRepository{}
	svc := NewService(repo, &mockOwnership{})

	resp, err := svc.GetCalendar(context.Background(), uuid.New(), uuid.New(), "month", "2026-03")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if resp.View != "month" {
		t.Errorf("expected view=month, got %s", resp.View)
	}
	if !repo.lastStart.Equal(dateUTC(2026, 3, 1)) {
		t.Errorf("expected window start 2026-03-01, got %s", repo.lastStart)
	}
	if !repo.lastEnd.Equal(dateUTC(2026, 3, 31)) {
		t.Errorf("expected window end 2026-03-31, got %s", repo.lastEnd)
	}
	if resp.WindowStart != "2026-03-01" || resp.WindowEnd != "2026-03-31" {
		t.Errorf("unexpected window in response: %s..%s", resp.WindowStart, resp.WindowEnd)
	}
}

func TestGetCalendar_WeekView_ResolvesSundayToSaturday(t *testing.T) {
	repo := &mockRepository{}
	svc := NewService(repo, &mockOwnership{})

	// 2026-03-09 is a Monday → week is 2026-03-08 (Sun) .. 2026-03-14 (Sat).
	resp, err := svc.GetCalendar(context.Background(), uuid.New(), uuid.New(), "week", "2026-03-09")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !repo.lastStart.Equal(dateUTC(2026, 3, 8)) {
		t.Errorf("expected week start 2026-03-08, got %s", repo.lastStart)
	}
	if !repo.lastEnd.Equal(dateUTC(2026, 3, 14)) {
		t.Errorf("expected week end 2026-03-14, got %s", repo.lastEnd)
	}
	if resp.WindowStart != "2026-03-08" || resp.WindowEnd != "2026-03-14" {
		t.Errorf("unexpected window in response: %s..%s", resp.WindowStart, resp.WindowEnd)
	}
}

func TestGetCalendar_ReturnsItemsWithFormattedDates(t *testing.T) {
	start := dateUTC(2026, 3, 4)
	due := dateUTC(2026, 3, 12)
	planID := uuid.New()
	repo := &mockRepository{
		items: []*models.ChecklistItem{
			{
				BaseDBDateModel: models.BaseDBDateModel{ID: uuid.New()},
				PlanID:          planID,
				Description:     "Build auth middleware",
				Scope:           "longterm",
				StartDate:       &start,
				DueDate:         &due,
			},
		},
	}
	svc := NewService(repo, &mockOwnership{})

	resp, err := svc.GetCalendar(context.Background(), planID, uuid.New(), "month", "2026-03")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(resp.Items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(resp.Items))
	}
	if resp.Items[0].StartDate != "2026-03-04" {
		t.Errorf("expected startDate 2026-03-04, got %s", resp.Items[0].StartDate)
	}
	if resp.Items[0].DueDate != "2026-03-12" {
		t.Errorf("expected dueDate 2026-03-12, got %s", resp.Items[0].DueDate)
	}
}

func TestGetCalendar_OwnershipFails_Returns403(t *testing.T) {
	otherUser := uuid.New()
	repo := &mockRepository{}
	svc := NewService(repo, &mockOwnership{ownedBy: otherUser})

	_, err := svc.GetCalendar(context.Background(), uuid.New(), uuid.New(), "month", "2026-03")
	if err == nil {
		t.Fatal("expected ownership error")
	}
	if err != constants.ErrForbidden {
		t.Errorf("expected ErrForbidden, got %v", err)
	}
}

func TestGetCalendar_InvalidView_ReturnsError(t *testing.T) {
	svc := NewService(&mockRepository{}, &mockOwnership{})
	if _, err := svc.GetCalendar(context.Background(), uuid.New(), uuid.New(), "year", "2026-03"); err == nil {
		t.Error("expected error for invalid view")
	}
}

func TestGetCalendar_InvalidDateFormat_ReturnsError(t *testing.T) {
	svc := NewService(&mockRepository{}, &mockOwnership{})
	if _, err := svc.GetCalendar(context.Background(), uuid.New(), uuid.New(), "month", "not-a-date"); err == nil {
		t.Error("expected error for invalid date in month view")
	}
	if _, err := svc.GetCalendar(context.Background(), uuid.New(), uuid.New(), "week", "2026-03"); err == nil {
		t.Error("expected error for month-format date in week view")
	}
}

func TestGetCalendar_DefaultsToMonth_WhenViewEmpty(t *testing.T) {
	svc := NewService(&mockRepository{}, &mockOwnership{})
	resp, err := svc.GetCalendar(context.Background(), uuid.New(), uuid.New(), "", "2026-03")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if resp.View != "month" {
		t.Errorf("expected default view=month, got %s", resp.View)
	}
}

package calendar

import (
	"context"
	"fmt"
	"time"

	"github.com/darkphotonKN/fireplace/services/api-gateway/internal/models"
	"github.com/google/uuid"
)

const dateLayout = "2006-01-02"

type CalendarRepository interface {
	GetItemsInWindow(ctx context.Context, planID uuid.UUID, windowStart, windowEnd time.Time) ([]*models.ChecklistItem, error)
}

// PlanOwnership verifies a plan belongs to a user. Implementations live in
// the plans package; the calendar package depends only on this minimal interface.
type PlanOwnership interface {
	AssertPlanOwnership(ctx context.Context, planID, userID uuid.UUID) error
}

type Service struct {
	repo      CalendarRepository
	ownership PlanOwnership
}

func NewService(repo CalendarRepository, ownership PlanOwnership) *Service {
	return &Service{repo: repo, ownership: ownership}
}

// GetCalendar resolves the requested window, verifies plan ownership,
// fetches checklist items overlapping the window, and formats the response.
func (s *Service) GetCalendar(ctx context.Context, planID, userID uuid.UUID, view, date string) (*GetCalendarResp, error) {
	if view == "" {
		view = "month"
	}
	windowStart, windowEnd, err := resolveWindow(view, date)
	if err != nil {
		return nil, err
	}

	if err := s.ownership.AssertPlanOwnership(ctx, planID, userID); err != nil {
		return nil, err
	}

	items, err := s.repo.GetItemsInWindow(ctx, planID, windowStart, windowEnd)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch calendar items: %w", err)
	}

	formatted := make([]CalendarItem, 0, len(items))
	for _, item := range items {
		formatted = append(formatted, CalendarItem{
			ID:          item.ID,
			Description: item.Description,
			Scope:       item.Scope,
			Done:        item.Done,
			StartDate:   formatDate(item.StartDate),
			DueDate:     formatDate(item.DueDate),
		})
	}

	return &GetCalendarResp{
		PlanID:      planID.String(),
		View:        view,
		WindowStart: windowStart.Format(dateLayout),
		WindowEnd:   windowEnd.Format(dateLayout),
		Items:       formatted,
	}, nil
}

func resolveWindow(view, date string) (time.Time, time.Time, error) {
	switch view {
	case "month":
		t, err := time.ParseInLocation("2006-01", date, time.UTC)
		if err != nil {
			return time.Time{}, time.Time{}, fmt.Errorf("invalid date for month view, expected YYYY-MM: %w", err)
		}
		start := t
		end := t.AddDate(0, 1, -1)
		return start, end, nil
	case "week":
		t, err := time.ParseInLocation(dateLayout, date, time.UTC)
		if err != nil {
			return time.Time{}, time.Time{}, fmt.Errorf("invalid date for week view, expected YYYY-MM-DD: %w", err)
		}
		// Sunday-start week (Sun..Sat).
		offset := int(t.Weekday()) // Sunday=0, Saturday=6
		start := t.AddDate(0, 0, -offset)
		end := start.AddDate(0, 0, 6)
		return start, end, nil
	default:
		return time.Time{}, time.Time{}, fmt.Errorf("invalid view: must be 'week' or 'month'")
	}
}

func formatDate(t *time.Time) string {
	if t == nil {
		return ""
	}
	return t.Format(dateLayout)
}

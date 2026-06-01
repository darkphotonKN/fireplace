package calendar

import (
	"context"
	"fmt"
	"time"

	planpb "github.com/darkphotonKN/fireplace/common/api/proto/plan"
	commonconstants "github.com/darkphotonKN/fireplace/common/constants"
	"github.com/google/uuid"
)

const dateLayout = "2006-01-02"

// PlanGateway is the slice of plan-service this package needs.
// Tests can substitute a fake.
type PlanGateway interface {
	AssertPlanOwnership(ctx context.Context, planID, userID uuid.UUID) error
	ListItemsInDateWindow(ctx context.Context, planID, userID uuid.UUID, windowStart, windowEnd time.Time) ([]*planpb.ChecklistItem, error)
}

type Service struct {
	plans PlanGateway
}

func NewService(plans PlanGateway) *Service {
	return &Service{plans: plans}
}

// GetCalendar resolves the requested window, verifies plan ownership against
// plan-service, fetches checklist items overlapping the window, and formats
// the response. The actual item data lives in plan-service-db; calendar
// keeps its own DB only for the future calendar_entries (slot pinning + recs).
func (s *Service) GetCalendar(ctx context.Context, planID, userID uuid.UUID, view, date string) (*GetCalendarOutput, error) {
	if view == "" {
		view = "month"
	}
	windowStart, windowEnd, err := resolveWindow(view, date)
	if err != nil {
		return nil, fmt.Errorf("calendar: get calendar: %w", err)
	}

	if err := s.plans.AssertPlanOwnership(ctx, planID, userID); err != nil {
		return nil, fmt.Errorf("calendar: get calendar: %w", err)
	}

	items, err := s.plans.ListItemsInDateWindow(ctx, planID, userID, windowStart, windowEnd)
	if err != nil {
		return nil, fmt.Errorf("calendar: get calendar: fetch items: %w", err)
	}

	formatted := make([]CalendarItem, 0, len(items))
	for _, item := range items {
		id, _ := uuid.Parse(item.Id)
		startDate := ""
		if item.StartDate != nil {
			startDate = item.StartDate.AsTime().Format(dateLayout)
		}
		dueDate := ""
		if item.DueDate != nil {
			dueDate = item.DueDate.AsTime().Format(dateLayout)
		}
		formatted = append(formatted, CalendarItem{
			ID:          id,
			Description: item.Description,
			Scope:       item.Scope,
			Done:        item.Done,
			StartDate:   startDate,
			DueDate:     dueDate,
		})
	}

	return &GetCalendarOutput{
		PlanID:      planID,
		View:        view,
		WindowStart: windowStart.Format(dateLayout),
		WindowEnd:   windowEnd.Format(dateLayout),
		Items:       formatted,
	}, nil
}

// resolveWindow translates (view, date) into a [start, end] date range.
// Ported verbatim from the monolith's calendar service.
func resolveWindow(view, date string) (time.Time, time.Time, error) {
	switch view {
	case "month":
		t, err := time.ParseInLocation("2006-01", date, time.UTC)
		if err != nil {
			return time.Time{}, time.Time{}, fmt.Errorf("%w: invalid date for month view, expected YYYY-MM: %v", commonconstants.ErrInvalidInput, err)
		}
		start := t
		end := t.AddDate(0, 1, -1)
		return start, end, nil
	case "week":
		t, err := time.ParseInLocation(dateLayout, date, time.UTC)
		if err != nil {
			return time.Time{}, time.Time{}, fmt.Errorf("%w: invalid date for week view, expected YYYY-MM-DD: %v", commonconstants.ErrInvalidInput, err)
		}
		offset := int(t.Weekday()) // Sunday=0, Saturday=6
		start := t.AddDate(0, 0, -offset)
		end := start.AddDate(0, 0, 6)
		return start, end, nil
	default:
		return time.Time{}, time.Time{}, fmt.Errorf("%w: invalid view: must be 'week' or 'month'", commonconstants.ErrInvalidInput)
	}
}

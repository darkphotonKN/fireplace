package calendargw

import (
	"context"
	"fmt"
	"time"

	planpb "github.com/darkphotonKN/fireplace/common/api/proto/plan"
	commonconstants "github.com/darkphotonKN/fireplace/common/constants"
	"github.com/google/uuid"
)

// The calendar domain runs IN-PROCESS in the gateway (ADR-0009 §1).
// calendar-service was folded back in, so there is no gRPC hop to calendar —
// though plan-service is still remote, and PlanGateway is how this reaches it.
//
// Calendar owns no data. Its former database held only calendar_entries for a
// not-yet-built slot-pinning feature; the read model is assembled entirely from
// plan-service's checklist items.

const dateLayout = "2006-01-02"

// PlanGateway is the slice of plan-service this package needs. Declared at the
// consumer so the gateway's plan client is free to grow methods nobody here
// calls, and so tests can substitute a fake.
type PlanGateway interface {
	AssertPlanOwnership(ctx context.Context, planID, userID uuid.UUID) error
	ListItemsInDateWindow(ctx context.Context, planID, userID uuid.UUID, windowStart, windowEnd time.Time) ([]*planpb.ChecklistItem, error)
}

// LocalClient satisfies CalendarClient against plan-service directly. It is the
// seam the gRPC Client used to occupy.
//
// The service-layer CalendarItem/GetCalendarOutput pair that calendar-service
// carried is gone: those types were field-identical to the transport types in
// model.go, and the handler's only job was copying one into the other. With the
// process boundary removed, that hop had nothing left to justify it.
type LocalClient struct {
	plans PlanGateway
}

func NewLocalClient(plans PlanGateway) *LocalClient {
	return &LocalClient{plans: plans}
}

// GetCalendar resolves the requested window, verifies plan ownership against
// plan-service, fetches checklist items overlapping the window, and formats the
// response.
func (c *LocalClient) GetCalendar(ctx context.Context, planID, userID uuid.UUID, view, date string) (*GetCalendarResp, error) {
	if view == "" {
		view = "month"
	}
	windowStart, windowEnd, err := resolveWindow(view, date)
	if err != nil {
		return nil, fmt.Errorf("calendar: get calendar: %w", err)
	}

	if err := c.plans.AssertPlanOwnership(ctx, planID, userID); err != nil {
		return nil, fmt.Errorf("calendar: get calendar: %w", err)
	}

	items, err := c.plans.ListItemsInDateWindow(ctx, planID, userID, windowStart, windowEnd)
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

	return &GetCalendarResp{
		PlanID:      planID.String(),
		View:        view,
		WindowStart: windowStart.Format(dateLayout),
		WindowEnd:   windowEnd.Format(dateLayout),
		Items:       formatted,
	}, nil
}

// resolveWindow translates (view, date) into a [start, end] date range.
// Ported verbatim from the monolith's calendar service, via calendar-service.
func resolveWindow(view, date string) (time.Time, time.Time, error) {
	switch view {
	case "month":
		t, err := time.ParseInLocation("2006-01", date, time.UTC)
		if err != nil {
			return time.Time{}, time.Time{}, fmt.Errorf("%w: invalid date for month view, expected YYYY-MM: %v", commonconstants.ErrInvalidInput, err)
		}
		return t, t.AddDate(0, 1, -1), nil
	case "week":
		t, err := time.ParseInLocation(dateLayout, date, time.UTC)
		if err != nil {
			return time.Time{}, time.Time{}, fmt.Errorf("%w: invalid date for week view, expected YYYY-MM-DD: %v", commonconstants.ErrInvalidInput, err)
		}
		offset := int(t.Weekday()) // Sunday=0, Saturday=6
		start := t.AddDate(0, 0, -offset)
		return start, start.AddDate(0, 0, 6), nil
	default:
		return time.Time{}, time.Time{}, fmt.Errorf("%w: invalid view: must be 'week' or 'month'", commonconstants.ErrInvalidInput)
	}
}

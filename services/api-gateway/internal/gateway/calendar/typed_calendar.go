package calendargw

import (
	"context"
	"net/http"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/darkphotonKN/fireplace/services/api-gateway/internal/apierr"
	"github.com/darkphotonKN/fireplace/services/api-gateway/internal/auth"
	"github.com/google/uuid"
)

// SERIALIZED calendar operation (FS-0004 §API surface, calendar).

// CalendarClient is the narrow seam this operation depends on.
type CalendarClient interface {
	GetCalendar(ctx context.Context, planID, userID uuid.UUID, view, date string) (*GetCalendarResp, error)
}

// --- transport mirror ------------------------------------------------------

// CalendarItemResponse mirrors CalendarItem field for field (R6).
type CalendarItemResponse struct {
	ID          uuid.UUID `json:"id"`
	Description string    `json:"description"`
	Scope       string    `json:"scope"`
	Done        bool      `json:"done"`
	StartDate   string    `json:"startDate"`
	DueDate     string    `json:"dueDate"`
}

// CalendarResponse mirrors GetCalendarResp.
type CalendarResponse struct {
	PlanID      string                 `json:"planId"`
	View        string                 `json:"view"`
	WindowStart string                 `json:"windowStart"`
	WindowEnd   string                 `json:"windowEnd"`
	Items       []CalendarItemResponse `json:"items"`
}

type CalendarOutput struct{ Body CalendarResponse }

// GetCalendarInput carries the plan id plus the window selectors.
//
// `view` is a plain string, deliberately not an enum: the legacy handler
// forwarded whatever arrived and let calendar-service decide, so constraining
// it here would turn a long-accepted request into a 422. The documented values
// live in the description instead.
type GetCalendarInput struct {
	PlanID uuid.UUID `path:"id" doc:"Plan id"`
	View   string    `query:"view" doc:"Window granularity: week or month. Unrecognised values are forwarded to calendar-service, not rejected"`
	Date   string    `query:"date" doc:"Anchor date: YYYY-MM-DD for week, YYYY-MM for month. Defaults to today's window when omitted"`
}

// resolveDate transcribes the legacy default: an absent date becomes today,
// formatted to match the requested view.
func (in *GetCalendarInput) resolveDate() string {
	if in.Date != "" {
		return in.Date
	}
	if in.View == "week" {
		return time.Now().UTC().Format("2006-01-02")
	}
	return time.Now().UTC().Format("2006-01")
}

func RegisterCalendarOperations(api huma.API, c CalendarClient,
	protect func(huma.Context, func(huma.Context)), secured []map[string][]string,
) {
	huma.Register(api, huma.Operation{
		OperationID: "getCalendar", Method: http.MethodGet,
		Path:        "/api/plans/{id}/calendar",
		Middlewares: huma.Middlewares{protect}, Security: secured,
		Summary: "Get a plan's Gantt calendar window",
		Description: "Returns the plan's scheduled checklist items for one window. " +
			"`view` selects granularity (`week` or `month`) and `date` anchors the window; " +
			"omitting `date` uses the current week or month.",
		Errors: []int{
			http.StatusUnauthorized, http.StatusForbidden, http.StatusNotFound,
			http.StatusUnprocessableEntity, http.StatusServiceUnavailable,
		},
	}, func(ctx context.Context, in *GetCalendarInput) (*CalendarOutput, error) {
		userID, ok := auth.UserIDFromCtx(ctx)
		if !ok {
			return nil, apierr.ProblemFor("calendargw: get calendar",
				apierr.ErrNoIdentity())
		}

		resp, err := c.GetCalendar(ctx, in.PlanID, userID, in.View, in.resolveDate())
		if err != nil {
			return nil, apierr.ProblemFor("calendargw: get calendar", err)
		}
		return &CalendarOutput{Body: toCalendarResponse(resp)}, nil
	})
}

// toCalendarResponse converts the client DTO to its transport mirror. Items is
// explicitly non-nil so an empty window marshals to [] and never null
// (FS-0004 §Edge States).
func toCalendarResponse(in *GetCalendarResp) CalendarResponse {
	out := CalendarResponse{
		PlanID:      in.PlanID,
		View:        in.View,
		WindowStart: in.WindowStart,
		WindowEnd:   in.WindowEnd,
		Items:       make([]CalendarItemResponse, 0, len(in.Items)),
	}
	for _, it := range in.Items {
		out.Items = append(out.Items, CalendarItemResponse{
			ID:          it.ID,
			Description: it.Description,
			Scope:       it.Scope,
			Done:        it.Done,
			StartDate:   it.StartDate,
			DueDate:     it.DueDate,
		})
	}
	return out
}

package insights

import (
	"context"
	"net/http"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/darkphotonKN/fireplace/services/api-gateway/internal/apierr"
	"github.com/darkphotonKN/fireplace/services/api-gateway/internal/auth"
	"github.com/google/uuid"
)

// Day plan operations (FS-02JBZ §API surface), served by insights-service over
// gRPC. Paths, types and fields say item and parent item, never "milestone"
// (ADR-0014, R61). New operations, so camelCase throughout (R57).

// DayPlanService is the gateway's seam to the day plan, satisfied by the
// insights-service gRPC client (internal/gateway/insights).
type DayPlanService interface {
	GetDayPlan(ctx context.Context, planID, userID uuid.UUID, date string) (DayPlanResponse, error)
	PlanMyDay(ctx context.Context, planID, userID uuid.UUID, date string) (DayPlanResponse, error)
	TickDayPlanEntry(ctx context.Context, entryID, userID uuid.UUID, itemID *uuid.UUID) (DayPlanEntryResponse, error)
	UntickDayPlanEntry(ctx context.Context, entryID, userID uuid.UUID) error
	SetItemRevisions(ctx context.Context, planID, userID, itemID uuid.UUID, enabled bool) error
}

// DayPlanResponse is FS-02JBZ's DayPlan, field for field.
type DayPlanResponse struct {
	PlanID             uuid.UUID              `json:"planId" format:"uuid" doc:"Plan id"`
	Date               string                 `json:"date" format:"date" doc:"The day plan's date as sent, YYYY-MM-DD"`
	Status             string                 `json:"status" enum:"planned,not_planned" doc:"planned once Plan my day has drawn for this date; not_planned until then"`
	Source             string                 `json:"source,omitempty" enum:"ai,rules" doc:"How the plan was drawn; present iff status is planned. rules is the basic plan"`
	GeneratedAt        *time.Time             `json:"generatedAt,omitempty" format:"date-time" doc:"When the plan was drawn; present iff status is planned, and unchanged when a redraw was served from the stored plan"`
	RefreshesRemaining int                    `json:"refreshesRemaining" minimum:"0" maximum:"3" doc:"Counted redraws left for this plan and date; 3 until the free first draw is used"`
	Groups             []DayPlanGroupResponse `json:"groups" maxItems:"3" nullable:"false" doc:"Zero to three groups, six entries at most across them; never null, empty when not_planned"`
}

// DayPlanGroupResponse is FS-02JBZ's DayPlanGroup: a label over entries.
type DayPlanGroupResponse struct {
	Label   string                 `json:"label" minLength:"1" maxLength:"40" doc:"Short topic label"`
	Entries []DayPlanEntryResponse `json:"entries" minItems:"1" maxItems:"6" nullable:"false" doc:"The group's entries, in order"`
}

// DayPlanEntryResponse is FS-02JBZ's DayPlanEntry.
type DayPlanEntryResponse struct {
	ID               uuid.UUID  `json:"id" format:"uuid" doc:"Entry id"`
	Kind             string     `json:"kind" enum:"existing_item,new_step" doc:"existing_item points at an item of the plan; new_step proposes a new child of a parent item"`
	ItemID           *uuid.UUID `json:"itemId,omitempty" format:"uuid" doc:"The item; present iff kind is existing_item"`
	ParentItemID     *uuid.UUID `json:"parentItemId,omitempty" format:"uuid" doc:"For new_step, the parent item the step goes under; for existing_item, the item's parent if it has one"`
	Text             string     `json:"text" doc:"The item's description when drawn, or the proposed step text"`
	Reason           string     `json:"reason" doc:"One line: why today"`
	Minutes          *int       `json:"minutes,omitempty" minimum:"5" maximum:"240" doc:"Rough minutes estimate; absent on rules entries"`
	Revision         *int       `json:"revision,omitempty" minimum:"1" maximum:"3" doc:"Revision number, set by the server only"`
	Ticked           bool       `json:"ticked" doc:"Day tick for this entry's date"`
	RevisionsEnabled *bool      `json:"revisionsEnabled,omitempty" doc:"Present only on existing_item entries of learning plans"`
}

type DayPlanOutput struct{ Body DayPlanResponse }

// DayPlanQueryInput: both parameters required; a malformed uuid or date is a
// shape failure (422) decided here, never forwarded (ADR-0005).
type DayPlanQueryInput struct {
	PlanID uuid.UUID `query:"planId" required:"true" format:"uuid" doc:"Plan id"`
	Date   string    `query:"date" required:"true" format:"date" doc:"The caller's local calendar date, YYYY-MM-DD. Must be within one day of the server's UTC date"`
}

// PlanMyDayRequest is planMyDay's body: writable fields only, both required;
// identity comes from the JWT, never the body (ADR-0005 §3). Unknown members
// are refused (huma's additionalProperties:false).
type PlanMyDayRequest struct {
	PlanID uuid.UUID `json:"planId" required:"true" format:"uuid" doc:"Plan id"`
	Date   string    `json:"date" required:"true" format:"date" doc:"The caller's local calendar date, YYYY-MM-DD. Must be within one day of the server's UTC date"`
}

type PlanMyDayInput struct {
	Body PlanMyDayRequest
}

// TickDayPlanEntryRequest is tickDayPlanEntry's body: writable fields only.
// itemId binds an unbound new_step entry to the item the user created from it;
// an existing_item entry takes none, or its own.
type TickDayPlanEntryRequest struct {
	ItemID *uuid.UUID `json:"itemId,omitempty" format:"uuid" doc:"The item created from a new_step entry (required to tick an unbound new_step); for an existing_item entry, omit it or send the entry's own item"`
}

// TickDayPlanEntryInput: the body is optional, so a bare PUT ticks an
// existing_item entry. A malformed id is a shape failure (422) decided here.
type TickDayPlanEntryInput struct {
	ID   uuid.UUID                `path:"id" format:"uuid" doc:"Entry id"`
	Body *TickDayPlanEntryRequest `required:"false"`
}

type DayPlanEntryOutput struct{ Body DayPlanEntryResponse }

type UntickDayPlanEntryInput struct {
	ID uuid.UUID `path:"id" format:"uuid" doc:"Entry id"`
}

// SetItemRevisionsRequest is setItemRevisions' body: writable fields only,
// both required; identity comes from the JWT, never the body (ADR-0005 §3).
type SetItemRevisionsRequest struct {
	PlanID  uuid.UUID `json:"planId" required:"true" format:"uuid" doc:"Plan id; the item must belong to it"`
	Enabled bool      `json:"enabled" required:"true" doc:"false switches the item's revisions off, cancelling its open revision; true switches them back on, and the item's next day tick schedules the next revision"`
}

// SetItemRevisionsInput: a malformed item id or body is a shape failure (422)
// decided here, never forwarded (ADR-0005).
type SetItemRevisionsInput struct {
	ItemID uuid.UUID `path:"itemId" format:"uuid" doc:"Item id"`
	Body   SetItemRevisionsRequest
}

// RegisterDayPlanOperations registers the day plan operations.
func RegisterDayPlanOperations(api huma.API, dayPlans DayPlanService, access PlanAccess,
	protect func(huma.Context, func(huma.Context)), secured []map[string][]string,
) {
	authorize := authorizer(access)

	huma.Register(api, huma.Operation{
		OperationID: "getDayPlan", Method: http.MethodGet,
		Path:        "/api/insights/day-plan",
		Middlewares: huma.Middlewares{protect}, Security: secured,
		Summary: "Get the day plan for a plan",
		Description: "Returns the caller's day plan for the plan and date: up to six entries in up to three " +
			"labelled groups, or status not_planned with no groups when Plan my day has not drawn one for " +
			"the date. A read never generates anything. A date more than one day from the server's UTC date " +
			"is refused with BRIEF_DATE_OUT_OF_RANGE.",
		Errors: []int{
			http.StatusBadRequest, http.StatusUnauthorized, http.StatusForbidden, http.StatusNotFound,
			http.StatusUnprocessableEntity, http.StatusServiceUnavailable,
		},
	}, func(ctx context.Context, in *DayPlanQueryInput) (*DayPlanOutput, error) {
		const op = "insights: get day plan"
		userID, err := authorize(ctx, op, in.PlanID)
		if err != nil {
			return nil, err
		}
		p, err := dayPlans.GetDayPlan(ctx, in.PlanID, userID, in.Date)
		return dayPlanOutput(op, p, err)
	})

	huma.Register(api, huma.Operation{
		OperationID: "planMyDay", Method: http.MethodPost,
		Path:        "/api/insights/day-plan",
		Middlewares: huma.Middlewares{protect}, Security: secured,
		Summary: "Plan my day",
		Description: "Draws or redraws the caller's day plan for the plan and date and returns it. If nothing " +
			"changed since the stored plan was drawn, the stored plan is returned and nothing is spent. The " +
			"first successful draw of a date is free; each later one spends one of 3 redraws per plan and " +
			"date (refreshesRemaining), and with none left it is " +
			"refused with 429 BRIEF_REFRESH_LIMIT_REACHED and the stored plan is unchanged. A date more than " +
			"one day from the server's UTC date is refused with BRIEF_DATE_OUT_OF_RANGE. If generation fails " +
			"(the platform's daily token budget is spent, the model is unreachable or times out, or its " +
			"output is invalid), a rule-based plan is drawn instead and returned with 200 and source rules; " +
			"it is stored, spends nothing and leaves the free draw unused, and the next draw tries the model " +
			"again. 503 SERVICE_UNAVAILABLE means the insights service could not be reached; retrying is correct.",
		Errors: []int{
			http.StatusBadRequest, http.StatusUnauthorized, http.StatusForbidden, http.StatusNotFound,
			http.StatusUnprocessableEntity, http.StatusTooManyRequests, http.StatusServiceUnavailable,
		},
	}, func(ctx context.Context, in *PlanMyDayInput) (*DayPlanOutput, error) {
		const op = "insights: plan my day"
		userID, err := authorize(ctx, op, in.Body.PlanID)
		if err != nil {
			return nil, err
		}
		p, err := dayPlans.PlanMyDay(ctx, in.Body.PlanID, userID, in.Body.Date)
		return dayPlanOutput(op, p, err)
	})

	// The day tick is entry-scoped: insights-service answers 404 for an entry
	// that is not the caller's, and checks the item against plan-service as
	// the caller, so no plan id is needed here.
	huma.Register(api, huma.Operation{
		OperationID: "tickDayPlanEntry", Method: http.MethodPut,
		Path:        "/api/insights/day-plan/entries/{id}/tick",
		Middlewares: huma.Middlewares{protect}, Security: secured,
		Summary: "Tick a day plan entry done for today",
		Description: "Marks the entry's item done for the day plan's date and returns the entry, ticked. This is " +
			"not the item's done, which it never changes; ticking the item itself done is a separate item " +
			"update. An existing_item entry takes no itemId, or its own. An unbound new_step entry needs " +
			"itemId, the item created from it, which must be a task under the entry's parent item in the same " +
			"plan; the entry is then bound to that item (kind existing_item, text and reason kept). Ticking a " +
			"ticked entry returns it unchanged. An entry that cannot be ticked (an unbound new_step without a " +
			"valid itemId, an item now archived or deleted, or an itemId that contradicts the entry's own) is " +
			"refused with DAY_PLAN_ENTRY_NOT_TICKABLE; a day plan date more than one day from the server's UTC " +
			"date with BRIEF_DATE_OUT_OF_RANGE. An entry that is not the caller's is 404.",
		Errors: []int{
			http.StatusBadRequest, http.StatusUnauthorized, http.StatusForbidden, http.StatusNotFound,
			http.StatusUnprocessableEntity, http.StatusServiceUnavailable,
		},
	}, func(ctx context.Context, in *TickDayPlanEntryInput) (*DayPlanEntryOutput, error) {
		const op = "insights: tick day plan entry"
		userID, ok := auth.UserIDFromCtx(ctx)
		if !ok {
			return nil, apierr.ProblemFor(op, apierr.ErrNoIdentity())
		}
		var itemID *uuid.UUID
		if in.Body != nil {
			itemID = in.Body.ItemID
		}
		e, err := dayPlans.TickDayPlanEntry(ctx, in.ID, userID, itemID)
		if err != nil {
			return nil, apierr.ProblemFor(op, err)
		}
		return &DayPlanEntryOutput{Body: e}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "untickDayPlanEntry", Method: http.MethodDelete,
		Path:        "/api/insights/day-plan/entries/{id}/tick",
		Middlewares: huma.Middlewares{protect}, Security: secured,
		DefaultStatus: http.StatusNoContent,
		Summary:       "Remove a day plan entry's tick",
		Description: "Removes the entry's day tick for the day plan's date. Unticking an unticked entry succeeds " +
			"and changes nothing. A day plan date more than one day from the server's UTC date is refused with " +
			"BRIEF_DATE_OUT_OF_RANGE. An entry that is not the caller's is 404.",
		Errors: []int{
			http.StatusBadRequest, http.StatusUnauthorized, http.StatusForbidden, http.StatusNotFound,
			http.StatusUnprocessableEntity, http.StatusServiceUnavailable,
		},
	}, func(ctx context.Context, in *UntickDayPlanEntryInput) (*struct{}, error) {
		const op = "insights: untick day plan entry"
		userID, ok := auth.UserIDFromCtx(ctx)
		if !ok {
			return nil, apierr.ProblemFor(op, apierr.ErrNoIdentity())
		}
		if err := dayPlans.UntickDayPlanEntry(ctx, in.ID, userID); err != nil {
			return nil, apierr.ProblemFor(op, err)
		}
		return &struct{}{}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "setItemRevisions", Method: http.MethodPut,
		Path:        "/api/insights/day-plan/items/{itemId}/revisions",
		Middlewares: huma.Middlewares{protect}, Security: secured,
		DefaultStatus: http.StatusNoContent,
		Summary:       "Switch revisions for an item",
		Description: "Switches the revision ladder on or off for one item of the caller's plan; on by default, " +
			"and per user on a shared plan. Off cancels the item's open revision and schedules no more. On " +
			"schedules nothing by itself: the item's next day tick schedules the next revision, never past " +
			"revision 3. Only learning plans revise; on a project plan the switch is accepted and has no " +
			"effect. A plan the caller cannot access, or an item not in the plan, is 404.",
		Errors: []int{
			http.StatusUnauthorized, http.StatusForbidden, http.StatusNotFound,
			http.StatusUnprocessableEntity, http.StatusServiceUnavailable,
		},
	}, func(ctx context.Context, in *SetItemRevisionsInput) (*struct{}, error) {
		const op = "insights: set item revisions"
		userID, err := authorize(ctx, op, in.Body.PlanID)
		if err != nil {
			return nil, err
		}
		if err := dayPlans.SetItemRevisions(ctx, in.Body.PlanID, userID, in.ItemID, in.Body.Enabled); err != nil {
			return nil, apierr.ProblemFor(op, err)
		}
		return &struct{}{}, nil
	})
}

// dayPlanOutput publishes a day plan from insights-service, or its failure as
// problem+json. Groups are never null on the wire.
func dayPlanOutput(op string, p DayPlanResponse, err error) (*DayPlanOutput, error) {
	if err != nil {
		return nil, apierr.ProblemFor(op, err)
	}
	if p.Groups == nil {
		p.Groups = []DayPlanGroupResponse{}
	}
	return &DayPlanOutput{Body: p}, nil
}

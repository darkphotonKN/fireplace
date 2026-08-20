package plangw

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/darkphotonKN/fireplace/services/api-gateway/internal/apierr"
	"github.com/darkphotonKN/fireplace/services/api-gateway/internal/auth"
	"github.com/google/uuid"
)

// SERIALIZED checklist operations (FS-0004 §API surface, checklists).
//
// Every domain rule these endpoints obey — two-tier nesting, task→note
// conversion blocked when an item has children, startDate before dueDate — is
// enforced by plan-service and described in prose on the request types. None of
// it is encoded as a schema constraint: the contract is a shape agreement, not
// a validator (ADR-0005).

// ChecklistsClient is the narrow seam these operations depend on.
type ChecklistsClient interface {
	CreateChecklist(ctx context.Context, planID, userID uuid.UUID, req CreateChecklistReq) (*ChecklistResp, error)
	GetChecklist(ctx context.Context, id, userID uuid.UUID) (*ChecklistResp, error)
	ListChecklists(ctx context.Context, planID, userID uuid.UUID, scope, itemType *string) ([]*ChecklistResp, error)
	ListArchivedChecklists(ctx context.Context, planID, userID uuid.UUID) ([]*ChecklistResp, error)
	ListUpcomingChecklists(ctx context.Context, planID, userID uuid.UUID) ([]*ChecklistResp, error)
	UpdateChecklist(ctx context.Context, id, userID uuid.UUID, req UpdateChecklistReq) (*ChecklistResp, error)
	UpdateChecklistDates(ctx context.Context, id, userID uuid.UUID, req UpdateDatesReq) (*ChecklistResp, error)
	ArchiveChecklist(ctx context.Context, id, userID uuid.UUID, archived bool) (*ChecklistResp, error)
	DeleteChecklist(ctx context.Context, id, userID uuid.UUID) error
}

// --- huma input/output wrappers -------------------------------------------

type ChecklistOutput struct{ Body ChecklistResp }
type ChecklistListOutput struct{ Body []ChecklistResp }

// PlanScopedInput is the path shape shared by the list operations.
type PlanScopedInput struct {
	PlanID uuid.UUID `path:"id" doc:"Plan id"`
}

// ListChecklistsInput adds the two optional filters.
//
// Plain strings, not pointers, for two independent reasons. Huma panics on a
// pointer query parameter outright. And the distinction a pointer would buy —
// absent versus explicitly empty — does not exist in this endpoint's behaviour:
// the legacy handler forwarded a filter only `if v := c.Query("scope"); v != ""`,
// so `?scope=` and omitting scope have always meant the same thing. Preserving
// a distinction the API never made would be inventing behaviour, not
// transcribing it.
type ListChecklistsInput struct {
	PlanID uuid.UUID `path:"id" doc:"Plan id"`
	Scope  string    `query:"scope" enum:"daily,longterm" doc:"Filter by scope; omit (or send empty) for all"`
	Type   string    `query:"type" enum:"task,note" doc:"Filter by item type; omit (or send empty) for all"`
}

// ChecklistItemInput carries both path params. The plan id is part of the URL
// even where the handler does not need it — the item id is globally unique —
// because the path expresses ownership and the legacy routes were shaped that
// way.
type ChecklistItemInput struct {
	PlanID      uuid.UUID `path:"id" doc:"Plan id"`
	ChecklistID uuid.UUID `path:"checklist_id" doc:"Checklist item id"`
}

type CreateChecklistInput struct {
	PlanID uuid.UUID `path:"id" doc:"Plan id"`
	Body   CreateChecklistReq
}

type UpdateChecklistInput struct {
	PlanID      uuid.UUID `path:"id" doc:"Plan id"`
	ChecklistID uuid.UUID `path:"checklist_id" doc:"Checklist item id"`
	Body        UpdateChecklistReq
}

type UpdateDatesInput struct {
	PlanID      uuid.UUID `path:"id" doc:"Plan id"`
	ChecklistID uuid.UUID `path:"checklist_id" doc:"Checklist item id"`
	Body        UpdateDatesReq
}

type ArchiveChecklistInput struct {
	PlanID      uuid.UUID `path:"id" doc:"Plan id"`
	ChecklistID uuid.UUID `path:"checklist_id" doc:"Checklist item id"`
	Body        ArchiveReq
}

func RegisterChecklistOperations(api huma.API, c ChecklistsClient,
	protect func(huma.Context, func(huma.Context)), secured []map[string][]string,
) {
	mw := huma.Middlewares{protect}
	errs := []int{
		http.StatusUnauthorized, http.StatusForbidden, http.StatusNotFound,
		http.StatusUnprocessableEntity, http.StatusServiceUnavailable,
	}
	writeErrs := append([]int{http.StatusBadRequest}, errs...)

	// identity resolves the caller once per handler. Unreachable behind the auth
	// middleware, but it must still produce a contract-shaped error rather than
	// a nil-pointer panic if the bridge ever yields nothing.
	identity := func(ctx context.Context, op string) (uuid.UUID, error) {
		userID, ok := auth.UserIDFromCtx(ctx)
		if !ok {
			return uuid.Nil, apierr.ProblemFor(op, apierr.ErrNoIdentity())
		}
		return userID, nil
	}

	huma.Register(api, huma.Operation{
		OperationID: "listChecklists", Method: http.MethodGet,
		Path:        "/api/plans/{id}/checklists",
		Middlewares: mw, Security: secured,
		Summary: "List a plan's checklist items",
		Description: "Returns non-archived items. `scope` and `type` are optional filters. " +
			"Omitting a filter and sending it empty are equivalent — both return every value.",
		Errors: errs,
	}, func(ctx context.Context, in *ListChecklistsInput) (*ChecklistListOutput, error) {
		userID, err := identity(ctx, "list checklist items")
		if err != nil {
			return nil, err
		}
		items, err := c.ListChecklists(ctx, in.PlanID, userID, optFilter(in.Scope), optFilter(in.Type))
		if err != nil {
			return nil, apierr.ProblemFor("list checklist items", err)
		}
		return &ChecklistListOutput{Body: derefChecklists(items)}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "listArchivedChecklists", Method: http.MethodGet,
		Path:        "/api/plans/{id}/checklists/archived",
		Middlewares: mw, Security: secured,
		Summary:     "List a plan's archived checklist items",
		Description: "Returns only archived items.",
		Errors:      errs,
	}, func(ctx context.Context, in *PlanScopedInput) (*ChecklistListOutput, error) {
		userID, err := identity(ctx, "list archived checklist items")
		if err != nil {
			return nil, err
		}
		items, err := c.ListArchivedChecklists(ctx, in.PlanID, userID)
		if err != nil {
			return nil, apierr.ProblemFor("list archived checklist items", err)
		}
		return &ChecklistListOutput{Body: derefChecklists(items)}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "listUpcomingChecklists", Method: http.MethodGet,
		Path:        "/api/plans/{id}/checklists/upcoming",
		Middlewares: mw, Security: secured,
		Summary:     "List a plan's upcoming checklist items",
		Description: "Returns items whose start date falls within the next week.",
		Errors:      errs,
	}, func(ctx context.Context, in *PlanScopedInput) (*ChecklistListOutput, error) {
		userID, err := identity(ctx, "list upcoming checklist items")
		if err != nil {
			return nil, err
		}
		items, err := c.ListUpcomingChecklists(ctx, in.PlanID, userID)
		if err != nil {
			return nil, apierr.ProblemFor("list upcoming checklist items", err)
		}
		return &ChecklistListOutput{Body: derefChecklists(items)}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "getChecklist", Method: http.MethodGet,
		Path:        "/api/plans/{id}/checklists/{checklist_id}",
		Middlewares: mw, Security: secured,
		Summary:     "Get a checklist item",
		Description: "Returns one checklist item by id.",
		Errors:      errs,
	}, func(ctx context.Context, in *ChecklistItemInput) (*ChecklistOutput, error) {
		userID, err := identity(ctx, "get checklist item")
		if err != nil {
			return nil, err
		}
		item, err := c.GetChecklist(ctx, in.ChecklistID, userID)
		if err != nil {
			return nil, apierr.ProblemFor("get checklist item", err)
		}
		return &ChecklistOutput{Body: *item}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "createChecklist", Method: http.MethodPost,
		Path:          "/api/plans/{id}/checklists",
		DefaultStatus: http.StatusCreated,
		Middlewares:   mw, Security: secured,
		Summary: "Create a checklist item",
		Description: "Creates a task or note under the plan. `scope` and `type` are optional; " +
			"plan-service defaults them to longterm and task. Parent nesting is validated " +
			"downstream — a parent must be a top-level item in the same plan.",
		Errors: writeErrs,
	}, func(ctx context.Context, in *CreateChecklistInput) (*ChecklistOutput, error) {
		userID, err := identity(ctx, "create checklist item")
		if err != nil {
			return nil, err
		}
		item, err := c.CreateChecklist(ctx, in.PlanID, userID, in.Body)
		if err != nil {
			return nil, apierr.ProblemFor("create checklist item", err)
		}
		return &ChecklistOutput{Body: *item}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "updateChecklist", Method: http.MethodPatch,
		Path:        "/api/plans/{id}/checklists/{checklist_id}",
		Middlewares: mw, Security: secured,
		Summary: "Update a checklist item",
		Description: "Partial update. `parentId` is three-state: send a UUID to re-parent, " +
			"null to clear, or omit to leave unchanged. Conversion and re-parenting rules are " +
			"enforced downstream.",
		Errors: writeErrs,
	}, func(ctx context.Context, in *UpdateChecklistInput) (*ChecklistOutput, error) {
		userID, err := identity(ctx, "update checklist item")
		if err != nil {
			return nil, err
		}
		item, err := c.UpdateChecklist(ctx, in.ChecklistID, userID, in.Body)
		if err != nil {
			return nil, apierr.ProblemFor("update checklist item", err)
		}
		return &ChecklistOutput{Body: *item}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "updateChecklistDates", Method: http.MethodPatch,
		Path:        "/api/plans/{id}/checklists/{checklist_id}/dates",
		Middlewares: mw, Security: secured,
		Summary: "Set or clear a checklist item's dates",
		Description: "Both dates are three-state: send \"YYYY-MM-DD\" to set, null to clear, " +
			"or omit to leave unchanged. When both are present, startDate must be on or before " +
			"dueDate — enforced downstream, not by this schema.",
		Errors: writeErrs,
	}, func(ctx context.Context, in *UpdateDatesInput) (*ChecklistOutput, error) {
		userID, err := identity(ctx, "update checklist dates")
		if err != nil {
			return nil, err
		}
		item, err := c.UpdateChecklistDates(ctx, in.ChecklistID, userID, in.Body)
		if err != nil {
			return nil, apierr.ProblemFor("update checklist dates", err)
		}
		return &ChecklistOutput{Body: *item}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "archiveChecklist", Method: http.MethodPatch,
		Path:        "/api/plans/{id}/checklists/{checklist_id}/archive",
		Middlewares: mw, Security: secured,
		Summary: "Archive or unarchive a checklist item",
		Description: "Takes a body with `archived`. It is a SETTER, not a toggle — sending " +
			"false unarchives.",
		Errors: writeErrs,
	}, func(ctx context.Context, in *ArchiveChecklistInput) (*ChecklistOutput, error) {
		userID, err := identity(ctx, "archive checklist item")
		if err != nil {
			return nil, err
		}
		item, err := c.ArchiveChecklist(ctx, in.ChecklistID, userID, in.Body.Archived)
		if err != nil {
			return nil, apierr.ProblemFor("archive checklist item", err)
		}
		return &ChecklistOutput{Body: *item}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "deleteChecklist", Method: http.MethodDelete,
		Path:          "/api/plans/{id}/checklists/{checklist_id}",
		DefaultStatus: http.StatusNoContent,
		Middlewares:   mw, Security: secured,
		Summary:     "Delete a checklist item",
		Description: "Deletes the item. Answers 204 with no body.",
		Errors:      errs,
	}, func(ctx context.Context, in *ChecklistItemInput) (*DeleteOutput, error) {
		userID, err := identity(ctx, "delete checklist item")
		if err != nil {
			return nil, err
		}
		if err := c.DeleteChecklist(ctx, in.ChecklistID, userID); err != nil {
			return nil, apierr.ProblemFor("delete checklist item", err)
		}
		return &DeleteOutput{}, nil
	})
}

// derefChecklists copies to values and is explicitly non-nil, so an empty
// result marshals to [] and never null (FS-0004 §Edge States).
func derefChecklists(in []*ChecklistResp) []ChecklistResp {
	out := make([]ChecklistResp, 0, len(in))
	for _, i := range in {
		if i != nil {
			out = append(out, *i)
		}
	}
	return out
}

// optFilter converts an empty filter to nil, which is exactly what the legacy
// handler did: a filter was forwarded downstream only when the query value was
// non-empty.
func optFilter(v string) *string {
	if v == "" {
		return nil
	}
	return &v
}

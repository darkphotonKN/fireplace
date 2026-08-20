package notes

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/danielgtaylor/huma/v2"
	commonconstants "github.com/darkphotonKN/fireplace/common/constants"
	"github.com/darkphotonKN/fireplace/services/api-gateway/internal/apierr"
	"github.com/darkphotonKN/fireplace/services/api-gateway/internal/auth"
	"github.com/google/uuid"
)

// SERIALIZED notes operations (FS-0004 §API surface, notes).
//
// Notes is a gateway-LOCAL domain: it is backed by this service's own `notes`
// table rather than a downstream service, so the transport mirror below is
// transcribed from model.go rather than from a protobuf message. The domain
// type `Note` never appears in the generated schema (R5).

// PlanOwnership answers "may this caller act on this plan?".
//
// plan-service deliberately does NOT enforce ownership on direct reads — it
// treats the gateway as a trusted caller and expects the gateway to assert
// ownership where it matters. Notes never asserted it, so every notes endpoint
// was once reachable by any authenticated user for any plan id.
type PlanOwnership interface {
	AssertPlanOwnership(ctx context.Context, planID, userID uuid.UUID) error
}

// NotesService is the narrow seam these operations depend on. *Service
// satisfies it; tests substitute a recorder.
type NotesService interface {
	CreateNote(planID uuid.UUID, req *CreateNoteReq) (*Note, error)
	GetNoteByID(id uuid.UUID) (*Note, error)
	GetNotesByPlanID(planID uuid.UUID, filters *FilterOptions) ([]Note, error)
	UpdateNote(id uuid.UUID, updates *UpdateNoteReq) (*Note, error)
	DeleteNote(id uuid.UUID) error
	GenerateAINotes(planID uuid.UUID, requestType string) ([]Note, error)
}

// --- transport mirror ------------------------------------------------------

// AIMetadataPayload mirrors AIMetadata. The column is JSONB, but its contents
// have a known shape, so it is declared as an object rather than left free-form.
// Used by both the response and the create request — same shape, one schema.
type AIMetadataPayload struct {
	GeneratedFrom string  `json:"generatedFrom"`
	Confidence    float64 `json:"confidence"`
	SourceContext string  `json:"sourceContext"`
	GeneratedAt   string  `json:"generatedAt"`
}

// NoteResponse mirrors Note field for field, including the camelCase names and
// the single `omitempty` (R6). Field names are transcribed, not normalized:
// notes uses camelCase filters while insights uses snake_case, and both stay.
type NoteResponse struct {
	ID             uuid.UUID          `json:"id"`
	PlanID         uuid.UUID          `json:"planId"`
	Content        string             `json:"content"`
	Type           string             `json:"type"`
	Priority       string             `json:"priority"`
	Tags           []string           `json:"tags"`
	RelatedTaskIDs []string           `json:"relatedTaskIds"`
	IsRead         bool               `json:"isRead"`
	IsDismissed    bool               `json:"isDismissed"`
	AIMetadata     *AIMetadataPayload `json:"aiMetadata,omitempty"`
	CreatedAt      time.Time          `json:"createdAt"`
	UpdatedAt      time.Time          `json:"updatedAt"`
}

// toNoteResponse converts one domain note to its transport mirror.
func toNoteResponse(n Note) NoteResponse {
	out := NoteResponse{
		ID:             n.ID,
		PlanID:         n.PlanID,
		Content:        n.Content,
		Type:           n.Type,
		Priority:       n.Priority,
		Tags:           n.Tags,
		RelatedTaskIDs: n.RelatedTaskIDs,
		IsRead:         n.IsRead,
		IsDismissed:    n.IsDismissed,
		CreatedAt:      n.CreatedAt,
		UpdatedAt:      n.UpdatedAt,
	}
	if n.AIMetadata != nil {
		out.AIMetadata = &AIMetadataPayload{
			GeneratedFrom: n.AIMetadata.GeneratedFrom,
			Confidence:    n.AIMetadata.Confidence,
			SourceContext: n.AIMetadata.SourceContext,
			GeneratedAt:   n.AIMetadata.GeneratedAt,
		}
	}
	return out
}

// toNoteResponses is explicitly non-nil, so an empty result marshals to [] and
// never null (FS-0004 §Edge States).
func toNoteResponses(in []Note) []NoteResponse {
	out := make([]NoteResponse, 0, len(in))
	for _, n := range in {
		out = append(out, toNoteResponse(n))
	}
	return out
}

// CreateNoteRequest mirrors CreateNoteReq. `content` carries no omitempty, so
// huma marks it required — matching today's `binding:"required"`.
type CreateNoteRequest struct {
	Content        string             `json:"content" doc:"Note body"`
	Type           string             `json:"type,omitempty" doc:"user | ai | warning | insight | suggestion"`
	Priority       string             `json:"priority,omitempty" doc:"low | medium | high | critical"`
	Tags           []string           `json:"tags,omitempty" doc:"Free-form tags"`
	RelatedTaskIDs []string           `json:"relatedTaskIds,omitempty" doc:"Checklist item ids this note refers to"`
	AIMetadata     *AIMetadataPayload `json:"aiMetadata,omitempty" doc:"Present on AI-generated notes"`
}

// UpdateNoteRequest mirrors UpdateNoteReq. Every field is optional and pointer-
// valued so an omitted field is distinguishable from an explicit zero value.
// Pointers are legal here: huma only rejects them on QUERY parameters.
type UpdateNoteRequest struct {
	Content     *string  `json:"content,omitempty" doc:"Replaces the note body"`
	Priority    *string  `json:"priority,omitempty" doc:"low | medium | high | critical"`
	Tags        []string `json:"tags,omitempty" doc:"Replaces the tag set"`
	IsRead      *bool    `json:"isRead,omitempty" doc:"Marks the note read or unread"`
	IsDismissed *bool    `json:"isDismissed,omitempty" doc:"Marks the note dismissed or not"`
}

// GenerateAINotesRequest mirrors GenerateAINotesReq. requestType is optional:
// the legacy handler defaulted it to "all" whenever the body was absent or
// unparseable, and an empty value keeps meaning "all".
type GenerateAINotesRequest struct {
	RequestType string `json:"requestType,omitempty" doc:"suggestion | warning | insight; empty or omitted means all"`
}

func (r *CreateNoteRequest) toDomain() *CreateNoteReq {
	out := &CreateNoteReq{
		Content:        r.Content,
		Type:           r.Type,
		Priority:       r.Priority,
		Tags:           r.Tags,
		RelatedTaskIDs: r.RelatedTaskIDs,
	}
	if r.AIMetadata != nil {
		out.AIMetadata = &AIMetadata{
			GeneratedFrom: r.AIMetadata.GeneratedFrom,
			Confidence:    r.AIMetadata.Confidence,
			SourceContext: r.AIMetadata.SourceContext,
			GeneratedAt:   r.AIMetadata.GeneratedAt,
		}
	}
	return out
}

func (r *UpdateNoteRequest) toDomain() *UpdateNoteReq {
	return &UpdateNoteReq{
		Content:     r.Content,
		Priority:    r.Priority,
		Tags:        r.Tags,
		IsRead:      r.IsRead,
		IsDismissed: r.IsDismissed,
	}
}

// --- huma input/output wrappers -------------------------------------------

type NoteOutput struct{ Body NoteResponse }

// NoteIDInput carries both path params. The plan id is part of the URL even
// though notes are looked up by note id alone — the path expresses ownership,
// and it is what the second authorization stage checks against.
type NoteIDInput struct {
	PlanID uuid.UUID `path:"id" doc:"Plan id"`
	NoteID uuid.UUID `path:"noteId" doc:"Note id"`
}

type CreateNoteInput struct {
	PlanID uuid.UUID `path:"id" doc:"Plan id"`
	Body   CreateNoteRequest
}

type UpdateNoteInput struct {
	PlanID uuid.UUID `path:"id" doc:"Plan id"`
	NoteID uuid.UUID `path:"noteId" doc:"Note id"`
	Body   UpdateNoteRequest
}

type GenerateAINotesInput struct {
	PlanID uuid.UUID `path:"id" doc:"Plan id"`
	Body   GenerateAINotesRequest
}
type NoteListOutput struct{ Body []NoteResponse }
type DeleteNoteOutput struct{}

// ListNotesInput carries the plan id plus the six filters.
//
// isRead/isDismissed are STRINGS, not *bool, for two reasons. Huma panics on a
// pointer query parameter outright. And the legacy parse is
// `if v := c.Query("isRead"); v != "" { b := v == "true" }` — so presence is
// "non-empty" and truth is "equals true", which means `?isRead=xyz` has always
// meant false rather than absent. An enum would turn that long-accepted request
// into a 422; a bool would lose the absent/false distinction entirely.
type ListNotesInput struct {
	PlanID        uuid.UUID `path:"id" doc:"Plan id"`
	Type          string    `query:"type" doc:"Filter by note type; omit (or send empty) for all"`
	Priority      string    `query:"priority" doc:"Filter by priority; omit (or send empty) for all"`
	IsRead        string    `query:"isRead" doc:"Filter by read state. Any value other than 'true' reads as false; omit for no filter"`
	IsDismissed   string    `query:"isDismissed" doc:"Filter by dismissed state. Any value other than 'true' reads as false; omit for no filter"`
	RelatedTaskID string    `query:"relatedTaskId" doc:"Filter to notes referencing this task id"`
	Tags          []string  `query:"tags,explode" doc:"Repeatable. ?tags=a&tags=b filters on both"`
}

// filters transcribes the legacy query parsing exactly.
func (in *ListNotesInput) filters() *FilterOptions {
	f := &FilterOptions{}
	if in.Type != "" {
		f.Type = in.Type
	}
	if in.Priority != "" {
		f.Priority = in.Priority
	}
	if in.IsRead != "" {
		isRead := in.IsRead == "true"
		f.IsRead = &isRead
	}
	if in.IsDismissed != "" {
		isDismissed := in.IsDismissed == "true"
		f.IsDismissed = &isDismissed
	}
	if in.RelatedTaskID != "" {
		f.RelatedTaskID = in.RelatedTaskID
	}
	// The legacy handler guarded the whole tag filter on c.Query("tags"), which
	// returns only the FIRST value — so `?tags=&tags=b` applied no tag filter at
	// all. Transcribed, not fixed (R6).
	if len(in.Tags) > 0 && in.Tags[0] != "" {
		tags := []string{}
		for _, tag := range in.Tags {
			if tag != "" {
				tags = append(tags, tag)
			}
		}
		if len(tags) > 0 {
			f.Tags = tags
		}
	}
	return f
}

// RegisterNotesOperations registers the serialized notes surface. Handlers may
// be nil: registration never invokes them.
func RegisterNotesOperations(api huma.API, svc NotesService, ownership PlanOwnership,
	protect func(huma.Context, func(huma.Context)), secured []map[string][]string,
) {
	mw := huma.Middlewares{protect}
	errs := []int{
		http.StatusUnauthorized, http.StatusNotFound,
		http.StatusUnprocessableEntity, http.StatusServiceUnavailable,
	}
	writeErrs := append([]int{http.StatusBadRequest}, errs...)

	// authorize resolves the caller, then asserts they may act on the plan.
	// A nil ownership checker fails CLOSED — the failure mode of an
	// authorization seam must never be "allow".
	authorize := func(ctx context.Context, op string, planID uuid.UUID) error {
		userID, ok := auth.UserIDFromCtx(ctx)
		if !ok {
			return apierr.ProblemFor("notes: "+op, apierr.ErrNoIdentity())
		}
		if ownership == nil {
			return apierr.ProblemFor("notes: "+op,
				fmt.Errorf("%w: ownership checker not configured", commonconstants.ErrForbidden))
		}
		if err := ownership.AssertPlanOwnership(ctx, planID, userID); err != nil {
			return apierr.ProblemFor("notes: "+op, err)
		}
		return nil
	}

	huma.Register(api, huma.Operation{
		OperationID: "listNotes", Method: http.MethodGet,
		Path:        "/api/plans/{id}/notes",
		Middlewares: mw, Security: secured,
		Summary: "List a plan's notes",
		Description: "Returns the plan's notes, optionally filtered. `tags` is repeatable — " +
			"`?tags=a&tags=b` filters on both. Omitting a filter and sending it empty are " +
			"equivalent. For `isRead`/`isDismissed`, any value other than `true` reads as false.",
		Errors: errs,
	}, func(ctx context.Context, in *ListNotesInput) (*NoteListOutput, error) {
		if err := authorize(ctx, "list", in.PlanID); err != nil {
			return nil, err
		}
		found, err := svc.GetNotesByPlanID(in.PlanID, in.filters())
		if err != nil {
			return nil, apierr.ProblemFor("notes: list", err)
		}
		return &NoteListOutput{Body: toNoteResponses(found)}, nil
	})

	// authorizeNote authorizes the PLAN in the path, then confirms the note
	// actually belongs to it.
	//
	// The plan check alone is not enough: these operations are keyed by note id
	// and the service looks a note up by that id ALONE, so without the second
	// stage a caller who owns any plan could read or delete a note in someone
	// else's plan just by putting their own plan id in the path. Ownership of
	// the container does not imply ownership of an arbitrary claimed item.
	authorizeNote := func(ctx context.Context, op string, planID, noteID uuid.UUID) error {
		if err := authorize(ctx, op, planID); err != nil {
			return err
		}
		note, err := svc.GetNoteByID(noteID)
		if err != nil {
			return apierr.ProblemFor("notes: "+op, err)
		}
		if note == nil || note.PlanID != planID {
			// Not found rather than forbidden: saying "wrong plan" would confirm
			// the note exists.
			return apierr.ProblemFor("notes: "+op,
				fmt.Errorf("%w: note %s is not in plan %s", commonconstants.ErrNotFound, noteID, planID))
		}
		return nil
	}

	huma.Register(api, huma.Operation{
		OperationID: "getNote", Method: http.MethodGet,
		Path:        "/api/plans/{id}/notes/{noteId}",
		Middlewares: mw, Security: secured,
		Summary:     "Get a note",
		Description: "Returns one note. Answers 404 when the note is not in the given plan, whether or not it exists elsewhere.",
		Errors:      errs,
	}, func(ctx context.Context, in *NoteIDInput) (*NoteOutput, error) {
		if err := authorizeNote(ctx, "get", in.PlanID, in.NoteID); err != nil {
			return nil, err
		}
		note, err := svc.GetNoteByID(in.NoteID)
		if err != nil {
			return nil, apierr.ProblemFor("notes: get", err)
		}
		return &NoteOutput{Body: toNoteResponse(*note)}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "createNote", Method: http.MethodPost,
		Path:          "/api/plans/{id}/notes",
		DefaultStatus: http.StatusCreated,
		Middlewares:   mw, Security: secured,
		Summary:     "Create a note",
		Description: "Creates a note on the plan and returns it. Answers 201.",
		Errors:      writeErrs,
	}, func(ctx context.Context, in *CreateNoteInput) (*NoteOutput, error) {
		if err := authorize(ctx, "create", in.PlanID); err != nil {
			return nil, err
		}
		note, err := svc.CreateNote(in.PlanID, in.Body.toDomain())
		if err != nil {
			return nil, apierr.ProblemFor("notes: create", err)
		}
		return &NoteOutput{Body: toNoteResponse(*note)}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "updateNote", Method: http.MethodPatch,
		Path:        "/api/plans/{id}/notes/{noteId}",
		Middlewares: mw, Security: secured,
		Summary:     "Update a note",
		Description: "Applies a partial update. Omitted fields are left unchanged.",
		Errors:      writeErrs,
	}, func(ctx context.Context, in *UpdateNoteInput) (*NoteOutput, error) {
		if err := authorizeNote(ctx, "update", in.PlanID, in.NoteID); err != nil {
			return nil, err
		}
		note, err := svc.UpdateNote(in.NoteID, in.Body.toDomain())
		if err != nil {
			return nil, apierr.ProblemFor("notes: update", err)
		}
		return &NoteOutput{Body: toNoteResponse(*note)}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "deleteNote", Method: http.MethodDelete,
		Path:          "/api/plans/{id}/notes/{noteId}",
		DefaultStatus: http.StatusNoContent,
		Middlewares:   mw, Security: secured,
		Summary:     "Delete a note",
		Description: "Deletes a note. Answers 204 with no body.",
		Errors:      errs,
	}, func(ctx context.Context, in *NoteIDInput) (*DeleteNoteOutput, error) {
		if err := authorizeNote(ctx, "delete", in.PlanID, in.NoteID); err != nil {
			return nil, err
		}
		if err := svc.DeleteNote(in.NoteID); err != nil {
			return nil, apierr.ProblemFor("notes: delete", err)
		}
		return &DeleteNoteOutput{}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "generateAINotes", Method: http.MethodPost,
		Path:        "/api/plans/{id}/notes/generate-ai",
		Middlewares: mw, Security: secured,
		Summary: "Generate AI notes for a plan",
		Description: "Generates notes from the plan's focus and checklist. " +
			"`requestType` selects one kind; an empty value or `{}` generates all. " +
			"A request with no body at all is rejected with 400.",
		Errors: writeErrs,
	}, func(ctx context.Context, in *GenerateAINotesInput) (*NoteListOutput, error) {
		if err := authorize(ctx, "generate ai", in.PlanID); err != nil {
			return nil, err
		}
		requestType := in.Body.RequestType
		if requestType == "" {
			requestType = "all"
		}
		generated, err := svc.GenerateAINotes(in.PlanID, requestType)
		if err != nil {
			return nil, apierr.ProblemFor("notes: generate ai", err)
		}
		return &NoteListOutput{Body: toNoteResponses(generated)}, nil
	})
}

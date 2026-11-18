package insights

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/darkphotonKN/fireplace/services/api-gateway/internal/apierr"
	"github.com/google/uuid"
)

// The note summary operation (FS-PGG2N R38-R44, §API surface), served by
// insights-service over gRPC. A new operation, so camelCase throughout.

// NoteSummaryService is the gateway's seam to insights-service's
// SummarizeNote, satisfied by the gRPC client (internal/gateway/insights).
type NoteSummaryService interface {
	SummarizeNote(ctx context.Context, planID, userID, itemID uuid.UUID) (string, error)
}

// SummarizeNoteRequest is summarizeNote's body: both ids required; identity
// comes from the JWT, never the body (ADR-0005 §3). Unknown members are
// refused (huma's additionalProperties:false).
type SummarizeNoteRequest struct {
	PlanID uuid.UUID `json:"planId" required:"true" format:"uuid" doc:"Plan id; the note must belong to it"`
	ItemID uuid.UUID `json:"itemId" required:"true" format:"uuid" doc:"The note to summarize"`
}

type SummarizeNoteInput struct {
	Body SummarizeNoteRequest
}

// NoteSummaryResponse is the text only: nothing is stored (R43).
type NoteSummaryResponse struct {
	Summary string `json:"summary" minLength:"1" maxLength:"280" doc:"One plain paragraph summarizing the note's body, in the note's language. Not saved: set it as the note's summary with updateChecklist"`
}

type NoteSummaryOutput struct{ Body NoteSummaryResponse }

// RegisterNoteSummaryOperations registers summarizeNote. Its only trigger is
// a user's press (R44, ADR-0019): it is a POST, and no read path calls it.
func RegisterNoteSummaryOperations(api huma.API, notes NoteSummaryService, access PlanAccess,
	protect func(huma.Context, func(huma.Context)), secured []map[string][]string,
) {
	authorize := authorizer(access)

	huma.Register(api, huma.Operation{
		OperationID: "summarizeNote", Method: http.MethodPost,
		Path:        "/api/insights/note-summary",
		Middlewares: huma.Middlewares{protect}, Security: secured,
		Summary: "Summarize a note with AI",
		Description: "Writes one plain paragraph of at most 280 characters summarizing the note's body and " +
			"returns it. Nothing is stored: to keep it, save it as the note's summary with updateChecklist. " +
			"The model reads only the first 8000 characters of the body. Each call is one metered model call. " +
			"404 NOT_FOUND: the note is missing, archived or in another plan. 400 ITEM_NOT_A_NOTE: the item is " +
			"a task. 429 LLM_BUDGET_EXHAUSTED: the platform's daily token budget is spent; retry after the next " +
			"UTC midnight. 503 NOTE_SUMMARY_FAILED: the model failed, timed out or wrote nothing; retrying is " +
			"correct. 503 SERVICE_UNAVAILABLE: the insights service could not be reached; retrying is correct.",
		Errors: []int{
			http.StatusBadRequest, http.StatusUnauthorized, http.StatusForbidden, http.StatusNotFound,
			http.StatusUnprocessableEntity, http.StatusTooManyRequests, http.StatusServiceUnavailable,
		},
	}, func(ctx context.Context, in *SummarizeNoteInput) (*NoteSummaryOutput, error) {
		const op = "insights: summarize note"
		userID, err := authorize(ctx, op, in.Body.PlanID)
		if err != nil {
			return nil, err
		}
		summary, err := notes.SummarizeNote(ctx, in.Body.PlanID, userID, in.Body.ItemID)
		if err != nil {
			return nil, apierr.ProblemFor(op, err)
		}
		return &NoteSummaryOutput{Body: NoteSummaryResponse{Summary: summary}}, nil
	})
}

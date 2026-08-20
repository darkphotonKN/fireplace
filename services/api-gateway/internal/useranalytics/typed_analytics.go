package useranalytics

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

// SERIALIZED analytics operation (FS-0004 §API surface, analytics).
//
// HONEST STATUS: the handler and repository behind this are documented stubs.
// The operation is published — path, param, security, and the success shape
// declared from model.go — but it answers 501 today. The success shape is
// therefore UNPROVEN until the data path lands, and this comment exists so that
// is impossible to forget. Serializing a stub is legitimate; claiming it
// verified would not be.

// AnalyticsService is the narrow seam this operation depends on. It is legal
// for this to be nil while the data path is unbuilt: the handler answers 501
// before it would ever be consulted.
type AnalyticsService interface {
	GetUserAnalytics(ctx context.Context, userID uuid.UUID, date time.Time) (*UserAnalytics, error)
}

// UserAnalyticsResponse mirrors UserAnalytics field for field (R6). Declared so
// the contract publishes the intended shape; NOT yet returned by any code path.
type UserAnalyticsResponse struct {
	ID               uuid.UUID `json:"id"`
	UserID           uuid.UUID `json:"userId"`
	Date             time.Time `json:"date"`
	TasksCompleted   int       `json:"tasksCompleted"`
	TasksTotal       int       `json:"tasksTotal"`
	CompletionRate   float64   `json:"completionRate"`
	CurrentStreak    int       `json:"currentStreak"`
	ActivePlansCount int       `json:"activePlansCount"`
	CreatedAt        time.Time `json:"createdAt"`
	UpdatedAt        time.Time `json:"updatedAt"`
}

type UserAnalyticsOutput struct{ Body UserAnalyticsResponse }

type GetUserAnalyticsInput struct {
	UserID uuid.UUID `path:"userId" doc:"User id"`
}

func RegisterAnalyticsOperations(api huma.API, _ AnalyticsService,
	protect func(huma.Context, func(huma.Context)), secured []map[string][]string,
) {
	huma.Register(api, huma.Operation{
		OperationID: "getUserAnalytics", Method: http.MethodGet,
		Path:        "/api/analytics/user/{userId}",
		Middlewares: huma.Middlewares{protect}, Security: secured,
		Summary: "Get a user's daily analytics",
		Description: "Returns per-user daily completion counts, rate, and streak. " +
			"NOT YET IMPLEMENTED: this operation answers 501 with code `NOT_IMPLEMENTED` " +
			"until its data path lands. The success shape below is the intended contract " +
			"and is unproven until then.",
		Errors: []int{
			http.StatusUnauthorized, http.StatusUnprocessableEntity,
			http.StatusNotImplemented, http.StatusServiceUnavailable,
		},
	}, func(ctx context.Context, in *GetUserAnalyticsInput) (*UserAnalyticsOutput, error) {
		if _, ok := auth.UserIDFromCtx(ctx); !ok {
			return nil, apierr.ProblemFor("useranalytics: get user analytics",
				apierr.ErrNoIdentity())
		}
		// Transcribed from the gin handler, which returned 501 unconditionally.
		return nil, apierr.ProblemFor("useranalytics: get user analytics",
			fmt.Errorf("%w: user analytics data path", commonconstants.ErrNotImplemented))
	})
}

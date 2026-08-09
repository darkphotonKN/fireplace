package authgw

import (
	"context"
	"net/http"

	pb "github.com/darkphotonKN/fireplace/common/api/proto/auth"
	"github.com/google/uuid"

	"github.com/danielgtaylor/huma/v2"
	"github.com/darkphotonKN/fireplace/services/api-gateway/internal/apierr"
	"github.com/darkphotonKN/fireplace/services/api-gateway/internal/auth"
)

// SERIALIZED profile operations (FS-0002 §API surface).
//
// This is the wiring layer: huma.Register makes the operation's shape
// machine-readable, which is what allows openapi.yaml to be DERIVED rather than
// authored. The gRPC call inside is the same one the legacy gin handler makes.
//
// Registration must run WITHOUT infrastructure: cmd/openapi calls this with a
// nil handler to emit the spec. Nothing here may touch the network or a DB —
// the handler closures capture h but are not invoked during generation.

// ProfileClient is the narrow seam the serialized operations depend on.
//
// Accept interfaces, return structs: *Client satisfies this, and a test double
// satisfies it without a gRPC connection. Registration previously took the
// concrete *Handler, which made every acceptance criterion in I-0004/I-0005
// untestable without a live auth-service — develop's "missing interface" gate.
type ProfileClient interface {
	GetProfileProto(ctx context.Context, id uuid.UUID) (*pb.User, error)
	UpdateProfileProto(ctx context.Context, id uuid.UUID, req UpdateProfileRequest) (*pb.User, error)
}

func RegisterOperations(api huma.API, c ProfileClient) {
	huma.Register(api, huma.Operation{
		OperationID: "getProfile",
		Method:      http.MethodGet,
		Path:        "/users/profile",
		Summary:     "Get the authenticated user's profile",
		Description: "Returns the profile of the user identified by the bearer token's `sub` claim.",
		Errors:      []int{http.StatusUnauthorized, http.StatusNotFound, http.StatusInternalServerError},
	}, func(ctx context.Context, _ *struct{}) (*GetProfileOutput, error) {
		userID, ok := auth.UserIDFromCtx(ctx)
		if !ok {
			return nil, apierr.ProblemFor("get profile", errUnauthorized())
		}
		u, err := c.GetProfileProto(ctx, userID)
		if err != nil {
			return nil, apierr.ProblemFor("get profile", err)
		}
		return &GetProfileOutput{Body: profileFromProto(u)}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "updateProfile",
		Method:      http.MethodPatch,
		Path:        "/users/profile",
		Summary:     "Update the authenticated user's profile",
		Description: "Partial update. An absent field and a null field both mean " +
			"\"leave unchanged\"; an empty string sets the field to empty. An empty " +
			"body is a valid no-op. Validation lives in auth-service, so an empty " +
			"name is rejected with 400, not 422.",
		Errors: []int{http.StatusBadRequest, http.StatusUnauthorized, http.StatusNotFound, http.StatusInternalServerError},
	}, func(ctx context.Context, in *UpdateProfileInput) (*UpdateProfileOutput, error) {
		userID, ok := auth.UserIDFromCtx(ctx)
		if !ok {
			return nil, apierr.ProblemFor("update profile", errUnauthorized())
		}
		u, err := c.UpdateProfileProto(ctx, userID, in.Body)
		if err != nil {
			return nil, apierr.ProblemFor("update profile", err)
		}
		return &UpdateProfileOutput{Body: profileFromProto(u)}, nil
	})
}

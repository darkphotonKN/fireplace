package authgw

import (
	"fmt"
	"time"

	pb "github.com/darkphotonKN/fireplace/common/api/proto/auth"
	commonconstants "github.com/darkphotonKN/fireplace/common/constants"
	"github.com/google/uuid"
)

// Transport types for the SERIALIZED profile surface (FS-0002 §API surface).
//
// These are the LAST hand-authored artifact in the contract chain: huma derives
// openapi.yaml from these signatures, and openapi-typescript derives the client
// from that. Nothing downstream is written by hand.
//
// Per ADR-0003 these are declared PER OPERATION and are never storage models.
// Note what that buys here concretely: models.User is a monolith leftover whose
// table was dropped in migration 000020 — it is backed by nothing, yet it still
// carries a `Password` field and snake_case date tags. Mapping straight from
// pb.User deletes that hop instead of laundering it.

// ProfileResponse is the published shape of a user's own profile.
// It publishes exactly these seven fields — `password` cannot appear here
// regardless of what any other struct grows later.
type ProfileResponse struct {
	ID          uuid.UUID `json:"id" doc:"User id"`
	Email       string    `json:"email" doc:"Email address" example:"a@b.com"`
	Name        string    `json:"name" doc:"Account name" example:"Kranti"`
	DisplayName *string   `json:"displayName" doc:"Optional display name" example:"kn"`
	Bio         *string   `json:"bio" doc:"Optional short biography"`
	CreatedAt   time.Time `json:"createdAt" doc:"When the account was created"`
	UpdatedAt   time.Time `json:"updatedAt" doc:"When the profile was last changed"`
}

// profileFromProto maps the auth-service response straight to the wire shape.
// Deliberately does NOT route through models.User (ADR-0003).
func profileFromProto(u *pb.User) ProfileResponse {
	id, _ := uuid.Parse(u.Id)
	return ProfileResponse{
		ID:          id,
		Email:       u.Email,
		Name:        u.Name,
		DisplayName: u.DisplayName,
		Bio:         u.Bio,
		CreatedAt:   u.CreatedAt.AsTime(),
		UpdatedAt:   u.UpdatedAt.AsTime(),
	}
}

// --- huma input/output wrappers -------------------------------------------
//
// huma locates the request/response body via a field literally named `Body`.
// These wrappers are what make the operation's shape machine-readable; they
// carry no logic.

type GetProfileOutput struct {
	Body ProfileResponse
}

type UpdateProfileInput struct {
	Body UpdateProfileRequest
}

type UpdateProfileOutput struct {
	Body ProfileResponse
}

// errUnauthorized is the sentinel used when the identity bridge yields nothing —
// which should be unreachable behind ProblemMiddleware, but must still map to a
// contract-shaped error rather than a panic.
func errUnauthorized() error {
	return fmt.Errorf("%w: no identity in context", commonconstants.ErrUnauthorized)
}

package authgw

import (
	"time"

	pb "github.com/darkphotonKN/fireplace/common/api/proto/auth"
	"github.com/google/uuid"
)

// Transport types for the SERIALIZED users surface (FS-0004 §API surface).
//
// Same rule as transport.go: these map STRAIGHT from the proto and never route
// through models.User. That matters more here than it did for profile, because
// models.User is what the legacy signin, get-user, and list-users bodies
// published — and it carries `json:"password,omitempty"` backed by a table that
// migration 000020 dropped. Nothing assigns that field today, so nothing leaks;
// mapping from the proto means nothing ever can.

// UserResponse is the published shape of a user.
//
// Date keys are camelCase, matching ProfileResponse (FS-0002). The legacy body
// spelled them created_at/updated_at because models.User embeds
// BaseDBDateModel, whose tags are snake_case. Transcribing that literally would
// have published two spellings of one entity's timestamps in a single
// document — a defect this retrofit would be introducing, not preserving, so it
// is corrected here as a deliberate break riding the same slice as the
// frontend cutover (ADR-0006 §3's reasoning, applied to a rename).
type UserResponse struct {
	ID          uuid.UUID `json:"id" doc:"User id"`
	Email       string    `json:"email" doc:"Email address" example:"a@b.com"`
	Name        string    `json:"name" doc:"Account name" example:"Kranti"`
	DisplayName *string   `json:"displayName" doc:"Optional display name" example:"kn"`
	Bio         *string   `json:"bio" doc:"Optional short biography"`
	CreatedAt   time.Time `json:"createdAt" doc:"When the account was created"`
	UpdatedAt   time.Time `json:"updatedAt" doc:"When the user was last changed"`
}

func userResponseFromProto(u *pb.User) UserResponse {
	id, _ := uuid.Parse(u.Id)
	return UserResponse{
		ID:          id,
		Email:       u.Email,
		Name:        u.Name,
		DisplayName: u.DisplayName,
		Bio:         u.Bio,
		CreatedAt:   u.CreatedAt.AsTime(),
		UpdatedAt:   u.UpdatedAt.AsTime(),
	}
}

// AuthResponse is what signin publishes: the token pair plus the user it
// belongs to.
//
// Field names are transcribed from the legacy LoginResponse exactly —
// including `userInfo`, which reads oddly next to `accessToken` but is what
// the frontend reads today and is not this feature's to rename.
//
// The expiry fields are NANOSECONDS. That is inherited from the monolith's
// int(time.Duration) and is preserved verbatim (FS-0004 R6); it is documented
// on the field rather than converted, because converting is a behaviour change.
type AuthResponse struct {
	RefreshToken     string       `json:"refreshToken" doc:"Refresh token"`
	AccessToken      string       `json:"accessToken" doc:"Bearer token for the Authorization header"`
	AccessExpiresIn  int64        `json:"accessExpiresIn" doc:"Access token lifetime in NANOSECONDS"`
	RefreshExpiresIn int64        `json:"refreshExpiresIn" doc:"Refresh token lifetime in NANOSECONDS"`
	UserInfo         UserResponse `json:"userInfo" doc:"The authenticated user"`
}

func authResponseFromProto(r *pb.AuthResponse) AuthResponse {
	return AuthResponse{
		RefreshToken:     r.RefreshToken,
		AccessToken:      r.AccessToken,
		AccessExpiresIn:  r.AccessExpiresIn,
		RefreshExpiresIn: r.RefreshExpiresIn,
		UserInfo:         userResponseFromProto(r.User),
	}
}

// SignupRequest is the body of POST /api/users/signup.
type SignupRequest struct {
	Email    string `json:"email" doc:"Email address" example:"a@b.com"`
	Password string `json:"password" doc:"Plaintext password; hashed by auth-service"`
	Name     string `json:"name" doc:"Account name" example:"Kranti"`
}

// SigninRequest is the body of POST /api/users/signin.
type SigninRequest struct {
	Email    string `json:"email" doc:"Email address" example:"a@b.com"`
	Password string `json:"password" doc:"Plaintext password"`
}

// --- huma input/output wrappers -------------------------------------------

type SignupInput struct {
	Body SignupRequest
}

// SignupOutput has NO Body field, and that is transcribed, not overlooked.
//
// client.SignUp returns a full AuthResponse with both tokens, and the legacy
// handler discards it, replying {statusCode, message} with no user data.
// Returning those tokens would be auto-login on signup — a genuine improvement
// and a genuine behaviour change, so it belongs to its own feature spec rather
// than riding a retrofit (FS-0004 R6).
type SignupOutput struct{}

type SigninInput struct {
	Body SigninRequest
}

type SigninOutput struct {
	Body AuthResponse
}

type GetUserInput struct {
	ID uuid.UUID `path:"id" doc:"User id"`
}

type GetUserOutput struct {
	Body UserResponse
}

type ListUsersOutput struct {
	Body []UserResponse
}

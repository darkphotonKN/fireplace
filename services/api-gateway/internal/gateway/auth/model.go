package authgw

import "github.com/darkphotonKN/fireplace/services/api-gateway/internal/models"

// HTTP request/response shapes — kept 1:1 with the monolith's previous user API
// so external clients (flow-client, etc.) see no change after the gRPC cutover.

type SignUpRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Name     string `json:"name"`
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type LoginResponse struct {
	RefreshToken     string       `json:"refreshToken"`
	AccessToken      string       `json:"accessToken"`
	AccessExpiresIn  int64        `json:"accessExpiresIn"`
	RefreshExpiresIn int64        `json:"refreshExpiresIn"`
	UserInfo         *models.User `json:"userInfo"`
}

// UpdateProfileRequest is a PARTIAL update: every field is optional and an
// absent field means "leave unchanged" (FS-0002 §Requirements 8-10).
//
// The `omitempty` tags are load-bearing, not cosmetic: huma marks a field
// REQUIRED unless the json tag carries omitempty. Without them the generated
// schema required all three, and an empty body `{}` — which FS-0002 R10 says
// is a valid no-op — would have been rejected with 422.
type UpdateProfileRequest struct {
	// STRICT: an undeclared member is a 422, not a silent no-op.
	//
	// huma's default (additionalProperties:false) is kept deliberately. On a
	// PATCH, silently ignoring an unknown member is the worst outcome — a
	// typo like {"biio":"…"} would return 200 while changing nothing, so the
	// user believes they saved something they did not. 422 names the problem.
	//
	// Safe here because the client ships with the server and consumes the
	// generated TypeScript client, so it cannot send an undeclared field by
	// accident. Revisit ONLY if this API gains consumers that deploy
	// independently — strict request bodies couple client and server releases.
	//
	// NOTE: `id` is deliberately absent. Identity comes from the JWT `sub`
	// claim, never the body — accepting a client-supplied id on "update MY
	// profile" is the trust-the-body-field hole ADR-0001 closes.
	Name        *string `json:"name,omitempty"`
	DisplayName *string `json:"displayName,omitempty"`
	Bio         *string `json:"bio,omitempty"`
}

type RefreshTokenRequest struct {
	RefreshToken string `json:"refreshToken"`
}

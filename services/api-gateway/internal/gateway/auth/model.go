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

type UpdateProfileRequest struct {
	Name        *string `json:"name"`
	DisplayName *string `json:"displayName"`
	Bio         *string `json:"bio"`
}

type RefreshTokenRequest struct {
	RefreshToken string `json:"refreshToken"`
}

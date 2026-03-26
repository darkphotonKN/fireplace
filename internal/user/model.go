package user

import "github.com/darkphotonKN/fireplace/internal/models"

type Response struct {
	models.BaseDBDateModel
	Email       string  `db:"email" json:"email"`
	Name        string  `db:"name" json:"name"`
	DisplayName *string `db:"display_name" json:"displayName"`
	Bio         *string `db:"bio" json:"bio"`
}

type LoginResponse struct {
	RefreshToken     string `json:"refreshToken"`
	AccessToken      string `json:"accessToken"`
	AccessExpiresIn  int    `json:"accessExpiresIn"`
	RefreshExpiresIn int    `json:"refreshExpiresIn"`

	UserInfo *models.User `json:"userInfo"`
}

type LoginRequest struct {
	Email    string `db:"email" json:"email"`
	Password string `db:"password" json:"password"`
}

type UpdateProfileRequest struct {
	Name        *string `json:"name"`
	DisplayName *string `json:"displayName"`
	Bio         *string `json:"bio"`
}

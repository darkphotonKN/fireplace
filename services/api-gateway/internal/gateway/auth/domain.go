package authgw

import (
	"time"

	"github.com/google/uuid"
)

// User mirrors the auth-service users table. The column is named "password"
// (not "hashed_password") to match the existing schema inherited from the
// monolith; Go field name HashedPassword keeps the intent visible. JSON tag
// "-" keeps the hash out of any responses.
type User struct {
	ID             uuid.UUID `db:"id" json:"id"`
	Email          string    `db:"email" json:"email"`
	Name           string    `db:"name" json:"name"`
	HashedPassword string    `db:"password" json:"-"`
	DisplayName    *string   `db:"display_name" json:"displayName,omitempty"`
	Bio            *string   `db:"bio" json:"bio,omitempty"`
	CreatedAt      time.Time `db:"created_at" json:"created_at"`
	UpdatedAt      time.Time `db:"updated_at" json:"updated_at"`
}

type SignUpInput struct {
	Email    string
	Password string
	Name     string
}

type SignInInput struct {
	Email    string
	Password string
}

type UpdateProfileParams struct {
	ID          uuid.UUID
	Name        *string
	DisplayName *string
	Bio         *string
}

type AuthTokens struct {
	User             *User
	AccessToken      string
	RefreshToken     string
	AccessExpiresIn  time.Duration
	RefreshExpiresIn time.Duration
}

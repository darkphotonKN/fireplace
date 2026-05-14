package auth

import (
	"time"

	"github.com/google/uuid"
)

// User mirrors the auth-service's users table. HashedPassword stays out of
// JSON output so handlers can return User directly without leaking credentials.
type User struct {
	ID             uuid.UUID `db:"id" json:"id"`
	Email          string    `db:"email" json:"email"`
	Name           string    `db:"name" json:"name"`
	HashedPassword string    `db:"hashed_password" json:"-"`
	CreatedAt      time.Time `db:"created_at" json:"created_at"`
	UpdatedAt      time.Time `db:"updated_at" json:"updated_at"`
}

// SignUpRequest is the service-layer input for creating a user.
type SignUpRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Name     string `json:"name"`
}

// SignInRequest is the service-layer input for authenticating a user.
type SignInRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

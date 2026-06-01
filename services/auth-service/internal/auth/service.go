package auth

import (
	"context"
	"errors"
	"fmt"
	"time"

	commonauth "github.com/darkphotonKN/fireplace/common/auth"
	commonbroker "github.com/darkphotonKN/fireplace/common/broker"
	commonconstants "github.com/darkphotonKN/fireplace/common/constants"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

// TTL defaults applied when env vars are unset or unparseable.
// 1h / 7d matches the original monolith — kept here as a safety net.
const (
	defaultAccessTTL  = time.Hour
	defaultRefreshTTL = 24 * 7 * time.Hour
)

type Repository interface {
	Create(ctx context.Context, u *User) error
	GetByID(ctx context.Context, id uuid.UUID) (*User, error)
	GetByEmail(ctx context.Context, email string) (*User, error)
	ListAll(ctx context.Context) ([]*User, error)
	UpdateProfile(ctx context.Context, in *UpdateProfileInput) (*User, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

type service struct {
	repo       Repository
	publishCh  commonbroker.Publisher
	jwtSecret  string
	accessTTL  time.Duration
	refreshTTL time.Duration
}

// NewService constructs the auth service. accessTTL / refreshTTL come from
// the env (ACCESS_TOKEN_TTL / REFRESH_TOKEN_TTL) — caller parses them with
// time.ParseDuration. Pass 0 to fall back to the const defaults above.
func NewService(repo Repository, publishCh commonbroker.Publisher, jwtSecret string, accessTTL, refreshTTL time.Duration) *service {
	if accessTTL <= 0 {
		accessTTL = defaultAccessTTL
	}
	if refreshTTL <= 0 {
		refreshTTL = defaultRefreshTTL
	}
	return &service{
		repo:       repo,
		publishCh:  publishCh,
		jwtSecret:  jwtSecret,
		accessTTL:  accessTTL,
		refreshTTL: refreshTTL,
	}
}

// (issueTokens defined below; keeps the field references compact)

func (s *service) SignUp(ctx context.Context, in *SignUpInput) (*AuthTokens, error) {
	if in.Email == "" || in.Password == "" || in.Name == "" {
		return nil, commonconstants.ErrInvalidInput
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(in.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("auth: hash password: %w", err)
	}

	u := &User{
		Email:          in.Email,
		Name:           in.Name,
		HashedPassword: string(hashed),
	}
	if err := s.repo.Create(ctx, u); err != nil {
		return nil, fmt.Errorf("auth: sign up: %w", err)
	}

	s.PublishUserCreated(ctx, u)
	return s.issueTokens(u)
}

func (s *service) SignIn(ctx context.Context, in *SignInInput) (*AuthTokens, error) {
	u, err := s.repo.GetByEmail(ctx, in.Email)
	if err != nil {
		// Business decision: a missing account must be indistinguishable from a
		// wrong password so we don't leak which emails are registered.
		if errors.Is(err, commonconstants.ErrNotFound) {
			return nil, commonconstants.ErrUnauthorized
		}
		return nil, fmt.Errorf("auth: sign in: %w", err)
	}
	if err := bcrypt.CompareHashAndPassword([]byte(u.HashedPassword), []byte(in.Password)); err != nil {
		return nil, commonconstants.ErrUnauthorized
	}
	return s.issueTokens(u)
}

func (s *service) Refresh(ctx context.Context, refreshToken string) (*AuthTokens, error) {
	userID, err := commonauth.ValidateRefreshToken(refreshToken, s.jwtSecret)
	if err != nil {
		return nil, commonconstants.ErrUnauthorized
	}
	u, err := s.repo.GetByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("auth: refresh: %w", err)
	}
	return s.issueTokens(u)
}

func (s *service) GetUser(ctx context.Context, id uuid.UUID) (*User, error) {
	u, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("auth: get user: %w", err)
	}
	return u, nil
}

func (s *service) ListUsers(ctx context.Context) ([]*User, error) {
	users, err := s.repo.ListAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("auth: list users: %w", err)
	}
	return users, nil
}

func (s *service) UpdateProfile(ctx context.Context, in *UpdateProfileInput) (*User, error) {
	if in.Name != nil && *in.Name == "" {
		return nil, fmt.Errorf("%w: name cannot be empty", commonconstants.ErrInvalidInput)
	}
	u, err := s.repo.UpdateProfile(ctx, in)
	if err != nil {
		return nil, fmt.Errorf("auth: update profile: %w", err)
	}
	s.PublishUserUpdated(ctx, u)
	return u, nil
}

func (s *service) DeleteUser(ctx context.Context, id uuid.UUID) error {
	if err := s.repo.Delete(ctx, id); err != nil {
		return fmt.Errorf("auth: delete user: %w", err)
	}
	s.PublishUserDeleted(ctx, id)
	return nil
}

func (s *service) issueTokens(u *User) (*AuthTokens, error) {
	access, err := commonauth.GenerateJWT(u.ID, commonauth.TokenTypeAccess, s.jwtSecret, s.accessTTL)
	if err != nil {
		return nil, fmt.Errorf("auth: issue access token: %w", err)
	}
	refresh, err := commonauth.GenerateJWT(u.ID, commonauth.TokenTypeRefresh, s.jwtSecret, s.refreshTTL)
	if err != nil {
		return nil, fmt.Errorf("auth: issue refresh token: %w", err)
	}
	return &AuthTokens{
		User:             u,
		AccessToken:      access,
		RefreshToken:     refresh,
		AccessExpiresIn:  s.accessTTL,
		RefreshExpiresIn: s.refreshTTL,
	}, nil
}

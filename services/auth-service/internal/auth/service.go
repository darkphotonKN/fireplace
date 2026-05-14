package auth

import (
	"context"
	"errors"
	"time"

	commonauth "github.com/darkphotonKN/fireplace/common/auth"
	commonbroker "github.com/darkphotonKN/fireplace/common/broker"
	commonconstants "github.com/darkphotonKN/fireplace/common/constants"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

const (
	accessTTL  = time.Hour
	refreshTTL = 24 * 7 * time.Hour
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
	repo      Repository
	publishCh commonbroker.Publisher
	jwtSecret string
}

func NewService(repo Repository, publishCh commonbroker.Publisher, jwtSecret string) *service {
	return &service{repo: repo, publishCh: publishCh, jwtSecret: jwtSecret}
}

func (s *service) SignUp(ctx context.Context, in *SignUpInput) (*AuthTokens, error) {
	if in.Email == "" || in.Password == "" || in.Name == "" {
		return nil, commonconstants.ErrInvalidInput
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(in.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	u := &User{
		Email:          in.Email,
		Name:           in.Name,
		HashedPassword: string(hashed),
	}
	if err := s.repo.Create(ctx, u); err != nil {
		return nil, err
	}

	s.PublishUserCreated(ctx, u)
	return s.issueTokens(u)
}

func (s *service) SignIn(ctx context.Context, in *SignInInput) (*AuthTokens, error) {
	u, err := s.repo.GetByEmail(ctx, in.Email)
	if err != nil {
		if errors.Is(err, commonconstants.ErrNotFound) {
			return nil, commonconstants.ErrUnauthorized
		}
		return nil, err
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
		return nil, err
	}
	return s.issueTokens(u)
}

func (s *service) GetUser(ctx context.Context, id uuid.UUID) (*User, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *service) ListUsers(ctx context.Context) ([]*User, error) {
	return s.repo.ListAll(ctx)
}

func (s *service) UpdateProfile(ctx context.Context, in *UpdateProfileInput) (*User, error) {
	if in.Name != nil && *in.Name == "" {
		return nil, errors.New("name cannot be empty")
	}
	u, err := s.repo.UpdateProfile(ctx, in)
	if err != nil {
		return nil, err
	}
	s.PublishUserUpdated(ctx, u)
	return u, nil
}

func (s *service) DeleteUser(ctx context.Context, id uuid.UUID) error {
	if err := s.repo.Delete(ctx, id); err != nil {
		return err
	}
	s.PublishUserDeleted(ctx, id)
	return nil
}

func (s *service) issueTokens(u *User) (*AuthTokens, error) {
	access, err := commonauth.GenerateJWT(u.ID, commonauth.TokenTypeAccess, s.jwtSecret, accessTTL)
	if err != nil {
		return nil, err
	}
	refresh, err := commonauth.GenerateJWT(u.ID, commonauth.TokenTypeRefresh, s.jwtSecret, refreshTTL)
	if err != nil {
		return nil, err
	}
	return &AuthTokens{
		User:             u,
		AccessToken:      access,
		RefreshToken:     refresh,
		AccessExpiresIn:  accessTTL,
		RefreshExpiresIn: refreshTTL,
	}, nil
}

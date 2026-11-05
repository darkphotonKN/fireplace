package authgw

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	commonauth "github.com/darkphotonKN/fireplace/common/auth"
	commonconstants "github.com/darkphotonKN/fireplace/common/constants"
	"github.com/google/uuid"
)

const testJWTSecret = "test-secret"

// fakeRepo answers GetByID only. Refresh touches nothing else, and an embedded
// nil Repository makes any other call panic rather than pass quietly.
type fakeRepo struct {
	Repository
	user *User
	err  error
}

func (f fakeRepo) GetByID(context.Context, uuid.UUID) (*User, error) {
	return f.user, f.err
}

func mint(t *testing.T, tokenType commonauth.TokenType, userID uuid.UUID, ttl time.Duration) string {
	t.Helper()
	tok, err := commonauth.GenerateJWT(userID, tokenType, testJWTSecret, ttl)
	if err != nil {
		t.Fatalf("mint token: %v", err)
	}
	return tok
}

func TestRefresh_DeletedUser_ReturnsUnauthorized(t *testing.T) {
	notFound := fmt.Errorf("get user by id: %w", commonconstants.ErrNotFound)
	svc := NewService(fakeRepo{err: notFound}, nil, testJWTSecret, 0, 0)

	_, err := svc.Refresh(context.Background(), mint(t, commonauth.TokenTypeRefresh, uuid.New(), time.Hour))

	if !errors.Is(err, commonconstants.ErrUnauthorized) {
		t.Fatalf("want ErrUnauthorized, got %v", err)
	}
	// The difference between "deleted" and "bad token" is never exposed (R4).
	if errors.Is(err, commonconstants.ErrNotFound) {
		t.Errorf("the not-found cause must not survive into the result: %v", err)
	}
}

func TestRefresh_RepositoryFailure_StaysWrappedAndNotUnauthorized(t *testing.T) {
	outage := errors.New("connection refused")
	svc := NewService(fakeRepo{err: outage}, nil, testJWTSecret, 0, 0)

	_, err := svc.Refresh(context.Background(), mint(t, commonauth.TokenTypeRefresh, uuid.New(), time.Hour))

	if !errors.Is(err, outage) {
		t.Fatalf("want the repository error wrapped, got %v", err)
	}
	if errors.Is(err, commonconstants.ErrUnauthorized) {
		t.Errorf("an outage must not read as a rejected token: %v", err)
	}
}

func TestRefresh_BadToken_ReturnsUnauthorized(t *testing.T) {
	userID := uuid.New()
	cases := map[string]string{
		"garbage":      "not-a-jwt",
		"empty":        "",
		"expired":      mint(t, commonauth.TokenTypeRefresh, userID, -time.Minute),
		"access token": mint(t, commonauth.TokenTypeAccess, userID, time.Hour),
	}
	for name, token := range cases {
		t.Run(name, func(t *testing.T) {
			// A user exists, so only the token check can refuse.
			svc := NewService(fakeRepo{user: &User{ID: userID}}, nil, testJWTSecret, 0, 0)
			if _, err := svc.Refresh(context.Background(), token); !errors.Is(err, commonconstants.ErrUnauthorized) {
				t.Fatalf("want ErrUnauthorized, got %v", err)
			}
		})
	}
}

// The window slides: a token with almost nothing left yields a full lifetime,
// measured from the exchange, and the result can itself be exchanged (R5).
func TestRefresh_NearlyExpiredToken_IssuesFullLifetimeThatChains(t *testing.T) {
	userID := uuid.New()
	const refreshTTL = 720 * time.Hour
	svc := NewService(fakeRepo{user: &User{ID: userID}}, nil, testJWTSecret, time.Hour, refreshTTL)

	first, err := svc.Refresh(context.Background(), mint(t, commonauth.TokenTypeRefresh, userID, time.Minute))
	if err != nil {
		t.Fatalf("first exchange: %v", err)
	}

	claims, err := commonauth.ParseToken(first.RefreshToken, testJWTSecret)
	if err != nil {
		t.Fatalf("parse new refresh token: %v", err)
	}
	exp, err := claims.GetExpirationTime()
	if err != nil {
		t.Fatalf("read exp: %v", err)
	}
	if remaining := time.Until(exp.Time); remaining < refreshTTL-time.Minute || remaining > refreshTTL {
		t.Errorf("new refresh token has %v left, want about %v", remaining, refreshTTL)
	}
	if first.RefreshExpiresIn != refreshTTL {
		t.Errorf("RefreshExpiresIn: want %v, got %v", refreshTTL, first.RefreshExpiresIn)
	}
	if got, err := commonauth.ValidateRefreshToken(first.RefreshToken, testJWTSecret); err != nil || got != userID {
		t.Errorf("new refresh token: want user %s, got %s (err %v)", userID, got, err)
	}

	if _, err := svc.Refresh(context.Background(), first.RefreshToken); err != nil {
		t.Errorf("the new refresh token must itself be exchangeable: %v", err)
	}
}

func TestNewService_ZeroTTLs_FallBackToConstDefaults(t *testing.T) {
	userID := uuid.New()
	svc := NewService(fakeRepo{user: &User{ID: userID}}, nil, testJWTSecret, 0, 0)

	tokens, err := svc.Refresh(context.Background(), mint(t, commonauth.TokenTypeRefresh, userID, time.Hour))
	if err != nil {
		t.Fatalf("refresh: %v", err)
	}
	if tokens.AccessExpiresIn != time.Hour {
		t.Errorf("access fallback: want 1h, got %v", tokens.AccessExpiresIn)
	}
	if tokens.RefreshExpiresIn != 720*time.Hour {
		t.Errorf("refresh fallback: want 720h, got %v", tokens.RefreshExpiresIn)
	}
}

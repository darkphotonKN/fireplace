package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	commonauth "github.com/darkphotonKN/fireplace/common/auth"
	commonconstants "github.com/darkphotonKN/fireplace/common/constants"
	"github.com/darkphotonKN/fireplace/common/errcode"
	authgw "github.com/darkphotonKN/fireplace/services/api-gateway/internal/gateway/auth"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"golang.org/x/net/context"
)

// POST /api/users/refresh (FS-ZESEF R1-R6), through the real registration path,
// the real LocalClient and the real service. Only the repository is fake, so
// these cover the whole route from JSON body to the status on the wire.

const refreshPath = "/api/users/refresh"

// usersRepo answers GetByID only. The embedded nil Repository makes any other
// call panic, so an exchange that reaches for anything else fails loudly.
type usersRepo struct {
	authgw.Repository
	user *authgw.User
	err  error
}

func (r usersRepo) GetByID(context.Context, uuid.UUID) (*authgw.User, error) {
	return r.user, r.err
}

func refreshEngine(repo authgw.Repository) *gin.Engine {
	svc := authgw.NewService(repo, nil, testSecret, time.Hour, 720*time.Hour)
	return buildEngineWithUsers(fakeProfileClient{user: sampleUser()}, authgw.NewLocalClient(svc))
}

// postRefresh sends no Authorization header: the token travels in the body.
func postRefresh(t *testing.T, e *gin.Engine, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, refreshPath, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	return rec
}

func refreshBody(t *testing.T, refreshToken string) string {
	t.Helper()
	raw, err := json.Marshal(map[string]string{"refreshToken": refreshToken})
	if err != nil {
		t.Fatalf("marshal body: %v", err)
	}
	return string(raw)
}

func mintFor(t *testing.T, tokenType commonauth.TokenType, userID uuid.UUID, ttl time.Duration) string {
	t.Helper()
	tok, err := commonauth.GenerateJWT(userID, tokenType, testSecret, ttl)
	if err != nil {
		t.Fatalf("mint token: %v", err)
	}
	return tok
}

func decodeAuthResponse(t *testing.T, rec *httptest.ResponseRecorder) authgw.AuthResponse {
	t.Helper()
	if rec.Code != http.StatusOK {
		t.Fatalf("want 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var out authgw.AuthResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode AuthResponse: %v: %s", err, rec.Body.String())
	}
	return out
}

func assertProblem(t *testing.T, rec *httptest.ResponseRecorder, wantStatus int, wantCode errcode.Code) {
	t.Helper()
	if rec.Code != wantStatus {
		t.Fatalf("want %d, got %d: %s", wantStatus, rec.Code, rec.Body.String())
	}
	if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, "application/problem+json") {
		t.Errorf("content-type: want application/problem+json, got %q", ct)
	}
	var p problemBody
	if err := json.Unmarshal(rec.Body.Bytes(), &p); err != nil {
		t.Fatalf("decode problem: %v: %s", err, rec.Body.String())
	}
	if p.Code != string(wantCode) {
		t.Errorf("code: want %s, got %q", wantCode, p.Code)
	}
}

func TestRefresh_ValidToken_ReturnsAuthResponseWhoseAccessTokenWorks(t *testing.T) {
	t.Setenv("JWT_SECRET", testSecret)
	userID := uuid.New()
	e := refreshEngine(usersRepo{user: &authgw.User{ID: userID, Email: "a@b.com", Name: "Kranti"}})

	got := decodeAuthResponse(t, postRefresh(t, e, refreshBody(t, mintFor(t, commonauth.TokenTypeRefresh, userID, time.Hour))))

	if got.UserInfo.ID != userID || got.UserInfo.Email != "a@b.com" {
		t.Errorf("userInfo: want the current user, got %+v", got.UserInfo)
	}
	if got.AccessExpiresIn != int64(time.Hour) || got.RefreshExpiresIn != int64(720*time.Hour) {
		t.Errorf("expiry fields must stay nanoseconds: access %d, refresh %d", got.AccessExpiresIn, got.RefreshExpiresIn)
	}
	if rec := do(t, e, http.MethodGet, "/api/users/profile", got.AccessToken); rec.Code != http.StatusOK {
		t.Errorf("new access token as bearer: want 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestRefresh_NewRefreshToken_HasFullLifetimeAndCanBeExchanged(t *testing.T) {
	t.Setenv("JWT_SECRET", testSecret)
	userID := uuid.New()
	e := refreshEngine(usersRepo{user: &authgw.User{ID: userID}})

	first := decodeAuthResponse(t, postRefresh(t, e, refreshBody(t, mintFor(t, commonauth.TokenTypeRefresh, userID, time.Minute))))

	claims, err := commonauth.ParseToken(first.RefreshToken, testSecret)
	if err != nil {
		t.Fatalf("parse new refresh token: %v", err)
	}
	exp, err := claims.GetExpirationTime()
	if err != nil {
		t.Fatalf("read exp: %v", err)
	}
	if left := time.Until(exp.Time); left < 720*time.Hour-time.Minute {
		t.Errorf("new refresh token has %v left, want a full 720h from the exchange", left)
	}

	decodeAuthResponse(t, postRefresh(t, e, refreshBody(t, first.RefreshToken)))
}

func TestRefresh_RejectedTokens_Return401Unauthenticated(t *testing.T) {
	t.Setenv("JWT_SECRET", testSecret)
	userID := uuid.New()
	e := refreshEngine(usersRepo{user: &authgw.User{ID: userID}})

	cases := map[string]string{
		"garbage":      "not-a-jwt",
		"expired":      mintFor(t, commonauth.TokenTypeRefresh, userID, -time.Minute),
		"access token": mintFor(t, commonauth.TokenTypeAccess, userID, time.Hour),
		// Not a shape failure: it reaches the token check (R3).
		"empty string": "",
	}
	for name, tok := range cases {
		t.Run(name, func(t *testing.T) {
			assertProblem(t, postRefresh(t, e, refreshBody(t, tok)), http.StatusUnauthorized, errcode.Unauthenticated)
		})
	}
}

func TestRefresh_DeletedUser_Returns401Unauthenticated(t *testing.T) {
	t.Setenv("JWT_SECRET", testSecret)
	notFound := fmt.Errorf("get user by id: %w", commonconstants.ErrNotFound)
	e := refreshEngine(usersRepo{err: notFound})

	rec := postRefresh(t, e, refreshBody(t, mintFor(t, commonauth.TokenTypeRefresh, uuid.New(), time.Hour)))

	assertProblem(t, rec, http.StatusUnauthorized, errcode.Unauthenticated)
}

func TestRefresh_BadShape_Returns422ValidationFailed(t *testing.T) {
	t.Setenv("JWT_SECRET", testSecret)
	e := refreshEngine(usersRepo{user: &authgw.User{ID: uuid.New()}})
	tok := mintFor(t, commonauth.TokenTypeRefresh, uuid.New(), time.Hour)

	cases := map[string]string{
		"missing refreshToken": `{}`,
		"unknown member":       fmt.Sprintf(`{"refreshToken":%q,"extra":true}`, tok),
	}
	for name, body := range cases {
		t.Run(name, func(t *testing.T) {
			assertProblem(t, postRefresh(t, e, body), http.StatusUnprocessableEntity, errcode.ValidationFailed)
		})
	}
}

func TestRefresh_RepositoryFailure_Returns500NotUnauthorized(t *testing.T) {
	t.Setenv("JWT_SECRET", testSecret)
	e := refreshEngine(usersRepo{err: errors.New("connection refused")})

	rec := postRefresh(t, e, refreshBody(t, mintFor(t, commonauth.TokenTypeRefresh, uuid.New(), time.Hour)))

	assertProblem(t, rec, http.StatusInternalServerError, errcode.Internal)
	if strings.Contains(rec.Body.String(), "connection refused") {
		t.Errorf("the cause must stay server-side: %s", rec.Body.String())
	}
}

func TestRefresh_ClientUnavailable_Returns503(t *testing.T) {
	t.Setenv("JWT_SECRET", testSecret)
	e := buildEngine(fakeProfileClient{user: sampleUser()})

	rec := postRefresh(t, e, refreshBody(t, "any-token"))

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("want 503, got %d: %s", rec.Code, rec.Body.String())
	}
}

// Both renderers on the real mount: the serialized group answers problem+json
// and a legacy route keeps its envelope.
func TestRefreshTokenAsBearer_Returns401InBothRenderers(t *testing.T) {
	t.Setenv("JWT_SECRET", testSecret)
	e := buildEngine(fakeProfileClient{user: sampleUser()})
	refresh := mintFor(t, commonauth.TokenTypeRefresh, uuid.New(), time.Hour)

	t.Run("problem+json", func(t *testing.T) {
		assertProblem(t, do(t, e, http.MethodGet, "/api/users/profile", refresh), http.StatusUnauthorized, errcode.Unauthenticated)
	})
	t.Run("legacy", func(t *testing.T) {
		rec := do(t, e, http.MethodGet, "/api"+legacyProbePath, refresh)
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("want 401, got %d", rec.Code)
		}
		if got := strings.TrimSpace(rec.Body.String()); got != `{"message":"unauthorized","statusCode":401}` {
			t.Errorf("legacy body changed: %s", got)
		}
	})
}

package config

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	pb "github.com/darkphotonKN/fireplace/common/api/proto/auth"
	commonauth "github.com/darkphotonKN/fireplace/common/auth"
	"github.com/darkphotonKN/fireplace/common/errcode"
	"github.com/darkphotonKN/fireplace/services/api-gateway/internal/auth"
	authgw "github.com/darkphotonKN/fireplace/services/api-gateway/internal/gateway/auth"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"golang.org/x/net/context"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// These exercise the REAL registration path (MountSerialized -> RegisterAPI),
// not a hand-built lookalike. That is what the SetupRouter/RegisterAPI split
// bought: the spike could only test a shape-equivalent engine because the real
// router needed Postgres, Consul and an API key. This does not.

const testSecret = "test-secret"

type fakeProfileClient struct {
	user *pb.User
	err  error
}

func (f fakeProfileClient) GetProfileProto(ctx context.Context, id uuid.UUID) (*pb.User, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.user, nil
}

func (f fakeProfileClient) UpdateProfileProto(ctx context.Context, id uuid.UUID, req authgw.UpdateProfileRequest) (*pb.User, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.user, nil
}

func sampleUser() *pb.User {
	dn, bio := "kn", "builds things"
	return &pb.User{
		Id:          uuid.New().String(),
		Email:       "a@b.com",
		Name:        "Kranti",
		DisplayName: &dn,
		Bio:         &bio,
		CreatedAt:   timestamppb.New(time.Unix(1700000000, 0)),
		UpdatedAt:   timestamppb.New(time.Unix(1800000000, 0)),
	}
}

// buildEngine mirrors SetupRouter's group structure and calls the same
// MountSerialized it does — but constructs no infrastructure.
func buildEngine(c authgw.ProfileClient) *gin.Engine {
	gin.SetMode(gin.TestMode)
	engine := gin.New()

	api := engine.Group("/api")
	api.POST("/users/signin", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"message": "public"}) })

	protected := api.Group("")
	protected.Use(auth.AuthMiddleware())
	// A legacy protected route, sibling to the serialized ones, using the
	// untouched middleware and the legacy envelope.
	protected.GET("/plans", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"statusCode": 200, "message": "ok", "result": []string{}})
	})

	MountSerialized(engine, APIDeps{Profile: c})
	return engine
}

func token(t *testing.T) string {
	t.Helper()
	tok, err := commonauth.GenerateJWT(uuid.New(), commonauth.TokenTypeAccess, testSecret, time.Hour)
	if err != nil {
		t.Fatalf("mint token: %v", err)
	}
	return tok
}

func do(t *testing.T, e *gin.Engine, method, path, bearer string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, nil)
	if bearer != "" {
		req.Header.Set("Authorization", "Bearer "+bearer)
	}
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	return rec
}

type problemBody struct {
	Type   string `json:"type"`
	Title  string `json:"title"`
	Status int    `json:"status"`
	Detail string `json:"detail"`
	Code   string `json:"code"`
	Errors []any  `json:"errors"`
}

// --- I-0002: mount, identity bridge, doc surface -------------------------

func TestMount_SerializedRouteInheritsAuth(t *testing.T) {
	t.Setenv("JWT_SECRET", testSecret)
	e := buildEngine(fakeProfileClient{user: sampleUser()})

	rec := do(t, e, http.MethodGet, "/api/users/profile", "")
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("want 401, got %d: %s", rec.Code, rec.Body.String())
	}
	// I-0003: the 401 fork must have replaced the legacy envelope.
	if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, "application/problem+json") {
		t.Errorf("401 content-type: want application/problem+json, got %q", ct)
	}
	var p problemBody
	if err := json.Unmarshal(rec.Body.Bytes(), &p); err != nil {
		t.Fatalf("decode problem: %v — %s", err, rec.Body.String())
	}
	if p.Code != string(errcode.Unauthenticated) {
		t.Errorf("code: want UNAUTHENTICATED, got %q", p.Code)
	}
	if p.Status != http.StatusUnauthorized {
		t.Errorf("status member: want 401, got %d", p.Status)
	}
}

func TestMount_LegacyProtectedRouteIsByteIdentical(t *testing.T) {
	t.Setenv("JWT_SECRET", testSecret)
	e := buildEngine(fakeProfileClient{user: sampleUser()})

	// The fork must NOT leak: legacy 401 keeps the old envelope exactly.
	rec := do(t, e, http.MethodGet, "/api/plans", "")
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("want 401, got %d", rec.Code)
	}
	if got := strings.TrimSpace(rec.Body.String()); got != `{"message":"unauthorized","statusCode":401}` {
		t.Errorf("legacy 401 body changed — the fork leaked.\ngot: %s", got)
	}
	if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, "application/json") {
		t.Errorf("legacy 401 content-type changed: %q", ct)
	}

	if rec := do(t, e, http.MethodGet, "/api/plans", token(t)); rec.Code != http.StatusOK {
		t.Errorf("legacy route with token: want 200, got %d", rec.Code)
	}
}

func TestMount_PublicSurfacesUnaffected(t *testing.T) {
	t.Setenv("JWT_SECRET", testSecret)
	e := buildEngine(fakeProfileClient{user: sampleUser()})

	t.Run("signin still public", func(t *testing.T) {
		if rec := do(t, e, http.MethodPost, "/api/users/signin", ""); rec.Code != http.StatusOK {
			t.Fatalf("want 200, got %d", rec.Code)
		}
	})
	t.Run("openapi document reachable WITHOUT a token", func(t *testing.T) {
		rec := do(t, e, http.MethodGet, "/api/openapi.yaml", "")
		if rec.Code != http.StatusOK {
			t.Fatalf("want 200, got %d", rec.Code)
		}
		if !strings.Contains(rec.Body.String(), "openapi: 3") {
			t.Errorf("not an OpenAPI 3 document:\n%s", rec.Body.String()[:200])
		}
	})
	t.Run("docs UI reachable WITHOUT a token", func(t *testing.T) {
		if rec := do(t, e, http.MethodGet, "/api/docs", ""); rec.Code != http.StatusOK {
			t.Fatalf("want 200, got %d", rec.Code)
		}
	})
}

// --- I-0004: GET /users/profile ------------------------------------------

func TestGetProfile_BareResourceNoEnvelope(t *testing.T) {
	t.Setenv("JWT_SECRET", testSecret)
	u := sampleUser()
	e := buildEngine(fakeProfileClient{user: u})

	rec := do(t, e, http.MethodGet, "/api/users/profile", token(t))
	if rec.Code != http.StatusOK {
		t.Fatalf("want 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var raw map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &raw); err != nil {
		t.Fatalf("decode: %v", err)
	}
	for _, banned := range []string{"statusCode", "message", "result", "password", "$schema"} {
		if _, present := raw[banned]; present {
			t.Errorf("response must not contain %q — got %v", banned, raw)
		}
	}
	want := map[string]any{
		"id": u.Id, "email": "a@b.com", "name": "Kranti",
		"displayName": "kn", "bio": "builds things",
	}
	for k, v := range want {
		if raw[k] != v {
			t.Errorf("%s: want %v, got %v", k, v, raw[k])
		}
	}
	// camelCase, not the entity's snake_case created_at/updated_at.
	if _, ok := raw["createdAt"]; !ok {
		t.Error("missing createdAt (entity used created_at — the contract renames it)")
	}
	if len(raw) != 7 {
		t.Errorf("FS-0002 R6 says exactly 7 published fields, got %d: %v", len(raw), raw)
	}
}

func TestGetProfile_IdentityComesFromRequestContext(t *testing.T) {
	t.Setenv("JWT_SECRET", testSecret)
	var seen uuid.UUID
	spy := profileClientFunc(func(ctx context.Context, id uuid.UUID) (*pb.User, error) {
		seen = id
		return sampleUser(), nil
	})
	e := buildEngine(spy)

	userID := uuid.New()
	tok, err := commonauth.GenerateJWT(userID, commonauth.TokenTypeAccess, testSecret, time.Hour)
	if err != nil {
		t.Fatalf("mint: %v", err)
	}
	if rec := do(t, e, http.MethodGet, "/api/users/profile", tok); rec.Code != http.StatusOK {
		t.Fatalf("want 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if seen != userID {
		t.Fatalf("identity bridge failed: handler saw %v, token carried %v", seen, userID)
	}
}

func TestGetProfile_DownstreamErrorsBecomeProblems(t *testing.T) {
	t.Setenv("JWT_SECRET", testSecret)
	cases := []struct {
		name       string
		err        error
		wantStatus int
		wantCode   errcode.Code
		mustNotLeak string
	}{
		{"deleted user", status.Error(codes.NotFound, "user 123 gone"), http.StatusNotFound, errcode.NotFound, "user 123 gone"},
		{"auth-service down", status.Error(codes.Unavailable, "dial tcp refused"), http.StatusInternalServerError, errcode.Internal, "dial tcp refused"},
		{"internal", status.Error(codes.Internal, "boom"), http.StatusInternalServerError, errcode.Internal, "boom"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			e := buildEngine(fakeProfileClient{err: tc.err})
			rec := do(t, e, http.MethodGet, "/api/users/profile", token(t))

			if rec.Code != tc.wantStatus {
				t.Errorf("status: want %d, got %d", tc.wantStatus, rec.Code)
			}
			if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, "application/problem+json") {
				t.Errorf("content-type: want problem+json, got %q", ct)
			}
			var p problemBody
			if err := json.Unmarshal(rec.Body.Bytes(), &p); err != nil {
				t.Fatalf("decode: %v — %s", err, rec.Body.String())
			}
			if p.Code != string(tc.wantCode) {
				t.Errorf("code: want %s, got %s", tc.wantCode, p.Code)
			}
			if p.Title == "" {
				t.Error("title empty")
			}
			if strings.Contains(rec.Body.String(), tc.mustNotLeak) {
				t.Errorf("leaked downstream detail %q: %s", tc.mustNotLeak, rec.Body.String())
			}
		})
	}
}

// profileClientFunc adapts a func to ProfileClient for the identity spy.
type profileClientFunc func(ctx context.Context, id uuid.UUID) (*pb.User, error)

func (f profileClientFunc) GetProfileProto(ctx context.Context, id uuid.UUID) (*pb.User, error) {
	return f(ctx, id)
}
func (f profileClientFunc) UpdateProfileProto(ctx context.Context, id uuid.UUID, _ authgw.UpdateProfileRequest) (*pb.User, error) {
	return f(ctx, id)
}

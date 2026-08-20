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
	authgw "github.com/darkphotonKN/fireplace/services/api-gateway/internal/gateway/auth"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"golang.org/x/net/context"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// I-0005: PATCH /api/users/profile semantics.
//
// SCOPE NOTE: the "unchanged" outcomes are auth-service's behaviour (its
// repository builds a dynamic SET clause and omits nil fields). What the
// GATEWAY decides — and therefore what these tests pin — is how JSON decodes
// into the pointers it forwards. That decoding is what determines the
// downstream outcome, so it is the right thing to assert here.

// recordingClient captures the decoded request the gateway forwarded.
type recordingClient struct {
	got  authgw.UpdateProfileRequest
	user *pb.User
	err  error
}

func (r *recordingClient) GetProfileProto(ctx context.Context, id uuid.UUID) (*pb.User, error) {
	return r.user, r.err
}

func (r *recordingClient) UpdateProfileProto(ctx context.Context, id uuid.UUID, req authgw.UpdateProfileRequest) (*pb.User, error) {
	r.got = req
	if r.err != nil {
		return nil, r.err
	}
	// Mirror auth-service's ONE validation rule so the 400 path is real.
	if req.Name != nil && *req.Name == "" {
		return nil, status.Error(codes.InvalidArgument, "name cannot be empty")
	}
	return r.user, nil
}

func patchReq(t *testing.T, e *gin.Engine, body, bearer string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPatch, "/api/users/profile", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if bearer != "" {
		req.Header.Set("Authorization", "Bearer "+bearer)
	}
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	return rec
}

func ptr(s string) *string { return &s }

func TestPatchProfile_SemanticsTable(t *testing.T) {
	t.Setenv("JWT_SECRET", testSecret)

	cases := []struct {
		name       string
		body       string
		wantStatus int
		wantBio    *string // what the gateway forwarded
		wantName   *string
	}{
		{"sets bio", `{"bio":"hi"}`, http.StatusOK, ptr("hi"), nil},
		{"empty string sets empty", `{"bio":""}`, http.StatusOK, ptr(""), nil},
		{"null is IGNORED — indistinguishable from absent", `{"bio":null}`, http.StatusOK, nil, nil},
		{"empty body is a valid no-op", `{}`, http.StatusOK, nil, nil},
		{"sets name", `{"name":"New"}`, http.StatusOK, nil, ptr("New")},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rc := &recordingClient{user: sampleUser()}
			e := buildEngine(rc)

			rec := patchReq(t, e, tc.body, token(t))
			if rec.Code != tc.wantStatus {
				t.Fatalf("status: want %d, got %d — %s", tc.wantStatus, rec.Code, rec.Body.String())
			}
			if !ptrEq(rc.got.Bio, tc.wantBio) {
				t.Errorf("forwarded bio: want %v, got %v", show(tc.wantBio), show(rc.got.Bio))
			}
			if !ptrEq(rc.got.Name, tc.wantName) {
				t.Errorf("forwarded name: want %v, got %v", show(tc.wantName), show(rc.got.Name))
			}
			// Success is always a bare resource.
			var raw map[string]any
			if err := json.Unmarshal(rec.Body.Bytes(), &raw); err != nil {
				t.Fatalf("decode: %v", err)
			}
			if _, bad := raw["result"]; bad {
				t.Error("response is enveloped — must be a bare resource")
			}
			if len(raw) != 7 {
				t.Errorf("want 7 published fields, got %d", len(raw))
			}
		})
	}
}

func TestPatchProfile_EmptyNameIs400NotUnprocessable(t *testing.T) {
	t.Setenv("JWT_SECRET", testSecret)
	rc := &recordingClient{user: sampleUser()}
	e := buildEngine(rc)

	rec := patchReq(t, e, `{"name":""}`, token(t))

	// The whole point of leaving validation downstream: the status must NOT
	// move to huma's 422.
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("want 400 (validation stays downstream), got %d — %s", rec.Code, rec.Body.String())
	}
	if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, "application/problem+json") {
		t.Errorf("content-type: want problem+json, got %q", ct)
	}

	var p problemBody
	if err := json.Unmarshal(rec.Body.Bytes(), &p); err != nil {
		t.Fatalf("decode: %v — %s", err, rec.Body.String())
	}
	if p.Code != string(errcode.ProfileNameEmpty) {
		t.Errorf("code: want PROFILE_NAME_EMPTY, got %q", p.Code)
	}
	// errors[] is present and EMPTY: gRPC carries no structured field detail,
	// which is exactly why the code has to carry the precision (FS-0002 R16).
	if p.Errors == nil {
		t.Error("errors[] should be present (empty), not absent")
	}
	if len(p.Errors) != 0 {
		t.Errorf("errors[] should be empty for a downstream failure, got %v", p.Errors)
	}
	if strings.Contains(rec.Body.String(), "name cannot be empty") {
		t.Errorf("leaked downstream text: %s", rec.Body.String())
	}
}

// STRICT input validation: an undeclared member is a 422.
//
// This is the correct status for "syntactically valid, semantically
// unprocessable". The alternative — silently ignoring it — returns 200 for a
// typo'd PATCH, so the user believes they saved a change that never happened.
// That failure is invisible; this one names itself.
func TestPatchProfile_UnknownFieldIsRejected(t *testing.T) {
	t.Setenv("JWT_SECRET", testSecret)

	cases := []struct {
		name string
		body string
	}{
		{"typo'd field name", `{"biio":"my new bio"}`},
		{"stray member alongside a valid one", `{"bio":"hi","nope":1}`},
		{"read-only field echoed back", `{"name":"Edited","id":"9f2c"}`},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rc := &recordingClient{user: sampleUser()}
			e := buildEngine(rc)

			rec := patchReq(t, e, tc.body, token(t))
			if rec.Code != http.StatusUnprocessableEntity {
				t.Fatalf("want 422, got %d — %s", rec.Code, rec.Body.String())
			}
			if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, "application/problem+json") {
				t.Errorf("content-type: want problem+json, got %q", ct)
			}
			var p problemBody
			if err := json.Unmarshal(rec.Body.Bytes(), &p); err != nil {
				t.Fatalf("decode: %v", err)
			}
			if p.Code != string(errcode.ValidationFailed) {
				t.Errorf("code: want VALIDATION_FAILED, got %q", p.Code)
			}
			// Nothing reached the downstream service.
			if rc.got.Name != nil || rc.got.Bio != nil {
				t.Errorf("rejected request must not be forwarded, got %+v", rc.got)
			}
		})
	}
}

// The strictness above is only safe because a client CANNOT send an undeclared
// field by accident — it consumes generated types. `id` is not updatable and is
// therefore absent from the request type by design: identity comes from the JWT.
func TestPatchProfile_IdIsNotAcceptedFromTheBody(t *testing.T) {
	t.Setenv("JWT_SECRET", testSecret)
	var seen uuid.UUID
	rc := &spyClient{user: sampleUser(), onUpdate: func(id uuid.UUID) { seen = id }}
	e := buildEngine(rc)

	tokenUser := uuid.New()
	tok, err := commonauth.GenerateJWT(tokenUser, commonauth.TokenTypeAccess, testSecret, time.Hour)
	if err != nil {
		t.Fatalf("mint: %v", err)
	}
	// A body claiming to be someone else must not change whose profile is updated.
	rec := patchReqRaw(t, e, `{"name":"Edited"}`, tok)
	if rec.Code != http.StatusOK {
		t.Fatalf("want 200, got %d — %s", rec.Code, rec.Body.String())
	}
	if seen != tokenUser {
		t.Fatalf("identity must come from the token: updated %v, token carried %v", seen, tokenUser)
	}
}

type spyClient struct {
	user     *pb.User
	onUpdate func(uuid.UUID)
}

func (s *spyClient) GetProfileProto(ctx context.Context, id uuid.UUID) (*pb.User, error) {
	return s.user, nil
}
func (s *spyClient) UpdateProfileProto(ctx context.Context, id uuid.UUID, _ authgw.UpdateProfileRequest) (*pb.User, error) {
	s.onUpdate(id)
	return s.user, nil
}

func patchReqRaw(t *testing.T, e *gin.Engine, body, bearer string) *httptest.ResponseRecorder {
	return patchReq(t, e, body, bearer)
}

func ptrEq(a, b *string) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return *a == *b
}

func show(p *string) string {
	if p == nil {
		return "<nil>"
	}
	return `"` + *p + `"`
}

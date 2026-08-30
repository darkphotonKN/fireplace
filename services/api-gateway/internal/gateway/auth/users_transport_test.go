package authgw

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	pb "github.com/darkphotonKN/fireplace/common/api/proto/auth"
	"github.com/darkphotonKN/fireplace/services/api-gateway/internal/models"
	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// --- legacy reference shapes ------------------------------------------------
//
// These two converters produced the body the gateway published BEFORE the
// serialization retrofit, and they are the "before" side of every comparison
// below — the baseline that proves FS-0004 R6 preservation, not production
// code. They lived on the authgw gRPC Client until auth-service was folded
// back in-process (ADR-0009 §1) and that client was deleted. They moved here
// rather than being deleted with it: without them these tests still compile
// against nothing and silently stop checking anything.

func userFromProto(u *pb.User) *models.User {
	id, _ := uuid.Parse(u.Id)
	return &models.User{
		BaseDBDateModel: models.BaseDBDateModel{
			ID:        id,
			CreatedAt: u.CreatedAt.AsTime(),
			UpdatedAt: u.UpdatedAt.AsTime(),
		},
		Email:       u.Email,
		Name:        u.Name,
		DisplayName: u.DisplayName,
		Bio:         u.Bio,
	}
}

func authRespToHTTP(r *pb.AuthResponse) *LoginResponse {
	return &LoginResponse{
		AccessToken:      r.AccessToken,
		RefreshToken:     r.RefreshToken,
		AccessExpiresIn:  r.AccessExpiresIn,
		RefreshExpiresIn: r.RefreshExpiresIn,
		UserInfo:         userFromProto(r.User),
	}
}

// populatedProtoUser returns a user with EVERY field set to a distinct non-zero
// value.
//
// Populated is the whole point. Optional fields carry omitempty, so a
// zero-valued fixture marshals to {} on both sides and the comparison passes
// while proving nothing — the failure mode that makes transcription tests
// worthless at exactly the scale where transcription errors happen.
func populatedProtoUser() *pb.User {
	display := "kn"
	bio := "builds things"
	return &pb.User{
		Id:          "550e8400-e29b-41d4-a716-446655440000",
		Email:       "a@b.com",
		Name:        "Kranti",
		DisplayName: &display,
		Bio:         &bio,
		CreatedAt:   timestamppb.New(time.Date(2026, 6, 1, 12, 30, 0, 0, time.UTC)),
		UpdatedAt:   timestamppb.New(time.Date(2026, 6, 2, 9, 15, 0, 0, time.UTC)),
	}
}

func marshalMap(t *testing.T, v any) map[string]any {
	t.Helper()
	raw, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	return m
}

// The retrofit is behaviour-preserving at the value level (FS-0004 R6): every
// value the legacy body published must still be published, and nothing new may
// appear. Field NAMES have one declared exception, asserted separately below.
func TestUserResponse_CarriesEveryLegacyValue(t *testing.T) {
	u := populatedProtoUser()

	legacy := marshalMap(t, userFromProto(u)) // what `result` holds today
	now := marshalMap(t, userResponseFromProto(u))

	// The one deliberate rename in this slice: models.User embeds
	// BaseDBDateModel, whose tags are snake_case, while ProfileResponse —
	// already published by FS-0002 for the same entity — is camelCase.
	// Transcribing literally would publish both spellings of one entity's
	// timestamps in a single document, which is a defect the retrofit would be
	// INTRODUCING rather than preserving.
	renamed := map[string]string{"created_at": "createdAt", "updated_at": "updatedAt"}

	for key, want := range legacy {
		lookup := key
		if to, ok := renamed[key]; ok {
			lookup = to
		}
		got, present := now[lookup]
		if !present {
			t.Errorf("field %q (published as %q) is missing from UserResponse", key, lookup)
			continue
		}
		if got != want {
			t.Errorf("field %q: legacy = %v, serialized = %v", key, want, got)
		}
	}

	if len(now) != len(legacy) {
		t.Errorf("field count changed: legacy has %d, serialized has %d — a retrofit adds nothing\nlegacy: %v\nnew:    %v",
			len(legacy), len(now), keysOf(legacy), keysOf(now))
	}
}

// models.User carries `json:"password,omitempty"` and is today the response
// type for signin, getUser and listUsers. It does not leak — userFromProto
// never assigns it and pb.User has no such field — but it is one careless
// assignment away. Serializing removes the hazard structurally; this test is
// what keeps it removed.
func TestUserResponse_NeverPublishesPassword(t *testing.T) {
	raw, err := json.Marshal(userResponseFromProto(populatedProtoUser()))
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if strings.Contains(strings.ToLower(string(raw)), "password") {
		t.Fatalf("UserResponse published a password field: %s", raw)
	}
}

// One entity, one spelling. ProfileResponse is already in the document; a
// second user-shaped type disagreeing with it on date keys is precisely the
// drift the contract layer exists to prevent.
func TestUserResponse_DateKeysMatchProfileResponse(t *testing.T) {
	u := populatedProtoUser()

	profile := marshalMap(t, profileFromProto(u))
	user := marshalMap(t, userResponseFromProto(u))

	for _, key := range []string{"createdAt", "updatedAt"} {
		if _, ok := profile[key]; !ok {
			t.Fatalf("ProfileResponse is missing %q — this test's premise is wrong", key)
		}
		if _, ok := user[key]; !ok {
			t.Errorf("UserResponse is missing %q; it must match ProfileResponse", key)
		}
	}
	for _, key := range []string{"created_at", "updated_at"} {
		if _, ok := user[key]; ok {
			t.Errorf("UserResponse still publishes snake_case %q alongside the camelCase form", key)
		}
	}
}

// AuthResponse is what signin returns. Its embedded user must be the same
// published shape as everywhere else — not models.User, which is what the
// legacy LoginResponse embedded.
func TestAuthResponse_CarriesEveryLegacyValue(t *testing.T) {
	resp := &pb.AuthResponse{
		User:             populatedProtoUser(),
		AccessToken:      "access-token-value",
		RefreshToken:     "refresh-token-value",
		AccessExpiresIn:  900000000000,
		RefreshExpiresIn: 604800000000000,
	}

	legacy := marshalMap(t, authRespToHTTP(resp))
	now := marshalMap(t, authResponseFromProto(resp))

	for _, key := range []string{"accessToken", "refreshToken", "accessExpiresIn", "refreshExpiresIn"} {
		if legacy[key] != now[key] {
			t.Errorf("field %q: legacy = %v, serialized = %v", key, legacy[key], now[key])
		}
	}
	if _, ok := now["userInfo"]; !ok {
		t.Error("userInfo is missing; the legacy body published it and the retrofit preserves names")
	}
}

func keysOf(m map[string]any) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}

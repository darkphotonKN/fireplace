package plangw

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
)

// The legacy body for a plan, exactly as it shipped, frozen as data.
//
// It is a literal rather than a call into the old mapping code because that
// code is deleted by this slice. A migration's baseline has to outlive the
// thing it replaces, so it is recorded here as a fact instead of derived from
// an implementation that no longer exists.
const legacyPlanBody = `{
  "id":          "550e8400-e29b-41d4-a716-446655440000",
  "userId":      "6ba7b810-9dad-11d1-80b4-00c04fd430c8",
  "name":        "Ship the contract layer",
  "focus":       "engineering",
  "description": "Retrofit the gateway surface",
  "planType":    "project",
  "dailyReset":  true,
  "created_at":  "2026-06-01T12:30:00Z",
  "updated_at":  "2026-06-02T09:15:00Z"
}`

func populatedPlan() PlanResp {
	return PlanResp{
		ID:          uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"),
		UserID:      uuid.MustParse("6ba7b810-9dad-11d1-80b4-00c04fd430c8"),
		Name:        "Ship the contract layer",
		Focus:       "engineering",
		Description: "Retrofit the gateway surface",
		PlanType:    "project",
		DailyReset:  true,
		CreatedAt:   time.Date(2026, 6, 1, 12, 30, 0, 0, time.UTC),
		UpdatedAt:   time.Date(2026, 6, 2, 9, 15, 0, 0, time.UTC),
	}
}

func asMap(t *testing.T, v any) map[string]any {
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

// Every value the legacy body published must still be published, under the
// same key — with two declared renames.
func TestPlanResp_CarriesEveryLegacyValue(t *testing.T) {
	var legacy map[string]any
	if err := json.Unmarshal([]byte(legacyPlanBody), &legacy); err != nil {
		t.Fatalf("legacy fixture is not valid JSON: %v", err)
	}
	now := asMap(t, populatedPlan())

	// The same rename applied to the users group: date keys become camelCase so
	// one entity is not published under two spellings across one document.
	// ChecklistResp already spells its dates camelCase (startDate, dueDate), so
	// PlanResp was the odd one out even within its own package.
	renamed := map[string]string{"created_at": "createdAt", "updated_at": "updatedAt"}

	for key, want := range legacy {
		lookup := key
		if to, ok := renamed[key]; ok {
			lookup = to
		}
		got, present := now[lookup]
		if !present {
			t.Errorf("field %q (published as %q) is missing from PlanResp", key, lookup)
			continue
		}
		if got != want {
			t.Errorf("field %q: legacy = %v, serialized = %v", key, want, got)
		}
	}

	if len(now) != len(legacy) {
		t.Errorf("field count changed: legacy %d, serialized %d — a retrofit adds nothing", len(legacy), len(now))
	}
}

func TestPlanResp_DateKeysAreCamelCase(t *testing.T) {
	m := asMap(t, populatedPlan())
	for _, want := range []string{"createdAt", "updatedAt"} {
		if _, ok := m[want]; !ok {
			t.Errorf("PlanResp is missing %q", want)
		}
	}
	for _, gone := range []string{"created_at", "updated_at"} {
		if _, ok := m[gone]; ok {
			t.Errorf("PlanResp still publishes snake_case %q", gone)
		}
	}
}

// Huma derives "required" from the ABSENCE of omitempty, while gin derives it
// from `binding:"required"`. Those are independent, and where they disagree the
// document lies about the API.
//
// CreatePlanReq.Description is the live case: gin never required it, but it
// carries no omitempty, so huma would publish it as required — turning an
// optional field into a 422 for anyone who trusts the contract.
func TestCreatePlanReq_OptionalFieldsAreNotPublishedAsRequired(t *testing.T) {
	required := map[string]bool{"name": true, "focus": true, "planType": true}

	for _, f := range jsonFieldsOf(t, CreatePlanReq{}) {
		humaRequired := !f.omitempty
		if humaRequired != required[f.name] {
			t.Errorf("field %q: gin requires=%v but huma would publish required=%v — "+
				"add omitempty to make it optional, or binding:\"required\" to make it required",
				f.name, required[f.name], humaRequired)
		}
	}
}

// UpdatePlanReq is a partial update: nothing may be required, or an empty PATCH
// body stops being a valid no-op.
func TestUpdatePlanReq_NothingIsRequired(t *testing.T) {
	for _, f := range jsonFieldsOf(t, UpdatePlanReq{}) {
		if !f.omitempty {
			t.Errorf("field %q lacks omitempty, so huma publishes it as required on a PATCH", f.name)
		}
	}
}

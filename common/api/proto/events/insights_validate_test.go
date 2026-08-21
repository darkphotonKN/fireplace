package events

import "testing"

// Valid ids for every graph-shape test. Placeholders would make those tests pass
// on the identifier check instead of the check they name — they would stay green
// with the two-tier rule deleted.
const (
	planID = "3f1a2b3c-4d5e-6f70-8192-a3b4c5d6e7f8"
	userID = "8c7b6a59-4d3e-2f10-9182-736455647382"
)

// item is a terse constructor so the tables below read as the shapes they describe.
func item(desc string, parent *int32) *InsightItem {
	return &InsightItem{Description: desc, Scope: "longterm", Type: "task", ParentIndex: parent}
}

func idx(i int32) *int32 { return &i }

// ev builds a well-addressed event so each test varies only its items.
func ev(items ...*InsightItem) *InsightGeneratedEvent {
	return &InsightGeneratedEvent{PlanId: planID, UserId: userID, Items: items}
}

func TestInsightGeneratedEvent_Validate_RejectsIllegalGraphs(t *testing.T) {
	tests := []struct {
		name  string
		items []*InsightItem
	}{
		// 0 is top-level, 1 nests under 0, 2 tries to nest under 1 — a third tier.
		{"third tier", []*InsightItem{item("top", nil), item("child", idx(0)), item("grandchild", idx(1))}},
		{"parent past the end", []*InsightItem{item("top", nil), item("orphan", idx(5))}},
		{"negative parent", []*InsightItem{item("top", nil), item("orphan", idx(-1))}},
		{"its own parent", []*InsightItem{item("top", nil), item("ouroboros", idx(1))}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := ev(tt.items...).Validate(); err == nil {
				t.Fatal("want error, got nil")
			}
		})
	}
}

func TestInsightGeneratedEvent_Validate_AcceptsLegalShapes(t *testing.T) {
	tests := []struct {
		name  string
		items []*InsightItem
	}{
		// A generation that produced nothing usable is a valid end state, not a
		// failure — the consumer marks the plan ready with zero items.
		{"no items at all", nil},
		{"flat, no nesting", []*InsightItem{item("a", nil), item("b", nil)}},
		{"two tiers", []*InsightItem{item("parent", nil), item("child", idx(0))}},
		// Index 0 is a legal parent. This is the case that breaks if parent_index
		// is not proto3-`optional`, because unset and 0 become indistinguishable.
		{"parent is index zero", []*InsightItem{item("parent", nil), item("child", idx(0))}},
		{"children before their parent in the array", []*InsightItem{item("child", idx(1)), item("parent", nil)}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := ev(tt.items...).Validate(); err != nil {
				t.Fatalf("want nil, got %v", err)
			}
		})
	}
}

func TestInsightGeneratedEvent_Validate_RejectsBadIdentifiers(t *testing.T) {
	tests := []struct {
		name, planID, userID string
	}{
		{"plan id not a uuid", "not-a-uuid", userID},
		{"user id not a uuid", planID, "42"},
		{"plan id empty", "", userID},
		{"user id empty", planID, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := &InsightGeneratedEvent{PlanId: tt.planID, UserId: tt.userID, Items: []*InsightItem{item("a", nil)}}
			if err := e.Validate(); err == nil {
				t.Fatal("want error, got nil")
			}
		})
	}
}

// The payload is the CREATABLE SUBSET of a checklist item. This pins that
// literally, because the failure mode is additive and well-intentioned: someone
// adds `sequence` for clarity (FS-0006 R19c — array order IS sequence, and a
// field can disagree with the order it sits in), or `start_date` because
// scheduling seems useful (R19d — AI scheduling is unshipped), or `id` because
// every other message has one. Each would pass review as a small convenience.
func TestInsightItem_CarriesOnlyCreatableFields(t *testing.T) {
	want := map[string]bool{"description": true, "scope": true, "type": true, "parent_index": true}

	fields := (&InsightItem{}).ProtoReflect().Descriptor().Fields()
	got := map[string]bool{}
	for i := 0; i < fields.Len(); i++ {
		got[string(fields.Get(i).Name())] = true
	}

	for name := range got {
		if !want[name] {
			t.Errorf("InsightItem has field %q — the database owns that, or order already carries it", name)
		}
	}
	for name := range want {
		if !got[name] {
			t.Errorf("InsightItem is missing field %q", name)
		}
	}
}

func TestInsightGenerationFailedEvent_Validate(t *testing.T) {
	tests := []struct {
		name    string
		event   *InsightGenerationFailedEvent
		wantErr bool
	}{
		{"exhausted", &InsightGenerationFailedEvent{PlanId: planID, UserId: userID,
			FailureClass: FailureClass_FAILURE_CLASS_GENERATION_EXHAUSTED}, false},
		{"stalled", &InsightGenerationFailedEvent{PlanId: planID, UserId: userID,
			FailureClass: FailureClass_FAILURE_CLASS_STALLED}, false},
		// A producer that forgot to set the class is a bug. Accepting it would
		// show the user generic copy for a failure nobody classified.
		{"unspecified class", &InsightGenerationFailedEvent{PlanId: planID, UserId: userID}, true},
		{"bad plan id", &InsightGenerationFailedEvent{PlanId: "nope", UserId: userID,
			FailureClass: FailureClass_FAILURE_CLASS_STALLED}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.event.Validate()
			if tt.wantErr && err == nil {
				t.Fatal("want error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("want nil, got %v", err)
			}
		})
	}
}

// R37 makes failure_class decide whether a retry action appears. An unrecognised
// class must fail toward NOT offering a button that spends money.
func TestFailureClass_Retryable(t *testing.T) {
	tests := []struct {
		class FailureClass
		want  bool
	}{
		{FailureClass_FAILURE_CLASS_GENERATION_EXHAUSTED, true},
		{FailureClass_FAILURE_CLASS_STALLED, true},
		{FailureClass_FAILURE_CLASS_UNSPECIFIED, false},
		{FailureClass(9999), false},
	}
	for _, tt := range tests {
		t.Run(tt.class.String(), func(t *testing.T) {
			if got := tt.class.Retryable(); got != tt.want {
				t.Errorf("Retryable() = %v, want %v", got, tt.want)
			}
		})
	}
}

package events

import "testing"

// item is a terse constructor so the tables below read as the shapes they describe.
func item(desc string, parent *int32) *GeneratedItem {
	return &GeneratedItem{Description: desc, Scope: "longterm", Type: "task", ParentIndex: parent}
}

func idx(i int32) *int32 { return &i }

func TestInsightGeneratedEvent_Validate_RejectsThirdTier(t *testing.T) {
	// 0 is top-level, 1 nests under 0, 2 tries to nest under 1 — a third tier.
	ev := &InsightGeneratedEvent{
		PlanId: "p", UserId: "u",
		Items: []*GeneratedItem{
			item("top", nil),
			item("child", idx(0)),
			item("grandchild", idx(1)),
		},
	}
	if err := ev.Validate(); err == nil {
		t.Fatal("want error for a parent_index pointing at an item that is itself nested, got nil")
	}
}

func TestInsightGeneratedEvent_Validate_RejectsOutOfRangeParent(t *testing.T) {
	tests := []struct {
		name   string
		parent *int32
	}{
		{"past the end", idx(5)},
		{"negative", idx(-1)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ev := &InsightGeneratedEvent{
				PlanId: "p", UserId: "u",
				Items: []*GeneratedItem{item("top", nil), item("orphan", tt.parent)},
			}
			if err := ev.Validate(); err == nil {
				t.Fatalf("want error for parent_index %d, got nil", *tt.parent)
			}
		})
	}
}

func TestInsightGeneratedEvent_Validate_AcceptsLegalShapes(t *testing.T) {
	tests := []struct {
		name  string
		items []*GeneratedItem
	}{
		// A generation that produced nothing usable is a valid end state, not a
		// failure — the consumer marks the plan ready with zero items.
		{"no items at all", nil},
		{"flat, no nesting", []*GeneratedItem{item("a", nil), item("b", nil)}},
		{"two tiers", []*GeneratedItem{item("parent", nil), item("child", idx(0))}},
		// Index 0 is a legal parent. This is the case that fails if parent_index
		// is not proto3-`optional`, because unset and 0 become indistinguishable.
		{"parent is index zero", []*GeneratedItem{item("parent", nil), item("child", idx(0))}},
		{"children before their parent in the array", []*GeneratedItem{item("child", idx(1)), item("parent", nil)}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ev := &InsightGeneratedEvent{PlanId: "p", UserId: "u", Items: tt.items}
			if err := ev.Validate(); err != nil {
				t.Fatalf("want nil, got %v", err)
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
func TestGeneratedItem_CarriesOnlyCreatableFields(t *testing.T) {
	want := map[string]bool{"description": true, "scope": true, "type": true, "parent_index": true}

	fields := (&GeneratedItem{}).ProtoReflect().Descriptor().Fields()
	got := map[string]bool{}
	for i := 0; i < fields.Len(); i++ {
		got[string(fields.Get(i).Name())] = true
	}

	for name := range got {
		if !want[name] {
			t.Errorf("GeneratedItem has field %q — the database owns that, or order already carries it", name)
		}
	}
	for name := range want {
		if !got[name] {
			t.Errorf("GeneratedItem is missing field %q", name)
		}
	}
}

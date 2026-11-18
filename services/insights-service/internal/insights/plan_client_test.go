package insights

import (
	"testing"
	"time"

	planpb "github.com/darkphotonKN/fireplace/common/api/proto/plan"
	"github.com/darkphotonKN/fireplace/services/insights-service/internal/dayplan"
	"github.com/darkphotonKN/fireplace/services/insights-service/internal/notesummary"
	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestToPlanItems_CarriesTheFieldsThePlanTreeNeeds(t *testing.T) {
	parent, child := uuid.New(), uuid.New()
	parentStr := parent.String()
	due := time.Date(2026, 10, 30, 0, 0, 0, 0, time.UTC)
	updated := time.Date(2026, 10, 24, 9, 30, 0, 0, time.UTC)

	got, err := toPlanItems([]*planpb.ChecklistItem{
		{Id: parent.String(), Description: "Ship", Type: "task", Scope: "longterm", DueDate: timestamppb.New(due), UpdatedAt: timestamppb.New(updated)},
		{Id: child.String(), ParentId: &parentStr, Description: "Step", Type: "note", Scope: "daily", Done: true, Archived: true},
	})
	if err != nil {
		t.Fatal(err)
	}

	if len(got) != 2 {
		t.Fatalf("got %d items, want 2 (the client does not filter)", len(got))
	}
	p, c := got[0], got[1]
	if p.ID != parent || p.ParentID != nil || p.DueDate == nil || !p.DueDate.Equal(due) || p.Type != "task" || p.Scope != "longterm" || !p.UpdatedAt.Equal(updated) {
		t.Fatalf("parent = %+v", p)
	}
	if c.ParentID == nil || *c.ParentID != parent || !c.Done || !c.Archived || c.DueDate != nil || c.Type != "note" || c.Scope != "daily" {
		t.Fatalf("child = %+v", c)
	}
}

func TestToPlanItems_MalformedIdIsAnError(t *testing.T) {
	if _, err := toPlanItems([]*planpb.ChecklistItem{{Id: "nope"}}); err == nil {
		t.Fatal("want an error for a malformed item id")
	}
}

func TestToPlanProfile_CarriesPlanTypeAndFocus(t *testing.T) {
	got := toPlanProfile(&planpb.Plan{Id: "p", Name: "Algorithms", Focus: "Graphs first", PlanType: "learning"})

	if got != (dayplan.PlanProfile{Type: "learning", Focus: "Graphs first"}) {
		t.Fatalf("profile = %+v", got)
	}
}

// FS-PGG2N R39 (ADR-0015): only owner and member may read; none and an
// unanswered relation fail closed.
func TestOwnerOrMember_OnlyOwnerAndMember(t *testing.T) {
	cases := map[planpb.PlanRelation]bool{
		planpb.PlanRelation_PLAN_RELATION_OWNER:       true,
		planpb.PlanRelation_PLAN_RELATION_MEMBER:      true,
		planpb.PlanRelation_PLAN_RELATION_NONE:        false,
		planpb.PlanRelation_PLAN_RELATION_UNSPECIFIED: false,
	}
	for rel, want := range cases {
		if got := ownerOrMember(rel); got != want {
			t.Errorf("ownerOrMember(%v) = %v, want %v", rel, got, want)
		}
	}
}

func TestToNoteSummaryItem_CarriesTypeBodyAndArchived(t *testing.T) {
	got := toNoteSummaryItem(&planpb.ChecklistItem{Id: "i", Type: "note", Description: "Body", Archived: true})

	if got != (notesummary.Item{Type: "note", Description: "Body", Archived: true}) {
		t.Fatalf("item = %+v", got)
	}
}

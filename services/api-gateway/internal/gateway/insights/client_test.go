package insightsgw

import (
	"testing"
	"time"

	pb "github.com/darkphotonKN/fireplace/common/api/proto/insights"
	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestToDayPlan_TranslatesThePlannedProtoPlan(t *testing.T) {
	planID, item, parent, a, b := uuid.New(), uuid.New(), uuid.New(), uuid.New(), uuid.New()
	itemStr, parentStr, source := item.String(), parent.String(), "ai"
	minutes, revision := int32(45), int32(2)
	at := time.Date(2026, 10, 25, 9, 0, 0, 0, time.UTC)

	got, err := toDayPlan(&pb.DayPlan{
		PlanId: planID.String(), Date: "2026-10-25", Status: "planned", Source: &source, GeneratedAt: timestamppb.New(at), RefreshesRemaining: 2,
		Groups: []*pb.DayPlanGroup{{Label: "Graphs", Entries: []*pb.DayPlanEntry{
			{Id: a.String(), Kind: "existing_item", ItemId: &itemStr, ParentItemId: &parentStr, Text: "t", Reason: "r", Minutes: &minutes, Revision: &revision},
			{Id: b.String(), Kind: "new_step", ParentItemId: &parentStr, Text: "n", Reason: "r"},
		}}},
	})
	if err != nil {
		t.Fatal(err)
	}

	if got.PlanID != planID || got.Date != "2026-10-25" || got.Status != "planned" || got.Source != "ai" ||
		got.GeneratedAt == nil || !got.GeneratedAt.Equal(at) || got.RefreshesRemaining != 2 || len(got.Groups) != 1 {
		t.Fatalf("plan = %+v", got)
	}
	e, n := got.Groups[0].Entries[0], got.Groups[0].Entries[1]
	if got.Groups[0].Label != "Graphs" || e.ID != a || e.Kind != "existing_item" || *e.ItemID != item || *e.ParentItemID != parent ||
		*e.Minutes != 45 || *e.Revision != 2 || e.Ticked || e.RevisionsEnabled != nil {
		t.Fatalf("existing entry = %+v", e)
	}
	if n.ItemID != nil || *n.ParentItemID != parent || n.Minutes != nil || n.Revision != nil {
		t.Fatalf("new step = %+v", n)
	}
}

func TestToDayPlan_NotPlannedHasNoSourceNoDrawTimeAndNonNilGroups(t *testing.T) {
	got, err := toDayPlan(&pb.DayPlan{PlanId: uuid.NewString(), Date: "2026-10-25", Status: "not_planned", RefreshesRemaining: 3})
	if err != nil || got.Source != "" || got.GeneratedAt != nil || got.Groups == nil || len(got.Groups) != 0 {
		t.Fatalf("plan = %+v err %v", got, err)
	}
}

func TestToDayPlan_MalformedIdIsAnError(t *testing.T) {
	bad := "nope"
	for name, p := range map[string]*pb.DayPlan{
		"plan id":  {PlanId: "nope"},
		"entry id": {PlanId: uuid.NewString(), Groups: []*pb.DayPlanGroup{{Entries: []*pb.DayPlanEntry{{Id: "nope"}}}}},
		"item id":  {PlanId: uuid.NewString(), Groups: []*pb.DayPlanGroup{{Entries: []*pb.DayPlanEntry{{Id: uuid.NewString(), ItemId: &bad}}}}},
		"parent":   {PlanId: uuid.NewString(), Groups: []*pb.DayPlanGroup{{Entries: []*pb.DayPlanEntry{{Id: uuid.NewString(), ParentItemId: &bad}}}}},
	} {
		if _, err := toDayPlan(p); err == nil {
			t.Fatalf("%s: want an error", name)
		}
	}
}

func TestToDayPlanEntry_CarriesTheDayTick(t *testing.T) {
	id, item := uuid.New(), uuid.New()
	itemStr := item.String()

	e, err := toDayPlanEntry(&pb.DayPlanEntry{Id: id.String(), Kind: "existing_item", ItemId: &itemStr, Text: "x", Reason: "y", Ticked: true})

	if err != nil || !e.Ticked || e.ID != id || *e.ItemID != item {
		t.Fatalf("entry = %+v err %v, want the ticked entry", e, err)
	}
}

func TestToDayPlanEntry_CarriesTheRevisionsSwitchOnlyWhenSet(t *testing.T) {
	id, item := uuid.New(), uuid.New()
	itemStr := item.String()
	for _, enabled := range []bool{true, false} {
		e, err := toDayPlanEntry(&pb.DayPlanEntry{Id: id.String(), Kind: "existing_item", ItemId: &itemStr, Text: "x", Reason: "y", RevisionsEnabled: &enabled})
		if err != nil || e.RevisionsEnabled == nil || *e.RevisionsEnabled != enabled {
			t.Fatalf("entry = %+v err %v, want revisionsEnabled %v", e, err, enabled)
		}
	}
	e, err := toDayPlanEntry(&pb.DayPlanEntry{Id: id.String(), Kind: "existing_item", ItemId: &itemStr, Text: "x", Reason: "y"})
	if err != nil || e.RevisionsEnabled != nil {
		t.Fatalf("entry = %+v err %v, want revisionsEnabled absent when unset", e, err)
	}
}

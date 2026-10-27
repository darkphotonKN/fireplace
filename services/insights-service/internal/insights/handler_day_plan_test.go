package insights

import (
	"context"
	"fmt"
	"testing"
	"time"

	pb "github.com/darkphotonKN/fireplace/common/api/proto/insights"
	commonconstants "github.com/darkphotonKN/fireplace/common/constants"
	"github.com/darkphotonKN/fireplace/common/errcode"
	commongrpc "github.com/darkphotonKN/fireplace/common/grpcerror"
	"github.com/darkphotonKN/fireplace/services/insights-service/internal/dayplan"
	"github.com/darkphotonKN/fireplace/services/insights-service/internal/guided"
	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type fakeDayPlans struct {
	plan    dayplan.DayPlan
	err     error
	gotPlan uuid.UUID
	gotUser uuid.UUID
	gotDate string
	// drew records that PlanMyDay, not GetDayPlan, was called.
	drew bool

	entry    dayplan.Entry
	gotEntry uuid.UUID
	gotItem  *uuid.UUID
	unticked bool

	gotEnabled *bool
}

func (f *fakeDayPlans) GetDayPlan(ctx context.Context, planID, userID uuid.UUID, date string) (dayplan.DayPlan, error) {
	f.gotPlan, f.gotUser, f.gotDate = planID, userID, date
	return f.plan, f.err
}

func (f *fakeDayPlans) PlanMyDay(ctx context.Context, planID, userID uuid.UUID, date string) (dayplan.DayPlan, error) {
	f.drew = true
	f.gotPlan, f.gotUser, f.gotDate = planID, userID, date
	return f.plan, f.err
}

func (f *fakeDayPlans) TickEntry(ctx context.Context, entryID, userID uuid.UUID, itemID *uuid.UUID) (dayplan.Entry, error) {
	f.gotEntry, f.gotUser, f.gotItem = entryID, userID, itemID
	return f.entry, f.err
}

func (f *fakeDayPlans) UntickEntry(ctx context.Context, entryID, userID uuid.UUID) error {
	f.unticked = true
	f.gotEntry, f.gotUser = entryID, userID
	return f.err
}

func (f *fakeDayPlans) SetItemRevisions(ctx context.Context, planID, userID, itemID uuid.UUID, enabled bool) error {
	f.gotPlan, f.gotUser, f.gotItem, f.gotEnabled = planID, userID, &itemID, &enabled
	return f.err
}

func TestHandlerPlanMyDay_MapsThePlannedDayPlan(t *testing.T) {
	planID, userID, item, parent, newParent := uuid.New(), uuid.New(), uuid.New(), uuid.New(), uuid.New()
	entryA, entryB := uuid.New(), uuid.New()
	generatedAt := time.Date(2026, 10, 25, 9, 0, 0, 0, time.UTC)
	minutes := 45
	plans := &fakeDayPlans{plan: dayplan.DayPlan{
		Key:     guided.DayKey{PlanID: planID, UserID: userID, Date: time.Date(2026, 10, 25, 0, 0, 0, 0, time.UTC)},
		Planned: true, Source: dayplan.SourceAI, GeneratedAt: generatedAt, RefreshesRemaining: 2,
		Groups: []dayplan.Group{{Label: "Graphs", Entries: []dayplan.Entry{
			{ID: entryA, Kind: dayplan.EntryExistingItem, ItemID: &item, ParentItemID: &parent, Text: "Solve 3 BFS problems", Reason: "due", Minutes: &minutes},
			{ID: entryB, Kind: dayplan.EntryNewStep, ParentItemID: &newParent, Text: "Do knapsack", Reason: "start"},
		}}},
	}}
	h := NewHandler(nil, plans)

	resp, err := h.PlanMyDay(context.Background(), &pb.PlanMyDayRequest{PlanId: planID.String(), UserId: userID.String(), Date: "2026-10-25"})

	if err != nil {
		t.Fatalf("PlanMyDay: %v", err)
	}
	if !plans.drew || plans.gotPlan != planID || plans.gotUser != userID || plans.gotDate != "2026-10-25" {
		t.Fatalf("service drew %v with plan %s user %s date %q", plans.drew, plans.gotPlan, plans.gotUser, plans.gotDate)
	}
	p := resp.GetDayPlan()
	if p.GetPlanId() != planID.String() || p.GetDate() != "2026-10-25" || p.GetStatus() != "planned" || p.GetSource() != "ai" ||
		!p.GetGeneratedAt().AsTime().Equal(generatedAt) || p.GetRefreshesRemaining() != 2 || len(p.GetGroups()) != 1 {
		t.Fatalf("day plan = %v", p)
	}
	g := p.GetGroups()[0]
	a, b := g.GetEntries()[0], g.GetEntries()[1]
	if g.GetLabel() != "Graphs" || a.GetId() != entryA.String() || a.GetKind() != "existing_item" || a.GetItemId() != item.String() ||
		a.GetParentItemId() != parent.String() || a.GetMinutes() != 45 || a.Revision != nil || a.GetTicked() {
		t.Fatalf("existing entry = %v", a)
	}
	if b.GetKind() != "new_step" || b.ItemId != nil || b.GetParentItemId() != newParent.String() || b.Minutes != nil {
		t.Fatalf("new step entry = %v", b)
	}
}

func TestHandlerGetDayPlan_NotPlannedHasNoSourceNoDrawTimeAndNoGroups(t *testing.T) {
	planID, userID := uuid.New(), uuid.New()
	plans := &fakeDayPlans{plan: dayplan.DayPlan{
		Key: guided.DayKey{PlanID: planID, UserID: userID, Date: time.Date(2026, 10, 25, 0, 0, 0, 0, time.UTC)}, Groups: []dayplan.Group{}, RefreshesRemaining: 3,
	}}
	h := NewHandler(nil, plans)

	resp, err := h.GetDayPlan(context.Background(), &pb.GetDayPlanRequest{PlanId: planID.String(), UserId: userID.String(), Date: "2026-10-25"})

	if err != nil {
		t.Fatalf("GetDayPlan: %v", err)
	}
	p := resp.GetDayPlan()
	if plans.drew || p.GetStatus() != "not_planned" || p.Source != nil || p.GeneratedAt != nil || len(p.GetGroups()) != 0 || p.GetRefreshesRemaining() != 3 {
		t.Fatalf("day plan = %v, want not_planned with nothing drawn", p)
	}
}

func TestHandlerDayPlan_MapsErrors(t *testing.T) {
	tests := []struct {
		name     string
		svcErr   error
		badPlan  bool
		wantCode codes.Code
		wantErr  errcode.Code
	}{
		{"date out of range", fmt.Errorf("x: %w", guided.ErrDateOutOfRange), false, codes.InvalidArgument, errcode.BriefDateOutOfRange},
		{"refresh limit reached", fmt.Errorf("x: %w", guided.ErrRefreshLimitReached), false, codes.ResourceExhausted, errcode.BriefRefreshLimitReached},
		{"plan not found", fmt.Errorf("x: %w", commonconstants.ErrNotFound), false, codes.NotFound, ""},
		{"malformed plan id", nil, true, codes.InvalidArgument, ""},
	}
	for _, tt := range tests {
		for op, call := range map[string]func(h *Handler, planID string) error{
			"GetDayPlan": func(h *Handler, planID string) error {
				_, err := h.GetDayPlan(context.Background(), &pb.GetDayPlanRequest{PlanId: planID, UserId: uuid.NewString(), Date: "2026-10-25"})
				return err
			},
			"PlanMyDay": func(h *Handler, planID string) error {
				_, err := h.PlanMyDay(context.Background(), &pb.PlanMyDayRequest{PlanId: planID, UserId: uuid.NewString(), Date: "2026-10-25"})
				return err
			},
		} {
			t.Run(op+"/"+tt.name, func(t *testing.T) {
				planID := uuid.NewString()
				if tt.badPlan {
					planID = "nope"
				}
				err := call(NewHandler(nil, &fakeDayPlans{err: tt.svcErr}), planID)

				if status.Code(err) != tt.wantCode {
					t.Fatalf("code = %s, want %s (err %v)", status.Code(err), tt.wantCode, err)
				}
				code, ok := commongrpc.DomainCode(err)
				if tt.wantErr != "" && (!ok || code != tt.wantErr) {
					t.Fatalf("domain code = %q (ok %v), want %q", code, ok, tt.wantErr)
				}
				if tt.wantErr == "" && ok {
					t.Fatalf("unexpected domain code %q", code)
				}
			})
		}
	}
}

func TestHandlerTickDayPlanEntry_ForwardsTheCallAndMapsTheTickedEntry(t *testing.T) {
	entryID, userID, item, parent := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	plans := &fakeDayPlans{entry: dayplan.Entry{
		ID: entryID, Kind: dayplan.EntryExistingItem, ItemID: &item, ParentItemID: &parent,
		Text: "Solve the climbing stairs problem", Reason: "start", Ticked: true,
	}}
	h := NewHandler(nil, plans)
	itemStr := item.String()

	resp, err := h.TickDayPlanEntry(context.Background(), &pb.TickDayPlanEntryRequest{EntryId: entryID.String(), UserId: userID.String(), ItemId: &itemStr})

	if err != nil {
		t.Fatalf("TickDayPlanEntry: %v", err)
	}
	if plans.gotEntry != entryID || plans.gotUser != userID || plans.gotItem == nil || *plans.gotItem != item {
		t.Fatalf("service got entry %s user %s item %v", plans.gotEntry, plans.gotUser, plans.gotItem)
	}
	e := resp.GetEntry()
	if e.GetId() != entryID.String() || e.GetKind() != "existing_item" || e.GetItemId() != item.String() || !e.GetTicked() {
		t.Fatalf("entry = %v, want the ticked entry", e)
	}

	if _, err := h.TickDayPlanEntry(context.Background(), &pb.TickDayPlanEntryRequest{EntryId: entryID.String(), UserId: userID.String()}); err != nil {
		t.Fatal(err)
	}
	if plans.gotItem != nil {
		t.Fatalf("item = %v, want none forwarded when none is sent", plans.gotItem)
	}
}

func TestHandlerUntickDayPlanEntry_ForwardsTheCall(t *testing.T) {
	entryID, userID := uuid.New(), uuid.New()
	plans := &fakeDayPlans{}

	_, err := NewHandler(nil, plans).UntickDayPlanEntry(context.Background(), &pb.UntickDayPlanEntryRequest{EntryId: entryID.String(), UserId: userID.String()})

	if err != nil || !plans.unticked || plans.gotEntry != entryID || plans.gotUser != userID {
		t.Fatalf("untick err %v, service unticked %v entry %s user %s", err, plans.unticked, plans.gotEntry, plans.gotUser)
	}
}

func TestHandlerDayTicks_MapErrors(t *testing.T) {
	tests := []struct {
		name     string
		svcErr   error
		bad      string
		wantCode codes.Code
		wantErr  errcode.Code
	}{
		{"not tickable", fmt.Errorf("x: %w", dayplan.ErrEntryNotTickable), "", codes.FailedPrecondition, errcode.DayPlanEntryNotTickable},
		{"date out of range", fmt.Errorf("x: %w", guided.ErrDateOutOfRange), "", codes.InvalidArgument, errcode.BriefDateOutOfRange},
		{"entry not found", fmt.Errorf("x: %w", dayplan.ErrEntryNotFound), "", codes.NotFound, ""},
		{"malformed entry id", nil, "entry", codes.InvalidArgument, ""},
		{"malformed user id", nil, "user", codes.InvalidArgument, ""},
		{"malformed item id", nil, "item", codes.InvalidArgument, ""},
	}
	for _, tt := range tests {
		for op, call := range map[string]func(h *Handler, entryID, userID string, itemID *string) error{
			"Tick": func(h *Handler, entryID, userID string, itemID *string) error {
				_, err := h.TickDayPlanEntry(context.Background(), &pb.TickDayPlanEntryRequest{EntryId: entryID, UserId: userID, ItemId: itemID})
				return err
			},
			"Untick": func(h *Handler, entryID, userID string, _ *string) error {
				_, err := h.UntickDayPlanEntry(context.Background(), &pb.UntickDayPlanEntryRequest{EntryId: entryID, UserId: userID})
				return err
			},
		} {
			if op == "Untick" && (tt.bad == "item" || tt.wantErr == errcode.DayPlanEntryNotTickable) {
				continue
			}
			t.Run(op+"/"+tt.name, func(t *testing.T) {
				entryID, userID := uuid.NewString(), uuid.NewString()
				var itemID *string
				switch tt.bad {
				case "entry":
					entryID = "nope"
				case "user":
					userID = "nope"
				case "item":
					itemID = ptrTo("nope")
				}
				err := call(NewHandler(nil, &fakeDayPlans{err: tt.svcErr}), entryID, userID, itemID)

				if status.Code(err) != tt.wantCode {
					t.Fatalf("code = %s, want %s (err %v)", status.Code(err), tt.wantCode, err)
				}
				code, ok := commongrpc.DomainCode(err)
				if tt.wantErr != "" && (!ok || code != tt.wantErr) {
					t.Fatalf("domain code = %q (ok %v), want %q", code, ok, tt.wantErr)
				}
				if tt.wantErr == "" && ok {
					t.Fatalf("unexpected domain code %q", code)
				}
			})
		}
	}
}

func ptrTo[T any](v T) *T { return &v }

func TestHandlerTickDayPlanEntry_MapsRevisionAndRevisionsSwitch(t *testing.T) {
	entryID, item := uuid.New(), uuid.New()
	h := NewHandler(nil, &fakeDayPlans{entry: dayplan.Entry{
		ID: entryID, Kind: dayplan.EntryExistingItem, ItemID: &item, Text: "x", Reason: "y", Ticked: true,
		Revision: ptrTo(2), RevisionsEnabled: ptrTo(false),
	}})

	resp, err := h.TickDayPlanEntry(context.Background(), &pb.TickDayPlanEntryRequest{EntryId: entryID.String(), UserId: uuid.NewString()})

	if err != nil {
		t.Fatal(err)
	}
	if e := resp.GetEntry(); e.Revision == nil || e.GetRevision() != 2 || e.RevisionsEnabled == nil || e.GetRevisionsEnabled() {
		t.Fatalf("entry = %v, want revision 2 and revisions_enabled false", e)
	}

	h = NewHandler(nil, &fakeDayPlans{entry: dayplan.Entry{ID: entryID, Kind: dayplan.EntryExistingItem, ItemID: &item, Text: "x", Reason: "y"}})
	resp, err = h.TickDayPlanEntry(context.Background(), &pb.TickDayPlanEntryRequest{EntryId: entryID.String(), UserId: uuid.NewString()})
	if err != nil {
		t.Fatal(err)
	}
	if e := resp.GetEntry(); e.RevisionsEnabled != nil {
		t.Fatalf("entry = %v, want revisions_enabled unset when the service leaves it unset", e)
	}
}

func TestHandlerSetItemRevisions_ForwardsTheCall(t *testing.T) {
	planID, userID, item := uuid.New(), uuid.New(), uuid.New()
	plans := &fakeDayPlans{}

	_, err := NewHandler(nil, plans).SetItemRevisions(context.Background(), &pb.SetItemRevisionsRequest{
		PlanId: planID.String(), UserId: userID.String(), ItemId: item.String(), Enabled: true,
	})

	if err != nil || plans.gotPlan != planID || plans.gotUser != userID || plans.gotItem == nil || *plans.gotItem != item ||
		plans.gotEnabled == nil || !*plans.gotEnabled {
		t.Fatalf("err %v; service got plan %s user %s item %v enabled %v", err, plans.gotPlan, plans.gotUser, plans.gotItem, plans.gotEnabled)
	}
}

func TestHandlerSetItemRevisions_MapsErrors(t *testing.T) {
	tests := []struct {
		name     string
		svcErr   error
		req      func(*pb.SetItemRevisionsRequest)
		wantCode codes.Code
	}{
		{"item not in the plan", fmt.Errorf("x: %w", dayplan.ErrItemNotFound), nil, codes.NotFound},
		{"plan not accessible", fmt.Errorf("x: %w", commonconstants.ErrNotFound), nil, codes.NotFound},
		{"malformed plan id", nil, func(r *pb.SetItemRevisionsRequest) { r.PlanId = "nope" }, codes.InvalidArgument},
		{"malformed user id", nil, func(r *pb.SetItemRevisionsRequest) { r.UserId = "nope" }, codes.InvalidArgument},
		{"malformed item id", nil, func(r *pb.SetItemRevisionsRequest) { r.ItemId = "nope" }, codes.InvalidArgument},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &pb.SetItemRevisionsRequest{PlanId: uuid.NewString(), UserId: uuid.NewString(), ItemId: uuid.NewString()}
			if tt.req != nil {
				tt.req(req)
			}
			_, err := NewHandler(nil, &fakeDayPlans{err: tt.svcErr}).SetItemRevisions(context.Background(), req)

			if status.Code(err) != tt.wantCode {
				t.Fatalf("code = %s, want %s (err %v)", status.Code(err), tt.wantCode, err)
			}
		})
	}
}

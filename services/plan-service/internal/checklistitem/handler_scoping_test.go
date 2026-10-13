package checklistitem

import (
	"context"
	"testing"

	pb "github.com/darkphotonKN/fireplace/common/api/proto/plan"
	commonconstants "github.com/darkphotonKN/fireplace/common/constants"
	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// planRecorder records the plan each item-by-id call was scoped to, and answers
// NotFound when told to, the way the service does for another plan's item.
type planRecorder struct {
	Service

	calls  int
	planID uuid.UUID
	itemID uuid.UUID
	err    error
}

func (r *planRecorder) record(planID, id uuid.UUID) (*Item, error) {
	r.calls++
	r.planID, r.itemID = planID, id
	if r.err != nil {
		return nil, r.err
	}
	return &Item{ID: id, PlanID: planID}, nil
}

func (r *planRecorder) GetInPlan(ctx context.Context, planID, id uuid.UUID) (*Item, error) {
	return r.record(planID, id)
}

func (r *planRecorder) Update(ctx context.Context, in UpdateItemInput) (*Item, error) {
	return r.record(in.PlanID, in.ID)
}

func (r *planRecorder) UpdateDates(ctx context.Context, in UpdateDatesInput) (*Item, error) {
	return r.record(in.PlanID, in.ID)
}

func (r *planRecorder) Archive(ctx context.Context, planID, id uuid.UUID, archived bool) (*Item, error) {
	return r.record(planID, id)
}

func (r *planRecorder) Delete(ctx context.Context, planID, id uuid.UUID) error {
	_, err := r.record(planID, id)
	return err
}

// itemRPCs calls each item-by-id RPC with the given plan id.
func itemRPCs(h *Handler, itemID uuid.UUID, planID string) map[string]func() error {
	ctx, id := context.Background(), itemID.String()
	return map[string]func() error{
		"GetItem": func() error {
			_, err := h.GetItem(ctx, &pb.GetItemRequest{Id: id, PlanId: planID})
			return err
		},
		"UpdateItem": func() error {
			_, err := h.UpdateItem(ctx, &pb.UpdateItemRequest{Id: id, PlanId: planID})
			return err
		},
		"UpdateItemDates": func() error {
			_, err := h.UpdateItemDates(ctx, &pb.UpdateItemDatesRequest{Id: id, PlanId: planID})
			return err
		},
		"ArchiveItem": func() error {
			_, err := h.ArchiveItem(ctx, &pb.ArchiveItemRequest{Id: id, PlanId: planID, Archived: true})
			return err
		},
		"DeleteItem": func() error {
			_, err := h.DeleteItem(ctx, &pb.DeleteItemRequest{Id: id, PlanId: planID})
			return err
		},
	}
}

// R12: an item-by-id request without a plan id is a caller bug. It is refused,
// never run unscoped.
func TestItemByIDRPCs_EmptyPlanID_IsInvalidArgument(t *testing.T) {
	for name := range itemRPCs(nil, uuid.New(), "") {
		t.Run(name, func(t *testing.T) {
			rec := &planRecorder{}
			call := itemRPCs(NewHandler(rec), uuid.New(), "")[name]
			if got := status.Code(call()); got != codes.InvalidArgument {
				t.Fatalf("code = %s, want InvalidArgument", got)
			}
			if rec.calls != 0 {
				t.Error("a request with no plan id reached the domain")
			}
		})
	}
}

func TestItemByIDRPCs_MalformedPlanID_IsInvalidArgument(t *testing.T) {
	for name := range itemRPCs(nil, uuid.New(), "") {
		t.Run(name, func(t *testing.T) {
			rec := &planRecorder{}
			call := itemRPCs(NewHandler(rec), uuid.New(), "not-a-uuid")[name]
			if got := status.Code(call()); got != codes.InvalidArgument {
				t.Fatalf("code = %s, want InvalidArgument", got)
			}
			if rec.calls != 0 {
				t.Error("a malformed plan id reached the domain")
			}
		})
	}
}

// R11: the plan id reaches the domain, and the domain's NotFound for another
// plan's item leaves the transport as NotFound.
func TestItemByIDRPCs_ScopeToThePlanID(t *testing.T) {
	planID, itemID := uuid.New(), uuid.New()
	for name := range itemRPCs(nil, itemID, "") {
		t.Run(name, func(t *testing.T) {
			rec := &planRecorder{}
			if err := itemRPCs(NewHandler(rec), itemID, planID.String())[name](); err != nil {
				t.Fatalf("own plan: %v", err)
			}
			if rec.planID != planID || rec.itemID != itemID {
				t.Errorf("domain got (plan %s, item %s), want (%s, %s)", rec.planID, rec.itemID, planID, itemID)
			}

			rec = &planRecorder{err: commonconstants.ErrNotFound}
			err := itemRPCs(NewHandler(rec), itemID, uuid.NewString())[name]()
			if got := status.Code(err); got != codes.NotFound {
				t.Fatalf("another plan: code = %s, want NotFound", got)
			}
		})
	}
}

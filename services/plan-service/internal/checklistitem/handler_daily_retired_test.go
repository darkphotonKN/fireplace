package checklistitem

import (
	"context"
	"testing"

	pb "github.com/darkphotonKN/fireplace/common/api/proto/plan"
	"github.com/darkphotonKN/fireplace/common/errcode"
	commongrpc "github.com/darkphotonKN/fireplace/common/grpcerror"
	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// FS-02JBZ R51: the daily scope is retired. Creating an item as daily, or
// updating one to daily, is refused with DAILY_SCOPE_RETIRED carried as an
// ErrorInfo detail on InvalidArgument, and nothing is written. These run the
// real handler over the real service, so they prove the code survives the gRPC
// boundary (commongrpc.Fail), not just that the service returns it.

func TestDailyScope_CreateAndUpdate_AreRefusedWithDailyScopeRetired(t *testing.T) {
	planA, _, itemA, _, _ := twoPlans()
	daily := ScopeDaily

	tests := []struct {
		name string
		call func(h *Handler) error
	}{
		{"create as daily", func(h *Handler) error {
			_, err := h.CreateItem(context.Background(), &pb.CreateItemRequest{
				PlanId: planA.String(), Description: "habit", Scope: ScopeDaily,
			})
			return err
		}},
		{"update to daily", func(h *Handler) error {
			_, err := h.UpdateItem(context.Background(), &pb.UpdateItemRequest{
				Id: itemA.ID.String(), PlanId: planA.String(), Scope: &daily,
			})
			return err
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// A fresh repo per case. It implements neither Create nor
			// CountItems, so a create that got past the rule would panic.
			copied := *itemA
			repo := newScopedRepo(&copied)
			h := NewHandler(NewService(repo, nil))

			err := tt.call(h)

			if status.Code(err) != codes.InvalidArgument {
				t.Fatalf("code = %v, want InvalidArgument (err %v)", status.Code(err), err)
			}
			code, ok := commongrpc.DomainCode(err)
			if !ok || code != errcode.DailyScopeRetired {
				t.Errorf("domain code = %q (ok %v), want %q", code, ok, errcode.DailyScopeRetired)
			}
			if repo.writes != 0 {
				t.Errorf("%d writes reached the repository, want 0", repo.writes)
			}
			if repo.rows[itemA.ID].Scope != ScopeLongterm {
				t.Errorf("item scope = %q, want it left longterm", repo.rows[itemA.ID].Scope)
			}
		})
	}
}

// The refusal is about daily only: an update that names longterm, or leaves
// scope alone, still goes through.
func TestDailyScope_UpdateToLongtermOrOmitted_StillUpdates(t *testing.T) {
	for _, scope := range []*string{nil, func() *string { s := ScopeLongterm; return &s }()} {
		planA, _, itemA, _, repo := twoPlans()
		desc := "renamed"
		_, err := NewHandler(NewService(repo, nil)).UpdateItem(context.Background(), &pb.UpdateItemRequest{
			Id: itemA.ID.String(), PlanId: planA.String(), Scope: scope, Description: &desc,
		})
		if err != nil {
			t.Fatalf("scope %v: err = %v, want nil", scope, err)
		}
		if repo.writes != 1 {
			t.Errorf("scope %v: writes = %d, want 1", scope, repo.writes)
		}
	}
}

// listingRepo answers ListByPlanID the way the table does once no daily row
// exists: a daily filter matches nothing.
type listingRepo struct {
	Repository
	rows []*Item
}

func (r *listingRepo) ListByPlanID(ctx context.Context, in ListItemsInput) ([]*Item, error) {
	var out []*Item
	for _, it := range r.rows {
		if in.Scope == nil || *in.Scope == it.Scope {
			out = append(out, it)
		}
	}
	return out, nil
}

// S15: reads are not refused. Listing with scope=daily is accepted and finds
// nothing, so an old client that still asks keeps working.
func TestDailyScope_ListingWithDaily_IsAnEmptyListNotAnError(t *testing.T) {
	planID := uuid.New()
	repo := &listingRepo{rows: []*Item{{ID: uuid.New(), PlanID: planID, Scope: ScopeLongterm, Type: TypeTask}}}
	daily := ScopeDaily

	resp, err := NewHandler(NewService(repo, nil)).ListItems(context.Background(), &pb.ListItemsRequest{
		PlanId: planID.String(), Scope: &daily,
	})

	if err != nil {
		t.Fatalf("err = %v, want nil", err)
	}
	if n := len(resp.GetItems()); n != 0 {
		t.Errorf("got %d items, want 0", n)
	}
}

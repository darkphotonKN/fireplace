package plan

import (
	"context"
	"testing"

	pb "github.com/darkphotonKN/fireplace/common/api/proto/plan"
	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestGetPlanAccess_AnswersTheRelationAndOwner(t *testing.T) {
	repo := newFakeRepo()
	owner, member := uuid.New(), uuid.New()
	planID := repo.addPlan(owner)
	repo.share(planID, member)
	h := NewHandler(newTestService(repo))

	cases := []struct {
		name      string
		planID    uuid.UUID
		user      uuid.UUID
		want      pb.PlanRelation
		wantOwner string
	}{
		{"owner", planID, owner, pb.PlanRelation_PLAN_RELATION_OWNER, owner.String()},
		{"member", planID, member, pb.PlanRelation_PLAN_RELATION_MEMBER, owner.String()},
		{"stranger", planID, uuid.New(), pb.PlanRelation_PLAN_RELATION_NONE, owner.String()},
		{"missing plan", uuid.New(), owner, pb.PlanRelation_PLAN_RELATION_NONE, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			resp, err := h.GetPlanAccess(context.Background(), &pb.GetPlanAccessRequest{
				PlanId: tc.planID.String(), UserId: tc.user.String(),
			})
			if err != nil {
				t.Fatalf("GetPlanAccess: %v (a missing plan must never be an error)", err)
			}
			if resp.Relation != tc.want {
				t.Errorf("relation = %s, want %s", resp.Relation, tc.want)
			}
			if resp.OwnerId != tc.wantOwner {
				t.Errorf("owner_id = %q, want %q", resp.OwnerId, tc.wantOwner)
			}
		})
	}
}

// R22: include_shared adds the plans shared with the caller; left unset the
// RPC keeps its owned-only meaning for every other caller.
func TestListPlans_IncludeShared_AddsSharedPlansOnlyWhenSet(t *testing.T) {
	repo := newFakeRepo()
	member, other := uuid.New(), uuid.New()
	owned := repo.addPlan(member)
	shared := repo.addPlan(other)
	repo.share(shared, member)
	repo.addPlan(other) // not shared
	h := NewHandler(newTestService(repo))

	cases := []struct {
		name          string
		includeShared bool
		want          []uuid.UUID
	}{
		{"unset: owned only", false, []uuid.UUID{owned}},
		{"set: owned and shared", true, []uuid.UUID{owned, shared}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			resp, err := h.ListPlans(context.Background(), &pb.ListPlansRequest{
				UserId: member.String(), IncludeShared: tc.includeShared,
			})
			if err != nil {
				t.Fatalf("ListPlans: %v", err)
			}
			got := map[string]bool{}
			for _, p := range resp.Plans {
				got[p.Id] = true
			}
			if len(resp.Plans) != len(tc.want) {
				t.Fatalf("got %d plans, want %d", len(resp.Plans), len(tc.want))
			}
			for _, id := range tc.want {
				if !got[id.String()] {
					t.Errorf("plan %s missing", id)
				}
			}
		})
	}
}

func TestGetPlanAccess_MalformedUUID_IsInvalidArgument(t *testing.T) {
	h := NewHandler(newTestService(newFakeRepo()))
	for name, req := range map[string]*pb.GetPlanAccessRequest{
		"plan_id": {PlanId: "nope", UserId: uuid.NewString()},
		"user_id": {PlanId: uuid.NewString(), UserId: "nope"},
		"empty":   {},
	} {
		t.Run(name, func(t *testing.T) {
			_, err := h.GetPlanAccess(context.Background(), req)
			if got := status.Code(err); got != codes.InvalidArgument {
				t.Fatalf("code = %s, want InvalidArgument", got)
			}
		})
	}
}

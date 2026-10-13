package plan

import (
	"context"
	"testing"

	pb "github.com/darkphotonKN/fireplace/common/api/proto/plan"
	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestAddPlanMember_AnswersTheMemberAndSince(t *testing.T) {
	repo := newFakeRepo()
	owner, member := uuid.New(), uuid.New()
	planID := repo.addPlan(owner)
	h := NewHandler(newTestService(repo))

	got, err := h.AddPlanMember(context.Background(), &pb.AddPlanMemberRequest{
		PlanId: planID.String(), UserId: owner.String(), MemberUserId: member.String(),
	})
	if err != nil {
		t.Fatalf("AddPlanMember: %v", err)
	}
	if got.UserId != member.String() || !got.Since.AsTime().Equal(repo.since[shareKey{planID, member}]) {
		t.Errorf("got %v, want %s since %s", got, member, repo.since[shareKey{planID, member}])
	}
}

// The MEMBER_IS_OWNER rule reaches the wire as InvalidArgument (the gateway
// attaches the code at its add-member operation).
func TestAddPlanMember_TheOwner_IsInvalidArgument(t *testing.T) {
	repo := newFakeRepo()
	owner := uuid.New()
	planID := repo.addPlan(owner)

	_, err := NewHandler(newTestService(repo)).AddPlanMember(context.Background(), &pb.AddPlanMemberRequest{
		PlanId: planID.String(), UserId: owner.String(), MemberUserId: owner.String(),
	})
	if got := status.Code(err); got != codes.InvalidArgument {
		t.Fatalf("code = %s, want InvalidArgument", got)
	}
}

func TestRemovePlanMember_NotAMember_IsNotFound(t *testing.T) {
	repo := newFakeRepo()
	owner := uuid.New()
	planID := repo.addPlan(owner)

	_, err := NewHandler(newTestService(repo)).RemovePlanMember(context.Background(), &pb.RemovePlanMemberRequest{
		PlanId: planID.String(), UserId: owner.String(), MemberUserId: uuid.NewString(),
	})
	if got := status.Code(err); got != codes.NotFound {
		t.Fatalf("code = %s, want NotFound", got)
	}
}

func TestRemovePlanMember_Member_IsRemoved(t *testing.T) {
	repo := newFakeRepo()
	owner, member := uuid.New(), uuid.New()
	planID := repo.addPlan(owner)
	repo.share(planID, member)

	_, err := NewHandler(newTestService(repo)).RemovePlanMember(context.Background(), &pb.RemovePlanMemberRequest{
		PlanId: planID.String(), UserId: owner.String(), MemberUserId: member.String(),
	})
	if err != nil {
		t.Fatalf("RemovePlanMember: %v", err)
	}
	if repo.shares[member][planID] {
		t.Error("the share row remains")
	}
}

func TestListPlanMembers_AnswersOwnerCreatedAtAndMembers(t *testing.T) {
	repo := newFakeRepo()
	owner, first, second := uuid.New(), uuid.New(), uuid.New()
	planID := repo.addPlan(owner)
	repo.share(planID, first)
	repo.share(planID, second)

	got, err := NewHandler(newTestService(repo)).ListPlanMembers(context.Background(), &pb.ListPlanMembersRequest{
		PlanId: planID.String(), UserId: first.String(),
	})
	if err != nil {
		t.Fatalf("ListPlanMembers: %v", err)
	}
	if got.OwnerId != owner.String() || !got.PlanCreatedAt.AsTime().Equal(repo.plans[planID].CreatedAt) {
		t.Errorf("owner/created = %s/%s", got.OwnerId, got.PlanCreatedAt.AsTime())
	}
	if len(got.Members) != 2 || got.Members[0].UserId != first.String() || got.Members[1].UserId != second.String() {
		t.Fatalf("members = %v, want [%s %s]", got.Members, first, second)
	}
	if got.Members[0].Since == nil {
		t.Error("a member has no since")
	}
}

func TestListPlanMembers_MissingPlan_IsNotFound(t *testing.T) {
	_, err := NewHandler(newTestService(newFakeRepo())).ListPlanMembers(context.Background(), &pb.ListPlanMembersRequest{
		PlanId: uuid.NewString(), UserId: uuid.NewString(),
	})
	if got := status.Code(err); got != codes.NotFound {
		t.Fatalf("code = %s, want NotFound", got)
	}
}

func TestMemberRPCs_MalformedUUID_IsInvalidArgument(t *testing.T) {
	h := NewHandler(newTestService(newFakeRepo()))
	ctx := context.Background()
	id := uuid.NewString()
	calls := map[string]func() error{
		"list plan_id": func() error {
			_, err := h.ListPlanMembers(ctx, &pb.ListPlanMembersRequest{PlanId: "nope", UserId: id})
			return err
		},
		"add member_user_id": func() error {
			_, err := h.AddPlanMember(ctx, &pb.AddPlanMemberRequest{PlanId: id, UserId: id, MemberUserId: "nope"})
			return err
		},
		"add user_id": func() error {
			_, err := h.AddPlanMember(ctx, &pb.AddPlanMemberRequest{PlanId: id, UserId: "", MemberUserId: id})
			return err
		},
		"remove plan_id": func() error {
			_, err := h.RemovePlanMember(ctx, &pb.RemovePlanMemberRequest{PlanId: "", UserId: id, MemberUserId: id})
			return err
		},
		"remove member_user_id": func() error {
			_, err := h.RemovePlanMember(ctx, &pb.RemovePlanMemberRequest{PlanId: id, UserId: id, MemberUserId: "x"})
			return err
		},
	}
	for name, call := range calls {
		t.Run(name, func(t *testing.T) {
			if got := status.Code(call()); got != codes.InvalidArgument {
				t.Fatalf("code = %s, want InvalidArgument", got)
			}
		})
	}
}

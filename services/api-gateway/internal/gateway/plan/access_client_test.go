package plangw

import (
	"testing"
	"time"

	pb "github.com/darkphotonKN/fireplace/common/api/proto/plan"
	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// The members answer keeps plan-service's add order and every since.
func TestPlanMembersFromProto(t *testing.T) {
	owner, first, second := uuid.New(), uuid.New(), uuid.New()
	created := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	got := planMembersFromProto(&pb.ListPlanMembersResponse{
		OwnerId:       owner.String(),
		PlanCreatedAt: timestamppb.New(created),
		Members: []*pb.PlanMember{
			{UserId: first.String(), Since: timestamppb.New(created.Add(time.Hour))},
			{UserId: second.String(), Since: timestamppb.New(created.Add(2 * time.Hour))},
		},
	})
	if got.OwnerID != owner || !got.PlanCreatedAt.Equal(created) {
		t.Errorf("owner/created = %s/%s", got.OwnerID, got.PlanCreatedAt)
	}
	want := []Membership{{first, created.Add(time.Hour)}, {second, created.Add(2 * time.Hour)}}
	if len(got.Members) != 2 || got.Members[0] != want[0] || got.Members[1] != want[1] {
		t.Errorf("members = %+v, want %+v", got.Members, want)
	}
}

// An answer the gateway does not recognise must read as no relation, so the
// access check built on it (FS-12ZMF slice 2) fails closed.
func TestRelationFromProto(t *testing.T) {
	cases := map[pb.PlanRelation]Relation{
		pb.PlanRelation_PLAN_RELATION_OWNER:       RelationOwner,
		pb.PlanRelation_PLAN_RELATION_MEMBER:      RelationMember,
		pb.PlanRelation_PLAN_RELATION_NONE:        RelationNone,
		pb.PlanRelation_PLAN_RELATION_UNSPECIFIED: RelationNone,
		pb.PlanRelation(99):                       RelationNone,
	}
	for in, want := range cases {
		if got := relationFromProto(in); got != want {
			t.Errorf("relationFromProto(%v) = %q, want %q", in, got, want)
		}
	}
}

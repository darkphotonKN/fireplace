package checklistitem

import (
	"context"
	"fmt"
	"testing"

	pb "github.com/darkphotonKN/fireplace/common/api/proto/plan"
	commonconstants "github.com/darkphotonKN/fireplace/common/constants"
	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// fakeService records what the transport handed the domain. It embeds Service so
// only the method under test needs implementing.
type fakeService struct {
	Service

	got ReorderInput
	out []*Item
	err error
}

func (f *fakeService) Reorder(ctx context.Context, in ReorderInput) ([]*Item, error) {
	f.got = in
	return f.out, f.err
}

// The transport's whole job here is translation: ids arrive as strings and must
// reach the domain as uuids IN THE ORDER SENT, since that order IS the request.
func TestReorderItems_TranslatesTheRequestAndReturnsTheSiblings(t *testing.T) {
	planID, parentID := uuid.New(), uuid.New()
	first, second := uuid.New(), uuid.New()

	svc := &fakeService{out: []*Item{
		{ID: first, PlanID: planID, Sequence: 1, Scope: ScopeDaily, Type: TypeTask},
		{ID: second, PlanID: planID, Sequence: 2, Scope: ScopeDaily, Type: TypeTask},
	}}
	h := NewHandler(svc)

	parent := parentID.String()
	resp, err := h.ReorderItems(context.Background(), &pb.ReorderItemsRequest{
		PlanId:   planID.String(),
		UserId:   uuid.New().String(),
		Scope:    ScopeDaily,
		ParentId: &parent,
		Ids:      []string{first.String(), second.String()},
	})
	if err != nil {
		t.Fatalf("ReorderItems: %v", err)
	}

	if svc.got.PlanID != planID || svc.got.Scope != ScopeDaily {
		t.Errorf("service received (%s, %s), want (%s, %s)", svc.got.PlanID, svc.got.Scope, planID, ScopeDaily)
	}
	if svc.got.ParentID == nil || *svc.got.ParentID != parentID {
		t.Errorf("service received parent %v, want %s", svc.got.ParentID, parentID)
	}
	if len(svc.got.IDs) != 2 || svc.got.IDs[0] != first || svc.got.IDs[1] != second {
		t.Errorf("service received ids %v, want [%s %s]", svc.got.IDs, first, second)
	}

	if len(resp.Items) != 2 {
		t.Fatalf("response carries %d items, want 2", len(resp.Items))
	}
	if resp.Items[0].Id != first.String() || resp.Items[0].Sequence != "1" {
		t.Errorf("first item = %s/%s, want %s/1", resp.Items[0].Id, resp.Items[0].Sequence, first)
	}
	if resp.Items[1].Sequence != "2" {
		t.Errorf("second item sequence = %s, want 2", resp.Items[1].Sequence)
	}
}

// A malformed id is a client fault, not a 500: it must leave the transport as
// InvalidArgument like every other unparseable id on this service.
func TestReorderItems_MalformedIdIsInvalidArgument(t *testing.T) {
	svc := &fakeService{}
	h := NewHandler(svc)

	_, err := h.ReorderItems(context.Background(), &pb.ReorderItemsRequest{
		PlanId: uuid.New().String(),
		Scope:  ScopeDaily,
		Ids:    []string{"not-a-uuid"},
	})

	if got := status.Code(err); got != codes.InvalidArgument {
		t.Fatalf("code = %s, want InvalidArgument (err: %v)", got, err)
	}
	if svc.got.IDs != nil {
		t.Error("a malformed id reached the domain")
	}
}

// duplicateService records the item the transport addressed, so the tests
// assert what reached the domain, not only the status.
type duplicateService struct {
	Service

	gotPlan, gotID uuid.UUID
	called         bool
	out            []*Item
	err            error
}

func (f *duplicateService) Duplicate(ctx context.Context, planID, id uuid.UUID) ([]*Item, error) {
	f.called, f.gotPlan, f.gotID = true, planID, id
	return f.out, f.err
}

func TestDuplicateItem_AddressesTheItemInItsPlanAndReturnsTheCopiesInOrder(t *testing.T) {
	planID, source := uuid.New(), uuid.New()
	copyID, childID := uuid.New(), uuid.New()
	intention := "why"
	svc := &duplicateService{out: []*Item{
		{ID: copyID, PlanID: planID, Sequence: 3, Type: TypeTask, Scope: ScopeLongterm, Intention: &intention},
		{ID: childID, PlanID: planID, Sequence: 1, Type: TypeTask, Scope: ScopeLongterm, ParentID: &copyID},
	}}

	resp, err := NewHandler(svc).DuplicateItem(context.Background(), &pb.DuplicateItemRequest{
		Id: source.String(), PlanId: planID.String(), UserId: uuid.New().String(),
	})
	if err != nil {
		t.Fatalf("DuplicateItem: %v", err)
	}

	if svc.gotPlan != planID || svc.gotID != source {
		t.Errorf("service received (%s, %s), want (%s, %s)", svc.gotPlan, svc.gotID, planID, source)
	}
	if len(resp.Items) != 2 || resp.Items[0].Id != copyID.String() || resp.Items[1].Id != childID.String() {
		t.Fatalf("response = %v, want [copy child]", resp.Items)
	}
	if resp.Items[0].Intention == nil || *resp.Items[0].Intention != intention {
		t.Errorf("copy intention = %v, want %q", resp.Items[0].Intention, intention)
	}
	if resp.Items[1].ParentId == nil || *resp.Items[1].ParentId != copyID.String() {
		t.Errorf("child parent = %v, want %s", resp.Items[1].ParentId, copyID)
	}
}

func TestDuplicateItem_ErrorsMapToTheirCodes(t *testing.T) {
	cases := []struct {
		name string
		req  *pb.DuplicateItemRequest
		err  error
		want codes.Code
	}{
		{"malformed id", &pb.DuplicateItemRequest{Id: "nope", PlanId: uuid.New().String()}, nil, codes.InvalidArgument},
		{"no plan id", &pb.DuplicateItemRequest{Id: uuid.New().String()}, nil, codes.InvalidArgument},
		{"archived source", &pb.DuplicateItemRequest{Id: uuid.New().String(), PlanId: uuid.New().String()},
			fmt.Errorf("%w: archived", commonconstants.ErrInvalidInput), codes.InvalidArgument},
		{"missing or foreign source", &pb.DuplicateItemRequest{Id: uuid.New().String(), PlanId: uuid.New().String()},
			fmt.Errorf("checklistitem: duplicate: %w", commonconstants.ErrNotFound), codes.NotFound},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			svc := &duplicateService{err: tc.err}

			_, err := NewHandler(svc).DuplicateItem(context.Background(), tc.req)

			if got := status.Code(err); got != tc.want {
				t.Fatalf("code = %s, want %s (err: %v)", got, tc.want, err)
			}
			if tc.err == nil && svc.called {
				t.Error("an unparseable request reached the domain")
			}
		})
	}
}

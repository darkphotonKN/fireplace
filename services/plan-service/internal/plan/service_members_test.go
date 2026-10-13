package plan

import (
	"context"
	"errors"
	"sort"
	"testing"

	commonconstants "github.com/darkphotonKN/fireplace/common/constants"
	"github.com/google/uuid"
)

// The fake's member half: shares carry the time they were written, and a
// re-add writes nothing (ON CONFLICT DO NOTHING).

func (f *fakeRepo) GetByID(ctx context.Context, id uuid.UUID) (*Plan, error) {
	if f.err != nil {
		return nil, f.err
	}
	p, ok := f.plans[id]
	if !ok {
		return nil, commonconstants.ErrNotFound
	}
	return p, nil
}

func (f *fakeRepo) CreateShare(ctx context.Context, planID, userID uuid.UUID) error {
	if f.err != nil {
		return f.err
	}
	f.share(planID, userID)
	return nil
}

func (f *fakeRepo) GetShare(ctx context.Context, planID, userID uuid.UUID) (*Member, error) {
	if !f.shares[userID][planID] {
		return nil, commonconstants.ErrNotFound
	}
	return &Member{UserID: userID, Since: f.since[shareKey{planID, userID}]}, nil
}

func (f *fakeRepo) ListShares(ctx context.Context, planID uuid.UUID) ([]*Member, error) {
	if f.err != nil {
		return nil, f.err
	}
	owner := f.plans[planID].UserID
	out := []*Member{}
	for user, planIDs := range f.shares {
		if planIDs[planID] && user != owner {
			out = append(out, &Member{UserID: user, Since: f.since[shareKey{planID, user}]})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Since.Before(out[j].Since) })
	return out, nil
}

func (f *fakeRepo) DeleteShare(ctx context.Context, planID, userID uuid.UUID) error {
	if !f.shares[userID][planID] {
		return commonconstants.ErrNotFound
	}
	delete(f.shares[userID], planID)
	return nil
}

func (f *fakeRepo) shareCount(planID uuid.UUID) int {
	n := 0
	for _, planIDs := range f.shares {
		if planIDs[planID] {
			n++
		}
	}
	return n
}

// R17: the owner adds another account, and the new member comes back with the
// time the share was written.
func TestAddMember_Owner_SharesThePlan(t *testing.T) {
	repo := newFakeRepo()
	owner, member := uuid.New(), uuid.New()
	planID := repo.addPlan(owner)

	got, err := newTestService(repo).AddMember(context.Background(), planID, owner, member)
	if err != nil {
		t.Fatalf("AddMember: %v", err)
	}
	if got.UserID != member || got.Since.IsZero() {
		t.Errorf("got %+v, want member %s with a since", got, member)
	}
	if !repo.shares[member][planID] {
		t.Error("no share row was written")
	}
}

// R17: re-adding is an idempotent success that keeps the original since and
// writes no second row.
func TestAddMember_Again_KeepsTheOriginalSince(t *testing.T) {
	repo := newFakeRepo()
	owner, member := uuid.New(), uuid.New()
	planID := repo.addPlan(owner)
	svc := newTestService(repo)

	first, err := svc.AddMember(context.Background(), planID, owner, member)
	if err != nil {
		t.Fatalf("first add: %v", err)
	}
	repo.share(planID, uuid.New()) // time moves on
	again, err := svc.AddMember(context.Background(), planID, owner, member)
	if err != nil {
		t.Fatalf("second add: %v", err)
	}
	if !again.Since.Equal(first.Since) {
		t.Errorf("since = %s, want the original %s", again.Since, first.Since)
	}
	if n := repo.shareCount(planID); n != 2 {
		t.Errorf("%d share rows, want 2 (the member once, plus the other)", n)
	}
}

// R17, ADR-0005: the owner is not a member. plan-service owns that rule and
// refuses with ErrInvalidInput, which the wire carries as InvalidArgument.
func TestAddMember_TheOwner_IsInvalidInput(t *testing.T) {
	repo := newFakeRepo()
	owner := uuid.New()
	planID := repo.addPlan(owner)

	_, err := newTestService(repo).AddMember(context.Background(), planID, owner, owner)
	if !errors.Is(err, commonconstants.ErrInvalidInput) {
		t.Fatalf("err = %v, want ErrInvalidInput", err)
	}
	if repo.shareCount(planID) != 0 {
		t.Error("a share row was written for the owner")
	}
}

// Defensive (the gateway checks first): only the plan's owner can add, and a
// missing plan is NotFound.
func TestAddMember_NotTheOwner_IsNotFound(t *testing.T) {
	repo := newFakeRepo()
	owner, member, other := uuid.New(), uuid.New(), uuid.New()
	planID := repo.addPlan(owner)
	repo.share(planID, member)
	svc := newTestService(repo)

	cases := map[string]struct{ plan, caller uuid.UUID }{
		"a member":     {planID, member},
		"a stranger":   {planID, uuid.New()},
		"missing plan": {uuid.New(), owner},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := svc.AddMember(context.Background(), tc.plan, tc.caller, other)
			if !errors.Is(err, commonconstants.ErrNotFound) {
				t.Fatalf("err = %v, want ErrNotFound", err)
			}
			if repo.shares[other][planID] {
				t.Error("a share row was written")
			}
		})
	}
}

// R19: the owner removes a member.
func TestRemoveMember_Owner_DeletesTheShare(t *testing.T) {
	repo := newFakeRepo()
	owner, member := uuid.New(), uuid.New()
	planID := repo.addPlan(owner)
	repo.share(planID, member)

	if err := newTestService(repo).RemoveMember(context.Background(), planID, owner, member); err != nil {
		t.Fatalf("RemoveMember: %v", err)
	}
	if repo.shares[member][planID] {
		t.Error("the share row remains")
	}
}

// R19: no such share (not a member, already removed, or the owner's own id,
// even with a stray share row) is NotFound. So is a caller who isn't the owner
// (defensive), and the share survives that.
func TestRemoveMember_NoSuchShare_IsNotFound(t *testing.T) {
	repo := newFakeRepo()
	owner, member := uuid.New(), uuid.New()
	planID := repo.addPlan(owner)
	repo.share(planID, member)
	repo.share(planID, owner) // a stray row must not make the owner removable
	svc := newTestService(repo)

	cases := map[string]struct{ plan, caller, target uuid.UUID }{
		"not a member":     {planID, owner, uuid.New()},
		"the owner's id":   {planID, owner, owner},
		"member as caller": {planID, member, member},
		"stranger caller":  {planID, uuid.New(), member},
		"missing plan":     {uuid.New(), owner, member},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			err := svc.RemoveMember(context.Background(), tc.plan, tc.caller, tc.target)
			if !errors.Is(err, commonconstants.ErrNotFound) {
				t.Fatalf("err = %v, want ErrNotFound", err)
			}
		})
	}
	if !repo.shares[member][planID] || !repo.shares[owner][planID] {
		t.Error("a refused removal deleted a share")
	}
}

// R15: the owner, the plan's creation time, and the members in add order.
func TestListMembers_OwnerAndMembersInAddOrder(t *testing.T) {
	repo := newFakeRepo()
	owner, first, second := uuid.New(), uuid.New(), uuid.New()
	planID := repo.addPlan(owner)
	repo.share(planID, first)
	repo.share(planID, second)
	repo.share(uuid.New(), uuid.New()) // another plan's share stays out

	got, err := newTestService(repo).ListMembers(context.Background(), planID)
	if err != nil {
		t.Fatalf("ListMembers: %v", err)
	}
	if got.OwnerID != owner || !got.PlanCreatedAt.Equal(repo.plans[planID].CreatedAt) {
		t.Errorf("owner/created = %s/%s", got.OwnerID, got.PlanCreatedAt)
	}
	if len(got.Members) != 2 || got.Members[0].UserID != first || got.Members[1].UserID != second {
		t.Fatalf("members = %+v, want [%s %s]", got.Members, first, second)
	}
}

func TestListMembers_MissingPlan_IsNotFound(t *testing.T) {
	_, err := newTestService(newFakeRepo()).ListMembers(context.Background(), uuid.New())
	if !errors.Is(err, commonconstants.ErrNotFound) {
		t.Fatalf("err = %v, want ErrNotFound", err)
	}
}

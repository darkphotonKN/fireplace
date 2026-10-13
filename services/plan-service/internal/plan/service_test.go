package plan

import (
	"context"
	"errors"
	"testing"
	"time"

	commonconstants "github.com/darkphotonKN/fireplace/common/constants"
	"github.com/google/uuid"
)

// fakeRepo stands in for the database. It embeds Repository so only the methods
// a test exercises need implementing; reaching any other one panics and names it.
type fakeRepo struct {
	Repository

	plans  map[uuid.UUID]*Plan              // plan id → plan
	shares map[uuid.UUID]map[uuid.UUID]bool // user id → plan ids shared with them
	since  map[shareKey]time.Time           // when each share row was written
	clock  time.Time                        // advances one second per share written
	err    error

	deletedSharesFor []uuid.UUID
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{
		plans:  map[uuid.UUID]*Plan{},
		shares: map[uuid.UUID]map[uuid.UUID]bool{},
		since:  map[shareKey]time.Time{},
		clock:  time.Date(2026, 10, 1, 9, 0, 0, 0, time.UTC),
	}
}

type shareKey struct{ plan, user uuid.UUID }

func (f *fakeRepo) addPlan(owner uuid.UUID) uuid.UUID {
	p := &Plan{ID: uuid.New(), UserID: owner, CreatedAt: time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)}
	f.plans[p.ID] = p
	return p.ID
}

func (f *fakeRepo) share(planID, userID uuid.UUID) {
	if f.shares[userID] == nil {
		f.shares[userID] = map[uuid.UUID]bool{}
	}
	if f.shares[userID][planID] {
		return // ON CONFLICT DO NOTHING
	}
	f.shares[userID][planID] = true
	f.clock = f.clock.Add(time.Second)
	f.since[shareKey{planID, userID}] = f.clock
}

func (f *fakeRepo) GetAccess(ctx context.Context, planID, userID uuid.UUID) (*AccessRow, error) {
	if f.err != nil {
		return nil, f.err
	}
	p, ok := f.plans[planID]
	if !ok {
		return nil, commonconstants.ErrNotFound
	}
	return &AccessRow{OwnerID: p.UserID, Shared: f.shares[userID][planID]}, nil
}

func (f *fakeRepo) ListByUser(ctx context.Context, userID uuid.UUID) ([]*Plan, error) {
	out := []*Plan{}
	for _, p := range f.plans {
		if p.UserID == userID {
			out = append(out, p)
		}
	}
	return out, nil
}

func (f *fakeRepo) ListAccessible(ctx context.Context, userID uuid.UUID) ([]*Plan, error) {
	out := []*Plan{}
	for _, p := range f.plans {
		if p.UserID == userID || f.shares[userID][p.ID] {
			out = append(out, p)
		}
	}
	return out, nil
}

func (f *fakeRepo) Delete(ctx context.Context, id, userID uuid.UUID) error {
	p, ok := f.plans[id]
	if !ok || p.UserID != userID {
		return commonconstants.ErrNotFound
	}
	delete(f.plans, id)
	for _, planIDs := range f.shares {
		delete(planIDs, id) // ON DELETE CASCADE
	}
	return nil
}

func (f *fakeRepo) DeleteSharesForUser(ctx context.Context, userID uuid.UUID) error {
	if f.err != nil {
		return f.err
	}
	f.deletedSharesFor = append(f.deletedSharesFor, userID)
	delete(f.shares, userID)
	return nil
}

func newTestService(repo Repository) *service { return NewService(repo, nil, nil) }

// R1–R3: the relation is exactly one of owner, member or none, and a plan that
// does not exist is none, never an error.
func TestGetAccess_AnswersTheRelation(t *testing.T) {
	repo := newFakeRepo()
	owner, member, stranger := uuid.New(), uuid.New(), uuid.New()
	planID := repo.addPlan(owner)
	repo.share(planID, member)
	svc := newTestService(repo)

	cases := []struct {
		name      string
		planID    uuid.UUID
		user      uuid.UUID
		want      Relation
		wantOwner uuid.UUID
	}{
		{"owner", planID, owner, RelationOwner, owner},
		{"member", planID, member, RelationMember, owner},
		{"stranger", planID, stranger, RelationNone, owner},
		{"missing plan", uuid.New(), owner, RelationNone, uuid.Nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := svc.GetAccess(context.Background(), tc.planID, tc.user)
			if err != nil {
				t.Fatalf("GetAccess: %v", err)
			}
			if got.Relation != tc.want {
				t.Errorf("relation = %q, want %q", got.Relation, tc.want)
			}
			if got.OwnerID != tc.wantOwner {
				t.Errorf("owner = %s, want %s", got.OwnerID, tc.wantOwner)
			}
		})
	}
}

// An owner with a stray share row on their own plan is still the owner.
func TestGetAccess_OwnerWithAShareRow_IsOwner(t *testing.T) {
	repo := newFakeRepo()
	owner := uuid.New()
	planID := repo.addPlan(owner)
	repo.share(planID, owner)

	got, err := newTestService(repo).GetAccess(context.Background(), planID, owner)
	if err != nil || got.Relation != RelationOwner {
		t.Fatalf("got %v, %v; want owner", got, err)
	}
}

// A failure to read is not "no relation": the caller must be able to fail
// closed on it rather than mistake it for a stranger.
func TestGetAccess_RepositoryFailure_IsAnError(t *testing.T) {
	repo := newFakeRepo()
	repo.err = commonconstants.ErrTransient

	if _, err := newTestService(repo).GetAccess(context.Background(), uuid.New(), uuid.New()); !errors.Is(err, commonconstants.ErrTransient) {
		t.Fatalf("err = %v, want ErrTransient", err)
	}
}

// R25: user.deleted removes the user's own plans and every share row naming
// them, and a second delivery succeeds as a no-op.
func TestCascadeDeleteForUser_ClearsTheUsersShares(t *testing.T) {
	repo := newFakeRepo()
	gone, other := uuid.New(), uuid.New()
	ownPlan := repo.addPlan(gone)
	othersPlan := repo.addPlan(other)
	repo.share(othersPlan, gone)
	svc := newTestService(repo)

	for delivery := 1; delivery <= 2; delivery++ {
		if err := svc.CascadeDeleteForUser(context.Background(), gone); err != nil {
			t.Fatalf("delivery %d: %v", delivery, err)
		}
	}

	if len(repo.shares[gone]) != 0 {
		t.Errorf("share rows for the deleted user remain: %v", repo.shares[gone])
	}
	if _, ok := repo.plans[ownPlan]; ok {
		t.Error("the deleted user's own plan remains")
	}
	if _, ok := repo.plans[othersPlan]; !ok {
		t.Error("another user's plan was deleted")
	}
	if len(repo.deletedSharesFor) != 2 || repo.deletedSharesFor[0] != gone {
		t.Errorf("shares deleted for %v, want %s on both deliveries", repo.deletedSharesFor, gone)
	}
}

// A share cleanup failure surfaces so the consumer's ack/requeue policy decides.
func TestCascadeDeleteForUser_ShareCleanupFailure_IsReturned(t *testing.T) {
	repo := newFakeRepo()
	repo.err = commonconstants.ErrTransient

	if err := newTestService(repo).CascadeDeleteForUser(context.Background(), uuid.New()); !errors.Is(err, commonconstants.ErrTransient) {
		t.Fatalf("err = %v, want ErrTransient", err)
	}
}

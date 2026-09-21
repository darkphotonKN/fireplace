package checklistitem

import (
	"context"
	"errors"
	"testing"

	commonconstants "github.com/darkphotonKN/fireplace/common/constants"
	"github.com/google/uuid"
)

// fakeRepo stands in for the database. It embeds Repository so only the methods
// a test actually exercises need implementing — the same idiom the gateway's
// recordingClient uses. A call to an unimplemented method panics, which is the
// point: it names the method the code reached that the test did not expect.
type fakeRepo struct {
	Repository

	siblings    []*Item             // what ListSiblings answers
	known       map[uuid.UUID]*Item // every row that exists anywhere
	siblingsErr error

	gotSiblings SiblingSetInput
	gotOrder    []uuid.UUID
	reorderErr  error
}

func (f *fakeRepo) ListSiblings(ctx context.Context, in SiblingSetInput) ([]*Item, error) {
	f.gotSiblings = in
	if f.siblingsErr != nil {
		return nil, f.siblingsErr
	}
	return f.siblings, nil
}

func (f *fakeRepo) GetByID(ctx context.Context, id uuid.UUID) (*Item, error) {
	if item, ok := f.known[id]; ok {
		return item, nil
	}
	return nil, commonconstants.ErrNotFound
}

// Reorder mimics the real transaction: dense positions 1..N in the order given,
// read back in stored order.
func (f *fakeRepo) Reorder(ctx context.Context, ids []uuid.UUID) ([]*Item, error) {
	f.gotOrder = append([]uuid.UUID(nil), ids...)
	if f.reorderErr != nil {
		return nil, f.reorderErr
	}
	out := make([]*Item, 0, len(ids))
	for i, id := range ids {
		item := &Item{ID: id, Sequence: i + 1}
		if known, ok := f.known[id]; ok {
			copied := *known
			copied.Sequence = i + 1
			item = &copied
		}
		out = append(out, item)
	}
	return out, nil
}

// siblingSet builds a plan's top-level set of n items, sequence 1..n, and a
// repo that knows about exactly those rows.
func siblingSet(planID uuid.UUID, scope string, parentID *uuid.UUID, n int) (*fakeRepo, []uuid.UUID) {
	repo := &fakeRepo{known: map[uuid.UUID]*Item{}}
	ids := make([]uuid.UUID, 0, n)
	for i := 0; i < n; i++ {
		item := &Item{
			ID:       uuid.New(),
			PlanID:   planID,
			Scope:    scope,
			ParentID: parentID,
			Sequence: i + 1,
		}
		repo.siblings = append(repo.siblings, item)
		repo.known[item.ID] = item
		ids = append(ids, item.ID)
	}
	return repo, ids
}

func newTestService(repo Repository) *service {
	return NewService(repo, nil)
}

// R2.5: the server assigns dense positions (1..N) across the set in the order
// given, and returns the reordered siblings.
func TestReorder_AssignsDensePositionsInTheOrderGiven(t *testing.T) {
	planID := uuid.New()
	repo, ids := siblingSet(planID, ScopeDaily, nil, 3)
	svc := newTestService(repo)

	want := []uuid.UUID{ids[2], ids[0], ids[1]}

	got, err := svc.Reorder(context.Background(), ReorderInput{
		PlanID: planID,
		Scope:  ScopeDaily,
		IDs:    want,
	})
	if err != nil {
		t.Fatalf("Reorder: unexpected error: %v", err)
	}
	if len(got) != len(want) {
		t.Fatalf("returned %d items, want %d", len(got), len(want))
	}
	for i, item := range got {
		if item.ID != want[i] {
			t.Errorf("position %d: id = %s, want %s", i, item.ID, want[i])
		}
		if item.Sequence != i+1 {
			t.Errorf("item %s: sequence = %d, want %d", item.ID, item.Sequence, i+1)
		}
	}
	if len(repo.gotOrder) != len(want) {
		t.Fatalf("repository received %d ids, want %d", len(repo.gotOrder), len(want))
	}
	for i, id := range repo.gotOrder {
		if id != want[i] {
			t.Errorf("repository position %d: id = %s, want %s", i, id, want[i])
		}
	}
}

// FS-0009 §Edge States: a request whose ids are not exactly a permutation of the
// set — here, one member left out — is refused whole.
func TestReorder_IdsMissingAMemberAreRefused(t *testing.T) {
	planID := uuid.New()
	repo, ids := siblingSet(planID, ScopeDaily, nil, 3)
	svc := newTestService(repo)

	_, err := svc.Reorder(context.Background(), ReorderInput{
		PlanID: planID,
		Scope:  ScopeDaily,
		IDs:    []uuid.UUID{ids[1], ids[0]}, // ids[2] is left out
	})

	if !isInvalidInput(err) {
		t.Fatalf("err = %v, want ErrInvalidInput", err)
	}
	if repo.gotOrder != nil {
		t.Error("the repository was asked to write an order that is not a permutation of the set")
	}
}

// An id that belongs to another parent, another scope or another plan is a
// stranger to this set: 400 VALIDATION_FAILED (FS-0009 §API surface). An id that
// exists nowhere is 404 NOT_FOUND — the two are told apart by whether the row
// exists at all.
func TestReorder_StrangerIdsAreRefused(t *testing.T) {
	planID := uuid.New()
	otherParent := uuid.New()

	cases := []struct {
		name      string
		stranger  *Item // nil means the id exists nowhere
		wantIsErr func(error) bool
		wantLabel string
	}{
		{
			name:      "another parent",
			stranger:  &Item{ID: uuid.New(), PlanID: planID, Scope: ScopeDaily, ParentID: &otherParent},
			wantIsErr: isInvalidInput,
			wantLabel: "ErrInvalidInput",
		},
		{
			name:      "another scope",
			stranger:  &Item{ID: uuid.New(), PlanID: planID, Scope: ScopeLongterm},
			wantIsErr: isInvalidInput,
			wantLabel: "ErrInvalidInput",
		},
		{
			name:      "another plan",
			stranger:  &Item{ID: uuid.New(), PlanID: uuid.New(), Scope: ScopeDaily},
			wantIsErr: isInvalidInput,
			wantLabel: "ErrInvalidInput",
		},
		{
			name:      "no such item",
			stranger:  nil,
			wantIsErr: isNotFound,
			wantLabel: "ErrNotFound",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			repo, ids := siblingSet(planID, ScopeDaily, nil, 2)
			strangerID := uuid.New()
			if tc.stranger != nil {
				strangerID = tc.stranger.ID
				repo.known[strangerID] = tc.stranger
			}
			svc := newTestService(repo)

			_, err := svc.Reorder(context.Background(), ReorderInput{
				PlanID: planID,
				Scope:  ScopeDaily,
				IDs:    []uuid.UUID{ids[0], strangerID, ids[1]},
			})

			if !tc.wantIsErr(err) {
				t.Fatalf("err = %v, want %s", err, tc.wantLabel)
			}
			if repo.gotOrder != nil {
				t.Error("the repository was asked to write an order containing a stranger")
			}
		})
	}
}

// A repeated id passes a naive membership-plus-length check while silently
// dropping a member, so it is refused explicitly.
func TestReorder_DuplicateIdsAreRefused(t *testing.T) {
	planID := uuid.New()
	repo, ids := siblingSet(planID, ScopeDaily, nil, 2)
	svc := newTestService(repo)

	_, err := svc.Reorder(context.Background(), ReorderInput{
		PlanID: planID,
		Scope:  ScopeDaily,
		IDs:    []uuid.UUID{ids[0], ids[0]},
	})

	if !isInvalidInput(err) {
		t.Fatalf("err = %v, want ErrInvalidInput", err)
	}
	if repo.gotOrder != nil {
		t.Error("the repository was asked to write an order containing the same id twice")
	}
}

// The scope names half of the sibling set's address, so a value outside the
// domain's two is refused before any row is read — the same rule every other
// method on this service applies.
func TestReorder_InvalidScopeIsRefused(t *testing.T) {
	planID := uuid.New()
	repo, ids := siblingSet(planID, ScopeDaily, nil, 2)
	svc := newTestService(repo)

	_, err := svc.Reorder(context.Background(), ReorderInput{
		PlanID: planID,
		Scope:  "weekly",
		IDs:    ids,
	})

	if !isInvalidInput(err) {
		t.Fatalf("err = %v, want ErrInvalidInput", err)
	}
	if repo.gotOrder != nil {
		t.Error("the repository was asked to write an order for an unknown scope")
	}
}

// Nothing to order is not an order. An empty ids list would otherwise read as
// "the set is empty", which is a claim about the server's state, not a request.
func TestReorder_EmptyIdsAreRefused(t *testing.T) {
	planID := uuid.New()
	// The set is empty too: an empty request is bad input in its own right, not
	// a vacuous truth about a set that happens to have no members.
	repo := &fakeRepo{known: map[uuid.UUID]*Item{}}
	svc := newTestService(repo)

	_, err := svc.Reorder(context.Background(), ReorderInput{
		PlanID: planID,
		Scope:  ScopeDaily,
		IDs:    nil,
	})

	if !isInvalidInput(err) {
		t.Fatalf("err = %v, want ErrInvalidInput", err)
	}
	if repo.gotOrder != nil {
		t.Error("the repository was asked to write an empty order")
	}
}

// An unknown plan — or a parent id that addresses nothing — has no sibling set
// to reorder: 404 NOT_FOUND (FS-0009 §API surface).
func TestReorder_UnknownSiblingSetIsNotFound(t *testing.T) {
	// ListSiblings answers nothing, while the id sent DOES exist — somewhere
	// else. The missing SET is what decides the status, so this is 404 and not
	// the 400 a stranger id would earn inside a set that exists.
	elsewhere := &Item{ID: uuid.New(), PlanID: uuid.New(), Scope: ScopeDaily}
	repo := &fakeRepo{known: map[uuid.UUID]*Item{elsewhere.ID: elsewhere}}
	svc := newTestService(repo)

	_, err := svc.Reorder(context.Background(), ReorderInput{
		PlanID: uuid.New(),
		Scope:  ScopeDaily,
		IDs:    []uuid.UUID{elsewhere.ID},
	})

	if !isNotFound(err) {
		t.Fatalf("err = %v, want ErrNotFound", err)
	}
	if repo.gotOrder != nil {
		t.Error("the repository was asked to write an order for a set that does not exist")
	}
}

// R2.3: sending the order a set already has writes the same numbers and errors
// on nothing.
func TestReorder_SendingTheSameOrderTwiceIsANoOp(t *testing.T) {
	planID := uuid.New()
	repo, ids := siblingSet(planID, ScopeLongterm, nil, 3)
	svc := newTestService(repo)
	in := ReorderInput{PlanID: planID, Scope: ScopeLongterm, IDs: ids}

	first, err := svc.Reorder(context.Background(), in)
	if err != nil {
		t.Fatalf("first Reorder: %v", err)
	}
	second, err := svc.Reorder(context.Background(), in)
	if err != nil {
		t.Fatalf("second Reorder: %v", err)
	}

	for i := range first {
		if first[i].ID != second[i].ID || first[i].Sequence != second[i].Sequence {
			t.Errorf("position %d changed between identical requests: %s/%d then %s/%d",
				i, first[i].ID, first[i].Sequence, second[i].ID, second[i].Sequence)
		}
	}
}

// R2.1: a top-level set is addressed with parentId nil, a child set with its
// parent's id — so the parent has to reach the query that reads the set.
func TestReorder_AddressesTheSetByParent(t *testing.T) {
	planID := uuid.New()
	parentID := uuid.New()
	repo, ids := siblingSet(planID, ScopeDaily, &parentID, 2)
	svc := newTestService(repo)

	if _, err := svc.Reorder(context.Background(), ReorderInput{
		PlanID:   planID,
		Scope:    ScopeDaily,
		ParentID: &parentID,
		IDs:      []uuid.UUID{ids[1], ids[0]},
	}); err != nil {
		t.Fatalf("Reorder: %v", err)
	}

	if repo.gotSiblings.ParentID == nil || *repo.gotSiblings.ParentID != parentID {
		t.Errorf("sibling set read with parent %v, want %s", repo.gotSiblings.ParentID, parentID)
	}
	if repo.gotSiblings.PlanID != planID || repo.gotSiblings.Scope != ScopeDaily {
		t.Errorf("sibling set read as (%s, %s), want (%s, %s)",
			repo.gotSiblings.PlanID, repo.gotSiblings.Scope, planID, ScopeDaily)
	}
}

// A failed write is reported as a failure; the caller never gets a list that
// implies an order the database does not hold (R2.2 at this layer — the
// transaction itself is the repository's job).
func TestReorder_WriteFailureIsNotReportedAsSuccess(t *testing.T) {
	planID := uuid.New()
	repo, ids := siblingSet(planID, ScopeDaily, nil, 2)
	repo.reorderErr = commonconstants.ErrTransient
	svc := newTestService(repo)

	items, err := svc.Reorder(context.Background(), ReorderInput{
		PlanID: planID,
		Scope:  ScopeDaily,
		IDs:    ids,
	})

	if err == nil {
		t.Fatal("a failed write returned no error")
	}
	if items != nil {
		t.Errorf("a failed write returned %d items", len(items))
	}
}

func isInvalidInput(err error) bool { return errors.Is(err, commonconstants.ErrInvalidInput) }
func isNotFound(err error) bool     { return errors.Is(err, commonconstants.ErrNotFound) }

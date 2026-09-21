package checklistitem

import (
	"context"
	"errors"
	"testing"
	"time"

	commonconstants "github.com/darkphotonKN/fireplace/common/constants"
	"github.com/google/uuid"
)

// The reorder write is the one piece of I-0055 that no unit test can reach: a
// fake repository proves the service calls it, not that the SQL does anything.
// `unnest($1::uuid[]) WITH ORDINALITY` joined against an UPDATE, and the row
// count that decides whether to abandon the transaction, are the database's
// behaviour and are exercised here against the real one.

func seqOf(t *testing.T, db interface {
	Get(dest interface{}, query string, args ...interface{}) error
}, id uuid.UUID) int {
	t.Helper()
	var seq int
	if err := db.Get(&seq, `SELECT sequence FROM checklist_items WHERE id = $1`, id); err != nil {
		t.Fatalf("read sequence: %v", err)
	}
	return seq
}

func TestIntegrationReorder_WritesDensePositionsInTheOrderGiven(t *testing.T) {
	db := openTestDB(t)
	planID := seedPlan(t, db)

	at := time.Date(2026, 3, 9, 12, 0, 0, 0, time.UTC)
	a, b, c := uuid.New(), uuid.New(), uuid.New()
	// Sparse, out-of-order starting sequences, as the global counter produces.
	seedItem(t, db, planID, a, 4100, at)
	seedItem(t, db, planID, b, 17, at.Add(time.Minute))
	seedItem(t, db, planID, c, 950, at.Add(2*time.Minute))

	got, err := NewRepository(db).Reorder(context.Background(), []uuid.UUID{c, a, b})
	if err != nil {
		t.Fatalf("reorder: %v", err)
	}

	if seqOf(t, db, c) != 1 || seqOf(t, db, a) != 2 || seqOf(t, db, b) != 3 {
		t.Fatalf("stored sequences are %d, %d, %d; want 1, 2, 3 for c, a, b",
			seqOf(t, db, c), seqOf(t, db, a), seqOf(t, db, b))
	}
	want := []uuid.UUID{c, a, b}
	if len(got) != len(want) {
		t.Fatalf("read back %d rows, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i].ID != want[i] {
			t.Fatalf("read-back position %d: got %s, want %s", i, got[i].ID, want[i])
		}
	}
}

func TestIntegrationReorder_VanishedIDAbandonsTheWholeWrite(t *testing.T) {
	db := openTestDB(t)
	planID := seedPlan(t, db)

	at := time.Date(2026, 3, 9, 12, 0, 0, 0, time.UTC)
	a, b := uuid.New(), uuid.New()
	seedItem(t, db, planID, a, 10, at)
	seedItem(t, db, planID, b, 20, at.Add(time.Minute))
	// An id that was in the set when it was validated and is gone by the write,
	// which is what an item deleted in another tab mid-drag looks like.
	gone := uuid.New()

	_, err := NewRepository(db).Reorder(context.Background(), []uuid.UUID{b, gone, a})

	if err == nil {
		t.Fatal("reorder with a vanished id reported success")
	}
	if !errors.Is(err, commonconstants.ErrNotFound) {
		t.Fatalf("got %v, want it to wrap ErrNotFound", err)
	}
	// The point of the transaction: the two surviving rows keep the sequences
	// they had. A partial write would leave b at 1 and a at 3.
	if seqOf(t, db, a) != 10 || seqOf(t, db, b) != 20 {
		t.Fatalf("sequences moved to %d and %d; want 10 and 20 — the write was not abandoned",
			seqOf(t, db, a), seqOf(t, db, b))
	}
}

func TestIntegrationListSiblings_AddressesTopLevelAndChildSetsSeparately(t *testing.T) {
	db := openTestDB(t)
	planID := seedPlan(t, db)

	at := time.Date(2026, 3, 9, 12, 0, 0, 0, time.UTC)
	parent, other, child := uuid.New(), uuid.New(), uuid.New()
	seedItem(t, db, planID, parent, 1, at)
	seedItem(t, db, planID, other, 2, at.Add(time.Minute))
	seedItem(t, db, planID, child, 3, at.Add(2*time.Minute))
	if _, err := db.Exec(
		`UPDATE checklist_items SET parent_id = $2 WHERE id = $1`, child, parent); err != nil {
		t.Fatalf("nest child: %v", err)
	}

	repo := NewRepository(db)
	ctx := context.Background()

	top, err := repo.ListSiblings(ctx, SiblingSetInput{PlanID: planID, Scope: ScopeLongterm})
	if err != nil {
		t.Fatalf("list top-level siblings: %v", err)
	}
	if len(top) != 2 || top[0].ID != parent || top[1].ID != other {
		t.Fatalf("top-level set is %v, want [%s %s]", ids(top), parent, other)
	}

	kids, err := repo.ListSiblings(ctx,
		SiblingSetInput{PlanID: planID, Scope: ScopeLongterm, ParentID: &parent})
	if err != nil {
		t.Fatalf("list child siblings: %v", err)
	}
	if len(kids) != 1 || kids[0].ID != child {
		t.Fatalf("child set is %v, want [%s]", ids(kids), child)
	}
}

func ids(items []*Item) []uuid.UUID {
	out := make([]uuid.UUID, 0, len(items))
	for _, it := range items {
		out = append(out, it.ID)
	}
	return out
}

func TestIntegrationListSiblings_TiedSequences_OrdersByCreatedAt(t *testing.T) {
	// ListSiblings arrived with I-0055, after I-0054 made every list order
	// total. It is a list order too: the set it returns is what the client
	// renders and drags against, so it must not reshuffle between reads.
	db := openTestDB(t)
	planID := seedPlan(t, db)

	base := time.Date(2026, 3, 9, 12, 0, 0, 0, time.UTC)
	oldest, newest := uuid.New(), uuid.New()
	seedItem(t, db, planID, newest, 7, base.Add(time.Hour))
	seedItem(t, db, planID, oldest, 7, base)

	got, err := NewRepository(db).ListSiblings(
		context.Background(), SiblingSetInput{PlanID: planID, Scope: ScopeLongterm})
	if err != nil {
		t.Fatalf("list siblings: %v", err)
	}

	if len(got) != 2 || got[0].ID != oldest || got[1].ID != newest {
		t.Fatalf("got %v, want [%s %s]", ids(got), oldest, newest)
	}
}

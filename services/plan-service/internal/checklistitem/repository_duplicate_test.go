package checklistitem

import (
	"context"
	"errors"
	"testing"
	"time"

	commonconstants "github.com/darkphotonKN/fireplace/common/constants"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// FS-F6AEE R2–R7 against the real table. The copy, its children and the
// placement are one transaction, and "directly after the source" is a property
// of the stored order, so none of it can be proven with a fake.

func strp(s string) *string { return &s }

// shape writes the columns seedItem leaves at their defaults. A nil intention
// leaves the column NULL.
func shape(t *testing.T, db *sqlx.DB, id uuid.UUID, parent *uuid.UUID, itemType string, done bool, intention *string) {
	t.Helper()
	if _, err := db.Exec(`
		UPDATE checklist_items
		SET parent_id = $2, type = $3, done = $4, intention = $5
		WHERE id = $1`, id, parent, itemType, done, intention); err != nil {
		t.Fatalf("shape item %s: %v", id, err)
	}
}

func siblingIDs(t *testing.T, db *sqlx.DB, planID uuid.UUID, parent *uuid.UUID) []uuid.UUID {
	t.Helper()
	got, err := NewRepository(db).ListSiblings(context.Background(),
		SiblingSetInput{PlanID: planID, Scope: ScopeLongterm, ParentID: parent})
	if err != nil {
		t.Fatalf("list siblings: %v", err)
	}
	return ids(got)
}

func sameIDs(a, b []uuid.UUID) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func countItems(t *testing.T, db *sqlx.DB, planID uuid.UUID) int {
	t.Helper()
	var n int
	if err := db.Get(&n, `SELECT COUNT(*) FROM checklist_items WHERE plan_id = $1`, planID); err != nil {
		t.Fatalf("count items: %v", err)
	}
	return n
}

func TestIntegrationDuplicate_TopLevelCopiesLiveChildrenAndLandsAfterTheSource(t *testing.T) {
	db := openTestDB(t)
	planID := seedPlan(t, db)

	at := time.Date(2026, 3, 9, 12, 0, 0, 0, time.UTC)
	day := time.Date(2026, 3, 11, 0, 0, 0, 0, time.UTC) // start_date/due_date are DATE columns
	a, source, b := uuid.New(), uuid.New(), uuid.New()
	seedItem(t, db, planID, a, 10, at)
	seedDatedItem(t, db, planID, source, 20, at.Add(time.Minute), day)
	seedItem(t, db, planID, b, 30, at.Add(2*time.Minute))
	shape(t, db, source, nil, TypeTask, true, strp("ship the beta"))

	doneKid, archivedKid, plainKid := uuid.New(), uuid.New(), uuid.New()
	seedItem(t, db, planID, doneKid, 5, at.Add(3*time.Minute))
	seedArchivedItem(t, db, planID, archivedKid, 6, at.Add(4*time.Minute))
	seedItem(t, db, planID, plainKid, 7, at.Add(5*time.Minute))
	shape(t, db, doneKid, &source, TypeTask, true, strp("first, the schema"))
	shape(t, db, archivedKid, &source, TypeTask, false, strp("set aside"))
	shape(t, db, plainKid, &source, TypeNote, false, nil)

	got, err := NewRepository(db).Duplicate(context.Background(), planID, source)
	if err != nil {
		t.Fatalf("duplicate: %v", err)
	}

	if len(got) != 3 {
		t.Fatalf("returned %d items, want the copy and its 2 live children", len(got))
	}
	cp := got[0]
	if cp.ID == source {
		t.Fatal("the copy reused the source's id")
	}
	if cp.Description != "item "+source.String()[:8] || cp.Type != TypeTask ||
		cp.Scope != ScopeLongterm || cp.ParentID != nil || cp.PlanID != planID {
		t.Errorf("copy = %+v, want the source's description, type, scope and no parent", cp)
	}
	if cp.Done || cp.Archived {
		t.Errorf("copy done=%v archived=%v, want both false", cp.Done, cp.Archived)
	}
	if cp.Intention == nil || *cp.Intention != "ship the beta" {
		t.Errorf("copy intention = %v, want %q", cp.Intention, "ship the beta")
	}
	if cp.StartDate == nil || cp.DueDate == nil ||
		!cp.StartDate.Equal(day) || !cp.DueDate.Equal(day) {
		t.Errorf("copy dates = %v / %v, want the source's", cp.StartDate, cp.DueDate)
	}

	first, second := got[1], got[2]
	if first.Description != "item "+doneKid.String()[:8] || second.Description != "item "+plainKid.String()[:8] {
		t.Errorf("children came back as %q, %q; want the done kid's then the plain kid's",
			first.Description, second.Description)
	}
	for _, kid := range []*Item{first, second} {
		if kid.ParentID == nil || *kid.ParentID != cp.ID {
			t.Errorf("child %s parent = %v, want the copy %s", kid.ID, kid.ParentID, cp.ID)
		}
		if kid.Done || kid.Archived {
			t.Errorf("child %s done=%v archived=%v, want both false", kid.ID, kid.Done, kid.Archived)
		}
		if kid.ID == doneKid || kid.ID == plainKid || kid.ID == archivedKid {
			t.Errorf("child %s is a source row, not a copy", kid.ID)
		}
	}
	if first.Intention == nil || *first.Intention != "first, the schema" {
		t.Errorf("first child intention = %v, want its source's", first.Intention)
	}
	if second.Intention != nil {
		t.Errorf("second child intention = %q, want NULL like its source", *second.Intention)
	}
	if second.Type != TypeNote {
		t.Errorf("second child type = %q, want note", second.Type)
	}

	if top := siblingIDs(t, db, planID, nil); !sameIDs(top, []uuid.UUID{a, source, cp.ID, b}) {
		t.Errorf("top-level set = %v, want [a source copy b] = %v", top, []uuid.UUID{a, source, cp.ID, b})
	}
	if kids := siblingIDs(t, db, planID, &cp.ID); !sameIDs(kids, []uuid.UUID{first.ID, second.ID}) {
		t.Errorf("copy's children = %v, want %v", kids, []uuid.UUID{first.ID, second.ID})
	}
	// The source's own children are untouched.
	if kids := siblingIDs(t, db, planID, &source); !sameIDs(kids, []uuid.UUID{doneKid, plainKid}) {
		t.Errorf("source's children = %v, want [%s %s]", kids, doneKid, plainKid)
	}
}

func TestIntegrationDuplicate_ChildSourceCopiesOnlyItselfUnderTheSameParent(t *testing.T) {
	db := openTestDB(t)
	planID := seedPlan(t, db)

	at := time.Date(2026, 3, 9, 12, 0, 0, 0, time.UTC)
	parent, k1, k2, k3 := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	seedItem(t, db, planID, parent, 1, at)
	seedItem(t, db, planID, k1, 1, at.Add(time.Minute))
	seedItem(t, db, planID, k2, 2, at.Add(2*time.Minute))
	seedItem(t, db, planID, k3, 3, at.Add(3*time.Minute))
	for _, k := range []uuid.UUID{k1, k2, k3} {
		shape(t, db, k, &parent, TypeTask, false, nil)
	}

	got, err := NewRepository(db).Duplicate(context.Background(), planID, k2)
	if err != nil {
		t.Fatalf("duplicate: %v", err)
	}

	if len(got) != 1 {
		t.Fatalf("returned %d items, want 1", len(got))
	}
	if got[0].ParentID == nil || *got[0].ParentID != parent {
		t.Errorf("copy parent = %v, want %s", got[0].ParentID, parent)
	}
	if got[0].Intention != nil {
		t.Errorf("copy intention = %q, want NULL like its source", *got[0].Intention)
	}
	if kids := siblingIDs(t, db, planID, &parent); !sameIDs(kids, []uuid.UUID{k1, k2, got[0].ID, k3}) {
		t.Errorf("children = %v, want [k1 k2 copy k3]", kids)
	}
}

func TestIntegrationDuplicate_LastOfSetBecomesTheNewLast(t *testing.T) {
	db := openTestDB(t)
	planID := seedPlan(t, db)

	at := time.Date(2026, 3, 9, 12, 0, 0, 0, time.UTC)
	a, last := uuid.New(), uuid.New()
	seedItem(t, db, planID, a, 1, at)
	seedItem(t, db, planID, last, 2, at.Add(time.Minute))

	got, err := NewRepository(db).Duplicate(context.Background(), planID, last)
	if err != nil {
		t.Fatalf("duplicate: %v", err)
	}
	if top := siblingIDs(t, db, planID, nil); !sameIDs(top, []uuid.UUID{a, last, got[0].ID}) {
		t.Errorf("top-level set = %v, want [a last copy]", top)
	}
}

func TestIntegrationDuplicate_NoteSourceGivesAnUndoneNote(t *testing.T) {
	db := openTestDB(t)
	planID := seedPlan(t, db)

	note := uuid.New()
	seedItem(t, db, planID, note, 1, time.Now())
	shape(t, db, note, nil, TypeNote, false, strp("remember"))

	got, err := NewRepository(db).Duplicate(context.Background(), planID, note)
	if err != nil {
		t.Fatalf("duplicate: %v", err)
	}
	if len(got) != 1 || got[0].Type != TypeNote || got[0].Done {
		t.Fatalf("got %+v, want one undone note", got)
	}
}

func TestIntegrationDuplicate_TiedSequencesPlaceTheCopyAfterTheSourceInStoredOrder(t *testing.T) {
	db := openTestDB(t)
	planID := seedPlan(t, db)

	// Three rows share a sequence; created_at decides their order.
	base := time.Date(2026, 3, 9, 12, 0, 0, 0, time.UTC)
	a, source, b := uuid.New(), uuid.New(), uuid.New()
	seedItem(t, db, planID, b, 7, base.Add(2*time.Hour))
	seedItem(t, db, planID, source, 7, base.Add(time.Hour))
	seedItem(t, db, planID, a, 7, base)

	got, err := NewRepository(db).Duplicate(context.Background(), planID, source)
	if err != nil {
		t.Fatalf("duplicate: %v", err)
	}
	if top := siblingIDs(t, db, planID, nil); !sameIDs(top, []uuid.UUID{a, source, got[0].ID, b}) {
		t.Errorf("top-level set = %v, want [a source copy b]", top)
	}
	if seqOf(t, db, a) != 1 || seqOf(t, db, source) != 2 || seqOf(t, db, got[0].ID) != 3 || seqOf(t, db, b) != 4 {
		t.Errorf("sequences not rewritten densely")
	}
}

// R6: a failure after the copy is inserted abandons everything — no copy, no
// copied child, no rewritten sequence.
func TestIntegrationDuplicate_FailureAfterTheFirstInsertRollsEverythingBack(t *testing.T) {
	db := openTestDB(t)
	planID := seedPlan(t, db)

	at := time.Date(2026, 3, 9, 12, 0, 0, 0, time.UTC)
	source, kid, b := uuid.New(), uuid.New(), uuid.New()
	seedItem(t, db, planID, source, 40, at)
	seedItem(t, db, planID, b, 90, at.Add(time.Minute))
	seedItem(t, db, planID, kid, 3, at.Add(2*time.Minute))
	shape(t, db, kid, &source, TypeTask, false, nil)

	repo := NewRepository(db)
	boom := errors.New("forced failure")
	repo.afterCopyInsert = func() error { return boom }

	_, err := repo.Duplicate(context.Background(), planID, source)

	if !errors.Is(err, boom) {
		t.Fatalf("err = %v, want the forced failure", err)
	}
	if n := countItems(t, db, planID); n != 3 {
		t.Errorf("plan holds %d items, want the 3 it was seeded with", n)
	}
	if seqOf(t, db, source) != 40 || seqOf(t, db, b) != 90 {
		t.Errorf("sequences moved to %d, %d; want 40, 90", seqOf(t, db, source), seqOf(t, db, b))
	}
}

// R8 at the write itself: the row is re-read under a lock, so an item archived
// or moved between the service's check and the transaction is still refused.
func TestIntegrationDuplicate_RefusesAnArchivedOrForeignSourceAndWritesNothing(t *testing.T) {
	db := openTestDB(t)
	planID, otherPlan := seedPlan(t, db), seedPlan(t, db)

	at := time.Date(2026, 3, 9, 12, 0, 0, 0, time.UTC)
	archived, foreign := uuid.New(), uuid.New()
	seedArchivedItem(t, db, planID, archived, 1, at)
	seedItem(t, db, otherPlan, foreign, 1, at)
	repo := NewRepository(db)

	if _, err := repo.Duplicate(context.Background(), planID, archived); !errors.Is(err, commonconstants.ErrInvalidInput) {
		t.Errorf("archived source: err = %v, want ErrInvalidInput", err)
	}
	if _, err := repo.Duplicate(context.Background(), planID, foreign); !errors.Is(err, commonconstants.ErrNotFound) {
		t.Errorf("source in another plan: err = %v, want ErrNotFound", err)
	}
	if _, err := repo.Duplicate(context.Background(), planID, uuid.New()); !errors.Is(err, commonconstants.ErrNotFound) {
		t.Errorf("missing source: err = %v, want ErrNotFound", err)
	}
	if countItems(t, db, planID) != 1 || countItems(t, db, otherPlan) != 1 {
		t.Error("a refused duplicate wrote a row")
	}
}

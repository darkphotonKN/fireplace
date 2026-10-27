package checklistitem

import (
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// FS-02JBZ R50 / S16: migration 000022 turns every daily row into a long-term
// one without losing anything, and places the migrated rows after the existing
// long-term siblings at the same level, in their old relative order.
//
// The whole test runs in ONE transaction that is always rolled back. The
// migration's UPDATE is global, so against a shared dev database it would
// otherwise convert real rows; inside a rolled-back transaction it converts
// nothing that outlives the test.

const retireDailyUp = "../../migrations/000022_retire_daily_scope.up.sql"
const retireDailyDown = "../../migrations/000022_retire_daily_scope.down.sql"

type migratedRow struct {
	ID        uuid.UUID  `db:"id"`
	Scope     string     `db:"scope"`
	Done      bool       `db:"done"`
	Archived  bool       `db:"archived"`
	ParentID  *uuid.UUID `db:"parent_id"`
	StartDate *time.Time `db:"start_date"`
	DueDate   *time.Time `db:"due_date"`
	Intention *string    `db:"intention"`
}

func TestIntegrationRetireDailyScope_MigratesEveryDailyRowAfterTheLongtermSiblings(t *testing.T) {
	db := openTestDB(t)
	up, err := os.ReadFile(retireDailyUp)
	if err != nil {
		t.Fatalf("read up migration: %v", err)
	}

	tx, err := db.Beginx()
	if err != nil {
		t.Fatalf("begin: %v", err)
	}
	defer tx.Rollback() //nolint:errcheck // rollback is the point

	plan, other := uuid.New(), uuid.New()
	for _, id := range []uuid.UUID{plan, other} {
		if _, err := tx.Exec(
			`INSERT INTO plans (id, user_id, name, plan_type, focus) VALUES ($1, $2, $3, $4, $5)`,
			id, uuid.New(), "FS-02JBZ migration fixture", "project", "fixture",
		); err != nil {
			t.Fatalf("seed plan: %v", err)
		}
	}

	base := time.Date(2026, 5, 1, 9, 0, 0, 0, time.UTC)
	insert := func(planID uuid.UUID, scope string, seq int, parent *uuid.UUID) uuid.UUID {
		t.Helper()
		id := uuid.New()
		if _, err := tx.Exec(`
			INSERT INTO checklist_items
				(id, plan_id, description, done, sequence, scope, type, parent_id, created_at, updated_at)
			VALUES ($1, $2, $3, false, $4, $5, 'task', $6, $7, $7)`,
			id, planID, scope+" item", seq, scope, parent, base.Add(time.Duration(seq)*time.Minute),
		); err != nil {
			t.Fatalf("seed item: %v", err)
		}
		return id
	}

	// Top level of plan: two long-term rows, then three daily rows whose old
	// relative order (by sequence) is d3, d1, d2 — deliberately interleaved with
	// the long-term sequences, so "after" has to be computed, not inherited.
	l1 := insert(plan, ScopeLongterm, 5, nil)
	l2 := insert(plan, ScopeLongterm, 9, nil)
	d3 := insert(plan, ScopeDaily, 1, nil)
	d1 := insert(plan, ScopeDaily, 3, nil)
	d2 := insert(plan, ScopeDaily, 7, nil)

	// Nested: a daily child under a long-term parent goes after that parent's
	// long-term children; a daily child under a daily parent keeps its parent.
	lc1 := insert(plan, ScopeLongterm, 20, &l1)
	dc1 := insert(plan, ScopeDaily, 2, &l1)
	dc2 := insert(plan, ScopeDaily, 4, &d1)

	// Another plan's long-term tail must not push this plan's rows down.
	insert(other, ScopeLongterm, 500, nil)
	od := insert(other, ScopeDaily, 1, nil)

	// d1 is done and carries dates and an intention; d2 is archived.
	if _, err := tx.Exec(`UPDATE checklist_items
		SET done = true, start_date = '2026-05-02', due_date = '2026-05-04', intention = 'keep me'
		WHERE id = $1`, d1); err != nil {
		t.Fatalf("decorate d1: %v", err)
	}
	if _, err := tx.Exec(`UPDATE checklist_items SET archived = true WHERE id = $1`, d2); err != nil {
		t.Fatalf("archive d2: %v", err)
	}

	if _, err := tx.Exec(string(up)); err != nil {
		t.Fatalf("up migration: %v", err)
	}

	var daily int
	if err := tx.Get(&daily, `SELECT count(*) FROM checklist_items WHERE scope = 'daily'`); err != nil {
		t.Fatalf("count daily: %v", err)
	}
	if daily != 0 {
		t.Errorf("%d daily rows remain after the migration, want 0", daily)
	}

	rows := map[uuid.UUID]migratedRow{}
	var all []migratedRow
	if err := tx.Select(&all, `
		SELECT id, scope, done, archived, parent_id, start_date, due_date, intention
		FROM checklist_items WHERE plan_id IN ($1, $2)`, plan, other); err != nil {
		t.Fatalf("read rows: %v", err)
	}
	for _, r := range all {
		rows[r.ID] = r
		if r.Scope != ScopeLongterm {
			t.Errorf("row %s scope = %q, want longterm", r.ID, r.Scope)
		}
	}
	if len(rows) != 10 {
		t.Fatalf("got %d rows, want 10 (nothing deleted)", len(rows))
	}

	got := rows[d1]
	if !got.Done {
		t.Error("d1 lost done")
	}
	if got.StartDate == nil || got.DueDate == nil ||
		got.StartDate.Format("2006-01-02") != "2026-05-02" || got.DueDate.Format("2006-01-02") != "2026-05-04" {
		t.Errorf("d1 dates = %v..%v, want 2026-05-02..2026-05-04", got.StartDate, got.DueDate)
	}
	if got.Intention == nil || *got.Intention != "keep me" {
		t.Errorf("d1 intention = %v, want \"keep me\"", got.Intention)
	}
	if !rows[d2].Archived {
		t.Error("d2 lost archived")
	}
	if p := rows[dc2].ParentID; p == nil || *p != d1 {
		t.Errorf("dc2 parent = %v, want d1", p)
	}
	if p := rows[dc1].ParentID; p == nil || *p != l1 {
		t.Errorf("dc1 parent = %v, want l1", p)
	}

	// Stored order, read the way every list reads it (sequence, then the
	// tie-break). Archived rows are included: they keep their place too.
	order := func(planID uuid.UUID, parent *uuid.UUID) []uuid.UUID {
		t.Helper()
		var ids []uuid.UUID
		if err := tx.Select(&ids, `
			SELECT id FROM checklist_items
			WHERE plan_id = $1 AND parent_id IS NOT DISTINCT FROM $2`+orderBySequence,
			planID, parent); err != nil {
			t.Fatalf("read order: %v", err)
		}
		return ids
	}
	assertOrder(t, "plan top level", order(plan, nil), []uuid.UUID{l1, l2, d3, d1, d2})
	assertOrder(t, "l1 children", order(plan, &l1), []uuid.UUID{lc1, dc1})
	assertOrder(t, "d1 children", order(plan, &d1), []uuid.UUID{dc2})

	var odSeq int
	if err := tx.Get(&odSeq, `SELECT sequence FROM checklist_items WHERE id = $1`, od); err != nil {
		t.Fatalf("read other: %v", err)
	}
	if odSeq != 501 {
		t.Errorf("other plan's migrated row sequence = %d, want 501 (after its own long-term tail)", odSeq)
	}
}

// The down migration is a documented no-op: which rows were daily is not
// recoverable, so it must run cleanly and change nothing.
func TestIntegrationRetireDailyScope_DownIsANoOp(t *testing.T) {
	db := openTestDB(t)
	down, err := os.ReadFile(retireDailyDown)
	if err != nil {
		t.Fatalf("read down migration: %v", err)
	}
	tx, err := db.Beginx()
	if err != nil {
		t.Fatalf("begin: %v", err)
	}
	defer tx.Rollback() //nolint:errcheck // rollback is the point

	before := scopeCounts(t, tx)
	if _, err := tx.Exec(string(down)); err != nil {
		t.Fatalf("down migration: %v", err)
	}
	after := scopeCounts(t, tx)
	if before != after {
		t.Errorf("down changed scopes: before %+v, after %+v", before, after)
	}
}

type scopeCount struct {
	Longterm int `db:"longterm"`
	Daily    int `db:"daily"`
}

func scopeCounts(t *testing.T, tx *sqlx.Tx) scopeCount {
	t.Helper()
	var c scopeCount
	if err := tx.Get(&c, `
		SELECT count(*) FILTER (WHERE scope = 'longterm') AS longterm,
		       count(*) FILTER (WHERE scope = 'daily')    AS daily
		FROM checklist_items`); err != nil {
		t.Fatalf("count scopes: %v", err)
	}
	return c
}

func assertOrder(t *testing.T, what string, got, want []uuid.UUID) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("%s: got %d rows %v, want %d %v", what, len(got), got, len(want), want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("%s: position %d got %s, want %s\nfull order: %v", what, i, got[i], want[i], got)
		}
	}
}

package checklistitem

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/darkphotonKN/fireplace/services/plan-service/internal/notecomment"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// FS-65B3N R4, R5 against the real tables: a note with a comment is refused
// as a task and left as it is; one without converts; and a comment create and
// a convert on the same note never both succeed.

func requireNoteComments(t *testing.T, db *sqlx.DB) {
	t.Helper()
	var exists bool
	if err := db.Get(&exists, `SELECT to_regclass('public.note_comments') IS NOT NULL`); err != nil || !exists {
		t.Skip("note_comments not migrated in the plan database")
	}
}

func seedComment(t *testing.T, db *sqlx.DB, itemID uuid.UUID) {
	t.Helper()
	if _, err := db.Exec(`INSERT INTO note_comments (item_id, author_id, body) VALUES ($1, $2, 'a later thought')`,
		itemID, uuid.New()); err != nil {
		t.Fatalf("seed comment: %v", err)
	}
}

func itemType(t *testing.T, db *sqlx.DB, id uuid.UUID) string {
	t.Helper()
	var typ string
	if err := db.Get(&typ, `SELECT type FROM checklist_items WHERE id = $1`, id); err != nil {
		t.Fatalf("read type: %v", err)
	}
	return typ
}

func toTask(id, planID uuid.UUID) UpdateItemInput {
	task := TypeTask
	return UpdateItemInput{ID: id, PlanID: planID, Type: &task, SetSummary: true}
}

func TestIntegrationNoteComments_ToTask_WithAComment_IsRefusedAndUnchanged(t *testing.T) {
	db := openTestDB(t)
	requireNoteComments(t, db)
	planID := seedPlan(t, db)
	id := uuid.New()
	gist := "the gist"
	seedNote(t, db, planID, id, 1, &gist)
	seedComment(t, db, id)

	body := "rewritten as a task"
	in := toTask(id, planID)
	in.Description = &body
	err := NewRepository(db).Update(context.Background(), in)

	if !errors.Is(err, ErrNoteHasComments) {
		t.Fatalf("Update = %v, want ErrNoteHasComments", err)
	}
	got, err := NewRepository(db).GetInPlan(context.Background(), planID, id)
	if err != nil {
		t.Fatalf("GetInPlan: %v", err)
	}
	if got.Type != TypeNote || got.Summary == nil || *got.Summary != gist || got.Description == body {
		t.Errorf("item = %+v, want the note unchanged", got)
	}
}

func TestIntegrationNoteComments_ToTask_WithoutComments_Converts(t *testing.T) {
	db := openTestDB(t)
	requireNoteComments(t, db)
	planID := seedPlan(t, db)
	id := uuid.New()
	gist := "the gist"
	seedNote(t, db, planID, id, 1, &gist)

	if err := NewRepository(db).Update(context.Background(), toTask(id, planID)); err != nil {
		t.Fatalf("Update: %v", err)
	}
	got, err := NewRepository(db).GetInPlan(context.Background(), planID, id)
	if err != nil {
		t.Fatalf("GetInPlan: %v", err)
	}
	if got.Type != TypeTask || got.Summary != nil {
		t.Errorf("item = type %q summary %v, want a task with no summary", got.Type, got.Summary)
	}
}

// Only the convert is refused: a note with comments stays editable, and a
// task asked to be a task again is not a convert.
func TestIntegrationNoteComments_OtherUpdates_AreUnaffected(t *testing.T) {
	db := openTestDB(t)
	requireNoteComments(t, db)
	planID := seedPlan(t, db)
	repo := NewRepository(db)
	ctx := context.Background()

	noteID := uuid.New()
	seedNote(t, db, planID, noteID, 1, nil)
	seedComment(t, db, noteID)
	body, note := "a longer body", TypeNote
	if err := repo.Update(ctx, UpdateItemInput{ID: noteID, PlanID: planID, Description: &body, Type: &note}); err != nil {
		t.Errorf("edit a commented note: %v", err)
	}

	taskID := uuid.New()
	seedItem(t, db, planID, taskID, 2, time.Now())
	if err := repo.Update(ctx, toTask(taskID, planID)); err != nil {
		t.Errorf("task to task: %v", err)
	}
	if err := repo.Update(ctx, UpdateItemInput{ID: taskID, PlanID: planID, Type: &note}); err != nil {
		t.Errorf("task to note: %v", err)
	}
	if typ := itemType(t, db, taskID); typ != TypeNote {
		t.Errorf("task to note: type = %q, want note", typ)
	}
}

// R5, lock ordering: a comment create holds the item FOR SHARE until it
// commits. A convert started meanwhile waits on its FOR UPDATE, then sees the
// committed comment and is refused.
func TestIntegrationNoteComments_ConvertWaitsForAnOpenCommentCreate(t *testing.T) {
	db := openTestDB(t)
	requireNoteComments(t, db)
	planID := seedPlan(t, db)
	id := uuid.New()
	seedNote(t, db, planID, id, 1, nil)
	ctx := context.Background()

	create, err := db.BeginTxx(ctx, nil)
	if err != nil {
		t.Fatalf("begin: %v", err)
	}
	defer create.Rollback()
	if _, err := create.Exec(`SELECT type FROM checklist_items WHERE id = $1 FOR SHARE`, id); err != nil {
		t.Fatalf("hold item: %v", err)
	}
	if _, err := create.Exec(`INSERT INTO note_comments (item_id, author_id, body) VALUES ($1, $2, 'mid-flight')`,
		id, uuid.New()); err != nil {
		t.Fatalf("insert: %v", err)
	}

	done := make(chan error, 1)
	go func() { done <- NewRepository(db).Update(ctx, toTask(id, planID)) }()

	select {
	case err := <-done:
		t.Fatalf("convert returned %v while the comment create held the item; want it to wait", err)
	case <-time.After(300 * time.Millisecond):
	}
	if err := create.Commit(); err != nil {
		t.Fatalf("commit: %v", err)
	}
	select {
	case err := <-done:
		if !errors.Is(err, ErrNoteHasComments) {
			t.Fatalf("convert = %v, want ErrNoteHasComments", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("convert still waiting after the comment create committed")
	}
	if typ := itemType(t, db, id); typ != TypeNote {
		t.Errorf("type = %q, want note", typ)
	}
}

// R5, end to end with the real comment create: run both at once many times.
// Exactly one wins each round, and no task is ever left with a comment.
func TestIntegrationNoteComments_CreateAndConvert_NeverBothSucceed(t *testing.T) {
	db := openTestDB(t)
	requireNoteComments(t, db)
	planID := seedPlan(t, db)
	items := NewRepository(db)
	comments := notecomment.NewRepository(db)
	ctx := context.Background()

	for round := 0; round < 25; round++ {
		id := uuid.New()
		seedNote(t, db, planID, id, round+1, nil)

		var wg sync.WaitGroup
		var createErr, convertErr error
		start := make(chan struct{})
		wg.Add(2)
		go func() {
			defer wg.Done()
			<-start
			_, createErr = comments.Create(ctx, notecomment.CreateInput{
				PlanID: planID, ItemID: id, AuthorID: uuid.New(), Body: "racing",
			})
		}()
		go func() {
			defer wg.Done()
			<-start
			convertErr = items.Update(ctx, toTask(id, planID))
		}()
		close(start)
		wg.Wait()

		if (createErr == nil) == (convertErr == nil) {
			t.Fatalf("round %d: create = %v, convert = %v; want exactly one to succeed", round, createErr, convertErr)
		}
		if createErr != nil && !errors.Is(createErr, notecomment.ErrNotANote) {
			t.Fatalf("round %d: create = %v, want ErrNotANote", round, createErr)
		}
		if convertErr != nil && !errors.Is(convertErr, ErrNoteHasComments) {
			t.Fatalf("round %d: convert = %v, want ErrNoteHasComments", round, convertErr)
		}
		var orphaned int
		if err := db.Get(&orphaned, `
			SELECT COUNT(*) FROM note_comments c JOIN checklist_items i ON i.id = c.item_id
			WHERE i.id = $1 AND i.type = 'task'`, id); err != nil {
			t.Fatalf("round %d: count: %v", round, err)
		}
		if orphaned != 0 {
			t.Fatalf("round %d: a task carries %d comments", round, orphaned)
		}
	}
}

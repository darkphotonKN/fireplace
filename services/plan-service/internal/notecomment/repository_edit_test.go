package notecomment

import (
	"context"
	"errors"
	"testing"
	"time"

	commonconstants "github.com/darkphotonKN/fireplace/common/constants"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// FS-65B3N R2, R16, R17, R19 against the real tables: every comment write is
// scoped plan → item → comment, an edit is the author's alone, and a delete
// the author's or the plan owner's.

type editFixture struct {
	db                    *sqlx.DB
	repo                  *repository
	planID, owner, noteID uuid.UUID
	author                uuid.UUID
	comment               *Comment
}

func newEditFixture(t *testing.T) editFixture {
	t.Helper()
	db := openTestDB(t)
	planID := seedPlan(t, db)
	var owner uuid.UUID
	if err := db.Get(&owner, `SELECT user_id FROM plans WHERE id = $1`, planID); err != nil {
		t.Fatal(err)
	}
	noteID := seedItem(t, db, planID, "note")
	author := uuid.New()
	repo := NewRepository(db)
	c, err := repo.Create(context.Background(), CreateInput{PlanID: planID, ItemID: noteID, AuthorID: author, Body: "original"})
	if err != nil {
		t.Fatalf("seed comment: %v", err)
	}
	return editFixture{db: db, repo: repo, planID: planID, owner: owner, noteID: noteID, author: author, comment: c}
}

func (f editFixture) ref(caller uuid.UUID) CommentRef {
	return CommentRef{PlanID: f.planID, ItemID: f.noteID, CommentID: f.comment.ID, UserID: caller}
}

func TestIntegrationGet_ScopedPlanItemComment(t *testing.T) {
	f := newEditFixture(t)
	otherNote := seedItem(t, f.db, f.planID, "note")
	otherPlan := seedPlan(t, f.db)
	ctx := context.Background()

	got, err := f.repo.Get(ctx, f.ref(uuid.New()))
	if err != nil || got.ID != f.comment.ID || got.AuthorID != f.author {
		t.Fatalf("Get = %+v, %v", got, err)
	}
	for name, ref := range map[string]CommentRef{
		"other note": {PlanID: f.planID, ItemID: otherNote, CommentID: f.comment.ID},
		"other plan": {PlanID: otherPlan, ItemID: f.noteID, CommentID: f.comment.ID},
		"missing":    {PlanID: f.planID, ItemID: f.noteID, CommentID: uuid.New()},
	} {
		if _, err := f.repo.Get(ctx, ref); !errors.Is(err, commonconstants.ErrNotFound) {
			t.Errorf("%s: err = %v, want ErrNotFound", name, err)
		}
	}
}

// R16: the author's edit replaces the body and sets edited_at, and leaves the
// item alone (R7).
func TestIntegrationUpdate_ByTheAuthor(t *testing.T) {
	f := newEditFixture(t)

	got, err := f.repo.Update(context.Background(), f.ref(f.author), "revised")

	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if got.Body != "revised" || got.EditedAt == nil || !got.CreatedAt.Equal(f.comment.CreatedAt) {
		t.Fatalf("comment = %+v, want the new body with edited_at", got)
	}
	var updated time.Time
	if err := f.db.Get(&updated, `SELECT updated_at FROM checklist_items WHERE id = $1`, f.noteID); err != nil || !updated.Equal(updatedLongAgo) {
		t.Fatalf("item updated_at = %v (%v), want unchanged", updated, err)
	}
}

// R16: the same body is a no-op; edited_at stays null.
func TestIntegrationUpdate_SameBody_IsANoOp(t *testing.T) {
	f := newEditFixture(t)

	got, err := f.repo.Update(context.Background(), f.ref(f.author), "original")

	if err != nil || got.Body != "original" || got.EditedAt != nil {
		t.Fatalf("comment = %+v, err = %v; want it unchanged", got, err)
	}
}

// R19: anyone but the author, the owner included, is ErrForbidden; a
// comment out of scope is ErrNotFound. Nothing changes.
func TestIntegrationUpdate_Refusals(t *testing.T) {
	f := newEditFixture(t)
	ctx := context.Background()

	for name, caller := range map[string]uuid.UUID{"owner": f.owner, "someone": uuid.New()} {
		if _, err := f.repo.Update(ctx, f.ref(caller), "hijack"); !errors.Is(err, commonconstants.ErrForbidden) {
			t.Errorf("%s: err = %v, want ErrForbidden", name, err)
		}
	}
	missing := f.ref(f.author)
	missing.CommentID = uuid.New()
	if _, err := f.repo.Update(ctx, missing, "x"); !errors.Is(err, commonconstants.ErrNotFound) {
		t.Errorf("missing: err = %v, want ErrNotFound", err)
	}
	if got, _ := f.repo.Get(ctx, f.ref(f.author)); got.Body != "original" || got.EditedAt != nil {
		t.Fatalf("comment = %+v, want unchanged", got)
	}
}

// R17, R19: the author and the plan's owner may delete; anyone else is
// ErrForbidden; a deleted comment is then ErrNotFound.
func TestIntegrationDelete(t *testing.T) {
	ctx := context.Background()
	t.Run("someone else", func(t *testing.T) {
		f := newEditFixture(t)
		if err := f.repo.Delete(ctx, f.ref(uuid.New())); !errors.Is(err, commonconstants.ErrForbidden) {
			t.Fatalf("err = %v, want ErrForbidden", err)
		}
		if n := countComments(t, f.db, f.noteID); n != 1 {
			t.Fatalf("%d comments left, want 1", n)
		}
	})
	for name, pick := range map[string]func(f editFixture) uuid.UUID{
		"author": func(f editFixture) uuid.UUID { return f.author },
		"owner":  func(f editFixture) uuid.UUID { return f.owner },
	} {
		t.Run(name, func(t *testing.T) {
			f := newEditFixture(t)
			if err := f.repo.Delete(ctx, f.ref(pick(f))); err != nil {
				t.Fatalf("Delete: %v", err)
			}
			if n := countComments(t, f.db, f.noteID); n != 0 {
				t.Fatalf("%d comments left, want 0", n)
			}
			if err := f.repo.Delete(ctx, f.ref(pick(f))); !errors.Is(err, commonconstants.ErrNotFound) {
				t.Fatalf("second delete: err = %v, want ErrNotFound", err)
			}
		})
	}
}

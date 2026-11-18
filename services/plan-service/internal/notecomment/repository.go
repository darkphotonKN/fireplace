package notecomment

import (
	"context"
	"fmt"

	commonconstants "github.com/darkphotonKN/fireplace/common/constants"
	commonhelpers "github.com/darkphotonKN/fireplace/common/utils"
	"github.com/darkphotonKN/fireplace/services/plan-service/internal/itemtype"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type repository struct {
	db *sqlx.DB
}

func NewRepository(db *sqlx.DB) *repository {
	return &repository{db: db}
}

func wrapDBErr(op string, err error) error {
	return commonhelpers.WrapDBErr("notecomment repo", op, err)
}

// itemTypeQuery reads the item's type only when it is in the plan, so another
// plan's item is no row, the same as a missing one (R2).
const itemTypeQuery = `SELECT type FROM checklist_items WHERE id = $1 AND plan_id = $2`

// noteOnly answers ErrNotANote for any type but note (R3).
func noteOnly(itemID uuid.UUID, itemType string) error {
	if itemType != itemtype.Note {
		return fmt.Errorf("item %s is a %s: %w", itemID, itemType, ErrNotANote)
	}
	return nil
}

// Create checks the item and inserts the comment in one transaction. The item
// row is held FOR SHARE until commit, so a concurrent change of its type
// waits for this insert, or this insert waits for it and then sees the new
// type (R5).
func (r *repository) Create(ctx context.Context, in CreateInput) (*Comment, error) {
	var c Comment
	err := commonhelpers.ExecTx(ctx, r.db, func(tx *sqlx.Tx) error {
		var itemType string
		if err := tx.GetContext(ctx, &itemType, itemTypeQuery+` FOR SHARE`, in.ItemID, in.PlanID); err != nil {
			return wrapDBErr("lock item "+in.ItemID.String(), err)
		}
		if err := noteOnly(in.ItemID, itemType); err != nil {
			return err
		}
		if err := tx.GetContext(ctx, &c, `
			INSERT INTO note_comments (item_id, author_id, body)
			VALUES ($1, $2, $3)
			RETURNING id, item_id, author_id, body, created_at, edited_at`,
			in.ItemID, in.AuthorID, in.Body); err != nil {
			return wrapDBErr("insert comment", err)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &c, nil
}

// List answers the note's comments oldest first, created_at then id (R11).
func (r *repository) List(ctx context.Context, planID, itemID uuid.UUID) ([]*Comment, error) {
	var itemType string
	if err := r.db.GetContext(ctx, &itemType, itemTypeQuery, itemID, planID); err != nil {
		return nil, wrapDBErr("get item "+itemID.String(), err)
	}
	if err := noteOnly(itemID, itemType); err != nil {
		return nil, err
	}
	comments := []*Comment{}
	if err := r.db.SelectContext(ctx, &comments, `
		SELECT id, item_id, author_id, body, created_at, edited_at
		FROM note_comments WHERE item_id = $1
		ORDER BY created_at ASC, id ASC`, itemID); err != nil {
		return nil, wrapDBErr("list comments", err)
	}
	return comments, nil
}

// commentColumns are a comment's columns, qualified for the scoped joins.
const commentColumns = `c.id, c.item_id, c.author_id, c.body, c.created_at, c.edited_at`

// scopedComment reads ref's comment only when it is on ref's item and that
// item is in ref's plan (R2).
const scopedComment = `
	SELECT ` + commentColumns + `
	FROM note_comments c
	JOIN checklist_items i ON i.id = c.item_id
	WHERE c.id = $1 AND c.item_id = $2 AND i.plan_id = $3`

// Get answers ref's comment; anything out of scope is ErrNotFound.
func (r *repository) Get(ctx context.Context, ref CommentRef) (*Comment, error) {
	var c Comment
	if err := r.db.GetContext(ctx, &c, scopedComment, ref.CommentID, ref.ItemID, ref.PlanID); err != nil {
		return nil, wrapDBErr("get comment "+ref.CommentID.String(), err)
	}
	return &c, nil
}

// Update replaces the body and sets edited_at, as ref.UserID, who must be the
// author (R19 backstop: ErrForbidden otherwise). The same body as stored
// changes nothing (R16). The row is locked from the read to the write.
func (r *repository) Update(ctx context.Context, ref CommentRef, body string) (*Comment, error) {
	var c Comment
	err := commonhelpers.ExecTx(ctx, r.db, func(tx *sqlx.Tx) error {
		if err := tx.GetContext(ctx, &c, scopedComment+` FOR UPDATE OF c`, ref.CommentID, ref.ItemID, ref.PlanID); err != nil {
			return wrapDBErr("lock comment "+ref.CommentID.String(), err)
		}
		if c.AuthorID != ref.UserID {
			return fmt.Errorf("comment %s is not %s's: %w", ref.CommentID, ref.UserID, commonconstants.ErrForbidden)
		}
		if c.Body == body {
			return nil
		}
		if err := tx.GetContext(ctx, &c, `
			UPDATE note_comments SET body = $2, edited_at = NOW() WHERE id = $1
			RETURNING id, item_id, author_id, body, created_at, edited_at`, ref.CommentID, body); err != nil {
			return wrapDBErr("update comment "+ref.CommentID.String(), err)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &c, nil
}

// Delete removes ref's comment for good, as ref.UserID, who must be its
// author or the plan's owner (R17, R19 backstop: ErrForbidden otherwise).
func (r *repository) Delete(ctx context.Context, ref CommentRef) error {
	return commonhelpers.ExecTx(ctx, r.db, func(tx *sqlx.Tx) error {
		var row struct {
			AuthorID uuid.UUID `db:"author_id"`
			OwnerID  uuid.UUID `db:"owner_id"`
		}
		if err := tx.GetContext(ctx, &row, `
			SELECT c.author_id, p.user_id AS owner_id
			FROM note_comments c
			JOIN checklist_items i ON i.id = c.item_id
			JOIN plans p ON p.id = i.plan_id
			WHERE c.id = $1 AND c.item_id = $2 AND i.plan_id = $3
			FOR UPDATE OF c`, ref.CommentID, ref.ItemID, ref.PlanID); err != nil {
			return wrapDBErr("lock comment "+ref.CommentID.String(), err)
		}
		if ref.UserID != row.AuthorID && ref.UserID != row.OwnerID {
			return fmt.Errorf("comment %s: %s is neither its author nor the plan's owner: %w",
				ref.CommentID, ref.UserID, commonconstants.ErrForbidden)
		}
		if _, err := tx.ExecContext(ctx, `DELETE FROM note_comments WHERE id = $1`, ref.CommentID); err != nil {
			return wrapDBErr("delete comment "+ref.CommentID.String(), err)
		}
		return nil
	})
}

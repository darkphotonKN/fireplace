// Package notecomment is comments on a note (FS-65B3N): dated entries by plan
// members, stored beside the checklist items they belong to. It decides the
// body rules and that only notes carry comments; who may read, add, edit or
// delete is the gateway's decision (ADR-0015).
//
// It deliberately reads checklist_items and plans, which checklistitem and
// plan own, inside its own transactions rather than calling those packages:
// the item's type is checked under a lock on its row in the same transaction
// as the comment insert, so a comment and a convert to task never both
// succeed (FS-65B3N R5), and a delete reads the plan's owner under the
// comment's lock.
package notecomment

import (
	"time"

	"github.com/google/uuid"
)

// MaxBodyLen is the longest comment body, in characters (Unicode code
// points), after trimming (R9).
const MaxBodyLen = 2000

// Comment is one comment on a note (R1). EditedAt is nil until the first edit.
type Comment struct {
	ID        uuid.UUID  `db:"id"`
	ItemID    uuid.UUID  `db:"item_id"`
	AuthorID  uuid.UUID  `db:"author_id"`
	Body      string     `db:"body"`
	CreatedAt time.Time  `db:"created_at"`
	EditedAt  *time.Time `db:"edited_at"`
}

// CreateInput adds a comment to ItemID in PlanID. AuthorID is the caller
// (R6), never a value the client sent.
type CreateInput struct {
	PlanID   uuid.UUID
	ItemID   uuid.UUID
	AuthorID uuid.UUID
	Body     string
}

// CommentRef addresses one comment as (plan, item, comment), on behalf of
// UserID, the caller (R2). Any mismatch along the chain is not found.
type CommentRef struct {
	PlanID    uuid.UUID
	ItemID    uuid.UUID
	CommentID uuid.UUID
	UserID    uuid.UUID
}

-- Migration: comments on a note (FS-65B3N R1).
-- A comment belongs to one checklist item, a note; deleting the item deletes
-- its comments. author_id has no FK: users live in the gateway's database
-- (ADR-0010). edited_at stays NULL until the first edit (R16). Which items may
-- carry comments (notes only, R3) is plan-service's rule, checked under a lock
-- on the item row (R5), not a constraint here.

CREATE TABLE note_comments (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    item_id    UUID NOT NULL REFERENCES checklist_items(id) ON DELETE CASCADE,
    author_id  UUID NOT NULL,
    body       TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    edited_at  TIMESTAMPTZ NULL
);

-- List order is (created_at, id) per note (R11).
CREATE INDEX idx_note_comments_item_created ON note_comments (item_id, created_at, id);

-- Migration: add type and parent_id to checklist_items.
-- Nested Items + Notes — see PRD #40 / issue #41.

ALTER TABLE checklist_items
    ADD COLUMN type      TEXT NOT NULL DEFAULT 'task',
    ADD COLUMN parent_id UUID REFERENCES checklist_items(id) ON DELETE CASCADE;

ALTER TABLE checklist_items
    ADD CONSTRAINT chk_checklist_type CHECK (type IN ('task', 'note')),
    ADD CONSTRAINT chk_note_not_done  CHECK (type = 'task' OR done = false);

CREATE INDEX idx_checklist_plan_parent ON checklist_items (plan_id, parent_id);

-- Two-tier nesting trigger: a row's parent must itself be a top-level row,
-- and a row that already has children cannot itself become a child.
CREATE OR REPLACE FUNCTION enforce_checklist_two_tier()
RETURNS TRIGGER AS $$
DECLARE
    parent_parent UUID;
    has_children  BOOLEAN;
BEGIN
    IF NEW.parent_id IS NOT NULL THEN
        SELECT parent_id INTO parent_parent
        FROM checklist_items
        WHERE id = NEW.parent_id;

        IF parent_parent IS NOT NULL THEN
            RAISE EXCEPTION 'two-tier nesting max: cannot nest under a child item (id=%)', NEW.parent_id
                USING ERRCODE = '23514';
        END IF;
    END IF;

    -- A row with existing children cannot itself become a child.
    IF TG_OP = 'UPDATE' AND NEW.parent_id IS NOT NULL AND (OLD.parent_id IS NULL OR OLD.parent_id <> NEW.parent_id) THEN
        SELECT EXISTS(SELECT 1 FROM checklist_items WHERE parent_id = NEW.id) INTO has_children;
        IF has_children THEN
            RAISE EXCEPTION 'two-tier nesting max: row % has children and cannot become a child', NEW.id
                USING ERRCODE = '23514';
        END IF;
    END IF;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_checklist_two_tier
    BEFORE INSERT OR UPDATE ON checklist_items
    FOR EACH ROW
    EXECUTE FUNCTION enforce_checklist_two_tier();

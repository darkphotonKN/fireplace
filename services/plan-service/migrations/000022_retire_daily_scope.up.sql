-- Migration: retire the daily scope (FS-02JBZ D12, R50, S16).
--
-- Every daily row becomes a long-term row, archived or not. Nothing is
-- deleted, and done, dates, parent_id, intention and archived are untouched:
-- only scope and sequence change.
--
-- Placement: within each sibling set (same plan, same parent), migrated rows
-- go after the existing long-term rows, in their old relative order (sequence,
-- then the lists' created_at, id tie-break). The tail is the largest long-term
-- sequence at that level, archived rows included, so nothing migrated can sort
-- ahead of any long-term sibling.
--
-- The scope CHECK constraint still allows 'daily': the value survives in the
-- schema and the contract so an old client's request is well-formed and its
-- refusal (DAILY_SCOPE_RETIRED) is a domain decision.

WITH longterm_tail AS (
    SELECT plan_id, parent_id, MAX(sequence) AS max_sequence
    FROM checklist_items
    WHERE scope = 'longterm'
    GROUP BY plan_id, parent_id
),
placed AS (
    SELECT d.id,
           COALESCE(t.max_sequence, 0)
             + row_number() OVER (
                 PARTITION BY d.plan_id, d.parent_id
                 ORDER BY d.sequence ASC, d.created_at ASC, d.id ASC
               ) AS new_sequence
    FROM checklist_items d
    LEFT JOIN longterm_tail t
      ON t.plan_id IS NOT DISTINCT FROM d.plan_id
     AND t.parent_id IS NOT DISTINCT FROM d.parent_id
    WHERE d.scope = 'daily'
)
UPDATE checklist_items c
SET scope = 'longterm',
    sequence = p.new_sequence
FROM placed p
WHERE c.id = p.id;

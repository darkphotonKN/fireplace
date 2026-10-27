-- A redraw replaces the plan only if the key still serves the draw it started
-- from (R21). A tick or untick changes what that draw shows without changing
-- the draw, so a redraw that read the plan before a tick could store a draw
-- that drops it. tick_version counts the ticks and unticks on the key: each
-- one moves it in its own transaction, and a redraw's save is conditional on
-- (draw_id, tick_version), so a redraw in flight across a tick loses, gives
-- its spend back and serves the stored plan, which shows the tick.
ALTER TABLE day_plans ADD COLUMN tick_version BIGINT NOT NULL DEFAULT 0;

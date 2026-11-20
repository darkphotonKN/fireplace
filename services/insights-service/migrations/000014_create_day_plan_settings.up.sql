-- The day plan setting (FS-02JBZ §Change: Hours a day, R93-R95): one user's
-- daily time on one plan, in whole minutes, 30 to 720 in steps of 30. Per
-- user: on a shared plan (FS-12ZMF) every member has their own. Plans are
-- referenced by id only: plan-service owns them, so there is no foreign key.
--
-- A row's existence is "asked": the user has saved a setting for the plan, and
-- is never asked again on any device (R94). daily_minutes NULL is "asked, no
-- budget": the day plan is drawn as it was before this change (R96). The CHECK
-- makes an off-step or out-of-range value unrepresentable, whoever writes it.
CREATE TABLE day_plan_settings (
    plan_id       UUID NOT NULL,
    user_id       UUID NOT NULL,
    daily_minutes INTEGER CHECK (
        daily_minutes IS NULL
        OR (daily_minutes BETWEEN 30 AND 720 AND daily_minutes % 30 = 0)
    ),
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    PRIMARY KEY (plan_id, user_id)
);

-- update_updated_at_column() is defined by migration 000001.
CREATE TRIGGER update_day_plan_settings_updated_at
    BEFORE UPDATE ON day_plan_settings
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

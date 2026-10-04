-- Накопления без денежной цели и срока
ALTER TABLE savings_goals
  ALTER COLUMN target DROP NOT NULL,
  ALTER COLUMN deadline DROP NOT NULL,
  DROP CONSTRAINT savings_goals_target_check,
  ADD CONSTRAINT savings_goals_target_check CHECK (target IS NULL OR target > 0);

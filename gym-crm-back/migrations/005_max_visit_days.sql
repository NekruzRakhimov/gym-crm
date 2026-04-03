ALTER TABLE tariffs ADD COLUMN IF NOT EXISTS max_visit_days INTEGER;
ALTER TABLE tariffs DROP COLUMN IF EXISTS max_visits_per_day;

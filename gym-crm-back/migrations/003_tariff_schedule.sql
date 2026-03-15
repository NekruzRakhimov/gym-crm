ALTER TABLE tariffs
  ADD COLUMN IF NOT EXISTS schedule_days VARCHAR(20) NOT NULL DEFAULT 'all',
  ADD COLUMN IF NOT EXISTS time_from VARCHAR(5),  -- "HH:MM" or NULL = no restriction
  ADD COLUMN IF NOT EXISTS time_to   VARCHAR(5);  -- "HH:MM" or NULL = no restriction

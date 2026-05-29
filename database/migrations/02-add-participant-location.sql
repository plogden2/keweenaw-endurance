-- Add home city/state location for participants (existing databases).
ALTER TABLE participants ADD COLUMN IF NOT EXISTS location VARCHAR(500);

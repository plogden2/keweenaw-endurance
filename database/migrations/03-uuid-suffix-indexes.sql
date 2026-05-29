-- Indexes for resolving entities by the last six characters of their UUID string.
CREATE INDEX IF NOT EXISTS idx_events_id_suffix ON events ((RIGHT(id::text, 6)));
CREATE INDEX IF NOT EXISTS idx_races_id_suffix ON races ((RIGHT(id::text, 6)));
CREATE INDEX IF NOT EXISTS idx_races_event_id_suffix ON races ((RIGHT(event_id::text, 6)));
CREATE INDEX IF NOT EXISTS idx_participants_id_suffix ON participants ((RIGHT(id::text, 6)));
CREATE INDEX IF NOT EXISTS idx_participants_race_id_suffix ON participants ((RIGHT(race_id::text, 6)));
CREATE INDEX IF NOT EXISTS idx_timing_checkpoints_id_suffix ON timing_checkpoints ((RIGHT(id::text, 6)));
CREATE INDEX IF NOT EXISTS idx_timing_checkpoints_race_id_suffix ON timing_checkpoints ((RIGHT(race_id::text, 6)));
CREATE INDEX IF NOT EXISTS idx_timing_records_id_suffix ON timing_records ((RIGHT(id::text, 6)));
CREATE INDEX IF NOT EXISTS idx_timing_records_participant_id_suffix ON timing_records ((RIGHT(participant_id::text, 6)));
CREATE INDEX IF NOT EXISTS idx_timing_records_checkpoint_id_suffix ON timing_records ((RIGHT(checkpoint_id::text, 6)));
CREATE INDEX IF NOT EXISTS idx_categories_id_suffix ON categories ((RIGHT(id::text, 6)));
CREATE INDEX IF NOT EXISTS idx_categories_race_id_suffix ON categories ((RIGHT(race_id::text, 6)));

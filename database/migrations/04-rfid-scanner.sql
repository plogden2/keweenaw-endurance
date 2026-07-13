-- RFID race scanner: tag associations, reader stations, timing record types, participant category.

-- Participant category assignment (nullable in DB; required in app layer for this feature)
ALTER TABLE participants
    ADD COLUMN IF NOT EXISTS category_id UUID REFERENCES categories(id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS idx_participants_category_id ON participants(category_id);

-- RFID tag associations (multi-tag per participant; tag_uid unique globally)
CREATE TABLE IF NOT EXISTS rfid_tag_associations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    participant_id UUID NOT NULL REFERENCES participants(id) ON DELETE CASCADE,
    tag_uid VARCHAR(100) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    active BOOLEAN NOT NULL DEFAULT true,
    CONSTRAINT uq_rfid_tag_associations_tag_uid UNIQUE (tag_uid)
);

CREATE INDEX IF NOT EXISTS idx_rfid_tag_associations_tag_uid ON rfid_tag_associations(tag_uid);
CREATE INDEX IF NOT EXISTS idx_rfid_tag_associations_participant_id ON rfid_tag_associations(participant_id);

-- Reader stations (event-scoped finish or checkpoint mode)
CREATE TABLE IF NOT EXISTS reader_stations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event_id UUID NOT NULL REFERENCES events(id) ON DELETE CASCADE,
    mode VARCHAR(50) NOT NULL DEFAULT 'finish'
        CHECK (mode IN ('finish', 'checkpoint')),
    checkpoint_id UUID REFERENCES timing_checkpoints(id) ON DELETE SET NULL,
    sequence_order INTEGER NOT NULL DEFAULT 0,
    name VARCHAR(255) NOT NULL,
    device_id VARCHAR(100),
    last_seen_at TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_reader_stations_event_id ON reader_stations(event_id);

-- Timing record extensions for RFID laps, karaoke bonus, and checkpoint passes
ALTER TABLE timing_records
    ADD COLUMN IF NOT EXISTS record_type VARCHAR(50) NOT NULL DEFAULT 'rfid_lap'
        CHECK (record_type IN ('rfid_lap', 'karaoke_bonus', 'checkpoint_pass'));

ALTER TABLE timing_records
    ADD COLUMN IF NOT EXISTS source_lap_id UUID REFERENCES timing_records(id) ON DELETE SET NULL;

ALTER TABLE timing_records
    ADD COLUMN IF NOT EXISTS station_id UUID REFERENCES reader_stations(id) ON DELETE SET NULL;

-- At most one karaoke_bonus per source lap
CREATE UNIQUE INDEX IF NOT EXISTS idx_timing_records_one_karaoke_per_source_lap
    ON timing_records (source_lap_id)
    WHERE record_type = 'karaoke_bonus' AND source_lap_id IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_timing_records_station_id ON timing_records(station_id);
CREATE INDEX IF NOT EXISTS idx_timing_records_record_type ON timing_records(record_type);

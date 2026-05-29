-- Database and user are created by POSTGRES_DB / POSTGRES_USER in docker-compose.
-- Enable UUID extension
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Create the events table
CREATE TABLE events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    description TEXT,
    event_date DATE NOT NULL,
    location VARCHAR(500),
    website_url VARCHAR(500),
    logo_url VARCHAR(500),
    status VARCHAR(50) CHECK (status IN ('upcoming', 'active', 'completed', 'cancelled')),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Create the races table
CREATE TABLE races (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event_id UUID REFERENCES events(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    race_type VARCHAR(50) CHECK (race_type IN ('time_based', 'lap_based')),
    distance_km DECIMAL(10,2),
    duration_minutes INTEGER,
    start_time TIMESTAMP,
    status VARCHAR(50) CHECK (status IN ('scheduled', 'active', 'finished', 'cancelled')),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Create the participants table
CREATE TABLE participants (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    race_id UUID REFERENCES races(id) ON DELETE CASCADE,
    bib_number VARCHAR(20) NOT NULL,
    first_name VARCHAR(100) NOT NULL,
    last_name VARCHAR(100) NOT NULL,
    gender VARCHAR(10) CHECK (gender IS NULL OR gender IN ('male', 'female', 'other')),
    age INTEGER,
    location VARCHAR(500),
    rfid_tag_uid VARCHAR(100),
    status VARCHAR(50) CHECK (status IN ('registered', 'started', 'finished', 'dnf', 'dns')),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Create the timing_checkpoints table
CREATE TABLE timing_checkpoints (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    race_id UUID REFERENCES races(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    checkpoint_type VARCHAR(50) CHECK (checkpoint_type IN ('start', 'finish', 'intermediate')),
    distance_from_start_km DECIMAL(10,2),
    location_description VARCHAR(500),
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Create the timing_records table
CREATE TABLE timing_records (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    participant_id UUID REFERENCES participants(id) ON DELETE CASCADE,
    checkpoint_id UUID REFERENCES timing_checkpoints(id) ON DELETE CASCADE,
    timestamp TIMESTAMP NOT NULL,
    local_timestamp TIMESTAMP NOT NULL,
    device_id VARCHAR(100),
    sync_status VARCHAR(50) DEFAULT 'synced' CHECK (sync_status IN ('synced', 'pending_sync', 'failed_sync')),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Create the categories table
CREATE TABLE categories (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    race_id UUID REFERENCES races(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    category_type VARCHAR(50) CHECK (category_type IN ('overall', 'male', 'female', 'age_group', 'custom')),
    age_min INTEGER,
    age_max INTEGER,
    gender_filter VARCHAR(10),
    display_order INTEGER DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Add indexes for performance
CREATE INDEX idx_events_status ON events(status);
CREATE INDEX idx_races_event_id ON races(event_id);
CREATE INDEX idx_participants_race_id ON participants(race_id);
CREATE UNIQUE INDEX idx_participants_race_bib ON participants(race_id, bib_number);
CREATE UNIQUE INDEX idx_participants_rfid_tag_uid ON participants(rfid_tag_uid)
    WHERE rfid_tag_uid IS NOT NULL AND rfid_tag_uid <> '';
CREATE INDEX idx_timing_checkpoints_race_id ON timing_checkpoints(race_id);
CREATE INDEX idx_timing_records_participant_id ON timing_records(participant_id);
CREATE INDEX idx_timing_records_checkpoint_id ON timing_records(checkpoint_id);
CREATE INDEX idx_categories_race_id ON categories(race_id);
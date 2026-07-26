-- Race teams: optional multi-racer units scored by average member laps.
-- Applied via GORM AutoMigrate in production; this file documents the schema
-- for seed scripts and manual Postgres environments.

CREATE TABLE IF NOT EXISTS teams (
    id UUID PRIMARY KEY,
    race_id UUID NOT NULL REFERENCES races(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    display_order INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    UNIQUE (race_id, name)
);

CREATE INDEX IF NOT EXISTS idx_teams_race_id ON teams(race_id);

ALTER TABLE participants
    ADD COLUMN IF NOT EXISTS team_id UUID REFERENCES teams(id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS idx_participants_team_id ON participants(team_id);

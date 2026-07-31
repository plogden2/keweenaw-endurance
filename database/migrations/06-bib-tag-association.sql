-- Bib-associated RFID tags: event-scoped bibs; associations point at bib_id.
-- Applied via GORM AutoMigrate + migrateTagAssociationsToBibs in production;
-- this file documents the target schema for seed scripts and manual Postgres.

CREATE TABLE IF NOT EXISTS bibs (
    id UUID PRIMARY KEY,
    event_id UUID NOT NULL REFERENCES events(id) ON DELETE CASCADE,
    bib_number VARCHAR(20) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    CONSTRAINT idx_bibs_event_number UNIQUE (event_id, bib_number)
);

CREATE INDEX IF NOT EXISTS idx_bibs_event_id ON bibs(event_id);

-- Target shape for rfid_tag_associations (replaces participant_id with bib_id).
-- Fresh installs: create with bib_id. Upgrades: AutoMigrate adds bib_id, backfill
-- runs, then participant_id is dropped.
CREATE TABLE IF NOT EXISTS rfid_tag_associations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    bib_id UUID NOT NULL REFERENCES bibs(id) ON DELETE CASCADE,
    tag_uid VARCHAR(100) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    active BOOLEAN NOT NULL DEFAULT true,
    CONSTRAINT uq_rfid_tag_associations_tag_uid UNIQUE (tag_uid)
);

CREATE INDEX IF NOT EXISTS idx_rfid_tag_associations_tag_uid ON rfid_tag_associations(tag_uid);
CREATE INDEX IF NOT EXISTS idx_rfid_tag_associations_bib_id ON rfid_tag_associations(bib_id);

-- Upgrade path (when participant_id still exists):
-- 1. CREATE TABLE bibs ... (above) — AutoMigrate Bib first; do NOT AutoMigrate
--    RFIDTagAssociation with NOT NULL bib_id until after backfill.
-- 2. ALTER TABLE rfid_tag_associations ADD COLUMN IF NOT EXISTS bib_id UUID REFERENCES bibs(id);
--    (nullable — existing association rows must survive)
-- 3. Backfill: for each association join participant → race → event, ensure Bib,
--    then UPDATE rfid_tag_associations SET bib_id = ? WHERE id = ?
--    Skip empty bib_number with log; refuse to drop participant_id if any bib_id is still NULL.
-- 4. Ensure Bibs for all participants with bib_number (even without tags)
-- 5. ALTER TABLE rfid_tag_associations DROP COLUMN IF EXISTS participant_id;
-- 6. ALTER TABLE rfid_tag_associations ALTER COLUMN bib_id SET NOT NULL;
-- 7. AutoMigrate final RFIDTagAssociation shape (indexes / FK).
-- Steps 2–6 run in a transaction on Postgres.

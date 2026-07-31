-- One-shot Postgres upgrade for existing volumes that still have
-- rfid_tag_associations.participant_id (pre feature/bib-tag-association).
-- Fresh installs: prefer GORM AutoMigrate via backend startup.
-- Safe to re-run after a successful upgrade (no-op when participant_id is gone).

BEGIN;

CREATE TABLE IF NOT EXISTS bibs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event_id UUID NOT NULL REFERENCES events(id) ON DELETE CASCADE,
    bib_number VARCHAR(20) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    CONSTRAINT idx_bibs_event_number UNIQUE (event_id, bib_number)
);

CREATE INDEX IF NOT EXISTS idx_bibs_event_id ON bibs(event_id);

-- Ensure bibs for every participant with a bib number (event-scoped).
INSERT INTO bibs (id, event_id, bib_number)
SELECT gen_random_uuid(), src.event_id, src.bib_number
FROM (
    SELECT DISTINCT r.event_id, TRIM(p.bib_number) AS bib_number
    FROM participants p
    JOIN races r ON r.id = p.race_id
    WHERE p.bib_number IS NOT NULL AND TRIM(p.bib_number) <> ''
) AS src
WHERE NOT EXISTS (
    SELECT 1 FROM bibs b
    WHERE b.event_id = src.event_id AND b.bib_number = src.bib_number
);

DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_schema = 'public'
          AND table_name = 'rfid_tag_associations'
          AND column_name = 'participant_id'
    ) THEN
        ALTER TABLE rfid_tag_associations
            ADD COLUMN IF NOT EXISTS bib_id UUID REFERENCES bibs(id);

        -- Drop unmigratable rows (empty bib) so seed/dev volumes can advance.
        DELETE FROM rfid_tag_associations a
        USING participants p
        WHERE a.participant_id = p.id
          AND (p.bib_number IS NULL OR TRIM(p.bib_number) = '');

        UPDATE rfid_tag_associations a
        SET bib_id = b.id
        FROM participants p
        JOIN races r ON r.id = p.race_id
        JOIN bibs b ON b.event_id = r.event_id AND b.bib_number = TRIM(p.bib_number)
        WHERE a.participant_id = p.id
          AND a.bib_id IS NULL
          AND p.bib_number IS NOT NULL
          AND TRIM(p.bib_number) <> '';

        IF EXISTS (SELECT 1 FROM rfid_tag_associations WHERE bib_id IS NULL) THEN
            RAISE EXCEPTION
                'cannot finish bib upgrade: % association(s) still have null bib_id',
                (SELECT COUNT(*) FROM rfid_tag_associations WHERE bib_id IS NULL);
        END IF;

        ALTER TABLE rfid_tag_associations DROP COLUMN participant_id;
        ALTER TABLE rfid_tag_associations ALTER COLUMN bib_id SET NOT NULL;
    END IF;
END $$;

CREATE UNIQUE INDEX IF NOT EXISTS idx_rfid_tag_associations_tag_uid
    ON rfid_tag_associations(tag_uid);
CREATE INDEX IF NOT EXISTS idx_rfid_tag_associations_bib_id
    ON rfid_tag_associations(bib_id);

COMMIT;

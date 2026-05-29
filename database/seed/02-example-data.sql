-- Example race timing data for local development and demos.
-- Idempotent: removes prior seed rows (fixed UUIDs) before inserting.

BEGIN;

-- Seed entity IDs (fixed for repeatable runs)
-- Event:  11111111-1111-4111-8111-111111111101
-- Races:  ...102 (half marathon), ...103 (lap endurance)
-- Checkpoints, participants, categories, and records use ...201+ ranges

DELETE FROM timing_records WHERE participant_id IN (
    SELECT id FROM participants WHERE race_id IN (
        '11111111-1111-4111-8111-111111111102',
        '11111111-1111-4111-8111-111111111103'
    )
);
DELETE FROM categories WHERE race_id IN (
    '11111111-1111-4111-8111-111111111102',
    '11111111-1111-4111-8111-111111111103'
);
DELETE FROM timing_checkpoints WHERE race_id IN (
    '11111111-1111-4111-8111-111111111102',
    '11111111-1111-4111-8111-111111111103'
);
DELETE FROM participants WHERE race_id IN (
    '11111111-1111-4111-8111-111111111102',
    '11111111-1111-4111-8111-111111111103'
);
DELETE FROM races WHERE id IN (
    '11111111-1111-4111-8111-111111111102',
    '11111111-1111-4111-8111-111111111103'
);
DELETE FROM events WHERE id = '11111111-1111-4111-8111-111111111101';

INSERT INTO events (id, name, description, event_date, location, website_url, status)
VALUES (
    '11111111-1111-4111-8111-111111111101',
    'Keweenaw Trail Fest 2026',
    'Annual trail running festival across the Keweenaw Peninsula.',
    '2026-06-14',
    'Copper Harbor, MI',
    'https://keweenawtrailfest.example',
    'active'
);

INSERT INTO races (id, event_id, name, race_type, distance_km, duration_minutes, start_time, status)
VALUES
    (
        '11111111-1111-4111-8111-111111111102',
        '11111111-1111-4111-8111-111111111101',
        'Copper Harbor Half Marathon',
        'time_based',
        21.10,
        NULL,
        '2026-06-14 08:00:00',
        'finished'
    ),
    (
        '11111111-1111-4111-8111-111111111103',
        '11111111-1111-4111-8111-111111111101',
        'Calumet 6-Hour Endurance',
        'lap_based',
        NULL,
        360,
        '2026-06-14 10:00:00',
        'active'
    );

INSERT INTO timing_checkpoints (id, race_id, name, checkpoint_type, distance_from_start_km, location_description, is_active)
VALUES
    ('11111111-1111-4111-8111-111111111201', '11111111-1111-4111-8111-111111111102', 'Start Line', 'start', 0.00, 'Copper Harbor marina', true),
    ('11111111-1111-4111-8111-111111111202', '11111111-1111-4111-8111-111111111102', 'Aid Station 10K', 'intermediate', 10.00, 'Estivant Pines trail junction', true),
    ('11111111-1111-4111-8111-111111111203', '11111111-1111-4111-8111-111111111102', 'Finish Line', 'finish', 21.10, 'Copper Harbor marina', true),
    ('11111111-1111-4111-8111-111111111204', '11111111-1111-4111-8111-111111111103', 'Loop Start/Finish', 'finish', 5.00, 'Calumet historic district loop', true);

INSERT INTO categories (id, race_id, name, category_type, age_min, age_max, gender_filter, display_order)
VALUES
    ('11111111-1111-4111-8111-111111111301', '11111111-1111-4111-8111-111111111102', 'Overall', 'overall', NULL, NULL, NULL, 0),
    ('11111111-1111-4111-8111-111111111302', '11111111-1111-4111-8111-111111111102', 'Men', 'male', NULL, NULL, 'male', 1),
    ('11111111-1111-4111-8111-111111111303', '11111111-1111-4111-8111-111111111102', 'Women', 'female', NULL, NULL, 'female', 2),
    ('11111111-1111-4111-8111-111111111304', '11111111-1111-4111-8111-111111111102', 'Masters 40+', 'age_group', 40, 99, NULL, 3),
    ('11111111-1111-4111-8111-111111111305', '11111111-1111-4111-8111-111111111103', 'Overall', 'overall', NULL, NULL, NULL, 0);

INSERT INTO participants (id, race_id, bib_number, first_name, last_name, gender, age, rfid_tag_uid, status)
VALUES
    ('11111111-1111-4111-8111-111111111401', '11111111-1111-4111-8111-111111111102', '101', 'Mike', 'Keweenaw', 'male', 38, 'E200001101', 'finished'),
    ('11111111-1111-4111-8111-111111111402', '11111111-1111-4111-8111-111111111102', '102', 'Sarah', 'Jenkins', 'female', 34, 'E200001102', 'finished'),
    ('11111111-1111-4111-8111-111111111403', '11111111-1111-4111-8111-111111111102', '103', 'James', 'Houghton', 'male', 45, 'E200001103', 'finished'),
    ('11111111-1111-4111-8111-111111111404', '11111111-1111-4111-8111-111111111102', '104', 'Emma', 'Copper', 'female', 29, 'E200001104', 'finished'),
    ('11111111-1111-4111-8111-111111111405', '11111111-1111-4111-8111-111111111102', '105', 'Tom', 'Marquette', 'male', 52, 'E200001105', 'finished'),
    ('11111111-1111-4111-8111-111111111406', '11111111-1111-4111-8111-111111111102', '106', 'Lisa', 'Ontonagon', 'female', 41, 'E200001106', 'finished'),
    ('11111111-1111-4111-8111-111111111407', '11111111-1111-4111-8111-111111111102', '107', 'Chris', 'Keweenaw', 'male', 31, 'E200001107', 'dnf'),
    ('11111111-1111-4111-8111-111111111408', '11111111-1111-4111-8111-111111111102', '108', 'Anna', 'Harbor', 'female', 36, 'E200001108', 'dns'),
    ('11111111-1111-4111-8111-111111111409', '11111111-1111-4111-8111-111111111103', '201', 'Pat', 'Runner', 'male', 42, 'E200002201', 'started'),
    ('11111111-1111-4111-8111-111111111410', '11111111-1111-4111-8111-111111111103', '202', 'Dana', 'Endure', 'female', 37, 'E200002202', 'started'),
    ('11111111-1111-4111-8111-111111111411', '11111111-1111-4111-8111-111111111103', '203', 'Alex', 'Loop', 'male', 28, 'E200002203', 'started'),
    ('11111111-1111-4111-8111-111111111412', '11111111-1111-4111-8111-111111111103', '204', 'Sam', 'Ultra', 'female', 33, 'E200002204', 'started');

-- Half marathon timing records (start + optional intermediate + finish)
INSERT INTO timing_records (id, participant_id, checkpoint_id, timestamp, local_timestamp, device_id, sync_status)
VALUES
    -- Bib 101 Mike Keweenaw — 1:38:42
    ('11111111-1111-4111-8111-111111111501', '11111111-1111-4111-8111-111111111401', '11111111-1111-4111-8111-111111111201', '2026-06-14 08:00:00', '2026-06-14 08:00:00', 'GATE-START', 'synced'),
    ('11111111-1111-4111-8111-111111111502', '11111111-1111-4111-8111-111111111401', '11111111-1111-4111-8111-111111111202', '2026-06-14 08:47:10', '2026-06-14 08:47:10', 'GATE-10K', 'synced'),
    ('11111111-1111-4111-8111-111111111503', '11111111-1111-4111-8111-111111111401', '11111111-1111-4111-8111-111111111203', '2026-06-14 09:38:42', '2026-06-14 09:38:42', 'GATE-FINISH', 'synced'),
    -- Bib 102 Sarah Jenkins — 1:42:15
    ('11111111-1111-4111-8111-111111111504', '11111111-1111-4111-8111-111111111402', '11111111-1111-4111-8111-111111111201', '2026-06-14 08:00:15', '2026-06-14 08:00:15', 'GATE-START', 'synced'),
    ('11111111-1111-4111-8111-111111111505', '11111111-1111-4111-8111-111111111402', '11111111-1111-4111-8111-111111111203', '2026-06-14 09:42:30', '2026-06-14 09:42:30', 'GATE-FINISH', 'synced'),
    -- Bib 103 James Houghton — 1:44:30
    ('11111111-1111-4111-8111-111111111506', '11111111-1111-4111-8111-111111111403', '11111111-1111-4111-8111-111111111201', '2026-06-14 08:00:30', '2026-06-14 08:00:30', 'GATE-START', 'synced'),
    ('11111111-1111-4111-8111-111111111507', '11111111-1111-4111-8111-111111111403', '11111111-1111-4111-8111-111111111203', '2026-06-14 09:45:00', '2026-06-14 09:45:00', 'GATE-FINISH', 'synced'),
    -- Bib 104 Emma Copper — 1:55:03
    ('11111111-1111-4111-8111-111111111508', '11111111-1111-4111-8111-111111111404', '11111111-1111-4111-8111-111111111201', '2026-06-14 08:01:00', '2026-06-14 08:01:00', 'GATE-START', 'synced'),
    ('11111111-1111-4111-8111-111111111509', '11111111-1111-4111-8111-111111111404', '11111111-1111-4111-8111-111111111203', '2026-06-14 09:56:03', '2026-06-14 09:56:03', 'GATE-FINISH', 'synced'),
    -- Bib 105 Tom Marquette — 2:01:18
    ('11111111-1111-4111-8111-111111111510', '11111111-1111-4111-8111-111111111405', '11111111-1111-4111-8111-111111111201', '2026-06-14 08:02:00', '2026-06-14 08:02:00', 'GATE-START', 'synced'),
    ('11111111-1111-4111-8111-111111111511', '11111111-1111-4111-8111-111111111405', '11111111-1111-4111-8111-111111111203', '2026-06-14 10:03:18', '2026-06-14 10:03:18', 'GATE-FINISH', 'synced'),
    -- Bib 106 Lisa Ontonagon — 2:12:00
    ('11111111-1111-4111-8111-111111111512', '11111111-1111-4111-8111-111111111406', '11111111-1111-4111-8111-111111111201', '2026-06-14 08:03:00', '2026-06-14 08:03:00', 'GATE-START', 'synced'),
    ('11111111-1111-4111-8111-111111111513', '11111111-1111-4111-8111-111111111406', '11111111-1111-4111-8111-111111111203', '2026-06-14 10:15:00', '2026-06-14 10:15:00', 'GATE-FINISH', 'synced'),
    -- Bib 107 Chris Keweenaw — DNF (start only)
    ('11111111-1111-4111-8111-111111111514', '11111111-1111-4111-8111-111111111407', '11111111-1111-4111-8111-111111111201', '2026-06-14 08:05:00', '2026-06-14 08:05:00', 'GATE-START', 'synced');

-- Lap endurance timing records (each loop crossing at finish checkpoint)
INSERT INTO timing_records (id, participant_id, checkpoint_id, timestamp, local_timestamp, device_id, sync_status)
VALUES
    -- Bib 201 Pat Runner — 4 laps
    ('11111111-1111-4111-8111-111111111601', '11111111-1111-4111-8111-111111111409', '11111111-1111-4111-8111-111111111204', '2026-06-14 10:52:00', '2026-06-14 10:52:00', 'LOOP-1', 'synced'),
    ('11111111-1111-4111-8111-111111111602', '11111111-1111-4111-8111-111111111409', '11111111-1111-4111-8111-111111111204', '2026-06-14 11:48:00', '2026-06-14 11:48:00', 'LOOP-1', 'synced'),
    ('11111111-1111-4111-8111-111111111603', '11111111-1111-4111-8111-111111111409', '11111111-1111-4111-8111-111111111204', '2026-06-14 12:44:00', '2026-06-14 12:44:00', 'LOOP-1', 'synced'),
    ('11111111-1111-4111-8111-111111111604', '11111111-1111-4111-8111-111111111409', '11111111-1111-4111-8111-111111111204', '2026-06-14 13:39:00', '2026-06-14 13:39:00', 'LOOP-1', 'synced'),
    -- Bib 202 Dana Endure — 3 laps
    ('11111111-1111-4111-8111-111111111605', '11111111-1111-4111-8111-111111111410', '11111111-1111-4111-8111-111111111204', '2026-06-14 10:58:00', '2026-06-14 10:58:00', 'LOOP-1', 'synced'),
    ('11111111-1111-4111-8111-111111111606', '11111111-1111-4111-8111-111111111410', '11111111-1111-4111-8111-111111111204', '2026-06-14 11:57:00', '2026-06-14 11:57:00', 'LOOP-1', 'synced'),
    ('11111111-1111-4111-8111-111111111607', '11111111-1111-4111-8111-111111111410', '11111111-1111-4111-8111-111111111204', '2026-06-14 12:55:00', '2026-06-14 12:55:00', 'LOOP-1', 'synced'),
    -- Bib 203 Alex Loop — 3 laps (slower)
    ('11111111-1111-4111-8111-111111111608', '11111111-1111-4111-8111-111111111411', '11111111-1111-4111-8111-111111111204', '2026-06-14 11:05:00', '2026-06-14 11:05:00', 'LOOP-1', 'synced'),
    ('11111111-1111-4111-8111-111111111609', '11111111-1111-4111-8111-111111111411', '11111111-1111-4111-8111-111111111204', '2026-06-14 12:08:00', '2026-06-14 12:08:00', 'LOOP-1', 'synced'),
    ('11111111-1111-4111-8111-111111111610', '11111111-1111-4111-8111-111111111411', '11111111-1111-4111-8111-111111111204', '2026-06-14 13:12:00', '2026-06-14 13:12:00', 'LOOP-1', 'synced'),
    -- Bib 204 Sam Ultra — 2 laps (still on course)
    ('11111111-1111-4111-8111-111111111611', '11111111-1111-4111-8111-111111111412', '11111111-1111-4111-8111-111111111204', '2026-06-14 11:02:00', '2026-06-14 11:02:00', 'LOOP-1', 'synced'),
    ('11111111-1111-4111-8111-111111111612', '11111111-1111-4111-8111-111111111412', '11111111-1111-4111-8111-111111111204', '2026-06-14 12:01:00', '2026-06-14 12:01:00', 'LOOP-1', 'synced');

-- Anchor the active lap race to the current clock so live race-flow visuals work in demos.
UPDATE races
SET start_time = date_trunc('minute', NOW() - INTERVAL '4 hours')
WHERE id = '11111111-1111-4111-8111-111111111103';

UPDATE timing_records SET timestamp = r.start_time + INTERVAL '52 minutes', local_timestamp = r.start_time + INTERVAL '52 minutes'
FROM races r WHERE timing_records.id = '11111111-1111-4111-8111-111111111601' AND r.id = '11111111-1111-4111-8111-111111111103';
UPDATE timing_records SET timestamp = r.start_time + INTERVAL '106 minutes', local_timestamp = r.start_time + INTERVAL '106 minutes'
FROM races r WHERE timing_records.id = '11111111-1111-4111-8111-111111111602' AND r.id = '11111111-1111-4111-8111-111111111103';
UPDATE timing_records SET timestamp = r.start_time + INTERVAL '160 minutes', local_timestamp = r.start_time + INTERVAL '160 minutes'
FROM races r WHERE timing_records.id = '11111111-1111-4111-8111-111111111603' AND r.id = '11111111-1111-4111-8111-111111111103';
UPDATE timing_records SET timestamp = r.start_time + INTERVAL '215 minutes', local_timestamp = r.start_time + INTERVAL '215 minutes'
FROM races r WHERE timing_records.id = '11111111-1111-4111-8111-111111111604' AND r.id = '11111111-1111-4111-8111-111111111103';
UPDATE timing_records SET timestamp = r.start_time + INTERVAL '58 minutes', local_timestamp = r.start_time + INTERVAL '58 minutes'
FROM races r WHERE timing_records.id = '11111111-1111-4111-8111-111111111605' AND r.id = '11111111-1111-4111-8111-111111111103';
UPDATE timing_records SET timestamp = r.start_time + INTERVAL '117 minutes', local_timestamp = r.start_time + INTERVAL '117 minutes'
FROM races r WHERE timing_records.id = '11111111-1111-4111-8111-111111111606' AND r.id = '11111111-1111-4111-8111-111111111103';
UPDATE timing_records SET timestamp = r.start_time + INTERVAL '175 minutes', local_timestamp = r.start_time + INTERVAL '175 minutes'
FROM races r WHERE timing_records.id = '11111111-1111-4111-8111-111111111607' AND r.id = '11111111-1111-4111-8111-111111111103';
UPDATE timing_records SET timestamp = r.start_time + INTERVAL '65 minutes', local_timestamp = r.start_time + INTERVAL '65 minutes'
FROM races r WHERE timing_records.id = '11111111-1111-4111-8111-111111111608' AND r.id = '11111111-1111-4111-8111-111111111103';
UPDATE timing_records SET timestamp = r.start_time + INTERVAL '128 minutes', local_timestamp = r.start_time + INTERVAL '128 minutes'
FROM races r WHERE timing_records.id = '11111111-1111-4111-8111-111111111609' AND r.id = '11111111-1111-4111-8111-111111111103';
UPDATE timing_records SET timestamp = r.start_time + INTERVAL '192 minutes', local_timestamp = r.start_time + INTERVAL '192 minutes'
FROM races r WHERE timing_records.id = '11111111-1111-4111-8111-111111111610' AND r.id = '11111111-1111-4111-8111-111111111103';
UPDATE timing_records SET timestamp = r.start_time + INTERVAL '62 minutes', local_timestamp = r.start_time + INTERVAL '62 minutes'
FROM races r WHERE timing_records.id = '11111111-1111-4111-8111-111111111611' AND r.id = '11111111-1111-4111-8111-111111111103';
UPDATE timing_records SET timestamp = r.start_time + INTERVAL '121 minutes', local_timestamp = r.start_time + INTERVAL '121 minutes'
FROM races r WHERE timing_records.id = '11111111-1111-4111-8111-111111111612' AND r.id = '11111111-1111-4111-8111-111111111103';

COMMIT;

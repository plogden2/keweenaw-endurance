-- All You Can East Bluffet 2026 (2026-08-01, East Bluff Bike Park, Copper Harbor, MI)
-- Source: https://www.copperharbortrails.org/bluffet
-- Regenerate: python database/seed/generate_bluffet_seed.py

BEGIN;

-- Event id: 0bcc65db-df20-417e-b746-075f0d11b945

DELETE FROM categories WHERE race_id IN (
    SELECT r.id FROM races r
    JOIN events e ON r.event_id = e.id
    WHERE e.name = 'All You Can East Bluffet'
);
DELETE FROM timing_checkpoints WHERE race_id IN (
    SELECT r.id FROM races r
    JOIN events e ON r.event_id = e.id
    WHERE e.name = 'All You Can East Bluffet'
);
DELETE FROM races WHERE event_id IN (
    SELECT e.id FROM events e
    WHERE e.name = 'All You Can East Bluffet'
);
DELETE FROM events WHERE name = 'All You Can East Bluffet';

INSERT INTO events (id, name, description, event_date, location, website_url, logo_url, status)
VALUES (
    '0bcc65db-df20-417e-b746-075f0d11b945',
    'All You Can East Bluffet',
    'Feast on the Copper Harbor Trails Club''s newest event - a brand new endurance enduro at East Bluff Bike Park. Spin the wheel, shred the trails, and push your limits all day long! Expert, intermediate, and youth classes with 6- and 12-hour options. Pedal to the top, spin the mountain bike wheel for your trail, punch your race plate, and repeat. Registration at copperharbortrails.org/bluffet.',
    '2026-08-01',
    'East Bluff Bike Park, Mandan Road, Copper Harbor, MI 49918',
    'https://www.copperharbortrails.org/bluffet',
    '/images/bluffet-2026-logo.png',
    'upcoming'
);

INSERT INTO races (id, event_id, name, race_type, distance_km, duration_minutes, start_time, status)
VALUES
    (
        '7246c2ba-bdc6-4118-9553-1752bc1bd103',
        '0bcc65db-df20-417e-b746-075f0d11b945',
        '12 Hour Expert All You Can East Bluffet',
        'lap_based',
        NULL,
        720,
        '2026-08-01 08:00:00',
        'scheduled'
    ),
    (
        'fd50ef6e-39a6-4580-b205-1f0c99b49d44',
        '0bcc65db-df20-417e-b746-075f0d11b945',
        '6 Hour Expert All You Can East Bluffet',
        'lap_based',
        NULL,
        360,
        '2026-08-01 08:00:00',
        'scheduled'
    ),
    (
        '921a0a62-ab97-4307-858e-5184bcda373a',
        '0bcc65db-df20-417e-b746-075f0d11b945',
        '12 Hour Intermediate All You Can East Bluffet',
        'lap_based',
        NULL,
        720,
        '2026-08-01 08:00:00',
        'scheduled'
    ),
    (
        '762d2507-4018-4200-a300-1b2eb83c98df',
        '0bcc65db-df20-417e-b746-075f0d11b945',
        '6 Hour Intermediate All You Can East Bluffet',
        'lap_based',
        NULL,
        360,
        '2026-08-01 08:00:00',
        'scheduled'
    ),
    (
        '1ee6ed51-7ab8-48c1-9ea1-05a6cfda5236',
        '0bcc65db-df20-417e-b746-075f0d11b945',
        'Kids'' Event',
        'lap_based',
        NULL,
        90,
        '2026-08-01 15:00:00',
        'scheduled'
    );

INSERT INTO timing_checkpoints (id, race_id, name, checkpoint_type, distance_from_start_km, location_description, is_active)
VALUES
    ('da99e1c5-8f43-4f40-8571-9692576de5dc', '7246c2ba-bdc6-4118-9553-1752bc1bd103', 'Start Line', 'start', 0.00, 'Bottom of East Bluff (Flo-Rion, Dueling Banjos)', true),
    ('595f24ba-7d25-47d2-8f7c-6a764fe096c9', '7246c2ba-bdc6-4118-9553-1752bc1bd103', 'Lap Check', 'finish', NULL, 'Bottom of East Bluff (Flo-Rion, Dueling Banjos) — race trackers', true),
    ('f500837a-3cc9-450b-8db4-c4012c2f0728', 'fd50ef6e-39a6-4580-b205-1f0c99b49d44', 'Start Line', 'start', 0.00, 'Bottom of East Bluff (Flo-Rion, Dueling Banjos)', true),
    ('1f509aec-ccae-46fe-8a67-1e3cf42fef12', 'fd50ef6e-39a6-4580-b205-1f0c99b49d44', 'Lap Check', 'finish', NULL, 'Bottom of East Bluff (Flo-Rion, Dueling Banjos) — race trackers', true),
    ('ea548ac5-e020-480a-a4b4-1711609d348b', '921a0a62-ab97-4307-858e-5184bcda373a', 'Start Line', 'start', 0.00, 'Bottom of East Bluff (Flo-Rion, Dueling Banjos)', true),
    ('9bc7e304-1612-4843-8124-ae27969ef465', '921a0a62-ab97-4307-858e-5184bcda373a', 'Lap Check', 'finish', NULL, 'Bottom of East Bluff (Flo-Rion, Dueling Banjos) — race trackers', true),
    ('dbae1b5e-d6a4-4023-8063-209e838c6540', '762d2507-4018-4200-a300-1b2eb83c98df', 'Start Line', 'start', 0.00, 'Bottom of East Bluff (Flo-Rion, Dueling Banjos)', true),
    ('a442d488-e8b4-4def-9019-b645a0cc0c23', '762d2507-4018-4200-a300-1b2eb83c98df', 'Lap Check', 'finish', NULL, 'Bottom of East Bluff (Flo-Rion, Dueling Banjos) — race trackers', true),
    ('6cca1d5c-515a-4af5-8306-4fc6ddf00eae', '1ee6ed51-7ab8-48c1-9ea1-05a6cfda5236', 'Start Line', 'start', 0.00, 'Bottom of East Bluff Campground Rd', true),
    ('1bf9cfd5-9977-446e-a3c4-7269fff2eeb9', '1ee6ed51-7ab8-48c1-9ea1-05a6cfda5236', 'Lap Check', 'finish', NULL, 'Bottom of East Bluff Campground Rd — race trackers', true);

INSERT INTO categories (id, race_id, name, category_type, age_min, age_max, gender_filter, display_order)
VALUES
    ('a733ce82-3186-4a80-ad68-7f7c138bc203', '7246c2ba-bdc6-4118-9553-1752bc1bd103', 'Overall', 'overall', NULL, NULL, NULL, 0),
    ('81c08279-9c87-4d72-9679-5296b4a7e9b3', '7246c2ba-bdc6-4118-9553-1752bc1bd103', 'Men', 'male', NULL, NULL, 'male', 1),
    ('40738360-abe5-4734-997a-7135d422c49a', '7246c2ba-bdc6-4118-9553-1752bc1bd103', 'Women', 'female', NULL, NULL, 'female', 2),
    ('d824726c-a63d-4fc0-b4ac-10d856426c0c', 'fd50ef6e-39a6-4580-b205-1f0c99b49d44', 'Overall', 'overall', NULL, NULL, NULL, 0),
    ('d9c4b1dc-b1f6-4073-92b1-215a6f60ce76', 'fd50ef6e-39a6-4580-b205-1f0c99b49d44', 'Men', 'male', NULL, NULL, 'male', 1),
    ('a76fde2e-447e-4cbe-93aa-b77765101209', 'fd50ef6e-39a6-4580-b205-1f0c99b49d44', 'Women', 'female', NULL, NULL, 'female', 2),
    ('4708a5b1-643d-42b6-bedb-3bb641e00343', '921a0a62-ab97-4307-858e-5184bcda373a', 'Overall', 'overall', NULL, NULL, NULL, 0),
    ('9d3d283a-b604-43b1-a292-a682f7200aee', '921a0a62-ab97-4307-858e-5184bcda373a', 'Men', 'male', NULL, NULL, 'male', 1),
    ('c7c29a51-6f02-47dd-bf76-ab03ce843ba4', '921a0a62-ab97-4307-858e-5184bcda373a', 'Women', 'female', NULL, NULL, 'female', 2),
    ('a4a37807-9782-4c4f-bcbb-2ad5e8ddf350', '762d2507-4018-4200-a300-1b2eb83c98df', 'Overall', 'overall', NULL, NULL, NULL, 0),
    ('f9989858-1f4b-421e-a845-6ecc59a82ece', '762d2507-4018-4200-a300-1b2eb83c98df', 'Men', 'male', NULL, NULL, 'male', 1),
    ('55b7a2dd-a2b7-4c55-9fed-175d272a8ce4', '762d2507-4018-4200-a300-1b2eb83c98df', 'Women', 'female', NULL, NULL, 'female', 2),
    ('8731ec4b-fe92-4975-9220-a0f6832423f4', '1ee6ed51-7ab8-48c1-9ea1-05a6cfda5236', 'Youth', 'age_group', 5, 12, NULL, 0);

COMMIT;

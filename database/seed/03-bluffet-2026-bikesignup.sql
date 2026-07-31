-- All You Can East Bluffet 2026 — BikeSignup production roster
-- Source: https://www.bikesignup.com/Race/FindARunner/?raceId=181840&perPage=50
-- Regenerate: python database/seed/generate_bluffet_bikesignup_seed.py
--
-- Event 1441674d-a011-471a-a601-722b88b117f5; races start 2026-08-01 12:00:00 America/Detroit
-- 46 participants (public list; anonymous registrants omitted)
-- bib_number NULL (unassigned); no bibs / RFID tag associations
-- Categories: Expert/Intermediate × Men/Women (12h/6h); Men/Women (kids)

BEGIN;

-- Allow unassigned bibs for race-morning assignment workflow
ALTER TABLE participants ALTER COLUMN bib_number DROP NOT NULL;

-- Clean prior Bluffet data (order respects FKs)
UPDATE timing_records SET station_id = NULL WHERE station_id IN (
    SELECT rs.id FROM reader_stations rs
    JOIN events e ON rs.event_id = e.id
    WHERE e.name = 'All You Can East Bluffet'
);
DELETE FROM reader_stations WHERE event_id IN (
    SELECT e.id FROM events e
    WHERE e.name = 'All You Can East Bluffet'
);
DELETE FROM timing_records WHERE participant_id IN (
    SELECT p.id FROM participants p
    WHERE p.race_id IN (
    SELECT r.id FROM races r
    JOIN events e ON r.event_id = e.id
    WHERE e.name = 'All You Can East Bluffet'
    )
);
DELETE FROM rfid_tag_associations WHERE bib_id IN (
    SELECT b.id FROM bibs b
    JOIN events e ON b.event_id = e.id
    WHERE e.name = 'All You Can East Bluffet'
);
DELETE FROM participants WHERE race_id IN (
    SELECT r.id FROM races r
    JOIN events e ON r.event_id = e.id
    WHERE e.name = 'All You Can East Bluffet'
);
DELETE FROM bibs WHERE event_id IN (
    SELECT e.id FROM events e
    WHERE e.name = 'All You Can East Bluffet'
);
DELETE FROM teams WHERE race_id IN (
    SELECT r.id FROM races r
    JOIN events e ON r.event_id = e.id
    WHERE e.name = 'All You Can East Bluffet'
);
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
    '1441674d-a011-471a-a601-722b88b117f5',
    'All You Can East Bluffet',
    'Feast on the Copper Harbor Trails Club''s newest event - a brand new endurance enduro at East Bluff Bike Park. Spin the wheel, shred the trails, and push your limits all day long! Expert, intermediate, and kids classes with 6- and 12-hour options plus a 90-minute kids race. Registration via BikeSignup.',
    '2026-08-01',
    'East Bluff Bike Park, Mandan Road, Copper Harbor, MI 49918',
    'https://www.copperharbortrails.org/bluffet',
    '/images/bluffet-2026-logo.png',
    'upcoming'
);

INSERT INTO races (id, event_id, name, race_type, distance_km, duration_minutes, start_time, status)
VALUES
    (
        '17da3ba1-2e09-4eb1-aeb3-d9dd5b6a394e',
        '1441674d-a011-471a-a601-722b88b117f5',
        '12 Hour',
        'lap_based',
        NULL,
        720,
        '2026-08-01 12:00:00',  -- America/Detroit
        'scheduled'
    ),
    (
        '209769a1-f723-4f70-ae90-466a46338684',
        '1441674d-a011-471a-a601-722b88b117f5',
        '6 Hour',
        'lap_based',
        NULL,
        360,
        '2026-08-01 12:00:00',  -- America/Detroit
        'scheduled'
    ),
    (
        '0e45ee85-800c-4e1f-a95b-4b92462e790a',
        '1441674d-a011-471a-a601-722b88b117f5',
        '90-Minute Kids',
        'lap_based',
        NULL,
        90,
        '2026-08-01 12:00:00',  -- America/Detroit
        'scheduled'
    );

INSERT INTO timing_checkpoints (id, race_id, name, checkpoint_type, distance_from_start_km, location_description, is_active)
VALUES
    ('c9ef4f76-8851-5cf6-a10a-7ed48b6f8b78', '17da3ba1-2e09-4eb1-aeb3-d9dd5b6a394e', 'Start Line', 'start', 0.00, 'Bottom of East Bluff (Flo-Rion, Dueling Banjos)', true),
    ('81ca12c0-dfec-512e-b605-7e1dfbcb63f5', '17da3ba1-2e09-4eb1-aeb3-d9dd5b6a394e', 'Lap Check', 'finish', NULL, 'Bottom of East Bluff (Flo-Rion, Dueling Banjos) — race trackers', true),
    ('32283593-3538-5e7f-804f-8e4d65f82752', '209769a1-f723-4f70-ae90-466a46338684', 'Start Line', 'start', 0.00, 'Bottom of East Bluff (Flo-Rion, Dueling Banjos)', true),
    ('5b7e8d76-8cc4-5e17-9147-9ed99a8df6fa', '209769a1-f723-4f70-ae90-466a46338684', 'Lap Check', 'finish', NULL, 'Bottom of East Bluff (Flo-Rion, Dueling Banjos) — race trackers', true),
    ('6be812e7-6991-5f35-a69f-ad81a90591c9', '0e45ee85-800c-4e1f-a95b-4b92462e790a', 'Start Line', 'start', 0.00, 'Bottom of East Bluff Campground Rd', true),
    ('31b14fd1-a863-57e1-b90f-45f1593cdd49', '0e45ee85-800c-4e1f-a95b-4b92462e790a', 'Lap Check', 'finish', NULL, 'Bottom of East Bluff Campground Rd — race trackers', true);

INSERT INTO categories (id, race_id, name, category_type, age_min, age_max, gender_filter, display_order)
VALUES
    ('c6c300d6-1c19-5b0b-9356-bd445cafd68d', '17da3ba1-2e09-4eb1-aeb3-d9dd5b6a394e', 'Expert Men', 'custom', NULL, NULL, 'male', 0),
    ('5fcddd89-2b73-5db9-b64f-b9fd54599aa0', '17da3ba1-2e09-4eb1-aeb3-d9dd5b6a394e', 'Expert Women', 'custom', NULL, NULL, 'female', 1),
    ('6d2d19e6-0552-5dca-ae67-d1189221fed9', '17da3ba1-2e09-4eb1-aeb3-d9dd5b6a394e', 'Intermediate Men', 'custom', NULL, NULL, 'male', 2),
    ('0b21d8e0-efa9-578d-a28d-d1577edb82a1', '17da3ba1-2e09-4eb1-aeb3-d9dd5b6a394e', 'Intermediate Women', 'custom', NULL, NULL, 'female', 3),
    ('335b231a-b711-5ab5-8078-fea1073ca3b0', '209769a1-f723-4f70-ae90-466a46338684', 'Expert Men', 'custom', NULL, NULL, 'male', 0),
    ('536975ac-d0ca-5424-bbab-c1604be4d89d', '209769a1-f723-4f70-ae90-466a46338684', 'Expert Women', 'custom', NULL, NULL, 'female', 1),
    ('c671ab5b-b5b5-58c3-b836-115888c9d27c', '209769a1-f723-4f70-ae90-466a46338684', 'Intermediate Men', 'custom', NULL, NULL, 'male', 2),
    ('6221ba18-7fb8-50a2-8ed0-42d03cdecdc8', '209769a1-f723-4f70-ae90-466a46338684', 'Intermediate Women', 'custom', NULL, NULL, 'female', 3),
    ('212d9db9-11f7-5ce7-8315-6b4092838bb0', '0e45ee85-800c-4e1f-a95b-4b92462e790a', 'Men', 'male', NULL, NULL, 'male', 0),
    ('96793b70-e74f-5d81-aa59-17aeea18d5c1', '0e45ee85-800c-4e1f-a95b-4b92462e790a', 'Women', 'female', NULL, NULL, 'female', 1);

-- participants: bib_number intentionally NULL (unassigned until race morning)
INSERT INTO participants (id, race_id, bib_number, first_name, last_name, gender, age, location, rfid_tag_uid, status, category_id, team_id)
VALUES
    (
        '3b01a5c1-16e3-54b3-a37a-bb9fe5b4f0cd',
        '0e45ee85-800c-4e1f-a95b-4b92462e790a',
        NULL,
        'F.',
        'Ahrens',
        'male',
        11,
        'Calumet, MI',
        NULL,
        'registered',
        '212d9db9-11f7-5ce7-8315-6b4092838bb0',
        NULL
    ),
    (
        'a901cfeb-e70a-5040-9be2-5fc9e8e08d3b',
        '0e45ee85-800c-4e1f-a95b-4b92462e790a',
        NULL,
        'T.',
        'Ahrens',
        'female',
        8,
        'Calumet, MI',
        NULL,
        'registered',
        '96793b70-e74f-5d81-aa59-17aeea18d5c1',
        NULL
    ),
    (
        '02ad3b0b-ed7d-5ed4-8e88-62a58cb4e0bc',
        '209769a1-f723-4f70-ae90-466a46338684',
        NULL,
        'Aunders',
        'Anderson',
        'male',
        15,
        'Calumet, MI',
        NULL,
        'registered',
        '335b231a-b711-5ab5-8078-fea1073ca3b0',
        NULL
    ),
    (
        '2ea61165-7c89-5854-be52-f6ec18c589e3',
        '209769a1-f723-4f70-ae90-466a46338684',
        NULL,
        'Kyia',
        'Anderson',
        'female',
        50,
        'Calumet, MI',
        NULL,
        'registered',
        '536975ac-d0ca-5424-bbab-c1604be4d89d',
        NULL
    ),
    (
        'f630b65d-0724-5983-ace2-f9ffaccd00bc',
        '209769a1-f723-4f70-ae90-466a46338684',
        NULL,
        'Tony',
        'Anderson',
        'male',
        32,
        'Chassell, MI',
        NULL,
        'registered',
        'c671ab5b-b5b5-58c3-b836-115888c9d27c',
        NULL
    ),
    (
        '52d5f1b3-2271-5594-80af-4f090edd6193',
        '209769a1-f723-4f70-ae90-466a46338684',
        NULL,
        'Lauren',
        'Atkinson',
        'female',
        23,
        'Northville, MI',
        NULL,
        'registered',
        '6221ba18-7fb8-50a2-8ed0-42d03cdecdc8',
        NULL
    ),
    (
        '291085ef-09b9-5d9c-be5c-2c4cb96af5d7',
        '209769a1-f723-4f70-ae90-466a46338684',
        NULL,
        'Austin',
        'Balken',
        'male',
        38,
        'Burnsville, MN',
        NULL,
        'registered',
        'c671ab5b-b5b5-58c3-b836-115888c9d27c',
        NULL
    ),
    (
        'a605ed31-3400-5370-b2ed-0ba680f6c6f5',
        '209769a1-f723-4f70-ae90-466a46338684',
        NULL,
        'Josh',
        'Blum',
        'male',
        48,
        'La Crosse, WI',
        NULL,
        'registered',
        '335b231a-b711-5ab5-8078-fea1073ca3b0',
        NULL
    ),
    (
        'f9996178-3c55-5d5d-929f-cbf333ce7f2f',
        '17da3ba1-2e09-4eb1-aeb3-d9dd5b6a394e',
        NULL,
        'Jason',
        'Brekhus',
        'male',
        49,
        'Minocqua, WI',
        NULL,
        'registered',
        'c6c300d6-1c19-5b0b-9356-bd445cafd68d',
        NULL
    ),
    (
        '95190f0f-9b3b-5eca-b744-05ff52262c77',
        '209769a1-f723-4f70-ae90-466a46338684',
        NULL,
        'Adrian',
        'Brokenshire',
        'female',
        24,
        'Troy, MI',
        NULL,
        'registered',
        '6221ba18-7fb8-50a2-8ed0-42d03cdecdc8',
        NULL
    ),
    (
        '92c4335e-feca-51c4-8776-62f19323050f',
        '17da3ba1-2e09-4eb1-aeb3-d9dd5b6a394e',
        NULL,
        'Benjamin',
        'Ciavola',
        'male',
        39,
        'Houghton, MI',
        NULL,
        'registered',
        'c6c300d6-1c19-5b0b-9356-bd445cafd68d',
        NULL
    ),
    (
        '7367c52d-be9e-52fa-975f-4f62b10f27e1',
        '17da3ba1-2e09-4eb1-aeb3-d9dd5b6a394e',
        NULL,
        'Mael',
        'Fleury',
        'male',
        20,
        'Rochester, MI',
        NULL,
        'registered',
        'c6c300d6-1c19-5b0b-9356-bd445cafd68d',
        NULL
    ),
    (
        '61afc878-1c92-5cdc-8922-215c6394efe2',
        '209769a1-f723-4f70-ae90-466a46338684',
        NULL,
        'Pierre',
        'Fleury',
        'male',
        49,
        'Rochester, MI',
        NULL,
        'registered',
        'c671ab5b-b5b5-58c3-b836-115888c9d27c',
        NULL
    ),
    (
        '82653553-d05d-5296-81a3-c61adcbf6f3c',
        '209769a1-f723-4f70-ae90-466a46338684',
        NULL,
        'Casey',
        'Goethel',
        'male',
        22,
        'Houghton, MI',
        NULL,
        'registered',
        '335b231a-b711-5ab5-8078-fea1073ca3b0',
        NULL
    ),
    (
        '2c79f96b-38b8-5126-aa23-be0d9c551a00',
        '209769a1-f723-4f70-ae90-466a46338684',
        NULL,
        'Michael',
        'Gumbleton',
        'male',
        33,
        'Grand Rapids, MI',
        NULL,
        'registered',
        'c671ab5b-b5b5-58c3-b836-115888c9d27c',
        NULL
    ),
    (
        '90fa1cf0-9778-5ee8-88af-3694d3832fd4',
        '209769a1-f723-4f70-ae90-466a46338684',
        NULL,
        'Brandon',
        'Hajdo-Fernandez',
        'male',
        41,
        'Atlantic Mine, MI',
        NULL,
        'registered',
        '335b231a-b711-5ab5-8078-fea1073ca3b0',
        NULL
    ),
    (
        '3a054e2f-7eae-5a28-8d9d-7b2f3c54a33a',
        '17da3ba1-2e09-4eb1-aeb3-d9dd5b6a394e',
        NULL,
        'Matthew',
        'Heirtzler',
        'male',
        21,
        'Northville, MI',
        NULL,
        'registered',
        'c6c300d6-1c19-5b0b-9356-bd445cafd68d',
        NULL
    ),
    (
        '556e9d33-52dc-5749-92c8-c5168a18b3ba',
        '209769a1-f723-4f70-ae90-466a46338684',
        NULL,
        'Tyler',
        'Hosking',
        'male',
        30,
        'Laurium, MI',
        NULL,
        'registered',
        'c671ab5b-b5b5-58c3-b836-115888c9d27c',
        NULL
    ),
    (
        'ff11a764-8a6b-59b9-84bc-50624a9a20ed',
        '17da3ba1-2e09-4eb1-aeb3-d9dd5b6a394e',
        NULL,
        'Adam',
        'Kazilsky',
        'male',
        33,
        'Duluth, MN',
        NULL,
        'registered',
        'c6c300d6-1c19-5b0b-9356-bd445cafd68d',
        NULL
    ),
    (
        'f370d1b4-8a8c-5238-aa13-838f88a92bcf',
        '209769a1-f723-4f70-ae90-466a46338684',
        NULL,
        'Zane',
        'Kile',
        'male',
        33,
        'Plymouth, MI',
        NULL,
        'registered',
        'c671ab5b-b5b5-58c3-b836-115888c9d27c',
        NULL
    ),
    (
        '60f40909-a183-5524-a2d1-30e14834ceb4',
        '209769a1-f723-4f70-ae90-466a46338684',
        NULL,
        'Amanda',
        'Kilpela',
        'female',
        39,
        'Atlantic Mine, MI',
        NULL,
        'registered',
        '6221ba18-7fb8-50a2-8ed0-42d03cdecdc8',
        NULL
    ),
    (
        'b3028521-8572-589c-8e85-9f6f3d9c54ef',
        '209769a1-f723-4f70-ae90-466a46338684',
        NULL,
        'Thomas',
        'Klarr',
        'male',
        33,
        'Howell, MI',
        NULL,
        'registered',
        'c671ab5b-b5b5-58c3-b836-115888c9d27c',
        NULL
    ),
    (
        '7448693c-ba13-5214-8af9-615ba2f11b50',
        '209769a1-f723-4f70-ae90-466a46338684',
        NULL,
        'Ian',
        'Klein',
        'male',
        14,
        'Hancock, MI',
        NULL,
        'registered',
        '335b231a-b711-5ab5-8078-fea1073ca3b0',
        NULL
    ),
    (
        '33722523-6c5d-57dc-bb06-41af40f8243c',
        '209769a1-f723-4f70-ae90-466a46338684',
        NULL,
        'Maggie',
        'Klein',
        'female',
        15,
        'Hancock, MI',
        NULL,
        'registered',
        '536975ac-d0ca-5424-bbab-c1604be4d89d',
        NULL
    ),
    (
        '3f4ac26e-6d72-5f92-b57d-b46656362eb5',
        '209769a1-f723-4f70-ae90-466a46338684',
        NULL,
        'Mark',
        'Klein',
        'male',
        44,
        'Hancock, MI',
        NULL,
        'registered',
        '335b231a-b711-5ab5-8078-fea1073ca3b0',
        NULL
    ),
    (
        '0bfa63d7-b7cf-55f7-9a01-1c68470acbdc',
        '17da3ba1-2e09-4eb1-aeb3-d9dd5b6a394e',
        NULL,
        'Josh',
        'Kocha',
        'male',
        42,
        'Marquette, MI',
        NULL,
        'registered',
        '6d2d19e6-0552-5dca-ae67-d1189221fed9',
        NULL
    ),
    (
        '9f6516e6-5b9c-5fa8-a00b-b62686f4cf08',
        '0e45ee85-800c-4e1f-a95b-4b92462e790a',
        NULL,
        'M.',
        'Kocha',
        'male',
        9,
        'Marquette, MI',
        NULL,
        'registered',
        '212d9db9-11f7-5ce7-8315-6b4092838bb0',
        NULL
    ),
    (
        'eadc870d-4193-5e7f-8152-46beb1b97baa',
        '17da3ba1-2e09-4eb1-aeb3-d9dd5b6a394e',
        NULL,
        'Sara',
        'Kocha',
        'female',
        40,
        'Marquette, MI',
        NULL,
        'registered',
        '0b21d8e0-efa9-578d-a28d-d1577edb82a1',
        NULL
    ),
    (
        'b0cac76a-a52b-574a-8973-0ea7e3f688b9',
        '209769a1-f723-4f70-ae90-466a46338684',
        NULL,
        'Sabrina',
        'Loven-Gulick',
        'female',
        34,
        'Calumet, MI',
        NULL,
        'registered',
        '6221ba18-7fb8-50a2-8ed0-42d03cdecdc8',
        NULL
    ),
    (
        '2da9530b-13c8-57f7-80ba-e613a38d473a',
        '17da3ba1-2e09-4eb1-aeb3-d9dd5b6a394e',
        NULL,
        'Lauren',
        'Luce',
        'female',
        37,
        'Marquette, MI',
        NULL,
        'registered',
        '0b21d8e0-efa9-578d-a28d-d1577edb82a1',
        NULL
    ),
    (
        '519b336e-3caf-5a28-bf48-b57efc126391',
        '17da3ba1-2e09-4eb1-aeb3-d9dd5b6a394e',
        NULL,
        'Matt',
        'Luce',
        'male',
        36,
        'Marquette, MI',
        NULL,
        'registered',
        '6d2d19e6-0552-5dca-ae67-d1189221fed9',
        NULL
    ),
    (
        '554381d2-6145-52d7-bbdc-8a3bb71e0471',
        '209769a1-f723-4f70-ae90-466a46338684',
        NULL,
        'Mike',
        'Macfarland',
        'male',
        37,
        'Houghton, MI',
        NULL,
        'registered',
        '335b231a-b711-5ab5-8078-fea1073ca3b0',
        NULL
    ),
    (
        '57e7d934-03e1-5613-86be-b6ce37995197',
        '209769a1-f723-4f70-ae90-466a46338684',
        NULL,
        'Karina',
        'Madigan',
        'female',
        25,
        'Calumet, MI',
        NULL,
        'registered',
        '6221ba18-7fb8-50a2-8ed0-42d03cdecdc8',
        NULL
    ),
    (
        'ed85789c-7ce2-5643-9d05-1e266604cdbe',
        '209769a1-f723-4f70-ae90-466a46338684',
        NULL,
        'Isabel',
        'Matthews',
        'female',
        22,
        'Eden Prairie, MI',
        NULL,
        'registered',
        '6221ba18-7fb8-50a2-8ed0-42d03cdecdc8',
        NULL
    ),
    (
        '2d7bed59-1d28-58f9-b529-1ad591970491',
        '209769a1-f723-4f70-ae90-466a46338684',
        NULL,
        'Ella',
        'Merklein',
        'female',
        23,
        'Hartford, WI',
        NULL,
        'registered',
        '6221ba18-7fb8-50a2-8ed0-42d03cdecdc8',
        NULL
    ),
    (
        'ae947f14-a253-57fc-84b8-5e5b4208d88a',
        '209769a1-f723-4f70-ae90-466a46338684',
        NULL,
        'Braden',
        'Miller',
        'male',
        21,
        'Lake Linden, MI',
        NULL,
        'registered',
        '335b231a-b711-5ab5-8078-fea1073ca3b0',
        NULL
    ),
    (
        'fb0a1fd3-7740-5235-aa77-a02ea2d5e272',
        '209769a1-f723-4f70-ae90-466a46338684',
        NULL,
        'Jack',
        'Miller',
        'male',
        24,
        'Lake Linden, MI',
        NULL,
        'registered',
        '335b231a-b711-5ab5-8078-fea1073ca3b0',
        NULL
    ),
    (
        '8c277b4c-c826-54c6-a16b-b25614113784',
        '209769a1-f723-4f70-ae90-466a46338684',
        NULL,
        'Sean',
        'Moisan',
        'male',
        28,
        'Wheat Ridge, CO',
        NULL,
        'registered',
        '335b231a-b711-5ab5-8078-fea1073ca3b0',
        NULL
    ),
    (
        'a2e698c1-6142-5460-886f-3ee4afcee9fe',
        '17da3ba1-2e09-4eb1-aeb3-d9dd5b6a394e',
        NULL,
        'Isaac',
        'Oldenburg',
        'male',
        21,
        'Houghton, MI',
        NULL,
        'registered',
        'c6c300d6-1c19-5b0b-9356-bd445cafd68d',
        NULL
    ),
    (
        'f976e876-202b-54ad-b8cd-0affbc003db7',
        '209769a1-f723-4f70-ae90-466a46338684',
        NULL,
        'Jayden',
        'Randall',
        'female',
        16,
        'Copper Harbor, MI',
        NULL,
        'registered',
        '536975ac-d0ca-5424-bbab-c1604be4d89d',
        NULL
    ),
    (
        '1941aa97-7954-5441-b498-08ddfc12775f',
        '209769a1-f723-4f70-ae90-466a46338684',
        NULL,
        'Lisa',
        'Randall',
        'female',
        48,
        'Copper Harbor, MI',
        NULL,
        'registered',
        '536975ac-d0ca-5424-bbab-c1604be4d89d',
        NULL
    ),
    (
        'f861c867-d4da-510c-ae9a-23648563814b',
        '209769a1-f723-4f70-ae90-466a46338684',
        NULL,
        'Darrell',
        'Robinette',
        'male',
        44,
        'Dollar Bay, MI',
        NULL,
        'registered',
        '335b231a-b711-5ab5-8078-fea1073ca3b0',
        NULL
    ),
    (
        'e5893b0d-dfa1-5fc8-84b8-08e98b24b0d1',
        '209769a1-f723-4f70-ae90-466a46338684',
        NULL,
        'Kendall',
        'Shoecraft',
        'male',
        40,
        'Ypsilanti, MI',
        NULL,
        'registered',
        '335b231a-b711-5ab5-8078-fea1073ca3b0',
        NULL
    ),
    (
        'efa5b8f7-ca7d-5c42-970c-1cb740a68710',
        '209769a1-f723-4f70-ae90-466a46338684',
        NULL,
        'Jonas',
        'Sublett',
        'male',
        47,
        'Rushford, MN',
        NULL,
        'registered',
        '335b231a-b711-5ab5-8078-fea1073ca3b0',
        NULL
    ),
    (
        '2c392ef4-9f53-54fe-9fe7-890ec66a9568',
        '209769a1-f723-4f70-ae90-466a46338684',
        NULL,
        'Ryan',
        'Verbrugge',
        'male',
        23,
        'Eden Prairie, MN',
        NULL,
        'registered',
        'c671ab5b-b5b5-58c3-b836-115888c9d27c',
        NULL
    ),
    (
        '4d8ea71e-2fc9-5164-8184-c29f149ec34f',
        '209769a1-f723-4f70-ae90-466a46338684',
        NULL,
        'Justus',
        'Witmer',
        'male',
        34,
        'Eden Prairie, MN',
        NULL,
        'registered',
        'c671ab5b-b5b5-58c3-b836-115888c9d27c',
        NULL
    );

COMMIT;

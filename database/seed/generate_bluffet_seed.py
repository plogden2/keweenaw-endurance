#!/usr/bin/env python3
"""Emit All You Can East Bluffet 2026 seed SQL (3 races, 100 racers, demo tags).

Requires migration 04-rfid-scanner.sql columns/tables:
  - participants.category_id
  - participants.rfid_tag_uid
  - rfid_tag_associations (id, participant_id, tag_uid, created_at, active)

UUIDs are deterministic (fixed event/race IDs + uuid5 children) so e2e fixtures
in frontend/e2e/fixtures/rfid.ts stay stable across regenerations.
"""

from __future__ import annotations

import os
import re
import uuid
from pathlib import Path

OUTPUT_SQL = Path(__file__).parent / "03-bluffet-2026.sql"
EVENT_NAME = "All You Can East Bluffet"

# Stable IDs matching frontend/e2e/fixtures/rfid.ts BLUFFET constants
EVENT_ID = "1441674d-a011-471a-a601-722b88b117f5"
RACE_12H_ID = "17da3ba1-2e09-4eb1-aeb3-d9dd5b6a394e"
RACE_6H_ID = "209769a1-f723-4f70-ae90-466a46338684"
RACE_KIDS_ID = "0e45ee85-800c-4e1f-a95b-4b92462e790a"

# Namespace for uuid5 child IDs (checkpoints, categories, participants, tags)
_NS = uuid.UUID(EVENT_ID)

# America/Detroit wall times on 2026-08-01 (EDT = UTC-4)
START_12H = "2026-08-01 08:00:00-04"
START_6H = "2026-08-01 08:00:00-04"
START_KIDS = "2026-08-01 15:00:00-04"

FIRST_NAMES = [
    "Alex", "Jordan", "Sam", "Casey", "Riley", "Morgan", "Taylor", "Quinn",
    "Avery", "Parker", "Reese", "Skyler", "Cameron", "Drew", "Emerson", "Finley",
    "Harper", "Hayden", "Jamie", "Kai", "Logan", "Noah", "Owen", "Peyton",
    "River", "Sage", "Shawn", "Sydney", "Terry", "Blair",
]
LAST_NAMES = [
    "Rivera", "Chen", "Nguyen", "Patel", "Brooks", "Keller", "Sullivan", "Ortiz",
    "Anders", "Berg", "Costa", "Diaz", "Ellis", "Frost", "Garcia", "Hahn",
    "Ito", "Jones", "Kim", "Lopez", "Meyer", "Nash", "Olsen", "Perez",
    "Quinn", "Reed", "Singh", "Torres", "Underwood", "Vega",
]


def stable_uuid(name: str) -> str:
    """Deterministic UUID5 under the Bluffet event namespace."""
    return str(uuid.uuid5(_NS, name))


def sql_str(value: str) -> str:
    return "'" + value.replace("'", "''") + "'"


def sql_null_or_str(value: str | None) -> str:
    if value is None:
        return "NULL"
    return sql_str(value)


def main() -> None:
    event_id = EVENT_ID

    races = [
        {
            "id": RACE_12H_ID,
            "name": "12 Hour",
            "duration": 720,
            "start": START_12H,
            "loc": "Bottom of East Bluff (Flo-Rion, Dueling Banjos)",
            "categories": [
                ("Intermediate Men", "custom", "male", 0),
                ("Intermediate Women", "custom", "female", 1),
                ("Advanced Men", "custom", "male", 2),
                ("Advanced Women", "custom", "female", 3),
            ],
            "participant_count": 40,
        },
        {
            "id": RACE_6H_ID,
            "name": "6 Hour",
            "duration": 360,
            "start": START_6H,
            "loc": "Bottom of East Bluff (Flo-Rion, Dueling Banjos)",
            "categories": [
                ("Intermediate Men", "custom", "male", 0),
                ("Intermediate Women", "custom", "female", 1),
                ("Advanced Men", "custom", "male", 2),
                ("Advanced Women", "custom", "female", 3),
            ],
            "participant_count": 40,
        },
        {
            "id": RACE_KIDS_ID,
            "name": "90-Minute Kids",
            "duration": 90,
            "start": START_KIDS,
            "loc": "Bottom of East Bluff Campground Rd",
            "categories": [
                ("Men", "male", "male", 0),
                ("Women", "female", "female", 1),
            ],
            "participant_count": 20,
        },
    ]

    checkpoint_rows: list[str] = []
    category_rows: list[str] = []
    # (participant_id, race_id, bib, first, last, gender, age, tag_uid, category_id)
    participants: list[tuple[str, str, str, str, str, str, int, str, str]] = []
    association_rows: list[str] = []

    name_idx = 0
    all_tag_uids: list[str] = []

    for race in races:
        race_id = race["id"]
        race_key = race["name"].lower().replace(" ", "-")
        loc = race["loc"]
        start_cp = stable_uuid(f"checkpoint:{race_key}:start")
        finish_cp = stable_uuid(f"checkpoint:{race_key}:finish")
        checkpoint_rows.append(
            f"    ('{start_cp}', '{race_id}', 'Start Line', 'start', 0.00, {sql_str(loc)}, true)"
        )
        checkpoint_rows.append(
            f"    ('{finish_cp}', '{race_id}', 'Lap Check', 'finish', NULL, "
            f"{sql_str(loc + ' — race trackers')}, true)"
        )

        cat_ids: list[tuple[str, str]] = []  # (id, gender)
        for name, ctype, gender, order in race["categories"]:
            cat_id = stable_uuid(f"category:{race_key}:{name}")
            cat_ids.append((cat_id, gender))
            category_rows.append(
                f"    ('{cat_id}', '{race_id}', {sql_str(name)}, {sql_str(ctype)}, "
                f"NULL, NULL, {sql_null_or_str(gender)}, {order})"
            )

        count = race["participant_count"]
        for i in range(count):
            cat_id, gender = cat_ids[i % len(cat_ids)]
            pid = stable_uuid(f"participant:{race_key}:{i + 1}")
            bib = str(i + 1)
            first = FIRST_NAMES[name_idx % len(FIRST_NAMES)]
            last = LAST_NAMES[(name_idx * 3) % len(LAST_NAMES)]
            name_idx += 1
            age = 10 + (i % 8) if race["name"] == "90-Minute Kids" else 25 + (i % 30)
            tag_uid = stable_uuid(f"tag:{race_key}:{i + 1}")
            all_tag_uids.append(tag_uid)
            participants.append(
                (pid, race_id, bib, first, last, gender, age, tag_uid, cat_id)
            )
            association_rows.append(
                f"    ('{stable_uuid(f'tag-assoc:{tag_uid}')}', '{pid}', "
                f"{sql_str(tag_uid)}, true)"
            )

    assert len(participants) == 100, f"expected 100 participants, got {len(participants)}"
    uuid_re = re.compile(
        r"^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$"
    )
    for tag_uid in all_tag_uids:
        assert uuid_re.match(tag_uid), f"tag_uid must be UUID, got {tag_uid!r}"
        assert not tag_uid.startswith("DEMO-TAG"), f"legacy DEMO-TAG prefix: {tag_uid}"
    assert len(category_rows) == 10, f"expected 10 categories, got {len(category_rows)}"

    if os.environ.get("DEBUG_TAGS") == "1":
        for tag_uid in all_tag_uids[:3]:
            print(tag_uid)

    event_filter = f"e.name = {sql_str(EVENT_NAME)}"
    race_subq = (
        "SELECT r.id FROM races r\n"
        "    JOIN events e ON r.event_id = e.id\n"
        f"    WHERE {event_filter}"
    )
    participant_subq = (
        "SELECT p.id FROM participants p\n"
        f"    WHERE p.race_id IN (\n    {race_subq}\n    )"
    )

    lines = [
        "-- All You Can East Bluffet 2026 (2026-08-01, East Bluff Bike Park, Copper Harbor, MI)",
        "-- Source: https://www.copperharbortrails.org/bluffet",
        "-- Regenerate: python database/seed/generate_bluffet_seed.py",
        "--",
        "-- Deterministic UUIDs (event/races match frontend/e2e/fixtures/rfid.ts BLUFFET)",
        "-- 3 races: 12 Hour (08:00), 6 Hour (08:00), 90-Minute Kids (15:00) America/Detroit",
        "-- Categories: Intermediate/Advanced × Men/Women (12h/6h); Men/Women (kids)",
        "-- 100 participants with category_id + deterministic tag UUIDs (uuid5)",
        "-- Requires: database/migrations/04-rfid-scanner.sql (category_id, rfid_tag_associations)",
        "",
        "BEGIN;",
        "",
        f"-- Event id: {event_id}",
        "",
        "-- Clean prior Bluffet seed (order respects FKs)",
        "DELETE FROM rfid_tag_associations WHERE participant_id IN (",
        f"    {participant_subq}",
        ");",
        "DELETE FROM participants WHERE race_id IN (",
        f"    {race_subq}",
        ");",
        "DELETE FROM categories WHERE race_id IN (",
        f"    {race_subq}",
        ");",
        "DELETE FROM timing_checkpoints WHERE race_id IN (",
        f"    {race_subq}",
        ");",
        "DELETE FROM races WHERE event_id IN (",
        "    SELECT e.id FROM events e",
        f"    WHERE {event_filter}",
        ");",
        f"DELETE FROM events WHERE name = {sql_str(EVENT_NAME)};",
        "",
        "INSERT INTO events (id, name, description, event_date, location, website_url, logo_url, status)",
        "VALUES (",
        f"    '{event_id}',",
        f"    {sql_str(EVENT_NAME)},",
        "    'Feast on the Copper Harbor Trails Club''s newest event - a brand new endurance enduro at East Bluff Bike Park. Spin the wheel, shred the trails, and push your limits all day long! Advanced, intermediate, and kids classes with 6- and 12-hour options plus a 90-minute kids race. Pedal to the top, spin the mountain bike wheel for your trail, punch your race plate, and repeat. Registration at copperharbortrails.org/bluffet.',",
        "    '2026-08-01',",
        "    'East Bluff Bike Park, Mandan Road, Copper Harbor, MI 49918',",
        "    'https://www.copperharbortrails.org/bluffet',",
        "    '/images/bluffet-2026-logo.png',",
        "    'upcoming'",
        ");",
        "",
        "INSERT INTO races (id, event_id, name, race_type, distance_km, duration_minutes, start_time, status)",
        "VALUES",
    ]

    race_values = []
    for race in races:
        race_values.append(
            "    (\n"
            f"        '{race['id']}',\n"
            f"        '{event_id}',\n"
            f"        {sql_str(race['name'])},\n"
            "        'lap_based',\n"
            "        NULL,\n"
            f"        {race['duration']},\n"
            f"        '{race['start']}',  -- America/Detroit\n"
            "        'scheduled'\n"
            "    )"
        )
    lines.append(",\n".join(race_values) + ";")
    lines.append("")
    lines.append(
        "INSERT INTO timing_checkpoints (id, race_id, name, checkpoint_type, distance_from_start_km, location_description, is_active)"
    )
    lines.append("VALUES")
    lines.append(",\n".join(checkpoint_rows) + ";")
    lines.append("")
    lines.append(
        "INSERT INTO categories (id, race_id, name, category_type, age_min, age_max, gender_filter, display_order)"
    )
    lines.append("VALUES")
    lines.append(",\n".join(category_rows) + ";")
    lines.append("")
    lines.append(
        "-- participants.category_id + rfid_tag_uid (migration 04); associations table for multi-tag lookups"
    )
    lines.append(
        "INSERT INTO participants (id, race_id, bib_number, first_name, last_name, gender, age, location, rfid_tag_uid, status, category_id)"
    )
    lines.append("VALUES")

    participant_rows = []
    for pid, race_id, bib, first, last, gender, age, tag_uid, cat_id in participants:
        participant_rows.append(
            "    (\n"
            f"        '{pid}',\n"
            f"        '{race_id}',\n"
            f"        {sql_str(bib)},\n"
            f"        {sql_str(first)},\n"
            f"        {sql_str(last)},\n"
            f"        {sql_str(gender)},\n"
            f"        {age},\n"
            "        'Copper Harbor, MI',\n"
            f"        {sql_str(tag_uid)},\n"
            "        'registered',\n"
            f"        '{cat_id}'\n"
            "    )"
        )
    lines.append(",\n".join(participant_rows) + ";")
    lines.append("")
    lines.append(
        "INSERT INTO rfid_tag_associations (id, participant_id, tag_uid, active)"
    )
    lines.append("VALUES")
    lines.append(",\n".join(association_rows) + ";")
    lines.append("")
    lines.append("COMMIT;")
    lines.append("")

    OUTPUT_SQL.write_text("\n".join(lines), encoding="utf-8")
    print(
        f"Wrote {OUTPUT_SQL} "
        f"({len(races)} races, {len(category_rows)} categories, "
        f"{len(participants)} participants, {len(association_rows)} tag associations)"
    )


if __name__ == "__main__":
    main()

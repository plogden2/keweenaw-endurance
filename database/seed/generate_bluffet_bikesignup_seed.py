#!/usr/bin/env python3
"""Emit All You Can East Bluffet 2026 production seed from BikeSignup roster.

Source: https://www.bikesignup.com/Race/FindARunner/?raceId=181840&perPage=50
Racers are seeded with NULL bib_number (unassigned) and no RFID tags/bib inventory.
Canonical event/race IDs match frontend e2e fixtures + reader paste sheet.

Regenerate: python database/seed/generate_bluffet_bikesignup_seed.py
"""

from __future__ import annotations

import uuid
from pathlib import Path

OUTPUT_SQL = Path(__file__).parent / "03-bluffet-2026-bikesignup.sql"
EVENT_NAME = "All You Can East Bluffet"

EVENT_ID = "1441674d-a011-471a-a601-722b88b117f5"
RACE_12H_ID = "17da3ba1-2e09-4eb1-aeb3-d9dd5b6a394e"
RACE_6H_ID = "209769a1-f723-4f70-ae90-466a46338684"
RACE_KIDS_ID = "0e45ee85-800c-4e1f-a95b-4b92462e790a"

_NS = uuid.UUID(EVENT_ID)

# races.start_time is timestamp-without-tz holding UTC wall time (see RaceService).
# 8:00 AM America/Detroit (EDT, UTC-4) on 2026-08-01 → 12:00 UTC.
START_ALL = "2026-08-01 12:00:00"

# BikeSignup Find-a-Participant (anonymous registrants are not listed).
# (first, last, race_key, skill, gender, age, city, state)
RACERS: list[tuple[str, str, str, str, str, int, str, str]] = [
    ("F.", "Ahrens", "kids", "", "male", 11, "Calumet", "MI"),
    ("T.", "Ahrens", "kids", "", "female", 8, "Calumet", "MI"),
    ("Aunders", "Anderson", "6-hour", "expert", "male", 15, "Calumet", "MI"),
    ("Kyia", "Anderson", "6-hour", "expert", "female", 50, "Calumet", "MI"),
    ("Tony", "Anderson", "6-hour", "intermediate", "male", 32, "Chassell", "MI"),
    ("Lauren", "Atkinson", "6-hour", "intermediate", "female", 23, "Northville", "MI"),
    ("Austin", "Balken", "6-hour", "intermediate", "male", 38, "Burnsville", "MN"),
    ("Josh", "Blum", "6-hour", "expert", "male", 48, "La Crosse", "WI"),
    ("Jason", "Brekhus", "12-hour", "expert", "male", 49, "Minocqua", "WI"),
    ("Adrian", "Brokenshire", "6-hour", "intermediate", "female", 24, "Troy", "MI"),
    ("Benjamin", "Ciavola", "12-hour", "expert", "male", 39, "Houghton", "MI"),
    ("Mael", "Fleury", "12-hour", "expert", "male", 20, "Rochester", "MI"),
    ("Pierre", "Fleury", "6-hour", "intermediate", "male", 49, "Rochester", "MI"),
    ("Casey", "Goethel", "6-hour", "expert", "male", 22, "Houghton", "MI"),
    ("Michael", "Gumbleton", "6-hour", "intermediate", "male", 33, "Grand Rapids", "MI"),
    ("Brandon", "Hajdo-Fernandez", "6-hour", "expert", "male", 41, "Atlantic Mine", "MI"),
    ("Matthew", "Heirtzler", "12-hour", "expert", "male", 21, "Northville", "MI"),
    ("Tyler", "Hosking", "6-hour", "intermediate", "male", 30, "Laurium", "MI"),
    ("Adam", "Kazilsky", "12-hour", "expert", "male", 33, "Duluth", "MN"),
    ("Zane", "Kile", "6-hour", "intermediate", "male", 33, "Plymouth", "MI"),
    ("Amanda", "Kilpela", "6-hour", "intermediate", "female", 39, "Atlantic Mine", "MI"),
    ("Thomas", "Klarr", "6-hour", "intermediate", "male", 33, "Howell", "MI"),
    ("Ian", "Klein", "6-hour", "expert", "male", 14, "Hancock", "MI"),
    ("Maggie", "Klein", "6-hour", "expert", "female", 15, "Hancock", "MI"),
    ("Mark", "Klein", "6-hour", "expert", "male", 44, "Hancock", "MI"),
    ("Josh", "Kocha", "12-hour", "intermediate", "male", 42, "Marquette", "MI"),
    ("M.", "Kocha", "kids", "", "male", 9, "Marquette", "MI"),
    ("Sara", "Kocha", "12-hour", "intermediate", "female", 40, "Marquette", "MI"),
    ("Sabrina", "Loven-Gulick", "6-hour", "intermediate", "female", 34, "Calumet", "MI"),
    ("Lauren", "Luce", "12-hour", "intermediate", "female", 37, "Marquette", "MI"),
    ("Matt", "Luce", "12-hour", "intermediate", "male", 36, "Marquette", "MI"),
    ("Mike", "Macfarland", "6-hour", "expert", "male", 37, "Houghton", "MI"),
    ("Karina", "Madigan", "6-hour", "intermediate", "female", 25, "Calumet", "MI"),
    ("Isabel", "Matthews", "6-hour", "intermediate", "female", 22, "Eden Prairie", "MI"),
    ("Ella", "Merklein", "6-hour", "intermediate", "female", 23, "Hartford", "WI"),
    ("Braden", "Miller", "6-hour", "expert", "male", 21, "Lake Linden", "MI"),
    ("Jack", "Miller", "6-hour", "expert", "male", 24, "Lake Linden", "MI"),
    ("Sean", "Moisan", "6-hour", "expert", "male", 28, "Wheat Ridge", "CO"),
    ("Isaac", "Oldenburg", "12-hour", "expert", "male", 21, "Houghton", "MI"),
    ("Jayden", "Randall", "6-hour", "expert", "female", 16, "Copper Harbor", "MI"),
    ("Lisa", "Randall", "6-hour", "expert", "female", 48, "Copper Harbor", "MI"),
    ("Darrell", "Robinette", "6-hour", "expert", "male", 44, "Dollar Bay", "MI"),
    ("Kendall", "Shoecraft", "6-hour", "expert", "male", 40, "Ypsilanti", "MI"),
    ("Jonas", "Sublett", "6-hour", "expert", "male", 47, "Rushford", "MN"),
    ("Ryan", "Verbrugge", "6-hour", "intermediate", "male", 23, "Eden Prairie", "MN"),
    ("Justus", "Witmer", "6-hour", "intermediate", "male", 34, "Eden Prairie", "MN"),
]


def stable_uuid(name: str) -> str:
    return str(uuid.uuid5(_NS, name))


def sql_str(value: str) -> str:
    return "'" + value.replace("'", "''") + "'"


def main() -> None:
    races = [
        {
            "id": RACE_12H_ID,
            "key": "12-hour",
            "name": "12 Hour",
            "duration": 720,
            "loc": "Bottom of East Bluff (Flo-Rion, Dueling Banjos)",
            "categories": [
                ("Expert Men", "custom", "male", 0),
                ("Expert Women", "custom", "female", 1),
                ("Intermediate Men", "custom", "male", 2),
                ("Intermediate Women", "custom", "female", 3),
            ],
        },
        {
            "id": RACE_6H_ID,
            "key": "6-hour",
            "name": "6 Hour",
            "duration": 360,
            "loc": "Bottom of East Bluff (Flo-Rion, Dueling Banjos)",
            "categories": [
                ("Expert Men", "custom", "male", 0),
                ("Expert Women", "custom", "female", 1),
                ("Intermediate Men", "custom", "male", 2),
                ("Intermediate Women", "custom", "female", 3),
            ],
        },
        {
            "id": RACE_KIDS_ID,
            "key": "kids",
            "name": "90-Minute Kids",
            "duration": 90,
            "loc": "Bottom of East Bluff Campground Rd",
            "categories": [
                ("Men", "male", "male", 0),
                ("Women", "female", "female", 1),
            ],
        },
    ]

    # Preserve finish checkpoint UUIDs used by reader paste sheet / bridge defaults.
    # Kids race key in demo seed was "90-minute-kids"; keep that for checkpoint IDs only.
    checkpoint_key = {
        "12-hour": "12-hour",
        "6-hour": "6-hour",
        "kids": "90-minute-kids",
    }

    cat_id_by_key: dict[tuple[str, str], str] = {}
    checkpoint_rows: list[str] = []
    category_rows: list[str] = []

    for race in races:
        race_id = race["id"]
        ck = checkpoint_key[race["key"]]
        loc = race["loc"]
        start_cp = stable_uuid(f"checkpoint:{ck}:start")
        finish_cp = stable_uuid(f"checkpoint:{ck}:finish")
        checkpoint_rows.append(
            f"    ('{start_cp}', '{race_id}', 'Start Line', 'start', 0.00, {sql_str(loc)}, true)"
        )
        checkpoint_rows.append(
            f"    ('{finish_cp}', '{race_id}', 'Lap Check', 'finish', NULL, "
            f"{sql_str(loc + ' — race trackers')}, true)"
        )
        for name, ctype, gender, order in race["categories"]:
            cat_id = stable_uuid(f"category:{race['key']}:{name}")
            cat_id_by_key[(race["key"], name)] = cat_id
            category_rows.append(
                f"    ('{cat_id}', '{race_id}', {sql_str(name)}, {sql_str(ctype)}, "
                f"NULL, NULL, {sql_str(gender)}, {order})"
            )

    race_id_by_key = {r["key"]: r["id"] for r in races}
    participant_rows: list[str] = []
    for i, (first, last, race_key, skill, gender, age, city, state) in enumerate(RACERS, start=1):
        if race_key == "kids":
            cat_name = "Men" if gender == "male" else "Women"
        else:
            skill_label = "Expert" if skill == "expert" else "Intermediate"
            gender_label = "Men" if gender == "male" else "Women"
            cat_name = f"{skill_label} {gender_label}"
        cat_id = cat_id_by_key[(race_key, cat_name)]
        race_id = race_id_by_key[race_key]
        pid = stable_uuid(f"bikesignup-participant:{i}:{last}:{first}")
        location = f"{city}, {state}"
        participant_rows.append(
            "    (\n"
            f"        '{pid}',\n"
            f"        '{race_id}',\n"
            "        NULL,\n"
            f"        {sql_str(first)},\n"
            f"        {sql_str(last)},\n"
            f"        {sql_str(gender)},\n"
            f"        {age},\n"
            f"        {sql_str(location)},\n"
            "        NULL,\n"
            "        'registered',\n"
            f"        '{cat_id}',\n"
            "        NULL\n"
            "    )"
        )

    assert len(RACERS) == 46, len(RACERS)

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
    bib_subq = (
        "SELECT b.id FROM bibs b\n"
        "    JOIN events e ON b.event_id = e.id\n"
        f"    WHERE {event_filter}"
    )

    lines = [
        "-- All You Can East Bluffet 2026 — BikeSignup production roster",
        "-- Source: https://www.bikesignup.com/Race/FindARunner/?raceId=181840&perPage=50",
        "-- Regenerate: python database/seed/generate_bluffet_bikesignup_seed.py",
        "--",
        f"-- Event {EVENT_ID}; races start {START_ALL} America/Detroit",
        f"-- {len(RACERS)} participants (public list; anonymous registrants omitted)",
        "-- bib_number NULL (unassigned); no bibs / RFID tag associations",
        "-- Categories: Expert/Intermediate × Men/Women (12h/6h); Men/Women (kids)",
        "",
        "BEGIN;",
        "",
        "-- Allow unassigned bibs for race-morning assignment workflow",
        "ALTER TABLE participants ALTER COLUMN bib_number DROP NOT NULL;",
        "",
        "-- Clean prior Bluffet data (order respects FKs)",
        "UPDATE timing_records SET station_id = NULL WHERE station_id IN (",
        "    SELECT rs.id FROM reader_stations rs",
        "    JOIN events e ON rs.event_id = e.id",
        f"    WHERE {event_filter}",
        ");",
        "DELETE FROM reader_stations WHERE event_id IN (",
        "    SELECT e.id FROM events e",
        f"    WHERE {event_filter}",
        ");",
        "DELETE FROM timing_records WHERE participant_id IN (",
        f"    {participant_subq}",
        ");",
        "DELETE FROM rfid_tag_associations WHERE bib_id IN (",
        f"    {bib_subq}",
        ");",
        "DELETE FROM participants WHERE race_id IN (",
        f"    {race_subq}",
        ");",
        "DELETE FROM bibs WHERE event_id IN (",
        "    SELECT e.id FROM events e",
        f"    WHERE {event_filter}",
        ");",
        "DELETE FROM teams WHERE race_id IN (",
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
        f"    '{EVENT_ID}',",
        f"    {sql_str(EVENT_NAME)},",
        "    'Feast on the Copper Harbor Trails Club''s newest event - a brand new endurance enduro at East Bluff Bike Park. Spin the wheel, shred the trails, and push your limits all day long! Expert, intermediate, and kids classes with 6- and 12-hour options plus a 90-minute kids race. Registration via BikeSignup.',",
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
            f"        '{EVENT_ID}',\n"
            f"        {sql_str(race['name'])},\n"
            "        'lap_based',\n"
            "        NULL,\n"
            f"        {race['duration']},\n"
            f"        '{START_ALL}',  -- America/Detroit\n"
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
        "-- participants: bib_number intentionally NULL (unassigned until race morning)"
    )
    lines.append(
        "INSERT INTO participants (id, race_id, bib_number, first_name, last_name, gender, age, location, rfid_tag_uid, status, category_id, team_id)"
    )
    lines.append("VALUES")
    lines.append(",\n".join(participant_rows) + ";")
    lines.append("")
    lines.append("COMMIT;")
    lines.append("")

    OUTPUT_SQL.write_text("\n".join(lines), encoding="utf-8")
    by_race = {k: sum(1 for r in RACERS if r[2] == k) for k in ("12-hour", "6-hour", "kids")}
    print(
        f"Wrote {OUTPUT_SQL} "
        f"({len(races)} races, {len(category_rows)} categories, "
        f"{len(participant_rows)} participants; by race {by_race})"
    )


if __name__ == "__main__":
    main()

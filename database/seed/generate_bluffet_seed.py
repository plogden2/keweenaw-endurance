#!/usr/bin/env python3
"""Emit Bluffet 2026 seed SQL with random UUIDs."""

from __future__ import annotations

import uuid
from pathlib import Path

OUTPUT_SQL = Path(__file__).parent / "03-bluffet-2026.sql"
EVENT_NAME = "All You Can East Bluffet"


def new_uuid() -> str:
    return str(uuid.uuid4())


def sql_str(value: str) -> str:
    return "'" + value.replace("'", "''") + "'"


def main() -> None:
    event_id = new_uuid()
    race_ids = [new_uuid() for _ in range(5)]
    race_names = [
        "12 Hour Expert All You Can East Bluffet",
        "6 Hour Expert All You Can East Bluffet",
        "12 Hour Intermediate All You Can East Bluffet",
        "6 Hour Intermediate All You Can East Bluffet",
        "Kids' Event",
    ]
    race_meta = [
        (None, 720, "2026-08-01 08:00:00"),
        (None, 360, "2026-08-01 08:00:00"),
        (None, 720, "2026-08-01 08:00:00"),
        (None, 360, "2026-08-01 08:00:00"),
        (None, 90, "2026-08-01 15:00:00"),
    ]

    checkpoint_rows = []
    category_rows = []
    for idx, race_id in enumerate(race_ids):
        start_id = new_uuid()
        finish_id = new_uuid()
        loc = (
            "Bottom of East Bluff Campground Rd"
            if idx == 4
            else "Bottom of East Bluff (Flo-Rion, Dueling Banjos)"
        )
        finish_loc = loc + (" — race trackers" if idx != 4 else " — race trackers")
        checkpoint_rows.append(
            f"    ('{start_id}', '{race_id}', 'Start Line', 'start', 0.00, {sql_str(loc)}, true)"
        )
        checkpoint_rows.append(
            f"    ('{finish_id}', '{race_id}', 'Lap Check', 'finish', NULL, {sql_str(finish_loc)}, true)"
        )
        if idx == 4:
            category_rows.append(
                f"    ('{new_uuid()}', '{race_id}', 'Youth', 'age_group', 5, 12, NULL, 0)"
            )
        else:
            for kind, label, ctype, gender in (
                ("overall", "Overall", "overall", "NULL"),
                ("male", "Men", "male", "'male'"),
                ("female", "Women", "female", "'female'"),
            ):
                category_rows.append(
                    f"    ('{new_uuid()}', '{race_id}', {sql_str(label)}, {sql_str(ctype)}, NULL, NULL, {gender}, "
                    f"{0 if kind == 'overall' else 1 if kind == 'male' else 2})"
                )

    event_filter = f"e.name = {sql_str(EVENT_NAME)}"
    lines = [
        "-- All You Can East Bluffet 2026 (2026-08-01, East Bluff Bike Park, Copper Harbor, MI)",
        "-- Source: https://www.copperharbortrails.org/bluffet",
        "-- Regenerate: python database/seed/generate_bluffet_seed.py",
        "",
        "BEGIN;",
        "",
        f"-- Event id: {event_id}",
        "",
        "DELETE FROM categories WHERE race_id IN (",
        "    SELECT r.id FROM races r",
        "    JOIN events e ON r.event_id = e.id",
        f"    WHERE {event_filter}",
        ");",
        "DELETE FROM timing_checkpoints WHERE race_id IN (",
        "    SELECT r.id FROM races r",
        "    JOIN events e ON r.event_id = e.id",
        f"    WHERE {event_filter}",
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
        "    'Feast on the Copper Harbor Trails Club''s newest event - a brand new endurance enduro at East Bluff Bike Park. Spin the wheel, shred the trails, and push your limits all day long! Expert, intermediate, and youth classes with 6- and 12-hour options. Pedal to the top, spin the mountain bike wheel for your trail, punch your race plate, and repeat. Registration at copperharbortrails.org/bluffet.',",
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
    for race_id, name, (distance, duration, start) in zip(race_ids, race_names, race_meta):
        distance_sql = "NULL" if distance is None else f"{distance}"
        race_values.append(
            "    (\n"
            f"        '{race_id}',\n"
            f"        '{event_id}',\n"
            f"        {sql_str(name)},\n"
            "        'lap_based',\n"
            f"        {distance_sql},\n"
            f"        {duration},\n"
            f"        '{start}',\n"
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
    lines.append("COMMIT;")
    lines.append("")

    OUTPUT_SQL.write_text("\n".join(lines), encoding="utf-8")
    print(f"Wrote {OUTPUT_SQL}")


if __name__ == "__main__":
    main()

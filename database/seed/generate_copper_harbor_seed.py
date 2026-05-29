#!/usr/bin/env python3
"""Fetch Copper Harbor Trails Fest 2025 results from RaceResult and emit seed SQL."""

from __future__ import annotations

import json
import re
import sys
import urllib.parse
import urllib.request
from dataclasses import dataclass
from datetime import datetime, timedelta
from pathlib import Path

EVENT_ID = 356809
BASE_URL = f"https://my.raceresult.com/{EVENT_ID}"
SOURCE_DIR = Path(__file__).parent / "source" / "copper-harbor-trails-fest-2025"
OUTPUT_SQL = Path(__file__).parent / "02-example-data.sql"

EVENT_UUID = "11111111-1111-4111-8111-111111111101"
RACE_UUIDS = {
    1: "11111111-1111-4111-8111-111111111102",  # Long XC
    2: "11111111-1111-4111-8111-111111111103",  # Medium XC
    3: "11111111-1111-4111-8111-111111111104",  # Short XC
}
CHECKPOINT_UUIDS = {
    (1, "start"): "11111111-1111-4111-8111-111111111201",
    (1, "finish"): "11111111-1111-4111-8111-111111111202",
    (2, "start"): "11111111-1111-4111-8111-111111111203",
    (2, "finish"): "11111111-1111-4111-8111-111111111204",
    (3, "start"): "11111111-1111-4111-8111-111111111205",
    (3, "finish"): "11111111-1111-4111-8111-111111111206",
}
CATEGORY_UUIDS = {
    (1, "overall"): "11111111-1111-4111-8111-111111111301",
    (1, "male"): "11111111-1111-4111-8111-111111111302",
    (1, "female"): "11111111-1111-4111-8111-111111111303",
    (2, "overall"): "11111111-1111-4111-8111-111111111304",
    (2, "male"): "11111111-1111-4111-8111-111111111305",
    (2, "female"): "11111111-1111-4111-8111-111111111306",
    (3, "overall"): "11111111-1111-4111-8111-111111111307",
    (3, "male"): "11111111-1111-4111-8111-111111111308",
    (3, "female"): "11111111-1111-4111-8111-111111111309",
}

RACE_META = {
    1: {
        "name": "Long XC Mountain Bike Race",
        "distance_km": 46.67,
        "start": "2025-08-30 10:00:00",
    },
    2: {
        "name": "Medium XC Mountain Bike Race",
        "distance_km": 24.14,
        "start": "2025-08-30 10:00:00",
    },
    3: {
        "name": "Short XC Mountain Bike Race",
        "distance_km": 11.27,
        "start": "2025-08-30 10:00:00",
    },
}

CONTEST_NAMES = {
    1: "Long XC Mountain Bike Race",
    2: "Medium XC Mountain Bike Race",
    3: "Short XC Mountain Bike Race",
}


@dataclass
class ParticipantRow:
    contest: int
    bib: str
    rr_id: str
    rank: str
    display_name: str
    city_state: str
    age_group: str
    finish_time: str | None
    status: str
    gender: str | None
    age: int | None
    first_name: str
    last_name: str


def fetch_json(url: str, params: dict | None = None) -> dict:
    if params:
        url = f"{url}?{urllib.parse.urlencode(params)}"
    request = urllib.request.Request(url, headers={"User-Agent": "Mozilla/5.0"})
    with urllib.request.urlopen(request, timeout=60) as response:
        return json.loads(response.read().decode("utf-8"))


def parse_elapsed_seconds(value: str) -> float | None:
    if not value or value.upper() in {"DNF", "DNS", "DSQ"}:
        return None
    match = re.match(r"(\d+):(\d+):(\d+(?:\.\d+)?)", value)
    if not match:
        return None
    hours, minutes, seconds = match.groups()
    return int(hours) * 3600 + int(minutes) * 60 + float(seconds)


def parse_gender(age_group: str) -> str | None:
    code = age_group.strip().upper()
    if code.startswith("M"):
        return "male"
    if code.startswith("F"):
        return "female"
    return None


def parse_age(age_group: str) -> int | None:
    match = re.search(r"[MF](\d+)\s*-\s*(\d+)", age_group, re.IGNORECASE)
    if match:
        low, high = int(match.group(1)), int(match.group(2))
        return (low + high) // 2
    if re.search(r"60\+", age_group):
        return 62
    return None


def split_name(display_name: str) -> tuple[str, str]:
    parts = display_name.strip().split()
    if not parts:
        return "Unknown", "Rider"
    if len(parts) == 1:
        return parts[0], "Rider"
    return parts[0], " ".join(parts[1:])


def sql_str(value: str | None) -> str:
    if value is None:
        return "NULL"
    return "'" + value.replace("'", "''") + "'"


def participant_uuid(index: int) -> str:
    return f"11111111-1111-4111-8111-111111111{400 + index}"


def timing_uuid(index: int) -> str:
    return f"11111111-1111-4111-8111-111111111{500 + index}"


def load_or_fetch_results() -> tuple[dict, dict[int, list[ParticipantRow]]]:
    SOURCE_DIR.mkdir(parents=True, exist_ok=True)

    config_path = SOURCE_DIR / "config.json"
    if config_path.exists():
        config = json.loads(config_path.read_text(encoding="utf-8"))
    else:
        config = fetch_json(f"{BASE_URL}/results/config", {"lang": "en"})
        config_path.write_text(json.dumps(config, indent=2), encoding="utf-8")

    key = config["key"]
    list_name = "Result Lists|Overall Results"
    participants_by_contest: dict[int, list[ParticipantRow]] = {}

    for contest in (1, 2, 3):
        list_path = SOURCE_DIR / f"contest-{contest}-overall.json"
        if list_path.exists():
            payload = json.loads(list_path.read_text(encoding="utf-8"))
        else:
            payload = fetch_json(
                f"{BASE_URL}/results/list",
                {
                    "key": key,
                    "listname": list_name,
                    "page": "results",
                    "contest": contest,
                    "r": "page",
                    "l": 0,
                },
            )
            list_path.write_text(json.dumps(payload, indent=2), encoding="utf-8")

        group_key = next(iter(payload["data"]))
        rows: list[ParticipantRow] = []
        for raw in payload["data"][group_key]:
            bib, rr_id, rank, display_name, city_state, age_group, finish_time, _mph = raw
            status = "dnf" if finish_time and finish_time.upper() == "DNF" else "finished"
            rows.append(
                ParticipantRow(
                    contest=contest,
                    bib=bib,
                    rr_id=rr_id,
                    rank=rank,
                    display_name=display_name,
                    city_state=city_state,
                    age_group=age_group,
                    finish_time=finish_time if status == "finished" else None,
                    status=status,
                    gender=parse_gender(age_group),
                    age=parse_age(age_group),
                    first_name=split_name(display_name)[0],
                    last_name=split_name(display_name)[1],
                )
            )
        participants_by_contest[contest] = rows

    return config, participants_by_contest


def build_sql(participants_by_contest: dict[int, list[ParticipantRow]]) -> str:
    race_ids = list(RACE_UUIDS.values())
    race_id_sql = ",\n        ".join(f"'{race_id}'" for race_id in race_ids)

    lines: list[str] = [
        "-- Real race results seeded from RaceResult event 356809.",
        "-- 2025 Copper Harbor Trails Fest - XC MTB (2025-08-30, Copper Harbor, MI)",
        "-- Source: https://my.raceresult.com/356809/",
        "-- Regenerate: python database/seed/generate_copper_harbor_seed.py",
        "",
        "BEGIN;",
        "",
        "-- Seed entity IDs (fixed for repeatable runs)",
        f"-- Event:  {EVENT_UUID}",
        "-- Races:  ...102 (long), ...103 (medium), ...104 (short)",
        "",
        "DELETE FROM timing_records WHERE participant_id IN (",
        "    SELECT id FROM participants WHERE race_id IN (",
        f"        {race_id_sql}",
        "    )",
        ");",
        "DELETE FROM categories WHERE race_id IN (",
        f"    {race_id_sql}",
        ");",
        "DELETE FROM timing_checkpoints WHERE race_id IN (",
        f"    {race_id_sql}",
        ");",
        "DELETE FROM participants WHERE race_id IN (",
        f"    {race_id_sql}",
        ");",
        "DELETE FROM races WHERE id IN (",
        f"    {race_id_sql}",
        ");",
        f"DELETE FROM events WHERE id = '{EVENT_UUID}';",
        "",
        "INSERT INTO events (id, name, description, event_date, location, website_url, status)",
        "VALUES (",
        f"    '{EVENT_UUID}',",
        "    '2025 Copper Harbor Trails Fest - XC MTB',",
        "    'Cross-country mountain bike races at the Copper Harbor Trails Fest (Superior Timing).',",
        "    '2025-08-30',",
        "    'Copper Harbor, MI',",
        "    'https://my.raceresult.com/356809/',",
        "    'completed'",
        ");",
        "",
        "INSERT INTO races (id, event_id, name, race_type, distance_km, duration_minutes, start_time, status)",
        "VALUES",
    ]

    race_values = []
    for contest, race_id in RACE_UUIDS.items():
        meta = RACE_META[contest]
        race_values.append(
            "    (\n"
            f"        '{race_id}',\n"
            f"        '{EVENT_UUID}',\n"
            f"        {sql_str(meta['name'])},\n"
            "        'time_based',\n"
            f"        {meta['distance_km']:.2f},\n"
            "        NULL,\n"
            f"        '{meta['start']}',\n"
            "        'finished'\n"
            "    )"
        )
    lines.append(",\n".join(race_values) + ";")
    lines.append("")
    lines.append(
        "INSERT INTO timing_checkpoints (id, race_id, name, checkpoint_type, distance_from_start_km, location_description, is_active)"
    )
    lines.append("VALUES")

    checkpoint_values = []
    for contest in (1, 2, 3):
        race_id = RACE_UUIDS[contest]
        distance = RACE_META[contest]["distance_km"]
        checkpoint_values.append(
            f"    ('{CHECKPOINT_UUIDS[(contest, 'start')]}', '{race_id}', 'Start Line', 'start', 0.00, 'Downtown Copper Harbor', true)"
        )
        checkpoint_values.append(
            f"    ('{CHECKPOINT_UUIDS[(contest, 'finish')]}', '{race_id}', 'Finish Line', 'finish', {distance:.2f}, 'Downtown Copper Harbor', true)"
        )
    lines.append(",\n".join(checkpoint_values) + ";")
    lines.append("")
    lines.append(
        "INSERT INTO categories (id, race_id, name, category_type, age_min, age_max, gender_filter, display_order)"
    )
    lines.append("VALUES")

    category_values = []
    for contest in (1, 2, 3):
        race_id = RACE_UUIDS[contest]
        category_values.append(
            f"    ('{CATEGORY_UUIDS[(contest, 'overall')]}', '{race_id}', 'Overall', 'overall', NULL, NULL, NULL, 0)"
        )
        category_values.append(
            f"    ('{CATEGORY_UUIDS[(contest, 'male')]}', '{race_id}', 'Men', 'male', NULL, NULL, 'male', 1)"
        )
        category_values.append(
            f"    ('{CATEGORY_UUIDS[(contest, 'female')]}', '{race_id}', 'Women', 'female', NULL, NULL, 'female', 2)"
        )
    lines.append(",\n".join(category_values) + ";")
    lines.append("")
    lines.append(
        "INSERT INTO participants (id, race_id, bib_number, first_name, last_name, gender, age, location, rfid_tag_uid, status)"
    )
    lines.append("VALUES")

    participant_values = []
    participant_index = 1
    participant_ids: list[tuple[ParticipantRow, str]] = []

    for contest in (1, 2, 3):
        race_id = RACE_UUIDS[contest]
        for row in participants_by_contest[contest]:
            pid = participant_uuid(participant_index)
            participant_ids.append((row, pid))
            participant_values.append(
                "    ("
                f"{sql_str(pid)}, "
                f"{sql_str(race_id)}, "
                f"{sql_str(row.bib)}, "
                f"{sql_str(row.first_name)}, "
                f"{sql_str(row.last_name)}, "
                f"{sql_str(row.gender)}, "
                f"{row.age if row.age is not None else 'NULL'}, "
                f"{sql_str(row.city_state)}, "
                f"{sql_str(f'RR{EVENT_ID}-{row.rr_id}')}, "
                f"{sql_str(row.status)}"
                ")"
            )
            participant_index += 1

    lines.append(",\n".join(participant_values) + ";")
    lines.append("")
    lines.append(
        "INSERT INTO timing_records (id, participant_id, checkpoint_id, timestamp, local_timestamp, device_id, sync_status)"
    )
    lines.append("VALUES")

    timing_values = []
    timing_index = 1

    for row, participant_id in participant_ids:
        contest = row.contest
        start_checkpoint = CHECKPOINT_UUIDS[(contest, "start")]
        finish_checkpoint = CHECKPOINT_UUIDS[(contest, "finish")]
        start_dt = datetime.strptime(RACE_META[contest]["start"], "%Y-%m-%d %H:%M:%S")
        start_ts = start_dt.strftime("%Y-%m-%d %H:%M:%S")

        timing_values.append(
            "    ("
            f"'{timing_uuid(timing_index)}', '{participant_id}', '{start_checkpoint}', "
            f"'{start_ts}', '{start_ts}', 'RR-START', 'synced'"
            ")"
        )
        timing_index += 1

        elapsed = parse_elapsed_seconds(row.finish_time or "")
        if elapsed is not None:
            finish_dt = start_dt + timedelta(seconds=elapsed)
            finish_ts = finish_dt.strftime("%Y-%m-%d %H:%M:%S")
            timing_values.append(
                "    ("
                f"'{timing_uuid(timing_index)}', '{participant_id}', '{finish_checkpoint}', "
                f"'{finish_ts}', '{finish_ts}', 'RR-FINISH', 'synced'"
                ")"
            )
            timing_index += 1

    lines.append(",\n".join(timing_values) + ";")
    lines.append("")
    lines.append("COMMIT;")
    lines.append("")
    return "\n".join(lines)


def main() -> int:
    fetch_only = "--fetch-only" in sys.argv
    _, participants_by_contest = load_or_fetch_results()
    total = sum(len(rows) for rows in participants_by_contest.values())
    print(f"Loaded {total} participants across {len(participants_by_contest)} contests")

    if fetch_only:
        return 0

    sql = build_sql(participants_by_contest)
    OUTPUT_SQL.write_text(sql, encoding="utf-8")
    print(f"Wrote {OUTPUT_SQL}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())

# CSV Race Export / Import Contract

**Purpose**: Disaster recovery when a laptop fails — export from healthy source (hosted or another station), import on replacement laptop, continue timing.

## Semantics

- **Live snapshot**: Each station continuously writes/updates a current CSV file for the configured event whenever relevant data changes (taps, karaoke, racers, tags, races, etc.). Recovery MUST NOT require a separate manual export action.
- **Copy (optional)**: Staff may copy/download the live file for USB/cloud backup; this is the same live snapshot, not a distinct export pipeline.
- **Import**: For the target event id (or create-from-file), **replace** local event-scoped data with CSV contents (authoritative restore). Does not delete unrelated events. After import, live CSV writing resumes on the new station.

## File format

- UTF-8, comma-separated, header row required
- Multiple logical sections separated by a sentinel row: `#SECTION,<name>`
- Timestamps ISO-8601 with offset

### SECTION `event`
| Column | Notes |
|--------|--------|
| id | UUID |
| name | |
| event_date | YYYY-MM-DD |
| location | |
| status | |

### SECTION `races`
| Column | Notes |
|--------|--------|
| id, event_id, name, race_type, duration_minutes, start_time, status | |

### SECTION `categories`
| Column | Notes |
|--------|--------|
| id, race_id, name, category_type, gender_filter, display_order | |

### SECTION `participants`
| Column | Notes |
|--------|--------|
| id, race_id, bib_number, first_name, last_name, gender, status, category_id | |

### SECTION `tags`
| Column | Notes |
|--------|--------|
| id, participant_id, tag_uid, created_at | |

### SECTION `checkpoints`
| Column | Notes |
|--------|--------|
| id, race_id, name, checkpoint_type, distance_from_start_km, is_active | |

### SECTION `timing_records`
| Column | Notes |
|--------|--------|
| id, participant_id, checkpoint_id, timestamp, local_timestamp, device_id, sync_status, record_type, source_lap_id, station_id | |

## Round-trip requirement

After continuous live-snapshot updates → copy file → import on empty station local DB, lap counts and tag lookups must match source for all participants in the file (SC-006). Live file MUST keep updating through offline periods on the source station.

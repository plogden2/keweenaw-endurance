# Race Teams Design

**Date:** 2026-07-26  
**Status:** Approved via advisor (user directed: no further clarifying questions; implement to completion)

## Goal

Add optional per-race teams of multiple racers. Team score is the average of member lap totals (RFID + karaoke). Expose team leaderboards, scan-popup team placement, race-flow team filters, PIN-gated team management, CSV round-trip, and seed a few teams into the Bluffet dress rehearsal.

## Architecture

**Approach A:** `teams` table + nullable `participants.team_id` FK. Compute averages on read (no score cache). Categories remain skill×gender; teams are a separate dimension.

### Schema

```text
teams:
  id UUID PK
  race_id UUID NOT NULL → races.id ON DELETE CASCADE
  name VARCHAR(255) NOT NULL
  display_order INT NOT NULL DEFAULT 0
  created_at TIMESTAMP
  UNIQUE(race_id, name)

participants.team_id UUID NULL → teams.id ON DELETE SET NULL
```

### Membership rules

- Optional; at most one team per racer.
- Min roster for a scored team: 2. Max soft-cap: 12.
- Creating a named team with zero members is allowed; leaderboard omits teams with &lt;2 members.
- `PUT .../members` with 1 id is rejected; 0 clears membership; 2–12 assigns (replaces roster).

### Scoring

```
avg_laps = sum(member scored laps) / roster_count
```

- Scored laps = `rfid_lap` + `karaoke_bonus` (same as individuals).
- Denominator is always current roster size (DNS/DNF/no-show stay in; usually 0 laps).
- Sort: `avg_laps` DESC → mean of members’ last RFID lap ASC (zero-lap members use race end = `start_time + duration`) → team name ASC (case-insensitive).
- Unteamed racers omitted from team board.

### API

```
GET/POST   /api/races/:id/teams
GET/PUT/DELETE /api/teams/:id
PUT        /api/teams/:id/members   { "participant_ids": ["..."] }
```

Participant JSON includes `team_id`. Live race payload adds `leaderboard_teams[]`:

```json
{
  "place": 1,
  "team_id": "...",
  "name": "East Bluff A",
  "avg_laps": 12.5,
  "member_count": 4,
  "mean_last_lap_at": "..."
}
```

Scan result (when teamed) adds: `team_id`, `team_name`, `team_placement`, `team_avg_laps`.

### UI

- Event live / race results: **Individuals | Teams** toggle (default Individuals).
- Race flow chart: Team filter (including “No team”).
- Racers (PIN): Teams list + assign dropdown / member editor.
- Lap popup: secondary line with team name · place · avg when teamed.

### CSV

- `#SECTION,teams`: `id,race_id,name,display_order`
- `participants` gains `team_id` column (nullable). Import order: teams before participants.

### Dress rehearsal

- 12 Hour race only: 4 teams × 4 members (“East Bluff A–D”).
- Stable uuid5 team IDs; assign first 16 twelve-hour participants in order.
- Assert team board present, race-flow team filter, and one teamed scan popup field.

## Out of scope

Cross-race teams, multi-team membership, team divisions/colors, cached score tables.

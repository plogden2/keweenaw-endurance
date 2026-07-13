# Feature Specification: RFID Race Scanner

**Feature Branch**: `002-rfid-race-scanner`

**Created**: 2026-07-12

**Status**: Ready for Implementation

**Prototypes**: Approved 2026-07-12 — `frontend/prototypes/002-rfid-race-scanner/` (see README decisions)

**Input**: User description: "Implement the full RFID scanner logic for a race. This should fully support the proxmark3 reader and writer. It should allow for easy entry/deletion for a race. There should be a Racers page for writing the chip for each racer. This should have a searchable list of all racers and a way to add a racer. It should be possible to adjust the bib number for each racer (default sequential). It should also be possible to program rfid tags for each racer with their racer uuid. It should be possible to program multiple tags in case one gets lost. When the laptop is configured as a reader for the race, it should always be able to read tags regardless of what page of the site the user is on. When a user taps their tag, a new lap with time is recorded, a popup with the racer name, current placement, and number of laps is displayed, and the mario kart new lap sound is played. There should be a 1 minute cooldown before the same racer can scan again. The default page once the race starts should be the live race flow & leaderboard. All taps and data should be stored in a db locally and synced to the online hosted database. If internet connection is lost, the reader station must keep functioning with current data locally. If the laptop fails, a new laptop should be able to load the database data in a csv and continue functioning. It should be possible to have an arbitrary number of readers and laptops. Start by implementing full e2e tests covering all use cases for this. Seed a demo race for the All You Can East Bluffet 2026 on August 1st. The race is a lap format with 12 hour, 6 hour, and 90-minute-kids races. There is further intermediate and advanced categories for the 12 hour and 6 hour races. All races are further broken down into men and women categories. The 6 and 12 hour races start at 8 am and the 90 minute race starts at 3pm. Seed 100 total racers. This lap has a karaoke bonus lap feature, where the racer will report they sang a song to the volunteer stationed at the reader and the volunteer will manually add that bonus lap after that racer scans their normal lap, so this should be an easy single click after the racer scans."

## Clarifications

### Session 2026-07-12

- Q: Reader station scope — one race, whole event, or multi-select? → A: One event per reader; station accepts taps for any racer in any active race at that event.
- Q: What do multiple readers represent? → A: Configurable either as equivalent finish stations or as ordered checkpoints; default is finish station (+1 lap per valid tap).
- Q: Who can operate race-day controls? → A: PIN login (1738) for management; live leaderboard is public/read-only without login.
- Q: When do RFID taps count as race laps? → A: Pre-start taps are test-only (no lap); counting starts when that racer’s race is active. A countdown to start MUST be displayed pre-start.
- Q: Lost / replaced RFID tags — revoke old tags? → A: All programmed tags stay active forever (no revoke in v1).
- Q: CSV recovery workflow? → A: Maintain a live current CSV on each station at all times (auto-updated); no manual export required for offline/failure continuity; import that file on a replacement laptop.
- Q: Live view & Racers search UX? → A: Debounced live search (no button). Live tabs 12h/6h/90m; overlap chart for concurrent races; default combined overall with category colors+legend; fullscreen rotator cycles flow+leaderboard side-by-side for active races.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Live RFID Lap Timing at Reader Station (Priority: P1)

A volunteer configures a laptop as a reader for an event. Racers in any active race under that event tap their RFID tags at the station. Each valid tap records a timed lap on the racer’s race, shows a confirmation popup with the racer’s name, current overall placement (color-coded by category), and lap count, plays the Mario Kart new-lap sound (no on-screen “playing sound” text), and updates the live leaderboard. The same racer cannot record another RFID lap within a 1-minute cooldown. Reading continues on every page of the site while the laptop remains configured as a reader. Before a race starts, the live race flow displays a countdown to that race’s start time; taps in that period are test reads only. Live view provides tabs for 12 Hour, 6 Hour, and 90 Minute, an overlap chart for concurrent races, combined overall boards with category legend, and a fullscreen rotator cycling flow + leaderboard side-by-side for active races. When any race at the event is running, opening the app lands on the live race flow and leaderboard by default.

**Why this priority**: Core race-day value — without reliable live lap recording and feedback, the timing system cannot run an event.

**Independent Test**: With a seeded race and programmed tags, configure a station as a reader, tap tags, verify lap records, popup content, sound, cooldown, overall leaderboard, tabs/overlap/rotator, and that reading works while navigating away from the live page.

**Acceptance Scenarios**:

1. **Given** a laptop is configured as a reader for an event with at least one active race, **When** a racer enrolled in an active race under that event taps a programmed tag, **Then** a lap is recorded on that racer’s race with the tap time, a popup shows the racer name, current overall placement, and number of laps, and the Mario Kart new-lap sound plays (without a “playing sound” label).
2. **Given** a racer just recorded a lap, **When** the same racer’s tag is tapped again within 1 minute, **Then** no new lap is recorded and the volunteer is informed that the racer is in cooldown.
3. **Given** a racer is in cooldown, **When** 1 minute has elapsed since their last RFID lap, **Then** their next tap records a new lap normally.
4. **Given** a laptop is configured as a reader, **When** the volunteer navigates to any other page in the site, **Then** tag taps continue to be detected and processed without reconfiguring the reader.
5. **Given** at least one race at the configured event is started/active, **When** a user opens the application on a reader or race-day view, **Then** the default landing view is the live race flow and leaderboard (covering active races at the event).
6. **Given** a valid lap was just recorded, **When** the live leaderboard is viewed, **Then** placements and lap counts reflect the new tap without requiring a manual refresh beyond normal live updates.
7. **Given** a laptop is configured as a reader for an event that has both 12-hour and 6-hour races active, **When** racers from each duration tap at the same station, **Then** each lap is attributed to the correct race without switching reader configuration.
8. **Given** a racer’s race is not yet started (still scheduled), **When** their tag is tapped at a configured reader, **Then** the system identifies the racer as a test read and does not record a race lap.
9. **Given** a racer’s race has been started/active, **When** their tag is tapped outside cooldown, **Then** a race lap is recorded normally.
10. **Given** a race at the configured event has a future start time and is not yet started, **When** staff view the live race flow, **Then** a countdown to that race’s start is displayed.
11. **Given** multiple races at the event have different start times (e.g. adults at 8:00 AM, kids at 3:00 PM), **When** the live race flow is viewed, **Then** countdowns (or clear next-start timing) are available for races that have not yet started.
12. **Given** the live event view is open, **When** the user selects race tabs, **Then** they can switch among 12 Hour, 6 Hour, and 90 Minute views.
13. **Given** 12 Hour and 6 Hour are both running, **When** the user opens the overlap view, **Then** both races are visible on a single chart.
14. **Given** one or more races are active, **When** fullscreen rotate is enabled, **Then** the display cycles race flow and leaderboard side-by-side for each active race.
15. **Given** any live leaderboard or scan popup placement, **When** no category filter is applied, **Then** the default is combined overall with category color coding and a legend.

---

### User Story 2 - Karaoke Bonus Lap After Scan (Priority: P1)

After a racer completes a normal RFID lap scan and reports that they sang a song, the volunteer at the reader can add a karaoke bonus lap with a single click on the scan confirmation UI. The bonus lap is attributed to that racer and counted toward their total laps/placement according to race rules.

**Why this priority**: Event-specific scoring mechanic required for All You Can East Bluffet; must be fast for volunteers under race-day pressure.

**Independent Test**: Record a normal lap for a racer, click the single karaoke bonus control on the scan popup, verify an additional bonus lap is stored and reflected in lap count/leaderboard.

**Acceptance Scenarios**:

1. **Given** a racer just completed a successful RFID lap scan and the confirmation popup is visible, **When** the volunteer clicks the karaoke bonus lap control once, **Then** one bonus lap is added for that racer without requiring another tag tap.
2. **Given** the confirmation popup for a scan is still available, **When** the volunteer has already added a karaoke bonus for that scan, **Then** a second accidental click does not silently create duplicate bonus laps for the same scan (the action is clearly one-shot or confirmable).
3. **Given** a karaoke bonus lap was added, **When** the leaderboard is viewed, **Then** the racer’s lap total includes the bonus lap and placement updates accordingly.
4. **Given** no recent successful scan for a racer is selected, **When** a volunteer looks for karaoke bonus, **Then** bonus addition is only offered in the context of a just-completed scan (not as an arbitrary free-form action disconnected from a scan).

---

### User Story 3 - Racers Page: Search, Add, Bibs, and Tag Programming (Priority: P1)

Race staff use a Racers page to manage participants for a race: search the full racer list, add a racer, adjust bib numbers (defaulting to sequential assignment), and program one or more Proxmark3-compatible RFID tags with the racer’s unique identifier so lost tags can be replaced without changing identity.

**Why this priority**: Without enrollment and tag programming, live scanning cannot attribute laps to people.

**Independent Test**: Open Racers for a race, add/search racers, change bibs, program multiple tags for one racer, and verify those tags all resolve to the same racer on scan.

**Acceptance Scenarios**:

1. **Given** a race exists, **When** staff open the Racers page, **Then** they see a searchable list of all racers in that race.
2. **Given** the Racers page is open, **When** staff add a new racer with required identity details, **Then** the racer appears in the list with a unique racer identifier and a default sequential bib number.
3. **Given** a racer exists, **When** staff change the bib number to a valid unused value for that race, **Then** the new bib is saved and shown in lists and race-day displays.
4. **Given** a racer is selected for tag programming, **When** staff write a tag via the Proxmark3 writer, **Then** the tag stores/associates the racer’s unique identifier and subsequent reads identify that racer.
5. **Given** a racer already has one programmed tag, **When** staff program an additional tag for the same racer, **Then** both tags identify the same racer and either can record laps (previous tags are not deactivated).
6. **Given** staff type in the Racers search field, **When** the debounce interval elapses, **Then** the list filters immediately without requiring a Search button click.

---

### User Story 4 - Easy Race Entry and Deletion (Priority: P2)

Organizers can create a race (or race day event with its duration-based races) and delete a race they no longer need, without complex multi-step administration.

**Why this priority**: Needed for setup and cleanup, but secondary to race-day timing once an event exists.

**Independent Test**: Create a race with lap-format settings, verify it appears for management, delete it, and verify it and its dependent race-day data are removed from active use.

**Acceptance Scenarios**:

1. **Given** an organizer is managing races, **When** they create a new race with name, date, and lap-format timing parameters, **Then** the race is available for racer management and reader configuration.
2. **Given** a race exists and is not required, **When** the organizer deletes it with confirmation, **Then** the race is removed from the active race list and racers in that race are no longer accepted for lap timing (event-level readers continue for remaining races).
3. **Given** deletion is requested, **When** the organizer has not confirmed, **Then** the race is not deleted.
4. **Given** a user has not entered the organizer PIN, **When** they attempt race create/delete or other management actions, **Then** access is denied until PIN `1738` is entered successfully.
5. **Given** a user has not entered the organizer PIN, **When** they open the live race flow and leaderboard, **Then** they can view it without logging in.

---

### User Story 5 - Local Persistence, Online Sync, and Offline Continuity (Priority: P1)

Every tap and race-management change is stored in a local database on the station and synced to the online hosted database when connectivity allows. If the internet drops, the reader keeps recording laps and updating local leaderboard state from local data. When connectivity returns, pending local changes sync to the hosted database.

**Why this priority**: Keweenaw race venues may have unreliable connectivity; stopping timing offline is unacceptable.

**Independent Test**: Record laps online, disconnect network, continue recording, reconnect, and verify all local taps appear in the hosted dataset without loss or silent overwrite of distinct events.

**Acceptance Scenarios**:

1. **Given** a reader station has connectivity, **When** a lap is recorded, **Then** it is stored locally and synced to the online hosted database.
2. **Given** the station loses internet connectivity, **When** racers continue tapping, **Then** laps are still recorded locally, popups and sound still work, and the local leaderboard continues to update.
3. **Given** the station was offline and recorded taps, **When** connectivity is restored, **Then** pending local taps and related changes sync to the online hosted database.
4. **Given** sync is in progress or pending, **When** staff view station status, **Then** they can tell whether the station is online, offline, or has unsynced local data.

---

### User Story 6 - CSV Disaster Recovery Across Laptops (Priority: P2)

Each reader/management station continuously maintains a **live current CSV snapshot** of event race data (racers, tags, laps, etc.) on local disk, updated whenever data changes—no manual export step required. If the network is lost, the live CSV keeps updating locally. If a laptop fails, staff copy that live CSV (or one from another healthy station) onto a replacement laptop and import so timing continues with current race state.

**Why this priority**: Hardware failure and offline periods must not leave staff without a recoverable snapshot; requiring a manual export before failure would interrupt recovery.

**Independent Test**: Record laps without clicking export; confirm the live CSV file updates; copy it to a fresh station; import; continue scanning with prior counts intact—including after a simulated offline period.

**Acceptance Scenarios**:

1. **Given** a station is recording race data, **When** laps or racer/tag changes occur, **Then** a live CSV snapshot on that station is updated automatically without a separate export action.
2. **Given** the station loses network connectivity, **When** taps continue, **Then** the live CSV continues to update locally with no interruption and no need to “export before going offline.”
3. **Given** a replacement laptop with the application, **When** staff import the live CSV from the failed or healthy station, **Then** racers, tag associations, and prior laps are available locally for continued operation.
4. **Given** import completed, **When** the laptop is configured as a reader for that event, **Then** new taps append correctly and live CSV writing resumes automatically.

---

### User Story 7 - Multiple Concurrent Reader Stations (Priority: P2)

An event may run an arbitrary number of reader laptops simultaneously. By default each station is a **finish station**: a valid tap adds one lap. Staff may instead configure a station as a **checkpoint** on an ordered course; in that mode, progressing toward a completed lap follows the expected checkpoint sequence. Shared race state converges through sync so leaderboards remain consistent across stations once data is exchanged.

**Why this priority**: Scale and redundancy for larger events; builds on single-station timing.

**Independent Test**: Run two or more finish-mode readers against the same event, record taps on each, sync, and verify all laps appear with cooldown respected once stations share recent taps. Separately configure a checkpoint-mode station and verify out-of-order taps do not incorrectly complete a lap.

**Acceptance Scenarios**:

1. **Given** multiple laptops are configured as finish-mode readers for the same event, **When** different racers tap at different stations, **Then** each valid tap is recorded as +1 lap on the correct race and contributes to shared results after sync.
2. **Given** a racer recorded a lap at finish station A within the last minute, **When** that racer taps at finish station B after stations have shared that recent tap (via sync or shared online state), **Then** station B enforces the same 1-minute cooldown.
3. **Given** stations were briefly isolated, **When** they sync, **Then** distinct taps from each station are preserved and obvious duplicates within the cooldown window for the same racer are not double-counted.
4. **Given** staff configure a new reader for an event without changing mode, **When** the station starts reading, **Then** it operates as a finish station (default).
5. **Given** a station is configured as a checkpoint in an ordered sequence, **When** a racer taps that checkpoint out of the expected order, **Then** the system does not award a completed lap from that tap alone and provides clear feedback that the tap was out of sequence (or otherwise not yet a full lap, per checkpoint rules).

---

### User Story 8 - Demo Seed: All You Can East Bluffet 2026 (Priority: P2)

The system ships with a seeded demo event “All You Can East Bluffet” on August 1, 2026, including three lap-format races, category structure, start times, and 100 racers so staff can demonstrate and test the full scanner workflow end-to-end.

**Why this priority**: Enables demos and comprehensive testing without manual event setup; supports TDD/e2e coverage of real event shape.

**Independent Test**: Load the demo seed and verify event metadata, race durations/start times, categories, and racer count; run scanner flows against seeded racers.

**Acceptance Scenarios**:

1. **Given** the demo seed is loaded, **When** staff open the event list, **Then** “All You Can East Bluffet” dated August 1, 2026 is present.
2. **Given** the demo event, **When** its races are inspected, **Then** there are three lap-format races: 12-hour, 6-hour, and 90-minute kids, with 6-hour and 12-hour starting at 8:00 AM and the 90-minute kids race starting at 3:00 PM (event-local time).
3. **Given** the 12-hour and 6-hour races, **When** categories are inspected, **Then** each includes Intermediate and Advanced, each further split into Men and Women (four categories per duration race).
4. **Given** the 90-minute kids race, **When** categories are inspected, **Then** it includes Men and Women categories (no Intermediate/Advanced split).
5. **Given** the demo seed, **When** racers are counted across the event, **Then** there are 100 racers total assigned across the races/categories.

---

### User Story 9 - End-to-End Test Coverage Before Delivery (Priority: P1)

Before the feature is considered complete, a full suite of end-to-end tests covers all user stories and edge cases in this specification (reader configuration, continuous background reading, lap recording, popup and sound, cooldown, karaoke bonus, racers CRUD/search/bibs, multi-tag programming, race create/delete, offline operation, sync, CSV export/import, multi-station behavior, and demo seed structure).

**Why this priority**: Project constitution and explicit product requirement: prove behavior with failing-then-passing e2e coverage before shipping scanner logic.

**Independent Test**: Run the e2e suite in isolation and confirm each acceptance scenario above has corresponding automated coverage (or an explicitly documented manual-only exception only where hardware cannot be simulated — with a simulated Proxmark3 path preferred).

**Acceptance Scenarios**:

1. **Given** the feature test suite, **When** it is executed in a clean environment with the demo seed, **Then** all automated e2e cases for the scenarios in this spec pass.
2. **Given** a new use case is listed in this spec, **When** implementation is proposed, **Then** an e2e test for that use case exists (or is updated) as part of delivery.

---

### Edge Cases

- Unknown or unprogrammed tag tapped at a reader: no lap recorded; volunteer sees a clear “unknown tag” message; no Mario Kart sound for success.
- Tag associated with a racer whose race is not active at the configured event: reject or warn; do not attribute a lap. If the race is merely not yet started, show a test-read identification instead of a successful lap.
- Pre-start / test read: no Mario Kart success sound for a race lap; feedback clearly indicates test/identification mode rather than a scored lap.
- Duplicate bib assignment attempted: prevent save and explain the conflict.
- Rapid double-tap / bounce on the same physical tag within cooldown: only the first valid lap counts.
- Karaoke bonus clicked after the scan popup was dismissed: bonus is not available for that scan unless the volunteer re-opens the last-scan context (if provided); otherwise they must wait for the next valid scan.
- Offline for extended periods on multiple stations: on reconnect, merge preserves distinct taps; cooldown-window duplicates for the same racer are reconciled without inventing extra laps.
- CSV import into a station that already has partial data for the same race: import clearly replaces or merges per documented recovery rules (default: recovery import establishes authoritative restored state for that race on the new laptop).
- Reader laptop loses Proxmark3 connection mid-race: volunteer is alerted; no silent “success” pops; reconnecting the device resumes reading without restarting the whole app if possible.
- Event deleted while stations still configured for it: stations stop accepting taps and prompt to select a valid event; deleting one race under an event only stops taps for racers in that race.
- Wrong PIN entered: management stays locked; no indication of the correct PIN; limited or delayed retries are optional for v1 (simple reject is acceptable).
- Out-of-sequence checkpoint tap: no completed lap awarded from that tap alone; volunteer/racer receives clear non-success feedback; finish-mode cooldown rules still apply to completed RFID laps only.
- Placement ties (same lap count): display uses a deterministic tie-break (e.g., earliest last-lap time) so popup placement is stable.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: System MUST support Proxmark3 hardware for both reading tags during a race and writing/programming tags for racers.
- **FR-002**: System MUST allow organizers to create a race and delete a race (with confirmation) through a simple management flow.
- **FR-003**: System MUST provide a Racers page per race with a searchable list of all racers and the ability to add a racer.
- **FR-003a**: Racers search MUST update the list as the user types using debounce (no separate Search button required).
- **FR-003b**: Bib numbers on the Racers list MUST display as plain text until clicked to edit; a save control MUST appear only after the bib value has changed.
- **FR-003c**: Tag programming MUST be initiated from the racer row (inline), not a separate standalone Program RFID section.
- **FR-004**: System MUST assign bib numbers sequentially by default when a racer is added, and MUST allow staff to adjust a racer’s bib number to another valid value for that race.
- **FR-005**: System MUST assign each racer a stable unique racer identifier and MUST program RFID tags with that identifier (not solely the bib number).
- **FR-006**: System MUST allow multiple RFID tags to be programmed and associated with the same racer so lost tags can be replaced or duplicated. In v1, all programmed associations remain active (no deactivate/revoke); any associated tag can record laps for that racer.
- **FR-007**: When a laptop is configured as a reader for an event, the system MUST accept and process tag reads continuously regardless of which site page is currently displayed, and MUST attribute each valid tap to the racer’s race among any active races at that event.
- **FR-008**: On a valid tag tap for a racer whose race is **active/started**, the system MUST record a new lap with the tap timestamp for that racer (on the race in which the racer is enrolled).
- **FR-008a**: On a valid tag tap for a racer whose race is **not yet started**, the system MUST treat the tap as a test read (show racer identity for hardware/check-in confirmation) and MUST NOT record a race lap.
- **FR-008b**: While a race is pre-start (scheduled with a future start time), the live race flow MUST display a countdown to that race’s start time.
- **FR-008c**: A race MUST become active for scoring when its `start_time` is reached (auto-start). Organizers with PIN MUST also be able to start or finish a race manually.
- **FR-009**: On a valid **scored** tag tap (race active), the system MUST show a popup including the racer’s name, current overall placement, and number of laps (no on-screen “playing sound” label).
- **FR-010**: On a valid **scored** tag tap (race active), the system MUST play the Mario Kart new-lap sound (or project-approved equivalent) without displaying a textual “playing sound” indicator.
- **FR-011**: System MUST enforce a 1-minute cooldown after an RFID lap for the same racer before another RFID lap can be recorded (event-wide across reader stations once stations share recent tap knowledge).
- **FR-012**: Once any race at the configured event has started, the default page for race-day use MUST be the live race flow and leaderboard for that event.
- **FR-013**: System MUST store all taps and relevant race data in a local database on each station.
- **FR-014**: System MUST sync local taps and relevant race data to the online hosted database when connectivity is available.
- **FR-015**: If internet connectivity is lost, a reader station MUST continue to function using local data (record laps, show feedback, update local standings).
- **FR-016**: System MUST continuously maintain a live current CSV snapshot of event race data on each station (updated on relevant data changes without a manual export step), and MUST support importing that CSV on another laptop so timing can resume after hardware failure. Optional copy/download of the live file is allowed for convenience; recovery MUST NOT depend on staff remembering to export before going offline or before a crash.
- **FR-017**: System MUST support an arbitrary number of reader laptops for the same event.
- **FR-017a**: Each reader station MUST be configurable as either a **finish station** (default: each valid tap awards +1 lap subject to cooldown) or a **checkpoint** station participating in an ordered checkpoint sequence for lap completion.
- **FR-017b**: When all active readers for an event are finish stations, the system MUST treat them as equivalent lap-count points (redundancy / multiple mats).
- **FR-017c**: When checkpoint mode is used, the system MUST only count a completed lap when the racer’s checkpoint progress satisfies the configured order/rules for that race.
- **FR-018**: After a successful RFID lap scan, the scan confirmation UI MUST offer a single-click control to add one karaoke bonus lap for that racer. If a karaoke bonus was already recorded for that scan, the UI MUST show that it was recorded instead of the add button.
- **FR-019**: Karaoke bonus laps MUST be stored distinctly enough to appear in totals/leaderboard and MUST NOT require a second RFID tap.
- **FR-020**: A full end-to-end automated test suite MUST cover the acceptance scenarios in this specification (with Proxmark3 interactions simulated where physical hardware is unavailable).
- **FR-021**: System MUST provide a demo seed for event “All You Can East Bluffet” on August 1, 2026, with:
  - Lap-format races: 12-hour, 6-hour, and 90-minute kids
  - 12-hour and 6-hour: Intermediate and Advanced, each with Men and Women
  - 90-minute kids: Men and Women only
  - Start times: 8:00 AM for 6-hour and 12-hour; 3:00 PM for 90-minute kids (America/Detroit local time unless otherwise configured)
  - 100 total seeded racers across the event, each assigned to a category
- **FR-022**: Live leaderboard and placement shown on scan popup MUST default to combined overall standings (with category color coding and legend), reflecting lap totals including karaoke bonus laps, using deterministic tie-breaking when lap counts are equal. Category-filtered boards MUST remain available.
- **FR-022a**: Live event view MUST provide tabs for 12 Hour, 6 Hour, and 90 Minute races; MUST allow overlapping races to be viewed on a single chart; and MUST offer a fullscreen rotating view that cycles race flow and leaderboard side-by-side for active races.
- **FR-023**: System MUST reject unknown tags with clear feedback and MUST NOT record a successful lap for unrecognized identifiers.
- **FR-024**: System MUST prevent duplicate bib numbers within the same race.
- **FR-025**: Karaoke bonus one-click MUST be available after a completed RFID lap recorded at a finish-mode station (or after a checkpoint sequence that completes a lap); intermediate checkpoint taps that do not complete a lap MUST NOT offer karaoke bonus.
- **FR-026**: System MUST protect management actions (race create/delete, racer add/edit, bib changes, RFID tag programming, reader station configuration, CSV import/copy tools, and similar setup controls) behind a shared organizer PIN login. The default PIN is `1738`.
- **FR-027**: The live race flow and leaderboard MUST be viewable without entering the PIN (public/read-only).
- **FR-028**: On an already-configured reader station, tap processing and karaoke bonus after a completed lap MUST work without re-entering the PIN for each action (PIN gates management access, not per-lap authorization).
- **FR-029**: New code for this feature MUST meet **100%** automated line/branch coverage for new packages and modules (unit + integration), with e2e covering every user story, before the feature is marked complete. Any exclusion MUST be listed in `quickstart.md` with rationale (e.g. Proxmark3 USB I/O shim covered by MockReader).

### Key Entities

- **Event**: A named race day (e.g., All You Can East Bluffet) with a calendar date and one or more races.
- **Race**: A lap-format contest under an event with duration, start time, status (scheduled/active/finished), and category structure.
- **Category**: A scoring/placement bucket within a race (e.g., Intermediate Men, Advanced Women, Kids Men).
- **Racer**: A participant in a race with unique identifier, name, bib number, category membership, and status.
- **RFID Tag Association**: Binding between a physical tag and a racer’s unique identifier; multiple associations per racer allowed; in v1 all associations remain active (no revocation).
- **Lap Record**: A timed lap attributed to a racer, with timestamp, source station, sync state, and type (standard RFID lap vs karaoke bonus).
- **Reader Station**: A laptop configured to read for a specific event, accepting taps for racers in any active race under that event, with a mode of **finish** (default) or **checkpoint**, online/offline status, and local datastore.
- **Checkpoint Progress**: Per-racer progress through an ordered checkpoint sequence when checkpoint-mode readers are in use; completing the sequence yields one lap.
- **Cooldown State**: Per-racer restriction preventing another RFID lap within 1 minute of the last RFID lap (bonus clicks are not RFID laps).

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: On a configured reader, 100% of taps from known programmed tags outside cooldown produce a stored lap, on-screen confirmation (name, placement, laps), and new-lap sound within 2 seconds of the tap being read.
- **SC-002**: 100% of RFID lap attempts by the same racer within 60 seconds of their last RFID lap are rejected without creating an extra lap.
- **SC-003**: Volunteers can add a karaoke bonus lap with one click after a scan; time from successful scan confirmation to bonus recorded is under 3 seconds of volunteer action.
- **SC-004**: Staff can add a racer, assign/adjust bib, and program a tag in under 2 minutes per racer in a normal desk setup.
- **SC-005**: A station that loses internet continues recording laps with zero lost taps attributable to connectivity loss for at least 4 hours of local operation, and syncs all pending taps after reconnect.
- **SC-006**: After importing a station’s live CSV (captured without a manual export step) onto a replacement laptop, race staff resume scanning with prior lap counts preserved for all racers present in the file; live CSV updates continue through offline periods on the source station.
- **SC-007**: At least 3 reader stations can operate against the same event concurrently without preventing valid distinct laps from being recorded and eventually visible in the shared results.
- **SC-008**: Demo seed loads with the correct event name/date, three races with specified start times and category matrix, and exactly 100 racers.
- **SC-009**: End-to-end automated tests cover every user story in this spec; the suite is the gate for feature completion (no story marked done without corresponding passing coverage).
- **SC-010**: Race organizers can create or delete a race in under 1 minute without technical assistance.
- **SC-011**: Users without the PIN can view the live race flow and leaderboard but cannot perform management actions; organizers unlock management by entering PIN `1738`.
- **SC-012**: Before a race starts, viewers of the live race flow can see an accurate countdown to that race’s start (updates at least once per second, within 1 second of true remaining time under normal clock sync).
- **SC-013**: Racers list filters within ~300ms of typing pause (debounce) without clicking Search.

## Assumptions

- Access control for this feature is a shared organizer **PIN** (default `1738`), not per-user accounts. Live leaderboard is public/read-only; management requires PIN unlock. Configured reader tap handling and karaoke bonus do not require re-PIN per tap.
- RFID taps count as scored laps only while that racer’s race status is active/started. Taps before start are test reads for identification/hardware check and do not add laps (supports staggered starts, e.g. kids race starting later the same day).
- Pre-start live race flow shows a countdown to each not-yet-started race’s start time (visible without PIN, alongside public leaderboard viewing).
- “SV locally” in the request means a local database on each reader/management laptop, kept in sync with the hosted online database.
- Placement on the scan popup and live boards defaults to **combined overall** standings with **color coding and a legend** for all categories (category-only filters remain available), using total laps (standard + karaoke bonus) with earliest last-lap time as the tie-breaker. Live view uses tabs for **12 Hour**, **6 Hour**, and **90 Minute**; overlapping races can share a **single chart**. A **fullscreen rotating** display cycles race flow + leaderboard side-by-side for active races.
- Racers search updates results as the user types (**debounced**); no search button required.
- A reader station is bound to an **event**, not a single race; it accepts taps for racers enrolled in any **active** race under that event (required for concurrent 12-hour and 6-hour racing on one finish line).
- Reader stations default to **finish** mode (+1 lap per valid tap). Staff may configure a station as a **checkpoint** instead; finish and checkpoint modes can coexist at an event when staff configure them that way. Demo / primary AYCEB flow assumes finish-mode readers.
- Karaoke bonus is offered only from the post-scan confirmation UI for the just-completed RFID lap; it does not bypass or reset the 1-minute RFID cooldown.
- Only one karaoke bonus may be attached per successful RFID scan via the one-click control (racers who sang multiple songs still get one bonus per qualifying lap scan unless future rules change).
- Multiple tags for one racer are all equally valid and remain active for the life of the race data in v1; there is no deactivate/revoke action. Lost-tag handling is “program an additional tag”; a found old tag still works if presented.
- Proxmark3 is the supported reader/writer; other RFID hardware is out of scope for this feature.
- Demo seed timezone for start times is America/Detroit (Eastern) on August 1, 2026.
- The 90-minute kids race uses Men/Women category labels as specified (not a separate age-group matrix).
- CSV recovery is the supported disaster-recovery path for a failed laptop; each station keeps a **live current CSV** updated automatically so staff need not run an export before network loss or failure. Full binary DB file clone is not required for v1 if CSV round-trip restores racers, tag associations, and laps.
- Multi-station cooldown enforcement is best-effort when stations are partitioned offline; after sync, cooldown-window duplicates for the same racer are reconciled so results do not over-count.
- Mario Kart new-lap audio is included as a licensed-or-rights-cleared asset for this project use, or a project-approved equivalent cue if redistribution rights block the exact clip — behavior remains “play distinctive new-lap success sound on valid lap.”
- Continuous background reading applies while the application is open and the station remains configured as a reader; OS-level sleep/closing the browser/app stops reading (expected).
- Existing race-timing project foundations (events, races, participants concepts) are extended by this feature rather than replaced wholesale.
- E2E tests may use a simulated Proxmark3 device interface so CI can run without physical hardware; physical device verification remains a recommended manual checklist on race hardware.
- Racer (UX term) maps to Participant in the API/data model.
- Unknown / wrong-race tags: reject with clear non-success feedback (no success sound).
- HTML prototypes under `frontend/prototypes/002-rfid-race-scanner/` are **approved** for Vue implementation (2026-07-12).

## Out of Scope

- RFID hardware other than Proxmark3
- Tag revoke/deactivate in v1
- Per-volunteer user accounts (PIN only for management)
- Public self-registration / online payment for racers
- Mobile native apps (web SPA on station laptops is in scope)

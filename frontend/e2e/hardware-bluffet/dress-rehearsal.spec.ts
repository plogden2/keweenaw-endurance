/**
 * East Bluffet hardware dress rehearsal — single ~32-minute wall-clock orchestrated
 * test against a real Proxmark3 reader (one physical chip rewritten per lap).
 *
 * This is NOT a fast CI spec. It owns a deterministic timeline (compressed
 * 30/15-minute races + a 5-minute kids race at T+20), drives the Proxmark
 * write→read path for every scored lap, and injects chaos (reader crash,
 * 5-minute hosted partition outage with offline bridge scoring) while three
 * Run via `npm run test:e2e:bluffet-hardware` (frontend/) — see Task 8 / README.
 */
import { test, devices } from '@playwright/test'
import type { Page } from '@playwright/test'
import fs from 'node:fs'
import path from 'node:path'
import { fileURLToPath } from 'node:url'

import { API_BASE, BLUFFET, armFinishStation, getBluffetEvent, pinLogin, pinToken } from '../fixtures/rfid'
import { appendIssue, createRunDir, writeStatus, type Issue, type IssueSeverity, type RunStatus } from './lib/artifacts'
import { sampleLapDelayMs, sleep } from './lib/clock'
import { addRacersToLapState, dueRacers, initLapState, removeRacer, scheduleNext, type LapState } from './lib/lapEngine'
import { programRacerAndAwaitLap, resolveLogicalUuid } from './lib/proxmark'
import { loadSeededRacers, pickDnfs, pickNoShows, type Racer } from './lib/roster'
import { addLateSignup, finishCheckpointId, firstCategoryId, refreshEmptyBibs, setCompressedStartTimes } from './lib/setup'
import {
  crashAndReopenReader,
  fetchLocalBridgeStatus,
  healPartition,
  partitionFromHosted,
  waitForReaderChip,
  type VideoContext,
} from './lib/chaos'
import {
  awaitCatchUp,
  churnOnce,
  pickFriends,
  serverLapsTotal,
  snapshotVisibleLaps,
  type Spectator,
} from './lib/spectators'

// REPO_ROOT resolved from this file's own path — NOT process.cwd(), which
// depends on where `npm run test:e2e:bluffet-hardware` happens to be invoked
// from and previously produced a fragile `e2e-artifacts` location outside the repo.
const HERE = path.dirname(fileURLToPath(import.meta.url))
const REPO_ROOT = path.resolve(HERE, '../../..')
const ARTIFACT_ROOT = path.join(REPO_ROOT, 'e2e-artifacts', 'bluffet-hardware')

const VIDEO_SIZE = { width: 1920, height: 1080 }

// Timeline offsets, all relative to tZero (= T+0, the moment the shared
// 30/15-minute start_time is set to "now + 2 minutes").
const LATE_SIGNUP_1_OFFSET_MS = -30_000
const LATE_SIGNUP_2_OFFSET_MS = -5_000
const LATE_SIGNUP_3_OFFSET_MS = 2 * 60_000
const DNF_WINDOW_START_MS = 3 * 60_000
const DNF_WINDOW_END_MS = 18 * 60_000
const DNF_COUNT = 10
const READER_CRASH_OFFSET_MS = 12 * 60_000
const KIDS_START_OFFSET_MS = 20 * 60_000
const OUTAGE_START_OFFSET_MS = 14 * 60_000
const OUTAGE_DURATION_MS = 5 * 60_000
const RACE_END_OFFSET_MS = 30 * 60_000
const FINALIZE_SETTLE_MS = 30_000
const TICK_MS = 2_000

type ReaderHandle = VideoContext

type OrchestratorState = {
  runId: string
  phase: string
  tZero: Date
  lapsScored: number
  pendingSync: number
  chaos: { apiOutage: boolean; readerDown: boolean }
  lastProxmark?: string
  lastError?: string
  criticalCount: number
}

test.describe('Hardware East Bluffet dress rehearsal', () => {
  test('runs the full ~32min wall-clock dress rehearsal with chaos and spectators', async (
    { browser, request },
  ) => {
    test.setTimeout(45 * 60 * 1000)

    fs.mkdirSync(ARTIFACT_ROOT, { recursive: true })
    const { runId, dir: runDir } = createRunDir(ARTIFACT_ROOT)
    process.env.BLUFFET_HW_ARTIFACT_DIR = runDir
    const videoDir = path.join(runDir, 'videos')
    fs.mkdirSync(videoDir, { recursive: true })
    const screenshotDir = path.join(runDir, 'screenshots')
    fs.mkdirSync(screenshotDir, { recursive: true })

    const tZero = new Date(Date.now() + 2 * 60_000)
    const state: OrchestratorState = {
      runId,
      phase: 'setup',
      tZero,
      lapsScored: 0,
      pendingSync: 0,
      chaos: { apiOutage: false, readerDown: false },
      criticalCount: 0,
    }

    function writeStatusNow() {
      const status: RunStatus = {
        runId: state.runId,
        phase: state.phase,
        tZeroIso: state.tZero.toISOString(),
        nowIso: new Date().toISOString(),
        elapsedSec: Math.round((Date.now() - state.tZero.getTime()) / 1000),
        lapsScored: state.lapsScored,
        pendingSync: state.pendingSync,
        chaos: { ...state.chaos },
        lastProxmark: state.lastProxmark,
        lastError: state.lastError,
        healthy: state.criticalCount === 0,
      }
      writeStatus(runDir, status)
    }

    async function screenshotFor(label: string, page: Page | undefined): Promise<string | undefined> {
      if (!page) return undefined
      try {
        const file = `${label}-${Date.now()}.png`
        await page.screenshot({ path: path.join(screenshotDir, file) })
        return path.join('screenshots', file)
      } catch {
        return undefined
      }
    }

    async function issue(
      severity: IssueSeverity,
      title: string,
      details: string,
      opts: { screenshot?: string } = {},
    ) {
      const entry: Issue = {
        ts: new Date().toISOString(),
        severity,
        title,
        details,
        phase: state.phase,
        screenshot: opts.screenshot,
      }
      appendIssue(runDir, entry)
      if (severity === 'critical') {
        state.criticalCount += 1
        state.lastError = title
      }
      writeStatusNow()
    }

    async function refreshPendingSync() {
      try {
        const local = await fetchLocalBridgeStatus()
        state.pendingSync = local.pending_count ?? 0
      } catch {
        try {
          const token = await pinToken(request)
          const res = await request.get(`${API_BASE}/api/rfid/sync-status`, {
            headers: { Authorization: `Bearer ${token}` },
          })
          if (res.ok()) {
            const body = await res.json()
            state.pendingSync = body.pending_count ?? 0
          }
        } catch {
          // best-effort telemetry only
        }
      }
    }

    /** Closes a context immediately (freeing browser resources) and returns its saved video path, if any. */
    async function closeAndCollectVideoPath(seg: ReaderHandle): Promise<string | undefined> {
      const video = seg.page?.video()
      try {
        await seg.context.close()
      } catch {
        // already closed — ignore
      }
      if (!video) return undefined
      try {
        return await video.path()
      } catch {
        return undefined
      }
    }

    async function finalizeVideoFromPaths(label: string, paths: string[]) {
      if (paths.length === 0) {
        await issue('minor', `No video captured for ${label}`, 'Context closed without a recorded video file.')
        return
      }
      fs.copyFileSync(paths[paths.length - 1], path.join(runDir, `${label}.webm`))
      if (paths.length > 1) {
        paths.slice(0, -1).forEach((p, i) => {
          fs.copyFileSync(p, path.join(runDir, `${label}-part${i + 1}.webm`))
        })
        await issue(
          'idea',
          `${label} recorded in ${paths.length} segments`,
          'A mid-race context restart split the recording. Extra segments were saved alongside ' +
            `${label}.webm (which is the final/longest segment) — ffmpeg concat if a single ` +
            'continuous file is needed for the side-by-side compose.',
        )
      }
    }

    let stopAll = false
    const background: Promise<void>[] = []
    const readerVideoPaths: string[] = []
    let reader: ReaderHandle | undefined
    let laptopCtx: Awaited<ReturnType<typeof browser.newContext>> | undefined
    let laptopPage: Page | undefined
    let iphoneCtx: Awaited<ReturnType<typeof browser.newContext>> | undefined
    let iphonePage: Page | undefined
    let videosSalvaged = false

    async function salvageVideos() {
      if (videosSalvaged) return
      videosSalvaged = true
      try {
        if (reader) {
          const p = await closeAndCollectVideoPath(reader)
          if (p) readerVideoPaths.push(p)
          reader = undefined
        }
        await finalizeVideoFromPaths('reader', readerVideoPaths)
        if (laptopCtx && laptopPage) {
          const p = await closeAndCollectVideoPath({ context: laptopCtx, page: laptopPage })
          await finalizeVideoFromPaths('spectator-laptop', p ? [p] : [])
        }
        if (iphoneCtx && iphonePage) {
          const p = await closeAndCollectVideoPath({ context: iphoneCtx, page: iphonePage })
          await finalizeVideoFromPaths('spectator-iphone', p ? [p] : [])
        }
      } catch {
        // best-effort — don't mask the real failure
      }
    }

    try {
      // ---------------------------------------------------------------
      // Phase: T-2:00 setup — event, contexts, roster, compressed start times
      // ---------------------------------------------------------------
      writeStatusNow()
      const event = await getBluffetEvent(request)
      const token = await pinToken(request)
      await armFinishStation(request, token, event.id, 'laptop-finish-1')

      const readerContext = await browser.newContext({
        viewport: VIDEO_SIZE,
        recordVideo: { dir: videoDir, size: VIDEO_SIZE },
      })
      let readerSeg = { context: readerContext, page: await readerContext.newPage() }
      reader = readerSeg
      await readerSeg.page.goto('/pin')
      await pinLogin(readerSeg.page)
      await readerSeg.page.goto('/station')
      await readerSeg.page
        .getByTestId('station-armed-indicator')
        .waitFor({ state: 'visible', timeout: 15_000 })
        .catch(() => {})
      await readerSeg.page.goto(`/events/${event.id}/live`)
      await readerSeg.page.getByTestId('live-view').waitFor({ state: 'visible', timeout: 20_000 })

      laptopCtx = await browser.newContext({
        viewport: VIDEO_SIZE,
        recordVideo: { dir: videoDir, size: VIDEO_SIZE },
      })
      laptopPage = await laptopCtx.newPage()
      await laptopPage.goto(`/events/${event.id}/live`)
      await laptopPage.getByTestId('live-view').waitFor({ state: 'visible', timeout: 20_000 })

      iphoneCtx = await browser.newContext({
        ...devices['iPhone 13'],
        recordVideo: { dir: videoDir, size: VIDEO_SIZE },
      })
      iphonePage = await iphoneCtx.newPage()
      await iphonePage.goto(`/events/${event.id}/live`)
      await iphonePage.getByTestId('live-view').waitFor({ state: 'visible', timeout: 20_000 })

      await setCompressedStartTimes(request, tZero)

      const { racers, skipped } = await loadSeededRacers(request)
      if (skipped.length > 0) {
        await issue(
          'minor',
          'Seeded racers missing logical tag UUID',
          `${skipped.length} participants had no rfid_tag_uid/association and were excluded ` +
            `from the lap rotation (never programmed onto the chip): bibs ` +
            `${skipped.slice(0, 10).map((r) => r.bib).join(', ')}${skipped.length > 10 ? ', …' : ''}`,
        )
      }

      // 9 no-shows are picked from the FULL seeded field (all 3 races) — a
      // kids racer can be a no-show same as a 30/15-minute racer. Only the
      // survivors are split by race: the 30/15-minute rotation starts at
      // T+0, but kids racers are held out of `lapState` until their own
      // start_time arrives at T+20 (see the T+20 gate in the main loop).
      const noShowIds = pickNoShows(racers, 9)
      const activeRacers: Racer[] = racers.filter((r) => !noShowIds.has(r.id))
      const nonKidsActiveRacers = activeRacers.filter((r) => r.raceId !== BLUFFET.races.kids.id)
      const kidsActiveRacers = activeRacers.filter((r) => r.raceId === BLUFFET.races.kids.id)
      const racerById = new Map<string, Racer>(racers.map((r) => [r.id, r]))

      // Spectators: 5 "friends" each, laptop from the 12h field, iphone from the
      // 6h field — pure search variety. Catch-up assertions always target the
      // 12-hour race because EventLive.vue only tags leaderboard cells on that tab.
      const spectatorLaptop: Spectator = {
        name: 'spectator-laptop',
        eventId: event.id,
        raceId: BLUFFET.races.twelveHour.id,
        friends: pickFriends(racers.filter((r) => r.raceId === BLUFFET.races.twelveHour.id), 1),
        page: laptopPage,
      }
      const spectatorIphone: Spectator = {
        name: 'spectator-iphone',
        eventId: event.id,
        raceId: BLUFFET.races.twelveHour.id,
        friends: pickFriends(racers.filter((r) => r.raceId === BLUFFET.races.sixHour.id), 2),
        page: iphonePage,
      }

      // ---------------------------------------------------------------
      // Background loops: status heartbeat, reader carousel, spectator churn.
      // These run concurrently with the sequential timeline below via the
      // event loop — no explicit interleaving needed.
      // ---------------------------------------------------------------
      background.push(
        (async () => {
          while (!stopAll) {
            await refreshPendingSync()
            writeStatusNow()
            await sleep(10_000)
          }
        })(),
      )

      const readerTabs = ['race-tab-12h', 'race-tab-6h', 'race-tab-90m'] as const
      background.push(
        (async () => {
          while (!stopAll) {
            const page = reader!.page
            try {
              const roll = Math.random()
              if (roll < 0.55) {
                const open = await page.getByTestId('fullscreen-rotator').isVisible().catch(() => false)
                if (!open) await page.getByTestId('fullscreen-rotator-toggle').click({ timeout: 3_000 })
              } else if (roll < 0.7) {
                await page.keyboard.press('Escape').catch(() => {})
                await page.goto(`/races/${BLUFFET.races.twelveHour.id}/racers`, { timeout: 15_000 })
                await sleep(1_000)
                await page.goto(`/events/${event.id}/live`, { timeout: 15_000 })
              } else if (roll < 0.85) {
                await page.keyboard.press('Escape').catch(() => {})
                await page.goto('/station', { timeout: 15_000 })
                await sleep(1_000)
                await page.goto(`/events/${event.id}/live`, { timeout: 15_000 })
              } else {
                const tab = readerTabs[Math.floor(Math.random() * readerTabs.length)]
                await page.getByTestId(tab).click({ timeout: 3_000 })
              }
            } catch {
              // best-effort UI churn — real problems surface via lap scoring assertions
            }
            await sleep(20_000 + Math.random() * 15_000)
          }
        })(),
      )

      background.push(
        (async () => {
          while (!stopAll) {
            await churnOnce(spectatorLaptop)
            await sleep(15_000 + Math.random() * 10_000)
          }
        })(),
      )
      background.push(
        (async () => {
          while (!stopAll) {
            await churnOnce(spectatorIphone)
            await sleep(15_000 + Math.random() * 10_000)
          }
        })(),
      )

      // ---------------------------------------------------------------
      // Pre-race: verify the write-tag path once, then 2 last-minute signups.
      // ---------------------------------------------------------------
      const probe = nonKidsActiveRacers[0]
      if (probe) {
        try {
          await programRacerAndAwaitLap({
            request,
            readerPage: reader!.page,
            participantId: probe.id,
            timeoutMs: 60_000,
            dismissAfter: true,
          })
          state.lastProxmark = `pre-race probe ok (bib ${probe.bib})`
          await issue(
            'idea',
            'Pre-race tag-write path verified',
            `Programmed + read bib ${probe.bib} before start_time — write-tag→Poll/WS path is healthy.`,
          )
        } catch (err) {
          await issue(
            'critical',
            'Pre-race tag-write probe failed',
            String(err),
            { screenshot: await screenshotFor('pre-race-probe', reader!.page) },
          )
        }
      }

      const cat12 = await firstCategoryId(request, BLUFFET.races.twelveHour.id)
      const cat6 = await firstCategoryId(request, BLUFFET.races.sixHour.id)

      await sleep(Math.max(0, tZero.getTime() + LATE_SIGNUP_1_OFFSET_MS - Date.now()))
      try {
        const late1 = await addLateSignup(request, BLUFFET.races.twelveHour.id, cat12, {
          first: 'Late',
          last: 'Signup1',
        })
        const racer: Racer = {
          id: late1.id,
          raceId: BLUFFET.races.twelveHour.id,
          bib: '',
          firstName: 'Late',
          lastName: 'Signup1',
          logicalTagUuid: '',
        }
        activeRacers.push(racer)
        nonKidsActiveRacers.push(racer)
        racerById.set(racer.id, racer)
      } catch (err) {
        await issue('critical', 'Late signup #1 failed', String(err))
      }

      await sleep(Math.max(0, tZero.getTime() + LATE_SIGNUP_2_OFFSET_MS - Date.now()))
      try {
        const late2 = await addLateSignup(request, BLUFFET.races.sixHour.id, cat6, {
          first: 'Late',
          last: 'Signup2',
        })
        const racer: Racer = {
          id: late2.id,
          raceId: BLUFFET.races.sixHour.id,
          bib: '',
          firstName: 'Late',
          lastName: 'Signup2',
          logicalTagUuid: '',
        }
        activeRacers.push(racer)
        nonKidsActiveRacers.push(racer)
        racerById.set(racer.id, racer)
      } catch (err) {
        await issue('critical', 'Late signup #2 failed', String(err))
      }

      // ---------------------------------------------------------------
      // T+0: races auto-start (start_time already set). Begin the lap engine.
      // ---------------------------------------------------------------
      await sleep(Math.max(0, tZero.getTime() - Date.now()))
      state.phase = 'racing'
      writeStatusNow()

      // Only the 30/15-minute rotation starts at T+0. Kids join at T+20 (see
      // the KIDS_START_OFFSET_MS gate below) via addRacersToLapState, which
      // preserves `scored` and existing nextDue entries instead of
      // re-initializing the whole map.
      let lapState: LapState = initLapState(nonKidsActiveRacers.map((r) => r.id), Date.now())

      // DNFs are drawn from the 30/15-minute field only — the entire DNF
      // window (T+3:00 → T+18:00) elapses before the kids race even starts.
      const dnfIds = [...pickDnfs(nonKidsActiveRacers.map((r) => r.id), DNF_COUNT)]
      const dnfWindowMs = DNF_WINDOW_END_MS - DNF_WINDOW_START_MS
      const dnfSchedule = dnfIds.map((id, i) => ({
        id,
        dueAt: tZero.getTime() + DNF_WINDOW_START_MS + (dnfWindowMs * i) / Math.max(1, dnfIds.length - 1),
      }))

      let signup3Done = false
      let kidsStarted = false
      let crashDone = false
      let outageStarted = false
      let outageEnded = false
      let outagePreSnapshot: { laptop: number; iphone: number } | undefined
      let outagePreServerLaps = 0
      let outagePreBridgePending = 0
      let outageLapsScored = 0
      let outageServerLeakReported = false

      const raceEndAt = tZero.getTime() + RACE_END_OFFSET_MS

      while (Date.now() < raceEndAt) {
        // Fresh clock every tick. Chaos/timeline MUST run before the lap engine —
        // each write→read can take 10–60s; draining all due racers first starved
        // outage/crash windows on the prod-like bridge path.
        let now = Date.now()

        // --- T+2:00 late signup #3, joins rotation immediately ---
        if (!signup3Done && now >= tZero.getTime() + LATE_SIGNUP_3_OFFSET_MS) {
          signup3Done = true
          try {
            const late3 = await addLateSignup(request, BLUFFET.races.sixHour.id, cat6, {
              first: 'Late',
              last: 'Signup3',
            })
            const racer: Racer = {
              id: late3.id,
              raceId: BLUFFET.races.sixHour.id,
              bib: '',
              firstName: 'Late',
              lastName: 'Signup3',
              logicalTagUuid: '',
            }
            activeRacers.push(racer)
            nonKidsActiveRacers.push(racer)
            racerById.set(racer.id, racer)
            lapState.nextDue.set(racer.id, Date.now() + sampleLapDelayMs())
          } catch (err) {
            await issue('critical', 'Late signup #3 (T+2:00) failed', String(err))
          }
        }

        // --- T+20: kids race start_time arrives — join its field into rotation ---
        // (No kids late signups exist in this timeline; if one is ever added,
        // it must be held back the same way until this flag flips.)
        if (!kidsStarted && now >= tZero.getTime() + KIDS_START_OFFSET_MS) {
          kidsStarted = true
          addRacersToLapState(lapState, kidsActiveRacers.map((r) => r.id), now)
          await issue(
            'idea',
            'Kids race joined lap rotation at T+20',
            `${kidsActiveRacers.length} kids racers added to the active lap rotation.`,
          )
        }

        // --- DNF drip: ~10 racers drop out of rotation across the mid-race window ---
        for (const entry of dnfSchedule) {
          if (!lapState.nextDue.has(entry.id)) continue
          if (now >= entry.dueAt) removeRacer(lapState, entry.id)
        }

        // --- reader crash + manual-entry recovery (once) ---
        if (!crashDone && now >= tZero.getTime() + READER_CRASH_OFFSET_MS) {
          crashDone = true
          await performReaderCrash()
        }

        // --- 5-minute hosted partition: bridge scores offline, spectators stale ---
        if (!outageStarted && now >= tZero.getTime() + OUTAGE_START_OFFSET_MS) {
          outageStarted = true
          state.chaos.apiOutage = true
          writeStatusNow()
          try {
            outagePreBridgePending = (await fetchLocalBridgeStatus()).pending_count
          } catch (err) {
            await issue(
              'critical',
              'Could not read local bridge status before outage',
              String(err),
            )
          }
          outagePreSnapshot = {
            laptop: await snapshotVisibleLaps(laptopPage).catch(() => 0),
            iphone: await snapshotVisibleLaps(iphonePage).catch(() => 0),
          }
          // Partition BEFORE snapshotting hosted laps. In-flight bridge→hosted
          // reads from the previous tick can land after OUTAGE_START; capturing
          // pre= too early falsely flags those as partition leaks.
          await partitionFromHosted([laptopCtx!, iphoneCtx!])
          try {
            await waitForReaderChip(reader!.page, 'offline', 45_000)
          } catch (err) {
            await issue(
              'critical',
              'Reader sync chip did not show Offline during partition',
              String(err),
              { screenshot: await screenshotFor('outage-offline-chip', reader!.page) },
            )
          }
          await sleep(2_000)
          // Hold until hosted lap total is stable so late WS reads from the
          // pre-partition window don't look like partition leaks.
          let stable = await totalScoredLaps().catch(() => 0)
          for (let i = 0; i < 5; i++) {
            await sleep(1_000)
            const again = await totalScoredLaps().catch(() => stable)
            if (again === stable) break
            stable = again
          }
          outagePreServerLaps = stable
        }
        if (outageStarted && !outageEnded) {
          if (now >= tZero.getTime() + OUTAGE_START_OFFSET_MS + OUTAGE_DURATION_MS) {
            outageEnded = true
            await healPartition([laptopCtx!, iphoneCtx!])
            state.chaos.apiOutage = false
            writeStatusNow()

            try {
              await waitForReaderChip(reader!.page, 'syncing', 90_000)
            } catch (err) {
              await issue(
                'minor',
                'Reader sync chip did not show Syncing after heal',
                String(err),
                { screenshot: await screenshotFor('outage-syncing-chip', reader!.page) },
              )
            }

            try {
              await waitForReaderChip(reader!.page, 'online_synced', 180_000)
            } catch (err) {
              await issue(
                'critical',
                'Reader sync chip did not reach Online · Synced after heal',
                String(err),
                { screenshot: await screenshotFor('outage-synced-chip', reader!.page) },
              )
            }

            const postHealTarget = outagePreServerLaps + outageLapsScored
            let postHealLaps = await totalScoredLaps().catch(() => outagePreServerLaps)
            const catchUpDeadline = Date.now() + 60_000
            while (postHealLaps < postHealTarget && Date.now() < catchUpDeadline) {
              await sleep(2_000)
              postHealLaps = await totalScoredLaps().catch(() => postHealLaps)
            }
            if (postHealLaps < postHealTarget) {
              await issue(
                'critical',
                'Hosted lap total did not catch up after automatic bridge sync',
                `expected ≥ ${postHealTarget} (pre=${outagePreServerLaps} ` +
                  `+ outage=${outageLapsScored}), got ${postHealLaps}`,
              )
            }

            try {
              const bridgeAfter = await fetchLocalBridgeStatus()
              if (bridgeAfter.pending_count > 0) {
                await issue(
                  'minor',
                  'Bridge still has pending laps after auto-sync',
                  `pending_count=${bridgeAfter.pending_count} after heal`,
                )
              }
            } catch {
              // best-effort
            }

            const laptopCatch = await awaitCatchUp(spectatorLaptop, request, { timeoutMs: 120_000 })
            if (!laptopCatch.caughtUp) {
              await issue(
                'critical',
                'Spectator laptop did not catch up after outage heal',
                `ui=${laptopCatch.uiTotal} server=${laptopCatch.serverTotal}`,
                { screenshot: await screenshotFor('outage-catchup-laptop', laptopPage) },
              )
            }
            const iphoneCatch = await awaitCatchUp(spectatorIphone, request, { timeoutMs: 120_000 })
            if (!iphoneCatch.caughtUp) {
              await issue(
                'critical',
                'Spectator iphone did not catch up after outage heal',
                `ui=${iphoneCatch.uiTotal} server=${iphoneCatch.serverTotal}`,
                { screenshot: await screenshotFor('outage-catchup-iphone', iphonePage) },
              )
            }
          } else {
            if (!outageServerLeakReported) {
              const serverNow = await totalScoredLaps().catch(() => outagePreServerLaps)
              if (serverNow > outagePreServerLaps) {
                outageServerLeakReported = true
                await issue(
                  'critical',
                  'Hosted lap total increased during bridge partition',
                  `pre=${outagePreServerLaps} now=${serverNow} — hosted should not score while bridge is partitioned.`,
                )
              }
            }

            if (outageLapsScored > 0) {
              try {
                const bridgePending = (await fetchLocalBridgeStatus()).pending_count
                if (bridgePending < outagePreBridgePending + outageLapsScored) {
                  await issue(
                    'minor',
                    'Local bridge pending queue lagging outage laps scored',
                    `expected ≥ ${outagePreBridgePending + outageLapsScored}, got ${bridgePending}`,
                  )
                }
              } catch {
                // best-effort during partition
              }
            }

            if (outagePreSnapshot) {
              const laptopNow = await snapshotVisibleLaps(laptopPage).catch(() => 0)
              const iphoneNow = await snapshotVisibleLaps(iphonePage).catch(() => 0)
              if (laptopNow > outagePreSnapshot.laptop || iphoneNow > outagePreSnapshot.iphone) {
                await issue(
                  'minor',
                  'Spectator saw fresh laps during hosted partition',
                  `laptop ${outagePreSnapshot.laptop}->${laptopNow}, iphone ${outagePreSnapshot.iphone}->${iphoneNow} ` +
                    '— expected no visible change while spectator contexts are offline.',
                )
                outagePreSnapshot = { laptop: laptopNow, iphone: iphoneNow }
              }
            }
          }
        }

        // --- lap engine: at most one write→read per tick so chaos stays on schedule ---
        now = Date.now()
        const inOutage = outageStarted && !outageEnded
        const due = dueRacers(lapState, now)
        if (due.length > 0) {
          const id = due[0]
          const racer = racerById.get(id)
          // Late signups may not have a tag association yet. During partition the
          // local bridge cannot mint one via hosted lookup — defer until online.
          if (inOutage && !racer?.logicalTagUuid) {
            lapState.nextDue.set(id, Date.now() + 30_000)
            await issue(
              'idea',
              'Deferred untagged racer during hosted partition',
              `bib=${racer?.bib ?? '?'} participant=${id} — no logical UUID yet; will retry after heal.`,
            )
          } else {
            try {
              await programRacerAndAwaitLap({
                request,
                readerPage: reader!.page,
                participantId: id,
                logicalTagUuid: racer?.logicalTagUuid,
                raceId: racer?.raceId,
                useLocalBridge: inOutage,
                timeoutMs: 60_000,
                dismissAfter: true,
              })
              // First successful write stamps the permanent UUID onto the chip and
              // usually creates/activates the association — cache it for outage path.
              if (racer && !racer.logicalTagUuid) {
                const resolved = await resolveLogicalUuid(request, id, racer.raceId)
                if (resolved) racer.logicalTagUuid = resolved
              }
              scheduleNext(lapState, id, Date.now())
              state.lapsScored = lapState.scored
              state.lastProxmark = `lap ok bib=${racer?.bib ?? id}`
              if (inOutage) outageLapsScored += 1
            } catch (err) {
              lapState.nextDue.set(id, Date.now() + 45_000)
              await issue(
                'critical',
                'Proxmark write-tag/read timeout',
                `bib=${racer?.bib ?? '?'} participant=${id}: ${String(err)}`,
                { screenshot: await screenshotFor('proxmark-timeout', reader!.page) },
              )
            }
          }
        }

        writeStatusNow()
        await sleep(TICK_MS)
      }

      /** Always-online ground truth across every race currently in play (kids counts as 0 laps before T+20). */
      async function totalScoredLaps(): Promise<number> {
        const raceIds = [BLUFFET.races.twelveHour.id, BLUFFET.races.sixHour.id, BLUFFET.races.kids.id]
        const totals = await Promise.all(raceIds.map((id) => serverLapsTotal(request, id).catch(() => 0)))
        return totals.reduce((a, b) => a + b, 0)
      }

      async function performReaderCrash() {
        state.chaos.readerDown = true
        writeStatusNow()
        await issue(
          'idea',
          'Reader crash chaos triggered',
          'Closing the reader browser context mid-race to simulate a hard crash; reopening with a fresh context.',
        )

        const preCrashLaps = await totalScoredLaps().catch(() => -1)

        const inRotation = activeRacers.filter((r) => lapState.nextDue.has(r.id))
        await refreshEmptyBibs(request, inRotation.filter((r) => !r.bib))
        const manualPicks = inRotation.filter((r) => r.bib).slice(0, 2)
        const oldSegmentPath = await closeAndCollectVideoPath(reader!)
        if (oldSegmentPath) readerVideoPaths.push(oldSegmentPath)

        try {
          reader = await crashAndReopenReader({ browser, request, eventId: event.id, videoDir })
        } catch (err) {
          await issue('critical', 'Reader failed to reopen after crash', String(err))
          state.chaos.readerDown = false
          return
        }

        // Assert no committed lap data was lost across the close/reopen —
        // BEFORE manual entry adds any recovery laps, so this only measures
        // whether the crash itself dropped previously-scored history.
        if (preCrashLaps >= 0) {
          const postReopenLaps = await totalScoredLaps().catch(() => preCrashLaps)
          if (postReopenLaps < preCrashLaps) {
            await issue(
              'critical',
              'Lap data lost across reader crash/reopen',
              `Server-side scored lap total dropped from ${preCrashLaps} (pre-crash) to ${postReopenLaps} ` +
                '(post-reopen) — previously committed laps disappeared. Manual-entry recovery laps have not ' +
                'been added yet at this point, so this is not explained by the recovery step itself.',
              { screenshot: await screenshotFor('reader-crash-lap-loss', reader!.page) },
            )
          }
        } else {
          await issue(
            'minor',
            'Could not verify lap-loss across reader crash',
            'Pre-crash server lap total could not be fetched, so post-reopen comparison was skipped.',
          )
        }

        for (const r of manualPicks) {
          try {
            const checkpointId = await finishCheckpointId(request, r.raceId)
            const manualToken = await pinToken(request)
            const res = await request.post(`${API_BASE}/api/rfid/manual-entry`, {
              headers: { Authorization: `Bearer ${manualToken}` },
              data: {
                race_id: r.raceId,
                checkpoint_id: checkpointId,
                bib_number: r.bib,
                timestamp: new Date().toISOString(),
                device_id: 'manual-recovery-1',
              },
            })
            if (!res.ok()) {
              await issue(
                'critical',
                'Manual entry failed after reader crash',
                `bib=${r.bib} race=${r.raceId}: ${res.status()} ${await res.text()}`,
              )
            } else {
              lapState.nextDue.set(r.id, Date.now() + sampleLapDelayMs())
              lapState.scored += 1
              state.lapsScored = lapState.scored
            }
          } catch (err) {
            await issue('critical', 'Manual entry threw after reader crash', String(err))
          }
        }

        state.chaos.readerDown = false
        writeStatusNow()
      }

      // ---------------------------------------------------------------
      // T+30 finalize: brief settle window for trailing laps, then teardown.
      // ---------------------------------------------------------------
      state.phase = 'finalizing'
      writeStatusNow()
      await sleep(FINALIZE_SETTLE_MS)

      stopAll = true
      await Promise.allSettled(background)

      state.phase = state.criticalCount > 0 ? 'failed' : 'done'
      writeStatusNow()

      if (state.criticalCount > 0) {
        throw new Error(
          `Dress rehearsal recorded ${state.criticalCount} critical issue(s) — see ${path.join(runDir, 'issues.md')}`,
        )
      }
    } catch (err) {
      stopAll = true
      await Promise.allSettled(background)
      const msg = String(err)
      if (!msg.includes('critical issue')) {
        await issue('critical', 'Orchestrator crashed', msg)
        state.phase = 'failed'
        writeStatusNow()
      }
      throw err
    } finally {
      await salvageVideos()
    }
  })
})

import fs from 'node:fs'
import path from 'node:path'

export type IssueSeverity = 'critical' | 'minor' | 'idea'
export type Issue = {
  ts: string
  severity: IssueSeverity
  title: string
  details: string
  phase: string
  screenshot?: string
}

export type RunStatus = {
  runId: string
  phase: string
  tZeroIso: string
  nowIso: string
  elapsedSec: number
  lapsScored: number
  pendingSync: number
  chaos: { apiOutage: boolean; readerDown: boolean }
  lastProxmark?: string
  lastError?: string
  healthy: boolean
}

/**
 * Creates (or reuses) the artifact directory for a run.
 *
 * `BLUFFET_HW_ARTIFACT_DIR`, if already set, wins — the
 * `scripts/run-bluffet-hardware.mjs` runner creates the run dir and exports
 * this env var *before* spawning Playwright so `playwright.config.ts` (which
 * reads it at module-load time for the JSON reporter's `outputFile`) and this
 * function agree on one runId/dir instead of each inventing their own.
 */
export function createRunDir(root = path.join(process.cwd(), '..', '..', 'e2e-artifacts', 'bluffet-hardware')) {
  const existingDir = process.env.BLUFFET_HW_ARTIFACT_DIR
  const dir = existingDir || path.join(root, new Date().toISOString().replace(/[:.]/g, '-'))
  const runId = path.basename(dir)
  fs.mkdirSync(dir, { recursive: true })
  const issuesJsonl = path.join(dir, 'issues.jsonl')
  const issuesMd = path.join(dir, 'issues.md')
  if (!fs.existsSync(issuesJsonl)) fs.writeFileSync(issuesJsonl, '')
  if (!fs.existsSync(issuesMd)) fs.writeFileSync(issuesMd, '# Issues\n\n')
  return { runId, dir }
}

export function writeStatus(dir: string, status: RunStatus) {
  fs.writeFileSync(path.join(dir, 'status.json'), JSON.stringify(status, null, 2))
}

export function appendIssue(dir: string, issue: Issue) {
  fs.appendFileSync(path.join(dir, 'issues.jsonl'), JSON.stringify(issue) + '\n')
  const shot = issue.screenshot ? ` ![shot](${issue.screenshot})` : ''
  fs.appendFileSync(
    path.join(dir, 'issues.md'),
    `- **[${issue.severity}]** ${issue.title} — ${issue.details} _(phase=${issue.phase}, ${issue.ts})_ ${shot}\n`,
  )
}

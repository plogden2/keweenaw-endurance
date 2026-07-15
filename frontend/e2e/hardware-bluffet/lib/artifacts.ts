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

export function createRunDir(root = path.join(process.cwd(), '..', '..', 'e2e-artifacts', 'bluffet-hardware')) {
  const runId = new Date().toISOString().replace(/[:.]/g, '-')
  const dir = path.join(root, runId)
  fs.mkdirSync(dir, { recursive: true })
  fs.writeFileSync(path.join(dir, 'issues.jsonl'), '')
  fs.writeFileSync(path.join(dir, 'issues.md'), '# Issues\n\n')
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

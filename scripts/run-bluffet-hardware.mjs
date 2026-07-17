#!/usr/bin/env node
/**
 * Thin pre-runner for the East Bluffet hardware dress rehearsal.
 *
 * `playwright.config.ts` reads `BLUFFET_HW_ARTIFACT_DIR` at module-load time
 * (for the JSON reporter's `outputFile`), but the spec itself doesn't set
 * that env var until well after Playwright has already started — too late
 * for the config to see it. This script creates the run directory and
 * exports the env var *before* spawning Playwright, so the config, the
 * spec's `createRunDir` (which now reuses this exact dir instead of
 * inventing a second runId), and every artifact (status.json, issues.md,
 * videos, playwright-report.json) all land in one place:
 *
 *   e2e-artifacts/bluffet-hardware/<runId>/
 */
import { spawn } from 'node:child_process'
import fs from 'node:fs'
import path from 'node:path'
import { fileURLToPath } from 'node:url'

const HERE = path.dirname(fileURLToPath(import.meta.url))
const REPO_ROOT = path.resolve(HERE, '..')
const ARTIFACT_ROOT = path.join(REPO_ROOT, 'e2e-artifacts', 'bluffet-hardware')
const FRONTEND_DIR = path.join(REPO_ROOT, 'frontend')

// --prod [optionalUrl]
const args = process.argv.slice(2)
let mode = 'prod-like'
let baseURL = process.env.E2E_BASE_URL || 'http://keweenawendurance.com'
const prodIdx = args.indexOf('--prod')
if (prodIdx !== -1) {
  mode = 'prod'
  const next = args[prodIdx + 1]
  baseURL = next && !next.startsWith('-') ? next : 'https://keweenawendurance.com'
  if (/keweenawendurance\.com/i.test(baseURL) && process.env.BLUFFET_HW_ALLOW_PROD !== '1') {
    console.error('Refusing --prod against live domain without BLUFFET_HW_ALLOW_PROD=1')
    process.exit(2)
  }
}

const playwrightArgs = [...args]
if (prodIdx !== -1) {
  playwrightArgs.splice(prodIdx, 1)
  if (playwrightArgs[prodIdx] && !playwrightArgs[prodIdx].startsWith('-')) {
    playwrightArgs.splice(prodIdx, 1)
  }
}

const apiURL = process.env.E2E_API_URL ?? baseURL

const runId = new Date().toISOString().replace(/[:.]/g, '-')
const runDir = path.join(ARTIFACT_ROOT, runId)

fs.mkdirSync(runDir, { recursive: true })
fs.writeFileSync(path.join(runDir, 'issues.jsonl'), '')
fs.writeFileSync(path.join(runDir, 'issues.md'), '# Issues\n\n')

console.log(`[run-bluffet-hardware] mode=${mode} baseURL=${baseURL} artifacts -> ${runDir}`)

const child = spawn(
  'npx',
  [
    'playwright',
    'test',
    '--config=e2e/hardware-bluffet/playwright.config.ts',
    ...playwrightArgs,
  ],
  {
    cwd: FRONTEND_DIR,
    stdio: 'inherit',
    shell: process.platform === 'win32',
    env: {
      ...process.env,
      BLUFFET_HW_ARTIFACT_DIR: runDir,
      BLUFFET_HW_MODE: mode,
      E2E_BASE_URL: baseURL,
      E2E_API_URL: apiURL,
    },
  },
)

child.on('exit', (code) => process.exit(code ?? 1))
child.on('error', (err) => {
  console.error('[run-bluffet-hardware] failed to spawn playwright:', err)
  process.exit(1)
})

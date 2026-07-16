import { readFileSync } from 'node:fs'
import { join } from 'node:path'
import { describe, expect, it } from 'vitest'

const mainCss = readFileSync(join(process.cwd(), 'src/assets/main.css'), 'utf8')

describe('default theme tokens', () => {
  it('defines Superior Forest token block on :root', () => {
    expect(mainCss).toMatch(/:root\s*\{[^}]*--ink:\s*#1a3f3d;/s)
    expect(mainCss).toMatch(/:root\s*\{[^}]*--accent:\s*var\(--ink\);|:root\s*\{[^}]*--accent:\s*#1a3f3d;/s)
    expect(mainCss).toMatch(/--accent-link:\s*#2f6b5a;/)
    expect(mainCss).toMatch(/--sage:\s*#a1b383;/)
    expect(mainCss).toMatch(/--copper:\s*#9b654e;/)
    expect(mainCss).toMatch(/--mist:\s*#eff1f0;/)
    expect(mainCss).toMatch(/--signal:\s*#c45c38;/)
    expect(mainCss).toMatch(/--success:\s*#27ae60;/)
    expect(mainCss).toMatch(/--muted:\s*#6b7a76;/)
    expect(mainCss).toMatch(/--border:\s*#d4dad7;/)
    expect(mainCss).toMatch(/--surface:\s*#ffffff;/)
    expect(mainCss).toMatch(/--ink-deep:\s*#203429;/)
  })

  it('uses Outfit on body and landscape wash on page background', () => {
    expect(mainCss).toMatch(/font-family:[^;]*Outfit/)
    expect(mainCss).toMatch(/linear-gradient\([^)]*#e8efe6[^)]*#eff1f0[^)]*#e8d5c8/)
  })

  it('wires shared chrome to tokens', () => {
    expect(mainCss).toMatch(/\.btn-primary\s*\{[^}]*background[^;]*var\(--accent\)/s)
    expect(mainCss).toMatch(/:focus-visible\s*\{[^}]*outline:[^;]*var\(--sage\)/s)
    expect(mainCss).toMatch(/\.status-active\s*\{[^}]*color:\s*var\(--success\)/s)
    expect(mainCss).toMatch(/\.status-cancelled\s*\{[^}]*color:\s*var\(--signal\)/s)
  })
})

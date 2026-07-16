import { readFileSync } from 'node:fs'
import { join } from 'node:path'
import { describe, expect, it } from 'vitest'

const files = [
  'src/components/ScanPopup.vue',
  'src/components/SyncStatus.vue',
  'src/components/RaceCard.vue',
  'src/components/ManualTimingForm.vue',
]

describe('shared component chrome tokens', () => {
  for (const file of files) {
    it(`${file} does not hardcode legacy blue chrome`, () => {
      const src = readFileSync(join(process.cwd(), file), 'utf8')
      const style = src.split('<style')[1] ?? ''
      expect(style).not.toMatch(/#3498db|#2980b9|#1a5276|#2c3e50/i)
    })
  }
})

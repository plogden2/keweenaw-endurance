import { readFileSync } from 'node:fs'
import { join } from 'node:path'
import { describe, expect, it } from 'vitest'

const files = [
  'src/views/Home.vue',
  'src/views/EventDetails.vue',
  'src/components/EventsTable.vue',
  'src/views/Timing.vue',
  'src/views/LiveTiming.vue',
  'src/views/RaceDetails.vue',
  'src/views/Racers.vue',
  'src/views/StationConfig.vue',
  'src/views/PinUnlock.vue',
  'src/views/CsvRecovery.vue',
]

describe('default view chrome tokens', () => {
  for (const file of files) {
    it(`${file} does not hardcode legacy default chrome hex`, () => {
      const src = readFileSync(join(process.cwd(), file), 'utf8')
      const style = src.split('<style')[1] ?? ''
      expect(style).not.toMatch(/#3498db|#2980b9|#1a5276|#2c3e50|#e74c3c|#c0392b/i)
    })
  }
})

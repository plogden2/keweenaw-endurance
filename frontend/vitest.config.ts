import { defineConfig, mergeConfig } from 'vitest/config'
import viteConfig from './vite.config'

export default mergeConfig(
  viteConfig,
  defineConfig({
    test: {
      globals: true,
      environment: 'jsdom',
      setupFiles: ['./src/test/setup.ts'],
      include: ['src/**/*.{test,spec}.{js,mjs,cjs,ts,mts,cts,jsx,tsx}'],
      exclude: ['node_modules', 'dist', 'e2e', 'coverage'],
      coverage: {
        reporter: ['text', 'json', 'html'],
        exclude: [
          'node_modules/',
          'src/test/',
          '**/*.config.js',
          '**/*.config.ts',
        ],
        // FR-029 / T013a: new RFID feature stores (and future composables) must stay at 100%.
        // Add paths here as feature modules land under stores/ and composables/.
        thresholds: {
          'src/stores/pinAuth.ts': {
            lines: 100,
            functions: 100,
            branches: 100,
            statements: 100,
          },
          'src/stores/station.ts': {
            lines: 100,
            functions: 100,
            branches: 100,
            statements: 100,
          },
          'src/composables/useReaderStation.ts': {
            lines: 100,
            functions: 100,
            branches: 100,
            statements: 100,
          },
        },
      },
    },
  }),
)

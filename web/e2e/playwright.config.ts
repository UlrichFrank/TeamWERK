import { defineConfig } from '@playwright/test'
import path from 'node:path'
import { fileURLToPath } from 'node:url'

const __dirname = path.dirname(fileURLToPath(import.meta.url))

// Single-Origin-Setup: die Prod-Binary (`teamwerk`) liefert die eingebettete SPA UND die
// API auf demselben Port — kein Vite-Proxy, kein zweiter Prozess, prod-nah und deterministisch.
// Voraussetzung: das Frontend-`dist` ist gebaut (der `make test-e2e`-Target macht `pnpm build`
// vorab; `go build` bettet es via embed.FS ein).
const ROOT = path.resolve(__dirname, '..', '..')
const PORT = 18080
const BASE = `http://localhost:${PORT}`

export default defineConfig({
  testDir: '.',
  reporter: 'list',
  retries: process.env.CI ? 2 : 0,
  timeout: 30_000,
  expect: { timeout: 10_000 },
  use: {
    baseURL: BASE,
    trace: 'on-first-retry',
  },
  webServer: {
    // Build der Test-Binary → deterministische DB seeden → serven (blockiert).
    command:
      'go build -o ./bin/teamwerk-e2e ./cmd/teamwerk && ' +
      './bin/teamwerk-e2e e2e-seed --db=./e2e.db && ' +
      `DB_PATH=./e2e.db PORT=${PORT} LOG_FORMAT=text JWT_SECRET=e2e-test-secret MAILER_DISABLED=true ` +
      `VIDEO_STORAGE_DIR=./storage/videos ./bin/teamwerk-e2e`,
    cwd: ROOT,
    url: `${BASE}/login`,
    reuseExistingServer: !process.env.CI,
    timeout: 120_000,
    stdout: 'pipe',
    stderr: 'pipe',
  },
})

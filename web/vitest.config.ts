import { defineConfig } from 'vitest/config'
import react from '@vitejs/plugin-react'
import { resolve } from 'path'

export default defineConfig({
  plugins: [react()],
  test: {
    // Vitest nur in src/ — die Playwright-E2E-Specs in e2e/ (import '@playwright/test')
    // dürfen NICHT vom Vitest-Runner gesammelt werden (sonst „Failed Suite" beim Laden).
    include: ['src/**/*.{test,spec}.{ts,tsx}'],
    environment: 'jsdom',
    setupFiles: ['./src/test/setup.ts'],
    globals: true,
    css: true,
    alias: {
      'virtual:pwa-register/react': resolve(__dirname, 'src/test/mocks/pwaRegister.ts'),
    },
  },
})

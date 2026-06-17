import { defineConfig } from 'vitest/config'
import react from '@vitejs/plugin-react'
import { resolve } from 'path'

export default defineConfig({
  plugins: [react()],
  test: {
    environment: 'jsdom',
    setupFiles: ['./src/test/setup.ts'],
    globals: true,
    css: true,
    alias: {
      'virtual:pwa-register/react': resolve(__dirname, 'src/test/mocks/pwaRegister.ts'),
    },
  },
})

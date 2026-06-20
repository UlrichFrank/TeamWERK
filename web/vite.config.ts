import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import { VitePWA } from 'vite-plugin-pwa'

export default defineConfig({
  plugins: [
    react(),
    VitePWA({
      strategies: 'injectManifest',
      srcDir: 'src',
      filename: 'sw.ts',
      registerType: 'autoUpdate',
      injectManifest: {
        // Keep index.html out of the precache — navigations are served via the
        // NetworkFirst 'app-shell' route in sw.ts so a deploy is picked up on
        // the next navigation instead of being pinned to the stale precache.
        globPatterns: ['**/*.{js,css,ico,png,svg,woff2}'],
      },
      manifest: {
        name: 'TeamWERK',
        short_name: 'TeamWERK',
        description: 'Team Stuttgart Verwaltungsplattform',
        theme_color: '#000000',
        background_color: '#FFFFFF',
        display: 'standalone',
        start_url: '/',
        icons: [
          {
            src: '/icons/icon-192.png',
            sizes: '192x192',
            type: 'image/png',
            purpose: 'any',
          },
          {
            src: '/icons/icon-512.png',
            sizes: '512x512',
            type: 'image/png',
            purpose: 'any',
          },
          {
            // Eigenes maskable Icon mit weißem Grund und Logo innerhalb der
            // 80%-Safe-Zone — verhindert die Android-Beschneidung des Randes.
            src: '/icons/icon-maskable-512.png',
            sizes: '512x512',
            type: 'image/png',
            purpose: 'maskable',
          },
        ],
      },
    }),
  ],
  server: {
    proxy: {
      '/api': 'http://localhost:8080',
    },
  },
  build: {
    outDir: '../cmd/teamwerk/web/dist',
    emptyOutDir: true,
  },
})

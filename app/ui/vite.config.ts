/// <reference types="vitest/config" />
import { fileURLToPath, URL } from 'node:url'
import tailwindcss from '@tailwindcss/vite'
import vue from '@vitejs/plugin-vue'
import { defineConfig } from 'vite'

export default defineConfig({
  plugins: [vue(), tailwindcss()],
  resolve: {
    alias: { '@': fileURLToPath(new URL('./src', import.meta.url)) },
  },
  build: {
    // The PocketBase app embeds this folder. See decision 00002.
    outDir: '../api/internal/webui/dist',
    emptyOutDir: true,
    // Every asset is a file. The Content Security Policy blocks data: URLs.
    assetsInlineLimit: 0,
  },
  server: {
    proxy: { '/v1': 'http://127.0.0.1:8090' },
  },
  test: {
    environment: 'jsdom',
    include: ['test/**/*.test.ts'],
    setupFiles: ['test/setup.ts'],
    coverage: {
      provider: 'v8',
      include: ['src/**'],
      reporter: ['text-summary'],
      // The end-to-end tests cover the pages. These limits stop a large fall.
      thresholds: { statements: 55, lines: 55 },
    },
  },
})

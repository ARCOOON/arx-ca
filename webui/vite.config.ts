import path from 'node:path'
import tailwindcss from '@tailwindcss/vite'
import vue from '@vitejs/plugin-vue'
import { defineConfig } from 'vite'

const srcDir = path.resolve(__dirname, './src')

export default defineConfig({
  plugins: [
    vue({
      features: {
        optionsAPI: false,
        prodDevtools: false,
        prodHydrationMismatchDetails: false,
        componentIdGenerator: 'filepath',
      },
    }),
    tailwindcss(),
  ],
  resolve: {
    extensions: ['.mjs', '.js', '.ts', '.jsx', '.tsx', '.json', '.vue'],
    alias: {
      '@': srcDir,
    },
  },
  server: {
    host: '0.0.0.0',
    port: 5173,
    proxy: {
      '/api': {
        target: 'http://127.0.0.1:8080',
        changeOrigin: true,
      },
    },
  },
  build: {
    outDir: 'dist',
    emptyOutDir: true,
    rollupOptions: {
      onwarn(warning, defaultHandler) {
        if (
          warning.code === 'INVALID_ANNOTATION' &&
          warning.message.includes('@vueuse/core')
        ) {
          return
        }
        defaultHandler(warning)
      },
    },
  },
})

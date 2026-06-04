import { fileURLToPath, URL } from 'node:url'
import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import tailwindcss from '@tailwindcss/vite'

const srcDir = fileURLToPath(new URL('./src', import.meta.url))

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
})

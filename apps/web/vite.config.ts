import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import { resolve } from 'path'
import { fileURLToPath } from 'url'

const rootDir = fileURLToPath(new URL('.', import.meta.url))
const backendPort = process.env.NOVEL_GENERATER_BACKEND_PORT || '19081'
const webPort = Number(process.env.NOVEL_GENERATER_WEB_PORT || 5174)
const webHost = process.env.NOVEL_GENERATER_WEB_HOST || '0.0.0.0'

export default defineConfig({
  plugins: [vue()],
  cacheDir: '/private/tmp/novel-generater-vite-cache',
  resolve: {
    alias: {
      '@': resolve(rootDir, 'src')
    }
  },
  server: {
    port: webPort,
    strictPort: true,
    host: webHost,
    allowedHosts: true,
    proxy: {
      '/api': {
        target: `http://127.0.0.1:${backendPort}`,
        changeOrigin: true
      }
    }
  }
})

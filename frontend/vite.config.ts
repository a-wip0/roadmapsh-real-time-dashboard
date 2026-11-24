import react from '@vitejs/plugin-react'
import { defineConfig } from 'vite'

// In dev, /api/* is proxied to the Go backend (path prefix stripped), so the
// browser sees a single origin and no CORS setup is needed.
export default defineConfig({
  plugins: [react()],
  server: {
    proxy: {
      '/api': {
        target: process.env.BACKEND_URL ?? 'http://localhost:8080',
        rewrite: (path) => path.replace(/^\/api/, ''),
      },
    },
  },
})

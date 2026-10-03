import tailwindcss from '@tailwindcss/vite'
import react from '@vitejs/plugin-react'
import { defineConfig } from 'vite'

// In development the Go API runs on :8080 (go run ./cmd/server).
// Proxying /api keeps the browser on a single origin, exactly like on Vercel,
// so there is no CORS configuration anywhere.
export default defineConfig({
  plugins: [react(), tailwindcss()],
  server: {
    proxy: {
      '/api': 'http://localhost:8080',
    },
  },
})

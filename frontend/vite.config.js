import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

export default defineConfig({
  plugins: [react()],
  server: {
    proxy: {
      '/api': {
        target: 'https://staging-parkir-pintar.piresc.dev',
        changeOrigin: true,
        secure: true,
      },
      '/health': {
        target: 'https://staging-parkir-pintar.piresc.dev',
        changeOrigin: true,
        secure: true,
      },
    },
  },
})

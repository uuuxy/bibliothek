import { defineConfig } from 'vite'
import { svelte } from '@sveltejs/vite-plugin-svelte'
import tailwindcss from '@tailwindcss/vite'
import path from 'path'

// https://vite.dev/config/
export default defineConfig({
  plugins: [
    svelte(),
    tailwindcss()
  ],
  resolve: {
    alias: {
      '$lib': path.resolve('src/inventur/lib')
    }
  },
  server: {
    proxy: {
      '/login': {
        target: 'http://127.0.0.1:8081',
        changeOrigin: true,
        secure: false
      },
      '/api': {
        target: 'http://127.0.0.1:8081',
        changeOrigin: true,
        secure: false
      },
      '/uploads': {
        target: 'http://127.0.0.1:8081',
        changeOrigin: true,
        secure: false
      },
      '/events': {
        target: 'http://127.0.0.1:8081',
        changeOrigin: true,
        secure: false,
        ws: true
      }
    }
  }
})



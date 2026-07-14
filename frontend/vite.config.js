import { defineConfig } from 'vite';
import { configDefaults } from 'vitest/config';
import { svelte } from '@sveltejs/vite-plugin-svelte';
import tailwindcss from '@tailwindcss/vite';
import { VitePWA } from 'vite-plugin-pwa';
import path from 'node:path';

// https://vite.dev/config/
export default defineConfig({
	plugins: [
		svelte(),
		tailwindcss(),
		VitePWA({
			registerType: 'autoUpdate',
			injectRegister: 'auto',
			workbox: {
				globPatterns: ['**/*.{js,css,html,ico,png,svg}']
			},
			manifest: {
				name: 'Schulbibliothek-Verwaltungssystem',
				short_name: 'Bibliothek',
				description: 'Verwaltungssystem für die Schulbibliothek',
				theme_color: '#0f172a',
				background_color: '#f8fafc',
				start_url: '/',
				display: 'standalone',
				icons: [
					{
						src: 'favicon.svg',
						sizes: 'any',
						type: 'image/svg+xml',
						purpose: 'any maskable'
					}
				]
			}
		})
	],
	test: {
		// Playwright-Specs (e2e/) laufen über `npm run test:e2e`, nicht über Vitest
		exclude: [...configDefaults.exclude, 'e2e/**']
	},
	resolve: {
		alias: {
			$lib: path.resolve('src/inventur/lib')
		}
	},
	server: {
		proxy: {
			'/login': {
				target: 'http://127.0.0.1:8083',
				changeOrigin: true,
				secure: false
			},
			'/api': {
				target: 'http://127.0.0.1:8083',
				changeOrigin: true,
				secure: false
			},
			'/uploads': {
				target: 'http://127.0.0.1:8083',
				changeOrigin: true,
				secure: false
			},
			'/events': {
				target: 'http://127.0.0.1:8083',
				changeOrigin: true,
				secure: false,
				ws: true
			}
		}
	}
});

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
				globPatterns: ['**/*.{js,css,html,ico,png,svg}'],
				// Der Service Worker beantwortet JEDE Navigation aus dem Cache mit der
				// App-Shell — das ist der Sinn einer SPA-PWA, aber nur für Pfade, die auch
				// die App meinen.
				//
				// Ohne diese Liste traf es auch die Server-Antworten: Ein Klick auf "PDF
				// herunterladen" ist eine Navigation (target="_blank"), der Service Worker
				// gab statt des Berichts die App-Shell zurück, die SPA startete auf
				// /api/bestellhistorie/bericht, fand den Pfad nicht in tabToPath und landete
				// auf ihrem Standard-Reiter. Für den Benutzer: "Ich klicke auf Herunterladen
				// und lande in der Ausleihe."
				//
				// Der Unterschied war von aussen kaum zu sehen, weil dieselbe URL per fetch
				// weiterhin das PDF lieferte — nur die Navigation nicht.
				navigateFallbackDenylist: [
					/^\/api\//,
					/^\/uploads\//,
					/^\/events$/,
					/^\/health$/,
					/^\/swagger/
				]
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

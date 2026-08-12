import { defineConfig } from 'vite';
import { configDefaults } from 'vitest/config';
import { svelte } from '@sveltejs/vite-plugin-svelte';
import tailwindcss from '@tailwindcss/vite';
import { VitePWA } from 'vite-plugin-pwa';
import path from 'node:path';

// Wohin der Entwicklungs-Server durchreicht.
//
// Bis zum 12.08.2026 stand hier fest `8083`. Das ist der Port des PRODUKTIONS-Stacks
// (docker-compose.yml); der lokale Stack läuft auf 8084 (docker-compose.local.yml) und
// `.env.example` setzt für den Start von Hand sogar 8081. Wer der dokumentierten
// Anleitung folgte, bekam auf http://localhost:5173 also eine Oberfläche, deren
// API-Aufrufe alle ins Leere liefen — ohne Fehlermeldung, die auf den Port zeigt.
//
// Vorgabe ist jetzt der lokale Stack. Wer sein Backend woanders hat, überschreibt:
//   VITE_API_TARGET=http://127.0.0.1:8081 npm run dev
const apiZiel = process.env.VITE_API_TARGET || 'http://127.0.0.1:8084';
const durchreichen = { target: apiZiel, changeOrigin: true, secure: false };

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
			'/login': durchreichen,
			'/api': durchreichen,
			'/uploads': durchreichen,
			'/events': { ...durchreichen, ws: true }
		}
	}
});

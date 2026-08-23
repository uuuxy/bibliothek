import { defineConfig } from 'vitest/config';
import { svelte } from '@sveltejs/vite-plugin-svelte';
import tailwindcss from '@tailwindcss/vite';
import path from 'node:path';

export default defineConfig({
	plugins: [svelte(), tailwindcss()],
	resolve: {
		alias: {
			$lib: path.resolve('src/inventur/lib')
		},
		// Ohne die browser-Condition löst Vitest Svelte auf den Server-Build auf; ein
		// render() scheitert dann mit „mount(...) is not available on the server", obwohl
		// environment: 'jsdom' gesetzt ist. Nötig, sobald Komponenten getestet werden.
		conditions: ['browser']
	},
	test: {
		include: ['src/**/*.{test,spec}.{js,ts}'],
		environment: 'jsdom',
		globals: true,
		// Coverage als lcov für SonarQube. Ohne diesen Bericht zählt dort JEDE Frontend-Zeile
		// als ungedeckt — SonarQube wertet fehlende Coverage als 0 %, nicht als „unbekannt“
		// (dieselbe Falle wie 2026-08-04 auf der Go-Seite). Seit die erste Analyse eine
		// Vergleichsbasis liefert, ist das Quality Gate scharf und riss genau daran.
		//
		// Gemessen wird nur ausgelieferter Code. Ausgenommen sind Testdateien selbst, der
		// Einstiegspunkt und die Hygiene-Ratschen (die lesen den Quelltext, statt Verhalten
		// zu haben). .svelte steht bewusst NICHT im include: SonarQube kann Svelte ohnehin
		// nicht parsen, ein lcov-Eintrag dafür wäre eine Zahl ohne Gegenstück im Bericht.
		coverage: {
			provider: 'v8',
			reporter: ['text-summary', 'lcov'],
			reportsDirectory: './coverage',
			include: ['src/**/*.js'],
			exclude: ['src/**/*.test.js', 'src/main.js', 'src/**/hygiene-quellen.js']
		}
	}
});

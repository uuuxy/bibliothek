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
		// Lücken von jsdom, einmal gestopft — siehe vitest.setup.js.
		setupFiles: ['./vitest.setup.js'],
		globals: true,
		// Coverage als lcov für SonarQube. Ohne diesen Bericht zählt dort jede Frontend-Zeile
		// als ungedeckt — SonarQube wertet fehlende Coverage als 0 %, nicht als „unbekannt“ —,
		// und das Quality Gate reißt daran.
		//
		// Gemessen wird nur ausgelieferter Code. Ausgenommen sind Testdateien selbst, der
		// Einstiegspunkt und die Hygiene-Ratschen (die lesen den Quelltext, statt Verhalten
		// zu haben). .svelte steht nicht im include: SonarQube kann Svelte ohnehin
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

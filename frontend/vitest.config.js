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
		globals: true
	}
});

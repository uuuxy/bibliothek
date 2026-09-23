import { describe, it, expect, vi, afterEach } from 'vitest';
import { render, fireEvent } from '@testing-library/svelte';
import OpacSearch from './OpacSearch.svelte';

// Der öffentliche Katalog zeigt höchstens 50 Titel (api/opac.go). Bis zum 23.09.2026 stand
// über der Liste dann „50 Treffer" — die Kappung des Servers, nicht die Zahl der Bücher.
// Seitdem nennt der Server alle im Kopf X-Treffer-Gesamt.

/** @param {number} anzahl @param {number} gesamt */
function antwort(anzahl, gesamt) {
	const titel = Array.from({ length: anzahl }, (_, i) => ({
		id: `t${i}`,
		titel: `Buch ${i}`,
		autor: '',
		verfuegbar: 1,
		gesamt: 1
	}));
	return {
		ok: true,
		json: async () => titel,
		headers: new Headers({ 'X-Treffer-Gesamt': String(gesamt) })
	};
}

/** @param {any} screen @param {string} text */
async function suche(screen, text) {
	await fireEvent.input(screen.getByRole('searchbox', { name: 'Im Medienkatalog suchen' }), {
		target: { value: text }
	});
	await vi.advanceTimersByTimeAsync(400);
}

describe('OpacSearch: Zahl der Treffer', () => {
	afterEach(() => {
		vi.useRealTimers();
		vi.unstubAllGlobals();
	});

	it('nennt die Gesamtzahl, wenn der Katalog bei 50 abschneidet', async () => {
		vi.useFakeTimers();
		vi.stubGlobal(
			'fetch',
			vi.fn(async () => antwort(50, 312))
		);
		const screen = render(OpacSearch);
		await suche(screen, 'Buch');
		expect(screen.getByText(/Gezeigt werden 50 von 312 Treffern/)).toBeTruthy();
	});

	it('schreibt die Zahl allein, wenn alle Treffer da sind', async () => {
		vi.useFakeTimers();
		vi.stubGlobal(
			'fetch',
			vi.fn(async () => antwort(3, 3))
		);
		const screen = render(OpacSearch);
		await suche(screen, 'Buch');
		expect(screen.getByText('3 Treffer')).toBeTruthy();
		expect(screen.queryByText(/Gezeigt werden/)).toBeNull();
	});
});

import { describe, it, expect, vi, afterEach } from 'vitest';
import { render, fireEvent } from '@testing-library/svelte';
import OpacSuche from './OpacSuche.svelte';
import { programmstand } from '../../stores/programmstand.svelte.js';

// Der öffentliche Katalog zeigt höchstens 50 Titel (api/portal/opac.go). Bis zum 23.09.2026 stand
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

describe('OpacSuche: Zahl der Treffer', () => {
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
		const screen = render(OpacSuche);
		await suche(screen, 'Buch');
		expect(screen.getByText(/Gezeigt werden 50 von 312 Treffern/)).toBeTruthy();
	});

	it('schreibt die Zahl allein, wenn alle Treffer da sind', async () => {
		vi.useFakeTimers();
		vi.stubGlobal(
			'fetch',
			vi.fn(async () => antwort(3, 3))
		);
		const screen = render(OpacSuche);
		await suche(screen, 'Buch');
		expect(screen.getByText('3 Treffer')).toBeTruthy();
		expect(screen.queryByText(/Gezeigt werden/)).toBeNull();
	});
});

// Der Katalog hat keine Anmeldung und keine Live-Leitung: Von einem Update erfährt ein offenes
// Fenster nur aus der Antwort auf die Suche.
describe('OpacSuche: neue Fassung', () => {
	afterEach(() => {
		vi.useRealTimers();
		vi.unstubAllGlobals();
		programmstand.eigener = '';
		programmstand.server = '';
	});

	it('zeigt nach der Suche den Hinweis, wenn der Server einen anderen Stand nennt', async () => {
		vi.useFakeTimers();
		programmstand.eigener = '/assets/index-alt111.js';
		const res = antwort(3, 3);
		res.headers.set('X-Oberflaechenstand', '/assets/index-neu222.js');
		vi.stubGlobal(
			'fetch',
			vi.fn(async () => res)
		);
		const screen = render(OpacSuche);
		expect(screen.queryByRole('status')).toBeNull();

		await suche(screen, 'Buch');
		expect(screen.getByRole('status').textContent).toContain(
			'Eine neue Fassung des Programms ist eingespielt.'
		);
		expect(screen.getByText('3 Treffer')).toBeTruthy();
	});
});

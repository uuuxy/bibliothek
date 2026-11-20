import { describe, it, expect, vi, afterEach } from 'vitest';
import { render, fireEvent } from '@testing-library/svelte';
import OpacSuche from './OpacSuche.svelte';
import { programmstand } from '../../stores/programmstand.svelte.js';

// Der öffentliche Katalog zeigt höchstens 50 Titel (internal/api/portal/opac.go). Bis zum 23.09.2026 stand
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
		expect(screen.getByText(/50 von 312 Treffern/)).toBeTruthy();
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
		expect(screen.queryByText(/von 312 Treffern/)).toBeNull();
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
			'Eine neue Fassung des Programms ist eingespielt'
		);
		expect(screen.getByText('3 Treffer')).toBeTruthy();
	});
});

// Scheiterte die Suche, stand „Keine Titel gefunden" da: Der Katalog hätte das Buch nicht,
// während nur der Server nicht antwortete.
describe('OpacSuche: die Suche scheitert', () => {
	afterEach(() => {
		vi.useRealTimers();
		vi.unstubAllGlobals();
	});

	it('nennt den Grund des Servers statt „Keine Titel gefunden" und sucht auf Wunsch neu', async () => {
		vi.useFakeTimers();
		const abruf = vi.fn(async () => ({
			ok: false,
			status: 503,
			headers: new Headers(),
			text: async () => JSON.stringify({ error: 'Die Datenbank antwortet nicht' })
		}));
		vi.stubGlobal('fetch', abruf);
		const screen = render(OpacSuche);
		await suche(screen, 'Buch');

		expect(screen.getByText('Katalog nicht geladen')).toBeTruthy();
		expect(screen.getByText('Die Datenbank antwortet nicht')).toBeTruthy();
		expect(screen.queryByText('Keine Titel gefunden')).toBeNull();

		abruf.mockImplementation(async () => /** @type {any} */ (antwort(3, 3)));
		await fireEvent.click(screen.getByRole('button', { name: 'Erneut versuchen' }));
		await vi.waitFor(() => expect(screen.getByText('3 Treffer')).toBeTruthy());
		expect(screen.queryByText('Katalog nicht geladen')).toBeNull();
	});

	it('sagt ohne Netz auf Deutsch, dass der Server nicht erreichbar ist', async () => {
		vi.useFakeTimers();
		vi.stubGlobal(
			'fetch',
			vi.fn(async () => {
				throw new TypeError('Failed to fetch');
			})
		);
		const screen = render(OpacSuche);
		await suche(screen, 'Buch');

		expect(screen.getByText('Keine Verbindung zum Server')).toBeTruthy();
		expect(screen.queryByText(/Failed to fetch/)).toBeNull();
		expect(screen.queryByText('Keine Titel gefunden')).toBeNull();
	});
});

// Eine langsame ältere Suche kam nach einer neueren an und überschrieb deren Treffer: Wer „Ma“
// tippte, kurz wartete und „Mathematik 7“ fertig schrieb, sah am Ende die Treffer für „Ma“.
describe('OpacSuche: Reihenfolge der Antworten', () => {
	afterEach(() => {
		vi.useRealTimers();
		vi.unstubAllGlobals();
	});

	it('zeigt die Treffer der neuesten Suche, auch wenn eine ältere später antwortet', async () => {
		vi.useFakeTimers();
		/** @type {(wert: any) => void} */
		let alteAntwort = () => {};
		const titel = (/** @type {string} */ name) => ({
			ok: true,
			json: async () => [{ id: name, titel: name, autor: '', verfuegbar: 1, gesamt: 1 }],
			headers: new Headers({ 'X-Treffer-Gesamt': '1' })
		});
		vi.stubGlobal(
			'fetch',
			vi
				.fn()
				.mockImplementationOnce(() => new Promise((erfuelle) => (alteAntwort = erfuelle)))
				.mockImplementationOnce(async () => titel('Mathematik 7 Arbeitsheft'))
		);
		const screen = render(OpacSuche);
		await suche(screen, 'Ma');
		await suche(screen, 'Mathematik 7');
		expect(screen.getByRole('heading', { name: 'Mathematik 7 Arbeitsheft' })).toBeTruthy();

		alteAntwort(titel('Malbuch'));
		await vi.advanceTimersByTimeAsync(0);
		expect(screen.queryByRole('heading', { name: 'Malbuch' })).toBeNull();
		expect(screen.getByRole('heading', { name: 'Mathematik 7 Arbeitsheft' })).toBeTruthy();
	});
});

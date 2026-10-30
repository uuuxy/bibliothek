import { describe, it, expect, vi, beforeAll, beforeEach } from 'vitest';
import { fireEvent, render, screen, waitFor } from '@testing-library/svelte';
import Katalogseite from './KatalogSuche.svelte';
import { apiFetch } from '../../lib/apiFetch.js';

/**
 * Der ganze Weg eines Kamera-Scans im Medienkatalog, an einem Stück: KameraScanner →
 * ThekenKamera → Suchpille → StartseitenFilter → Seite → Kachelraster.
 *
 * Grund für diesen Test (Rückmeldung 18.09.2026): „wenn ich bei medienkatalog die kamera
 * verwende passiert nichts — ein Fenster öffnet sich eine Millisekunde und dann nichts."
 * Genau das steht hier: gescannt wird die EAN vom Buchrücken, im Bestand steht dieselbe
 * ISBN mit Bindestrichen. Das Raster hat keinen Leer-Zustand — „nichts gefunden" sieht
 * deshalb aus wie „nichts passiert".
 *
 * Ersetzt wird NUR der Erkenner (er braucht eine Kamera) und das Laden der Bücher. Der
 * Weg dazwischen läuft echt; er ist die Frage.
 */
const GESCANNT = '9783060130764';

vi.mock('$lib/components/scanner/barcodeErkennung.js', async (importOriginal) => ({
	...(await importOriginal()),
	erzeugeBarcodeErkenner: async () => ({
		name: 'test',
		detector: { detect: async () => [{ rawValue: '9783060130764' }] }
	})
}));

// Die Frage nach einem Exemplar (GET /api/buecher/exemplare/erkennen); ohne eigene Antwort
// trägt keine Nummer ein Exemplar.
vi.mock('../../lib/apiFetch.js', async (importOriginal) => ({
	.../** @type {any} */ (await importOriginal()),
	apiFetch: vi.fn(async () => ({ ok: true, json: async () => ({ treffer: null }) }))
}));

vi.mock('$lib/startseiten_api.js', async (importOriginal) => ({
	...(await importOriginal()),
	buecherLaden: async () => [
		{ id: 'b1', title: 'Green Line 3', isbn: '978-3-06-013076-4', author: 'Cornelsen' },
		{ id: 'b2', title: 'Ganz anderes Buch', isbn: '9783127335712', author: 'Klett' }
	]
}));

beforeAll(() => {
	// Lücken von jsdom, die nur dieser Weg braucht: eine Kamera und ein Video, das spielt.
	Object.defineProperty(navigator, 'mediaDevices', {
		configurable: true,
		value: { getUserMedia: async () => ({ getTracks: () => [] }) }
	});
	HTMLMediaElement.prototype.play = vi.fn().mockResolvedValue(undefined);
	Object.defineProperty(HTMLMediaElement.prototype, 'readyState', {
		configurable: true,
		get: () => 2
	});
});

// Zurück auf „kein Exemplar": Die Antwort eines Falls gälte sonst im nächsten weiter.
beforeEach(() => {
	vi.mocked(apiFetch).mockReset();
});

describe('Kamera-Scan im Medienkatalog', () => {
	it('legt den gescannten Code ins Suchfeld und zeigt den Titel mit dieser ISBN', async () => {
		render(Katalogseite);
		const feld = /** @type {HTMLInputElement} */ (
			await screen.findByLabelText(/Suchen nach Titel/)
		);

		screen.getByRole('button', { name: /Kamera-Barcode-Scanner/ }).click();
		// Erste Hälfte: Der Code kommt überhaupt an — über drei Bindungen hinweg.
		await waitFor(() => expect(feld.value).toBe(GESCANNT), { timeout: 4000 });

		// Zweite Hälfte, und die eigentliche Beschwerde: Es muss auch etwas zu sehen sein.
		// Im Bestand steht „978-3-06-013076-4", der Scanner liefert Ziffern ohne Trenner.
		expect(screen.getByText('Green Line 3')).toBeTruthy();
		expect(screen.queryByText('Ganz anderes Buch')).toBeNull();
	});

	it('sagt es, wenn zum gesuchten Code nichts im Bestand steht', async () => {
		render(Katalogseite);
		const feld = /** @type {HTMLInputElement} */ (
			await screen.findByLabelText(/Suchen nach Titel/)
		);

		// Ohne diese Meldung sähe „nicht im Bestand" aus wie „die Kamera hat nichts gelesen":
		// Das Kachelraster wäre schlicht leer.
		await fireEvent.input(feld, { target: { value: '9781234567897' } });

		expect(await screen.findByText('Keine Bücher gefunden')).toBeTruthy();
		expect(screen.getByText(/9781234567897/)).toBeTruthy();
		expect(screen.queryByText('Green Line 3')).toBeNull();
	});
});

// Eine Exemplarnummer steht in keiner Angabe eines Titels. Gescannt liefert ein Littera-Etikett
// nicht die aufgedruckte Nummer, sondern einen EAN-13 (101634 → 1016340039562); die Theke fand
// das Buch, der Katalog nicht. Jetzt fragt die Suche den Server und zeigt den Titel des Exemplars.
describe('Exemplarnummer im Medienkatalog', () => {
	/** @param {string} wert */
	async function sucheNach(wert) {
		render(Katalogseite);
		const feld = /** @type {HTMLInputElement} */ (
			await screen.findByLabelText(/Suchen nach Titel/)
		);
		await fireEvent.input(feld, { target: { value: wert } });
	}

	it('zeigt den Titel des Exemplars zum gescannten Littera-Etikett', async () => {
		vi.mocked(apiFetch).mockImplementation(
			async (/** @type {any} */ url) =>
				/** @type {any} */ ({
					ok: true,
					json: async () => ({
						treffer: String(url).includes('code=1016340039562')
							? { id: 'ex-1', titel_id: 'b2', barcode: '101634' }
							: null
					})
				})
		);
		await sucheNach('1016340039562');

		expect(await screen.findByText('Exemplar 101634 gehört zu „Ganz anderes Buch“.')).toBeTruthy();
		expect(screen.getByText('Ganz anderes Buch')).toBeTruthy();
		expect(screen.queryByText('Green Line 3')).toBeNull();
		expect(screen.queryByText('Keine Bücher gefunden')).toBeNull();
	});

	it('sagt, wenn sich die Nummer nicht prüfen ließ, statt „kein Exemplar“', async () => {
		vi.mocked(apiFetch).mockResolvedValue(
			/** @type {any} */ ({ ok: false, status: 500, json: async () => ({ error: 'weg' }) })
		);
		await sucheNach('101634');

		expect(await screen.findByText(/ließ sich nicht prüfen/)).toBeTruthy();
	});

	// Der Katalog führt nur Titel mit einem Exemplar im Bestand. Ein ausgesondertes Exemplar,
	// dessen Titel sonst keins hat, ist gefunden und heißt nicht „kein Exemplar".
	it('nennt ein ausgesondertes Exemplar, dessen Titel nicht in der Liste steht', async () => {
		vi.mocked(apiFetch).mockResolvedValue(
			/** @type {any} */ ({
				ok: true,
				json: async () => ({
					treffer: {
						id: 'ex-9',
						titel_id: 'nicht-in-der-liste',
						barcode: 'B-00042',
						ausgesondert: true
					}
				})
			})
		);
		await sucheNach('B-00042');

		expect(await screen.findByText(/Exemplar B-00042 ist ausgesondert/)).toBeTruthy();
		expect(screen.getByText(/unter „Ohne Exemplare“/)).toBeTruthy();
		expect(screen.queryByText(/Weder ein Titel noch ein Exemplar/)).toBeNull();
	});

	// „Nichts gefunden" vor der Antwort hieße für das Buch in der Hand: nicht im Bestand.
	it('sagt nicht „nichts gefunden“, solange die Frage nach dem Exemplar läuft', async () => {
		vi.mocked(apiFetch).mockReturnValue(new Promise(() => {}));
		await sucheNach('101634');

		await new Promise((r) => setTimeout(r, 400));
		expect(screen.queryByText('Keine Bücher gefunden')).toBeNull();
	});

	it('fragt bei einem Wort ohne Ziffer nicht nach einem Exemplar', async () => {
		vi.mocked(apiFetch).mockClear();
		await sucheNach('Green');

		expect(screen.getByText('Green Line 3')).toBeTruthy();
		await new Promise((r) => setTimeout(r, 400));
		expect(vi.mocked(apiFetch)).not.toHaveBeenCalled();
	});
});

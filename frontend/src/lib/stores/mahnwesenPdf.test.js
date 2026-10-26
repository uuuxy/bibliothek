import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';

vi.mock('./snackbars.svelte.js', () => ({
	snackbars: { addToast: vi.fn() }
}));

vi.mock('../apiFetch.js', async (importOriginal) => ({
	.../** @type {any} */ (await importOriginal()),
	apiFetch: vi.fn()
}));

import { apiFetch } from '../apiFetch.js';
import { useMahnwesenPdf } from './mahnwesenPdf.svelte.js';
import { snackbars } from './snackbars.svelte.js';
import { PDF_BLOCKIERT } from '../utils/pdfImFenster.js';

const apiFetchMock = vi.mocked(apiFetch);

/**
 * Antwort mit PDF-Blob, wie sie die Print-Endpunkte liefern.
 * @returns {any}
 */
function mockPdfResponse() {
	return { ok: true, blob: async () => new Blob(['%PDF-1.4'], { type: 'application/pdf' }) };
}

/**
 * Ein Fehler, wie ihn apierrors schickt: {"error": "…"}.
 * @param {string} meldung
 * @returns {any}
 */
function mockErrorResponse(meldung) {
	return { ok: false, status: 500, text: async () => JSON.stringify({ error: meldung }) };
}

/** @type {{ href: string, download: string }[]} */
let clicks;
/** @type {{ closed: boolean, close: any, location: { replace: any } } | null} */
let fenster;
beforeEach(() => {
	vi.clearAllMocks();
	clicks = [];
	// jsdom kennt createObjectURL und window.open nicht. Der Spion am Anker hält fest, dass
	// nichts als Datei heruntergeladen wird.
	URL.createObjectURL = vi.fn(() => 'blob:mock-url');
	URL.revokeObjectURL = vi.fn();
	fenster = { closed: false, close: vi.fn(), location: { replace: vi.fn() } };
	vi.stubGlobal(
		'open',
		vi.fn(() => fenster)
	);
	vi.spyOn(HTMLAnchorElement.prototype, 'click').mockImplementation(
		/** @this {HTMLAnchorElement} */ function () {
			clicks.push({ href: this.href, download: this.download });
		}
	);
});

afterEach(() => {
	vi.restoreAllMocks();
	vi.unstubAllGlobals();
});

describe('useMahnwesenPdf.druckeGewaehlteMahnungen', () => {
	const schuelerListe = [
		{
			schueler_id: 's1',
			medien: [{ ausleihe_id: 'a1' }, { ausleihe_id: 'a2' }, { titel: 'ohne ausleihe_id' }]
		},
		{ schueler_id: 's2', medien: [{ ausleihe_id: 'a3' }] },
		{ schueler_id: 's3', medien: [] }
	];

	it('tut nichts bei leerer Auswahl', async () => {
		const store = useMahnwesenPdf();
		await store.druckeGewaehlteMahnungen(new Set(), () => schuelerListe, vi.fn());
		expect(apiFetch).not.toHaveBeenCalled();
	});

	it('sammelt die ausleih_ids der Auswahl, zeigt das PDF im Tab und räumt auf', async () => {
		apiFetchMock.mockResolvedValueOnce(mockPdfResponse());
		const store = useMahnwesenPdf();
		const gewaehlteKennungen = new Set(['s1', 's2']);
		const refreshData = vi.fn(async () => {});

		await store.druckeGewaehlteMahnungen(gewaehlteKennungen, () => schuelerListe, refreshData);

		expect(apiFetch).toHaveBeenCalledWith('/api/admin/mahnungen/bulk-print', {
			method: 'POST',
			body: JSON.stringify({ ausleih_ids: ['a1', 'a2', 'a3'] })
		});
		expect(fenster?.location.replace).toHaveBeenCalledWith('blob:mock-url');
		expect(clicks, 'die Briefe tragen Namen und Anschriften; keine Datei').toEqual([]);
		expect(gewaehlteKennungen.size).toBe(0);
		expect(refreshData).toHaveBeenCalledTimes(1);
		expect(store.drucktPdf).toBe(false);
	});

	// Der Druck zählt die Mahnung (api/mahnwesen/mahnbriefe.go). Ginge die Anfrage hinaus, obwohl
	// der Tab blockiert ist, wäre gemahnt, ohne dass jemand den Brief sieht.
	it('fragt bei blockiertem Tab nicht an und behält die Auswahl', async () => {
		fenster = null;
		const store = useMahnwesenPdf();
		const gewaehlteKennungen = new Set(['s2']);
		const refreshData = vi.fn();

		await store.druckeGewaehlteMahnungen(gewaehlteKennungen, () => schuelerListe, refreshData);

		expect(apiFetch).not.toHaveBeenCalled();
		expect(snackbars.addToast).toHaveBeenCalledWith(PDF_BLOCKIERT, 'warning');
		expect(gewaehlteKennungen.size).toBe(1);
		expect(refreshData).not.toHaveBeenCalled();
		expect(store.drucktPdf).toBe(false);
	});

	it('meldet eine Auswahl ohne überfällige Medien, ohne den Server zu rufen', async () => {
		const store = useMahnwesenPdf();
		const gewaehlteKennungen = new Set(['s3']);

		await store.druckeGewaehlteMahnungen(gewaehlteKennungen, () => schuelerListe, vi.fn());

		expect(apiFetch).not.toHaveBeenCalled();
		expect(snackbars.addToast).toHaveBeenCalledWith(
			'Keine überfälligen Medien für die ausgewählten Schüler gefunden.',
			'info'
		);
		expect(store.drucktPdf).toBe(false);
	});

	it('setzt drucktPdf während des Ladens und danach zurück', async () => {
		/** @type {(value: any) => void} */
		let resolveFetch = () => {};
		apiFetchMock.mockImplementationOnce(
			() =>
				new Promise((res) => {
					resolveFetch = res;
				})
		);
		const store = useMahnwesenPdf();

		const pending = store.druckeGewaehlteMahnungen(new Set(['s2']), () => schuelerListe, vi.fn());
		expect(store.drucktPdf).toBe(true);

		resolveFetch(mockPdfResponse());
		await pending;
		expect(store.drucktPdf).toBe(false);
	});

	it('behält Auswahl und Daten bei einem Serverfehler', async () => {
		apiFetchMock.mockResolvedValueOnce(mockErrorResponse('Drucker brennt'));
		const store = useMahnwesenPdf();
		const gewaehlteKennungen = new Set(['s2']);
		const refreshData = vi.fn();

		await store.druckeGewaehlteMahnungen(gewaehlteKennungen, () => schuelerListe, refreshData);

		// Die Meldung des Servers als Satz, ohne Klammern und ohne „Fehler: Error:" davor.
		expect(snackbars.addToast).toHaveBeenCalledTimes(1);
		expect(snackbars.addToast).toHaveBeenCalledWith('Drucker brennt', 'error');
		expect(gewaehlteKennungen.size).toBe(1); // Auswahl bleibt für einen zweiten Versuch
		expect(refreshData).not.toHaveBeenCalled();
		expect(store.drucktPdf).toBe(false);
	});
});

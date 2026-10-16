import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';

vi.mock('./snackbars.svelte.js', () => ({
	snackbars: { addToast: vi.fn() }
}));

vi.mock('../apiFetch.js', () => ({
	apiFetch: vi.fn()
}));

import { apiFetch } from '../apiFetch.js';
import { useMahnwesenPdf } from './mahnwesenPdf.svelte.js';
import { snackbars } from './snackbars.svelte.js';

const apiFetchMock = vi.mocked(apiFetch);

/**
 * Antwort mit PDF-Blob, wie sie die Print-Endpunkte liefern.
 * @returns {any}
 */
function mockPdfResponse() {
	return { ok: true, blob: async () => new Blob(['%PDF-1.4'], { type: 'application/pdf' }) };
}

/**
 * @param {string} text
 * @returns {any}
 */
function mockErrorResponse(text) {
	return { ok: false, statusText: 'Internal Server Error', text: async () => text };
}

/** @type {{ href: string, download: string }[]} */
let clicks;
/** @type {ReturnType<typeof vi.fn>} */
beforeEach(() => {
	vi.clearAllMocks();
	clicks = [];
	// jsdom kennt createObjectURL nicht; der Anchor-Spy fängt den Download ab,
	// ohne document.createElement zu verbiegen.
	URL.createObjectURL = vi.fn(() => 'blob:mock-url');
	URL.revokeObjectURL = vi.fn();
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

const heute = () => new Date().toISOString().slice(0, 10);

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

	it('sammelt die ausleih_ids der Auswahl, lädt das PDF und räumt auf', async () => {
		apiFetchMock.mockResolvedValueOnce(mockPdfResponse());
		const store = useMahnwesenPdf();
		const gewaehlteKennungen = new Set(['s1', 's2']);
		const refreshData = vi.fn(async () => {});

		await store.druckeGewaehlteMahnungen(gewaehlteKennungen, () => schuelerListe, refreshData);

		expect(apiFetch).toHaveBeenCalledWith('/api/admin/mahnungen/bulk-print', {
			method: 'POST',
			body: JSON.stringify({ ausleih_ids: ['a1', 'a2', 'a3'] })
		});
		expect(clicks).toEqual([{ href: 'blob:mock-url', download: `mahnbriefe_${heute()}.pdf` }]);
		expect(gewaehlteKennungen.size).toBe(0);
		expect(refreshData).toHaveBeenCalledTimes(1);
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

		expect(snackbars.addToast).toHaveBeenCalledTimes(1);
		expect(String(vi.mocked(snackbars.addToast).mock.calls[0][0])).toContain('Drucker brennt');
		expect(gewaehlteKennungen.size).toBe(1); // Auswahl bleibt für einen zweiten Versuch
		expect(refreshData).not.toHaveBeenCalled();
		expect(store.drucktPdf).toBe(false);
	});
});

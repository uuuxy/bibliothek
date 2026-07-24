import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';

vi.mock('./toastStore.svelte.js', () => ({
	toastStore: { addToast: vi.fn() }
}));

vi.mock('../apiFetch.js', () => ({
	apiFetch: vi.fn()
}));

import { apiFetch } from '../apiFetch.js';
import { useMahnwesenPdf } from './mahnwesenPdf.svelte.js';
import { toastStore } from './toastStore.svelte.js';

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

describe('useMahnwesenPdf.downloadPDF', () => {
	it('lädt die globale Mahnliste und stößt den Browser-Download an', async () => {
		apiFetchMock.mockResolvedValueOnce(mockPdfResponse());
		const store = useMahnwesenPdf();

		await store.downloadPDF();

		expect(apiFetch).toHaveBeenCalledWith('/api/mahnwesen/pdf');
		expect(clicks).toEqual([{ href: 'blob:mock-url', download: `mahnliste_${heute()}.pdf` }]);
		expect(URL.revokeObjectURL).toHaveBeenCalledWith('blob:mock-url');
		expect(store.pdfLoading).toBe(false);
	});

	it('setzt pdfLoading während des Ladens und danach zurück', async () => {
		/** @type {(value: any) => void} */
		let resolveFetch = () => {};
		apiFetchMock.mockImplementationOnce(
			() =>
				new Promise((res) => {
					resolveFetch = res;
				})
		);
		const store = useMahnwesenPdf();

		const pending = store.downloadPDF();
		expect(store.pdfLoading).toBe(true);

		resolveFetch(mockPdfResponse());
		await pending;
		expect(store.pdfLoading).toBe(false);
	});

	it('meldet Fehler per alert und lädt nichts herunter', async () => {
		apiFetchMock.mockResolvedValueOnce(mockErrorResponse(''));
		const store = useMahnwesenPdf();

		await store.downloadPDF();

		expect(clicks).toEqual([]);
		expect(toastStore.addToast).toHaveBeenCalledTimes(1);
		expect(store.pdfLoading).toBe(false);
	});
});

describe('useMahnwesenPdf.printSelectedMahnungen', () => {
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
		await store.printSelectedMahnungen(new Set(), () => schuelerListe, vi.fn());
		expect(apiFetch).not.toHaveBeenCalled();
	});

	it('sammelt die ausleih_ids der Auswahl, lädt das PDF und räumt auf', async () => {
		apiFetchMock.mockResolvedValueOnce(mockPdfResponse());
		const store = useMahnwesenPdf();
		const selectedIds = new Set(['s1', 's2']);
		const refreshData = vi.fn(async () => {});

		await store.printSelectedMahnungen(selectedIds, () => schuelerListe, refreshData);

		expect(apiFetch).toHaveBeenCalledWith('/api/admin/mahnungen/bulk-print', {
			method: 'POST',
			body: JSON.stringify({ ausleih_ids: ['a1', 'a2', 'a3'] })
		});
		expect(clicks).toEqual([{ href: 'blob:mock-url', download: `Mahnliste_Bulk_${heute()}.pdf` }]);
		expect(selectedIds.size).toBe(0);
		expect(refreshData).toHaveBeenCalledTimes(1);
		expect(store.pdfLoading).toBe(false);
	});

	it('meldet eine Auswahl ohne überfällige Medien, ohne den Server zu rufen', async () => {
		const store = useMahnwesenPdf();
		const selectedIds = new Set(['s3']);

		await store.printSelectedMahnungen(selectedIds, () => schuelerListe, vi.fn());

		expect(apiFetch).not.toHaveBeenCalled();
		expect(toastStore.addToast).toHaveBeenCalledWith(
			'Keine überfälligen Medien für die ausgewählten Schüler gefunden.',
			'info'
		);
		expect(store.pdfLoading).toBe(false);
	});

	it('behält Auswahl und Daten bei einem Serverfehler', async () => {
		apiFetchMock.mockResolvedValueOnce(mockErrorResponse('Drucker brennt'));
		const store = useMahnwesenPdf();
		const selectedIds = new Set(['s2']);
		const refreshData = vi.fn();

		await store.printSelectedMahnungen(selectedIds, () => schuelerListe, refreshData);

		expect(toastStore.addToast).toHaveBeenCalledTimes(1);
		expect(String(vi.mocked(toastStore.addToast).mock.calls[0][0])).toContain('Drucker brennt');
		expect(selectedIds.size).toBe(1); // Auswahl bleibt für einen zweiten Versuch
		expect(refreshData).not.toHaveBeenCalled();
		expect(store.pdfLoading).toBe(false);
	});
});

describe('useMahnwesenPdf.downloadElternPDF', () => {
	it('lädt die Eltern-Mahnbriefe herunter', async () => {
		apiFetchMock.mockResolvedValueOnce(mockPdfResponse());
		const store = useMahnwesenPdf();

		await store.downloadElternPDF();

		expect(apiFetch).toHaveBeenCalledWith('/api/reports/overdue-pdf');
		expect(clicks).toEqual([{ href: 'blob:mock-url', download: `mahnbriefe_${heute()}.pdf` }]);
		expect(store.globalErrorToast).toBeNull();
		expect(store.elternPdfLoading).toBe(false);
	});

	it('zeigt Fehler als Toast und blendet ihn nach 4s wieder aus', async () => {
		vi.useFakeTimers();
		try {
			apiFetchMock.mockResolvedValueOnce(mockErrorResponse('Keine Adressdaten'));
			const store = useMahnwesenPdf();

			await store.downloadElternPDF();

			expect(store.globalErrorToast).toContain('Keine Adressdaten');
			expect(store.elternPdfLoading).toBe(false);

			await vi.advanceTimersByTimeAsync(4000);
			expect(store.globalErrorToast).toBeNull();
		} finally {
			vi.useRealTimers();
		}
	});
});

describe('useMahnwesenPdf.downloadKlassePDF', () => {
	it('tut nichts ohne ausgewählte Klasse', async () => {
		const store = useMahnwesenPdf();
		await store.downloadKlassePDF('');
		expect(apiFetch).not.toHaveBeenCalled();
	});

	it('lädt das Klassen-PDF mit der Klasse im Pfad und Dateinamen', async () => {
		apiFetchMock.mockResolvedValueOnce(mockPdfResponse());
		const store = useMahnwesenPdf();

		await store.downloadKlassePDF('4a');

		expect(apiFetch).toHaveBeenCalledWith('/api/print/mahnung/klasse/4a');
		expect(clicks).toEqual([{ href: 'blob:mock-url', download: 'Mahnliste_Klasse_4a.pdf' }]);
		expect(store.klassePdfLoading).toBe(false);
	});

	it('nutzt die Fallback-Meldung, wenn der Server keinen Fehlertext liefert', async () => {
		apiFetchMock.mockResolvedValueOnce(mockErrorResponse(''));
		const store = useMahnwesenPdf();

		await store.downloadKlassePDF('4a');

		expect(store.globalErrorToast).toContain('Keine überfälligen Ausleihen gefunden');
		expect(store.klassePdfLoading).toBe(false);
	});
});

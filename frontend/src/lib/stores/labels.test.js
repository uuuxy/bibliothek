import { describe, it, expect, vi, beforeEach } from 'vitest';

vi.mock('./toastStore.svelte.js', () => ({
	toastStore: { addToast: vi.fn() }
}));

vi.mock('../apiFetch.js', () => ({
	apiFetch: vi.fn()
}));

import { apiFetch } from '../apiFetch.js';
import { labelStore } from './labels.svelte.js';

const apiFetchMock = vi.mocked(apiFetch);

/** @param {string[]} nummern @returns {any} */
const exemplare = (nummern) => ({
	ok: true,
	json: async () => nummern.map((barcode_id) => ({ barcode_id }))
});

/** Die Nummern der Etiketten, die auf den Bogen kämen. */
const aufDemBogen = () => labelStore.finalLabels.map((l) => l.barcode_id);

// Schritt 2 des Druck-Centers hakt die vorhandenen Exemplare eines Titels vor; was abgewählt
// wird, darf weder in der Vorschau noch im Druck stehen. Das Kästchen schreibt `checked` an
// das Exemplar selbst (bind:checked), der Bogen liest es dort.
describe('labelStore: vorhandene Exemplare abwählen', () => {
	beforeEach(async () => {
		vi.clearAllMocks();
		apiFetchMock.mockResolvedValueOnce(exemplare(['B-1', 'B-2', 'B-3']));
		labelStore.generationMode = 'existing';
		await labelStore.selectBookTitle({ id: 't1', titel: 'Titel', autor: 'Autorin' });
	});

	it('hakt alle Exemplare des Titels vor', () => {
		expect(aufDemBogen()).toEqual(['B-1', 'B-2', 'B-3']);
	});

	it('nimmt ein abgewähltes Exemplar vom Bogen', () => {
		expect(aufDemBogen()).toEqual(['B-1', 'B-2', 'B-3']);
		labelStore.existingCopies[1].checked = false;
		expect(aufDemBogen()).toEqual(['B-1', 'B-3']);
	});

	it('druckt und vermerkt nur, was angehakt ist', async () => {
		expect(aufDemBogen()).toHaveLength(3);
		labelStore.existingCopies[0].checked = false;

		vi.stubGlobal('open', vi.fn());
		URL.createObjectURL = vi.fn(() => 'blob:probe');
		apiFetchMock.mockResolvedValueOnce(
			/** @type {any} */ ({ ok: true, blob: async () => new Blob(['%PDF']) })
		);
		apiFetchMock.mockResolvedValueOnce(/** @type {any} */ ({ ok: true }));
		await labelStore.triggerPrint();

		const [druck, vermerk] = apiFetchMock.mock.calls.slice(-2);
		expect(druck[0]).toBe('/api/print/labels');
		expect(
			JSON.parse(String(druck[1]?.body)).items.map((/** @type {any} */ i) => i.BarcodeID)
		).toEqual(['B-2', 'B-3']);
		expect(vermerk[0]).toBe('/api/exemplare/etiketten-gedruckt');
		expect(JSON.parse(String(vermerk[1]?.body)).barcode_ids).toEqual(['B-2', 'B-3']);
		vi.unstubAllGlobals();
	});
});

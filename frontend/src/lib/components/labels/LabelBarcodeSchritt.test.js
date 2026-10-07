import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, fireEvent, waitFor } from '@testing-library/svelte';

vi.mock('../../stores/toastStore.svelte.js', () => ({
	toastStore: { addToast: vi.fn() }
}));

vi.mock('../../apiFetch.js', () => ({
	apiFetch: vi.fn()
}));

import { apiFetch } from '../../apiFetch.js';
import { labelStore } from '../../stores/labels.svelte.js';
import LabelBarcodeSchritt from './LabelBarcodeSchritt.svelte';

const apiFetchMock = vi.mocked(apiFetch);
const KEIN_EXEMPLAR = 'Zu diesem Titel gibt es kein Exemplar, das ein Etikett bekommen kann.';

// Schritt 2 unterscheidet „nicht geladen“ von „kein Exemplar“: Der zweite Satz legt nahe, neue
// Barcodes zu erzeugen, und darf nach einem gescheiterten Abruf nicht dastehen.
describe('Druck-Center, Schritt 2: Exemplare nicht geladen', () => {
	beforeEach(() => {
		vi.clearAllMocks();
		labelStore.generationMode = 'existing';
	});

	it('zeigt den Ladefehler mit „Erneut versuchen“ und lädt darauf die Liste', async () => {
		apiFetchMock.mockResolvedValueOnce(/** @type {any} */ ({ ok: false, status: 500 }));
		await labelStore.selectBookTitle({ id: 't1', titel: 'Titel', autor: '' });
		const k = render(LabelBarcodeSchritt);

		expect(k.getByRole('alert').textContent).toContain('Exemplare nicht geladen');
		expect(k.queryByText(KEIN_EXEMPLAR)).toBeNull();

		apiFetchMock.mockResolvedValueOnce(
			/** @type {any} */ ({ ok: true, json: async () => [{ barcode_id: 'B-0041873' }] })
		);
		await fireEvent.click(k.getByRole('button', { name: 'Erneut versuchen' }));

		await waitFor(() => expect(k.getByText('B-0041873')).toBeTruthy());
		expect(k.queryByRole('alert')).toBeNull();
	});

	it('nennt einen Titel ohne Exemplar weiter beim Namen', async () => {
		apiFetchMock.mockResolvedValueOnce(/** @type {any} */ ({ ok: true, json: async () => [] }));
		await labelStore.selectBookTitle({ id: 't2', titel: 'Titel', autor: '' });
		const k = render(LabelBarcodeSchritt);

		expect(k.getByText(KEIN_EXEMPLAR)).toBeTruthy();
		expect(k.queryByRole('alert')).toBeNull();
	});
});

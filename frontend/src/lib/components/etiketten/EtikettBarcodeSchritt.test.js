import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, fireEvent, waitFor } from '@testing-library/svelte';

vi.mock('../../stores/snackbars.svelte.js', () => ({
	snackbars: { addToast: vi.fn() }
}));

// leseApiFehler bleibt echt: Ohne ihn wirft schon das Lesen einer Ablehnung (ARCHITEKTUR.md 8.14).
vi.mock('../../apiFetch.js', async (importOriginal) => ({
	.../** @type {any} */ (await importOriginal()),
	apiFetch: vi.fn()
}));

import { apiFetch } from '../../apiFetch.js';
import { etikettenStore } from '../../stores/etiketten.svelte.js';
import EtikettBarcodeSchritt from './EtikettBarcodeSchritt.svelte';
import { serverAntwort } from '../../serverAntwort.js';

const apiFetchMock = vi.mocked(apiFetch);
const KEIN_EXEMPLAR = 'Zu diesem Titel gibt es kein Exemplar, das ein Etikett bekommen kann.';

// Schritt 2 unterscheidet „nicht geladen“ von „kein Exemplar“: Der zweite Satz sagt, der Titel
// habe nichts zu drucken, und darf nach einem gescheiterten Abruf nicht dastehen.
describe('Druck-Center, Schritt 2: Exemplare nicht geladen', () => {
	beforeEach(() => {
		vi.clearAllMocks();
	});

	it('zeigt den Ladefehler mit „Erneut versuchen“ und lädt darauf die Liste', async () => {
		apiFetchMock.mockResolvedValueOnce(
			serverAntwort(503, { error: 'Die Datenbank antwortet nicht' })
		);
		await etikettenStore.waehleTitel({ id: 't1', titel: 'Titel', autor: '' });
		const k = render(EtikettBarcodeSchritt);

		expect(k.getByRole('alert').textContent).toContain('Exemplare nicht geladen');
		expect(k.getByRole('alert').textContent).toContain('Die Datenbank antwortet nicht');
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
		await etikettenStore.waehleTitel({ id: 't2', titel: 'Titel', autor: '' });
		const k = render(EtikettBarcodeSchritt);

		expect(k.getByText(KEIN_EXEMPLAR)).toBeTruthy();
		expect(k.queryByRole('alert')).toBeNull();
	});
});

// Hinter der Nummer steht die Zustandsnotiz. Fehlt sie, steht dort ein Wort für den Zustand:
// „Neuwertig" am Exemplar im Bestand, „Bestellt" an einem, das noch nicht eingetroffen ist.
describe('Druck-Center, Schritt 2: Angabe hinter der Nummer', () => {
	beforeEach(() => {
		vi.clearAllMocks();
	});

	it('nennt ein bestelltes Exemplar ohne Notiz „Bestellt“', async () => {
		apiFetchMock.mockResolvedValueOnce(
			/** @type {any} */ ({
				ok: true,
				json: async () => [
					{ barcode_id: 'B-0000001', zustand_notiz: '', im_bestand: true },
					{ barcode_id: 'B-0000002', zustand_notiz: '', im_bestand: false },
					{ barcode_id: 'B-0000003', zustand_notiz: 'Im Zulauf - Buchhandlung', im_bestand: false },
					{ barcode_id: 'B-0000004', zustand_notiz: 'Einband lose', im_bestand: true }
				]
			})
		);
		await etikettenStore.waehleTitel({ id: 't3', titel: 'Titel', autor: '' });
		const k = render(EtikettBarcodeSchritt);
		await waitFor(() => expect(k.getByText('B-0000002')).toBeTruthy());

		const zeile = (/** @type {string} */ nummer) =>
			k.getByText(nummer).closest('label')?.textContent ?? '';
		expect(zeile('B-0000001')).toContain('(Neuwertig)');
		expect(zeile('B-0000002')).toContain('(Bestellt)');
		expect(zeile('B-0000003')).toContain('(Im Zulauf - Buchhandlung)');
		expect(zeile('B-0000004')).toContain('(Einband lose)');
	});
});

// Gedruckt wird nur die Nummer eines Exemplars, das es gibt: Eine Nummer, die der Browser
// abzählt, steht in keiner Zählung, und eine Bestellung könnte sie ein zweites Mal vergeben.
describe('Druck-Center, Schritt 2: nur vorhandene Exemplare', () => {
	beforeEach(() => {
		vi.clearAllMocks();
	});

	it('bietet keine Art an, Nummern ohne Exemplar zu drucken', async () => {
		apiFetchMock.mockResolvedValueOnce(
			/** @type {any} */ ({ ok: true, json: async () => [{ barcode_id: 'B-0041874' }] })
		);
		await etikettenStore.waehleTitel({ id: 't5', titel: 'Titel', autor: '' });
		const k = render(EtikettBarcodeSchritt);

		await waitFor(() => expect(k.getByText('B-0041874')).toBeTruthy());
		expect(k.queryByRole('button', { name: 'Neue Barcodes' })).toBeNull();
		expect(k.queryByLabelText(/Start-Ziffer/)).toBeNull();
		expect(k.getByRole('heading', { name: '2. Exemplare wählen' })).toBeTruthy();
	});
});

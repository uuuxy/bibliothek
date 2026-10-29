import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, fireEvent } from '@testing-library/svelte';
import { flushSync } from 'svelte';

vi.mock('../../stores/snackbars.svelte.js', () => ({
	snackbars: { addToast: vi.fn() }
}));

vi.mock('../../apiFetch.js', async (importOriginal) => ({
	.../** @type {any} */ (await importOriginal()),
	apiFetch: vi.fn()
}));

// pdf.js zeichnet in jsdom nicht; gemessen wird, welchen Bogen die Vorschau vom Server holt.
vi.mock('./pdfSeite.js', () => ({ ersteSeiteAlsBild: vi.fn() }));

import { apiFetch } from '../../apiFetch.js';
import { serverAntwort } from '../../serverAntwort.js';
import { ersteSeiteAlsBild } from './pdfSeite.js';
import { etikettenStore } from '../../stores/etiketten.svelte.js';
import EtikettVorschau from './EtikettVorschau.svelte';

const PDF = new Uint8Array([37, 80, 68, 70]).buffer;

/** Antworten nach Adresse: die Exemplare des Titels und das Druck-PDF. @param {number} anzahl */
function server(anzahl, pdfAntwort = () => ({ ok: true, arrayBuffer: async () => PDF })) {
	vi.mocked(apiFetch).mockImplementation(async (adresse) => {
		if (String(adresse).endsWith('/exemplare')) {
			return /** @type {any} */ ({
				ok: true,
				json: async () =>
					Array.from({ length: anzahl }, (_, i) => ({ barcode_id: `B-${1000 + i}` }))
			});
		}
		return /** @type {any} */ (pdfAntwort());
	});
}

/** Die Rümpfe der Aufträge an /api/print/labels. */
const auftraege = () =>
	vi
		.mocked(apiFetch)
		.mock.calls.filter(([adresse]) => adresse === '/api/print/labels')
		.map(([, optionen]) => JSON.parse(String(optionen?.body)));

/** @param {number} anzahl */
async function waehleTitelMit(anzahl) {
	await etikettenStore.waehleTitel({ id: `t${anzahl}`, titel: 'Titel', autor: 'Autorin' });
	flushSync();
}

beforeEach(() => {
	vi.clearAllMocks();
	vi.mocked(ersteSeiteAlsBild).mockResolvedValue(new Blob(['png']));
	URL.createObjectURL = vi.fn(() => 'blob:vorschau');
	URL.revokeObjectURL = vi.fn();
	etikettenStore.formatId = 'zweckform_l4760';
	etikettenStore.barcodeArt = 'code39';
	etikettenStore.startFeld = 1;
});

afterEach(() => {
	etikettenStore.formatId = 'zweckform_l4760';
	etikettenStore.barcodeArt = 'code39';
	etikettenStore.startFeld = 1;
});

// Die Vorschau ist das Druck-PDF des Servers: derselbe Auftrag wie beim Drucken, nur für den
// ersten Bogen. Eine eigene Zeichnung im Browser fehlten Schulname, Jahr und Vermerk.
describe('Druck-Center: Die Vorschau zeigt das Druck-PDF', () => {
	it('holt den ersten Bogen vom Server und zeigt seine erste Seite', async () => {
		server(2);
		const screen = render(EtikettVorschau);
		await waehleTitelMit(2);

		const blatt = await screen.findByAltText('Erster Bogen, wie er gedruckt wird');
		expect(blatt.getAttribute('src')).toBe('blob:vorschau');
		expect(auftraege()).toEqual([
			{
				formatId: 'zweckform_l4760',
				startPosition: 1,
				isQR: false,
				items: [
					{ BarcodeID: 'B-1000', Titel: 'Titel', Autor: 'Autorin' },
					{ BarcodeID: 'B-1001', Titel: 'Titel', Autor: 'Autorin' }
				]
			}
		]);
		expect(ersteSeiteAlsBild).toHaveBeenCalledWith(PDF, expect.any(Number));
	});

	it('schickt Format, Code und Startposition mit und nur die Etiketten des ersten Bogens', async () => {
		server(60);
		const screen = render(EtikettVorschau);
		await waehleTitelMit(60);
		etikettenStore.formatId = 'standard_52';
		etikettenStore.barcodeArt = 'qr';
		etikettenStore.startFeld = 3;
		flushSync();

		await screen.findByAltText('Erster Bogen, wie er gedruckt wird');
		const auftrag = auftraege().at(-1);
		expect([auftrag.formatId, auftrag.isQR, auftrag.startPosition]).toEqual([
			'standard_52',
			true,
			3
		]);
		// 52 Felder, zwei davon schon abgezogen.
		expect(auftrag.items).toHaveLength(50);
		expect(screen.getByText('Bogen 1 von 2 · 60 Etiketten')).toBeTruthy();
	});

	// Das Eingabefeld nimmt auch eine Zahl hinter dem letzten Feld an. Ohne Grenze blieb der erste
	// Bogen im Browser leer, und der Server setzte die Etiketten an eine andere Stelle.
	it('setzt eine Startposition hinter dem letzten Feld auf das letzte Feld', async () => {
		server(5);
		const screen = render(EtikettVorschau);
		await waehleTitelMit(5);
		etikettenStore.startFeld = 30;
		flushSync();

		await screen.findByAltText('Erster Bogen, wie er gedruckt wird');
		const auftrag = auftraege().at(-1);
		expect([auftrag.startPosition, auftrag.items.length]).toEqual([21, 1]);
	});

	it('fragt bei schnellen Änderungen einmal, nach der letzten', async () => {
		server(5);
		const screen = render(EtikettVorschau);
		await waehleTitelMit(5);
		for (const feld of [2, 3, 4]) {
			etikettenStore.startFeld = feld;
			flushSync();
		}

		await screen.findByAltText('Erster Bogen, wie er gedruckt wird');
		expect(auftraege().map((a) => a.startPosition)).toEqual([4]);
	});

	it('nennt den Grund, wenn der Server ablehnt, und versucht es erneut', async () => {
		let abgelehnt = true;
		server(2, () =>
			abgelehnt
				? serverAntwort(403, { error: 'Keine Berechtigung für diese Aktion' })
				: { ok: true, arrayBuffer: async () => PDF }
		);
		const screen = render(EtikettVorschau);
		await waehleTitelMit(2);

		expect(await screen.findByText('Vorschau nicht geladen')).toBeTruthy();
		expect(screen.getByText('Keine Berechtigung für diese Aktion')).toBeTruthy();
		abgelehnt = false;
		await fireEvent.click(screen.getByRole('button', { name: 'Erneut versuchen' }));
		expect(await screen.findByAltText('Erster Bogen, wie er gedruckt wird')).toBeTruthy();
		expect(auftraege()).toHaveLength(2);
	});

	it('zeigt bei einem einzigen Bogen keine Zeile darunter', async () => {
		server(21);
		const screen = render(EtikettVorschau);
		await waehleTitelMit(21);

		await screen.findByAltText('Erster Bogen, wie er gedruckt wird');
		expect(screen.queryByText(/^Bogen 1 von/)).toBeNull();
	});
});

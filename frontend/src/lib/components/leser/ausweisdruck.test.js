import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { erzeugeAusweisdruck, mitAusweisnummer } from './ausweisdruck.svelte.js';
import { idStore, vergissDesignLadezustand } from '../../designer/idDesignerStore.svelte.js';
import { apiFetch } from '../../apiFetch.js';
import { snackbars } from '../../stores/snackbars.svelte.js';

// leseApiFehler bleibt echt (ARCHITEKTUR.md 8.14).
vi.mock('../../apiFetch.js', async (importOriginal) => ({
	.../** @type {any} */ (await importOriginal()),
	apiFetch: vi.fn()
}));
vi.mock('../../stores/snackbars.svelte.js', () => ({ snackbars: { addToast: vi.fn() } }));

// Dieses Modul entscheidet am zentral gespeicherten printMode, ob Karten oder Etiketten aus
// dem Drucker kommen. Wer den Zustand liest, lädt ihn auch selbst: Die Druckfläche
// (AusweisStapeldruck) hängt nur im Baum, solange Karten markiert sind.

/** Lässt die anstehenden Microtasks (fetch → json → applyDesign) durchlaufen. */
const stillhalten = () => new Promise((fertig) => setTimeout(fertig, 0));

beforeEach(() => {
	vi.clearAllMocks();
	// Sitzungs-Merker zurück: Jeder Fall hier prüft das VERHALTEN BEIM ERSTEN LADEN.
	vergissDesignLadezustand();
	idStore.printMode = 'card';
	idStore.etikettFormat = 'zweckform_l4760';
});

describe('erzeugeAusweisdruck: lädt das zentrale Design selbst', () => {
	it('übernimmt einen zentral gespeicherten Etikettenmodus ohne fremde Hilfe', async () => {
		vi.mocked(apiFetch).mockResolvedValue(
			/** @type {any} */ ({
				ok: true,
				json: async () => ({ printMode: 'etikett', etikettFormat: 'avery_3475' })
			})
		);

		const druck = erzeugeAusweisdruck();
		await stillhalten();

		expect(apiFetch).toHaveBeenCalledWith('/api/ausweis-layout');
		expect(druck.etikettModus).toBe(true);
		expect(druck.maxPosition).toBe(24); // avery_3475: 3×8
	});

	it('sagt es laut, wenn die Druckeinstellung nicht zu laden ist', async () => {
		// Der stille Rückfall auf 'card' wäre genau der falsche Ausdruck: Karten auf
		// Kartenrohlinge, obwohl die Schule Etiketten eingestellt hat.
		vi.mocked(apiFetch).mockRejectedValue(new Error('Netz weg'));

		const druck = erzeugeAusweisdruck();
		await stillhalten();

		expect(druck.etikettModus).toBe(false);
		expect(snackbars.addToast).toHaveBeenCalledWith(
			expect.stringContaining('Ausweis-Design nicht geladen'),
			'error'
		);
	});

	it('lädt nicht erneut, wenn das Design in dieser Sitzung schon geladen wurde', async () => {
		// Der zweite GET war nicht nur überflüssig — er überschrieb frische, noch nicht
		// fertig gespeicherte Designer-Änderungen mit dem alten Serverstand.
		vi.mocked(apiFetch).mockResolvedValue(
			/** @type {any} */ ({ ok: true, json: async () => ({ printMode: 'etikett' }) })
		);

		erzeugeAusweisdruck();
		await stillhalten();
		expect(apiFetch).toHaveBeenCalledTimes(1);

		// Zweiter Bildschirm derselben Sitzung — z. B. Schülerdatei erneut geöffnet,
		// nachdem der Designer inzwischen auf 'card' zurückgestellt hat (nur im Store).
		idStore.printMode = 'card';
		const druck = erzeugeAusweisdruck();
		await stillhalten();

		expect(apiFetch).toHaveBeenCalledTimes(1);
		expect(druck.etikettModus).toBe(false); // der Store gilt, nicht der alte Serverstand
	});
});

describe('erzeugeAusweisdruck: Karten über den Druck des Browsers', () => {
	afterEach(() => vi.unstubAllGlobals());

	const seitenregelDa = () =>
		[...document.head.querySelectorAll('style')].some((s) =>
			s.textContent?.includes('85.6mm 53.98mm')
		);

	it('setzt Modus, Seite und Seitenregel für den Druck und räumt sie danach ab', async () => {
		vi.mocked(apiFetch).mockResolvedValue(
			/** @type {any} */ ({ ok: true, json: async () => ({ printMode: 'card' }) })
		);
		/** @type {{ modus: string | null, seite: string | null, seitenregel: boolean }[]} */
		const gesehen = [];
		vi.stubGlobal('print', () => {
			gesehen.push({
				modus: document.body.getAttribute('data-print-mode'),
				seite: document.body.getAttribute('data-print-side'),
				seitenregel: seitenregelDa()
			});
		});
		const druck = erzeugeAusweisdruck();
		await stillhalten();

		await druck.drucke([{ id: 's1', barcode_id: 'A-00001' }]);

		expect(gesehen).toEqual([{ modus: 'card', seite: 'front', seitenregel: true }]);
		// Bliebe etwas stehen, druckte die nächste Seite dieses Tabs im Kartenformat.
		expect(document.body.hasAttribute('data-print-mode')).toBe(false);
		expect(document.body.hasAttribute('data-print-side')).toBe(false);
		expect(seitenregelDa()).toBe(false);
	});

	// Ohne das gespeicherte Design gälte der Kartendruck mit den Standardwerten, auch wenn
	// die Schule Etiketten eingestellt hat.
	it('druckt nicht, solange das Design nicht zu laden ist', async () => {
		vi.mocked(apiFetch).mockRejectedValue(new Error('Keine Verbindung zum Server.'));
		const drucken = vi.fn();
		vi.stubGlobal('print', drucken);
		const druck = erzeugeAusweisdruck();
		await stillhalten();
		vi.mocked(snackbars.addToast).mockClear();

		await druck.drucke([{ id: 's1', barcode_id: 'A-00001' }]);

		expect(drucken).not.toHaveBeenCalled();
		expect(snackbars.addToast).toHaveBeenCalledWith(
			'Nicht gedruckt, das Ausweis-Design ist nicht geladen: Keine Verbindung zum Server.',
			'error'
		);
	});

	it('lädt das Design beim Drucken nach, wenn der erste Abruf gescheitert war', async () => {
		vi.mocked(apiFetch).mockRejectedValueOnce(new Error('Netz weg'));
		vi.mocked(apiFetch).mockResolvedValueOnce(
			/** @type {any} */ ({ ok: true, json: async () => ({ printMode: 'card' }) })
		);
		const drucken = vi.fn();
		vi.stubGlobal('print', drucken);
		const druck = erzeugeAusweisdruck();
		await stillhalten();

		await druck.drucke([{ id: 's1', barcode_id: 'A-00001' }]);

		expect(apiFetch).toHaveBeenCalledTimes(2);
		expect(drucken).toHaveBeenCalledTimes(1);
	});
});

describe('erzeugeAusweisdruck: Leser ohne Ausweisnummer', () => {
	afterEach(() => vi.unstubAllGlobals());

	const anna = { id: 's1', vorname: 'Anna', nachname: 'Berg', barcode_id: 'A-00001' };
	const katrin = { id: 'l2', vorname: 'Katrin', nachname: 'Wendlandt', barcode_id: null };

	it('liefert für den Kartendruck nur Leser mit Nummer', () => {
		expect(mitAusweisnummer([anna, katrin])).toEqual([anna]);
	});

	it('druckt die Karten der anderen und nennt, wer ohne Nummer ist', async () => {
		vi.mocked(apiFetch).mockResolvedValue(
			/** @type {any} */ ({ ok: true, json: async () => ({ printMode: 'card' }) })
		);
		const drucken = vi.fn();
		vi.stubGlobal('print', drucken);
		const druck = erzeugeAusweisdruck();
		await stillhalten();

		await druck.drucke([anna, katrin]);

		expect(drucken).toHaveBeenCalledTimes(1);
		expect(snackbars.addToast).toHaveBeenCalledWith(
			'Ohne Ausweisnummer kein Ausweis für Katrin Wendlandt. Die Nummer wird in der Akte unter „Stammdaten & Adresse“ eingetragen.',
			'warning'
		);
	});

	it('schickt dem Etikettenbogen nur Leser mit Nummer', async () => {
		vi.stubGlobal(
			'open',
			vi.fn(() => ({ closed: false, close: vi.fn(), location: { replace: vi.fn() } }))
		);
		vi.mocked(apiFetch).mockResolvedValue(
			/** @type {any} */ ({ ok: true, json: async () => ({ printMode: 'etikett' }) })
		);
		const druck = erzeugeAusweisdruck();
		await stillhalten();
		vi.mocked(apiFetch).mockClear();
		vi.mocked(apiFetch).mockResolvedValue(
			/** @type {any} */ ({ ok: false, json: async () => ({}) })
		);

		await druck.drucke([anna, katrin]);

		const bogen = vi
			.mocked(apiFetch)
			.mock.calls.find(([url]) => url === '/api/print/schueler-etiketten');
		expect(JSON.parse(String(bogen?.[1]?.body)).schuelerIds).toEqual(['s1']);
	});

	it('druckt nichts, wenn keiner eine Nummer hat', async () => {
		vi.mocked(apiFetch).mockResolvedValue(
			/** @type {any} */ ({ ok: true, json: async () => ({ printMode: 'card' }) })
		);
		const drucken = vi.fn();
		vi.stubGlobal('print', drucken);
		const druck = erzeugeAusweisdruck();
		await stillhalten();

		await druck.drucke([katrin]);

		expect(drucken).not.toHaveBeenCalled();
		expect(snackbars.addToast).toHaveBeenCalledWith(
			expect.stringContaining('Katrin Wendlandt'),
			'warning'
		);
	});
});

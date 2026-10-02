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

// Ein ausgesondertes Exemplar steht nicht mehr im Regal; ein Etikett dafür wäre immer falsch.
// Dieselbe Regel wie in der Liste „Fehlende Etiketten“ (repository.EtikettOffenBedingung).
// Bestellte Exemplare bleiben: Ihr Etikett kann vor der Lieferung gedruckt werden.
describe('labelStore: ausgesonderte Exemplare', () => {
	it('bietet sie nicht an, druckt sie nicht und nennt ihre Zahl', async () => {
		vi.clearAllMocks();
		apiFetchMock.mockResolvedValueOnce(
			/** @type {any} */ ({
				ok: true,
				json: async () => [
					{ barcode_id: 'B-1', ist_ausgesondert: false, im_bestand: true },
					{ barcode_id: 'B-2', ist_ausgesondert: true, im_bestand: false },
					{ barcode_id: 'B-3', ist_ausgesondert: false, im_bestand: false }
				]
			})
		);
		labelStore.generationMode = 'existing';
		await labelStore.selectBookTitle({ id: 't2', titel: 'Titel', autor: '' });

		expect(labelStore.existingCopies.map((c) => c.barcode_id)).toEqual(['B-1', 'B-3']);
		expect(aufDemBogen()).toEqual(['B-1', 'B-3']);
		expect(labelStore.ausgesondertAnzahl).toBe(1);
	});
});

// Über der Liste steht ein Kästchen für alle und ein Feld für die Nummer. Die Nummern stammen
// aus den gemeinsamen Prüffällen: 58968 ist ein Littera-Exemplar, dessen Etikett den
// EAN-13 5896800039556 trägt; B-100016 ist B-10001 mit dem Prüfzeichen früherer Etiketten.
describe('labelStore: alle, keine und einzelne Exemplare wählen', () => {
	beforeEach(async () => {
		vi.clearAllMocks();
		apiFetchMock.mockResolvedValueOnce(exemplare(['B-1', 'B-2', 'B-10001', '58968']));
		labelStore.generationMode = 'existing';
		labelStore.exemplarSuche = '';
		await labelStore.selectBookTitle({ id: 't3', titel: 'Titel', autor: '' });
	});

	it('zählt, wie viele Exemplare gewählt sind', () => {
		expect(labelStore.auswahl).toEqual({ gewaehlt: 4, gesamt: 4 });
		labelStore.existingCopies[0].checked = false;
		expect(labelStore.auswahl).toEqual({ gewaehlt: 3, gesamt: 4 });
	});

	it('nimmt mit einem Griff alle vom Bogen und setzt alle wieder darauf', () => {
		expect(labelStore.auswahlSichtbar).toBe('alle');
		labelStore.setzeSichtbare(false);
		expect(aufDemBogen()).toEqual([]);
		expect(labelStore.auswahlSichtbar).toBe('keine');

		labelStore.existingCopies[1].checked = true;
		expect(labelStore.auswahlSichtbar).toBe('teil');
		labelStore.setzeSichtbare(true);
		expect(aufDemBogen()).toEqual(['B-1', 'B-2', 'B-10001', '58968']);
	});

	it('zeigt beim Tippen nur die Exemplare, deren Nummer den Text enthält', () => {
		labelStore.exemplarSuche = ' b-1 ';
		expect(labelStore.sichtbareExemplare.map((c) => c.barcode_id)).toEqual(['B-1', 'B-10001']);
	});

	it('wählt mit dem Kästchen nur, was zu sehen ist; das Übrige bleibt, wie es war', () => {
		labelStore.exemplarSuche = 'B-';
		labelStore.setzeSichtbare(false);
		expect(aufDemBogen()).toEqual(['58968']);
		expect(labelStore.auswahlSichtbar).toBe('keine');
	});

	it('setzt die getippte Nummer mit der Eingabetaste auf den Bogen und leert das Feld', () => {
		labelStore.setzeSichtbare(false);
		labelStore.exemplarSuche = 'b-2';
		expect(labelStore.uebernimmNummer()).toBe(true);
		expect(aufDemBogen()).toEqual(['B-2']);
		expect(labelStore.exemplarSuche).toBe('');
	});

	// Das Feld sucht nur Nummern dieses Titels, kein Wort. Groß und klein zählen deshalb nicht,
	// auch hinter der Vorsilbe: Die Zeile steht beim Tippen ja schon da.
	it('nimmt eine von Hand getippte Nummer auch in anderer Schreibung', async () => {
		apiFetchMock.mockResolvedValueOnce(exemplare(['B-KL7', 'B-KL8']));
		await labelStore.selectBookTitle({ id: 't5', titel: 'Titel', autor: '' });
		labelStore.setzeSichtbare(false);
		labelStore.exemplarSuche = 'b-kl7';
		expect(labelStore.uebernimmNummer()).toBe(true);
		expect(aufDemBogen()).toEqual(['B-KL7']);
	});

	it('liest ein gescanntes Littera-Etikett und ein Etikett mit Prüfzeichen wie die Theke', () => {
		labelStore.setzeSichtbare(false);
		labelStore.exemplarSuche = '5896800039556';
		expect(labelStore.uebernimmNummer()).toBe(true);
		labelStore.exemplarSuche = 'B-100016';
		expect(labelStore.uebernimmNummer()).toBe(true);
		expect(aufDemBogen()).toEqual(['B-10001', '58968']);
	});

	it('nimmt nichts, wenn die Nummer nicht zu diesem Titel gehört oder nur ein Teil getippt ist', () => {
		labelStore.setzeSichtbare(false);
		for (const eingabe of ['B-9', 'B-', '1', 'A-1', '']) {
			labelStore.exemplarSuche = eingabe;
			expect(labelStore.uebernimmNummer(), `Eingabe „${eingabe}“`).toBe(false);
			expect(labelStore.exemplarSuche, 'die Eingabe bleibt stehen').toBe(eingabe);
		}
		expect(aufDemBogen()).toEqual([]);
	});

	it('beginnt bei einem anderen Titel wieder ohne Suchtext', async () => {
		labelStore.exemplarSuche = 'B-1';
		apiFetchMock.mockResolvedValueOnce(exemplare(['B-7']));
		await labelStore.selectBookTitle({ id: 't4', titel: 'Anderer Titel', autor: '' });
		expect(labelStore.exemplarSuche).toBe('');
		expect(labelStore.sichtbareExemplare.map((c) => c.barcode_id)).toEqual(['B-7']);
	});
});

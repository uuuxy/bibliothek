import { describe, it, expect, vi, beforeEach } from 'vitest';

vi.mock('./snackbars.svelte.js', () => ({
	snackbars: { addToast: vi.fn() }
}));

vi.mock('../apiFetch.js', async (importOriginal) => ({
	.../** @type {any} */ (await importOriginal()),
	apiFetch: vi.fn()
}));

import { apiFetch } from '../apiFetch.js';
import { etikettenStore } from './etiketten.svelte.js';
import { snackbars } from './snackbars.svelte.js';
import { serverAntwort } from '../serverAntwort.js';

const apiFetchMock = vi.mocked(apiFetch);

/** @param {string[]} nummern @returns {any} */
const exemplare = (nummern) => ({
	ok: true,
	json: async () => nummern.map((barcode_id) => ({ barcode_id }))
});

/** Die Nummern der Etiketten, die auf den Bogen kämen. */
const aufDemBogen = () => etikettenStore.bogen.map((l) => l.barcode_id);

// Schritt 2 des Druck-Centers hakt die vorhandenen Exemplare eines Titels vor; was abgewählt
// wird, darf weder in der Vorschau noch im Druck stehen. Das Kästchen schreibt `checked` an
// das Exemplar selbst (bind:checked), der Bogen liest es dort.
describe('etikettenStore: vorhandene Exemplare abwählen', () => {
	beforeEach(async () => {
		vi.clearAllMocks();
		apiFetchMock.mockResolvedValueOnce(exemplare(['B-1', 'B-2', 'B-3']));
		await etikettenStore.waehleTitel({ id: 't1', titel: 'Titel', autor: 'Autorin' });
	});

	it('hakt alle Exemplare des Titels vor', () => {
		expect(aufDemBogen()).toEqual(['B-1', 'B-2', 'B-3']);
	});

	it('nimmt ein abgewähltes Exemplar vom Bogen', () => {
		expect(aufDemBogen()).toEqual(['B-1', 'B-2', 'B-3']);
		etikettenStore.vorhandeneExemplare[1].checked = false;
		expect(aufDemBogen()).toEqual(['B-1', 'B-3']);
	});

	it('druckt und vermerkt nur, was angehakt ist', async () => {
		expect(aufDemBogen()).toHaveLength(3);
		etikettenStore.vorhandeneExemplare[0].checked = false;

		vi.stubGlobal(
			'open',
			vi.fn(() => ({ closed: false, close: vi.fn(), location: { replace: vi.fn() } }))
		);
		URL.createObjectURL = vi.fn(() => 'blob:probe');
		apiFetchMock.mockResolvedValueOnce(
			/** @type {any} */ ({ ok: true, blob: async () => new Blob(['%PDF']) })
		);
		apiFetchMock.mockResolvedValueOnce(/** @type {any} */ ({ ok: true }));
		await etikettenStore.drucke();

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
describe('etikettenStore: ausgesonderte Exemplare', () => {
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
		await etikettenStore.waehleTitel({ id: 't2', titel: 'Titel', autor: '' });

		expect(etikettenStore.vorhandeneExemplare.map((c) => c.barcode_id)).toEqual(['B-1', 'B-3']);
		expect(aufDemBogen()).toEqual(['B-1', 'B-3']);
		expect(etikettenStore.ausgesondertAnzahl).toBe(1);
	});
});

// Scheitert der Abruf der Exemplare, sagt der Store das eigens. Eine leere Liste hieße „kein
// Exemplar“, und der Bogen bliebe leer, ohne dass jemand den Grund sieht.
describe('etikettenStore: Exemplare nicht geladen', () => {
	const titel = { id: 't9', titel: 'Titel', autor: '' };

	beforeEach(() => {
		vi.clearAllMocks();
	});

	it('meldet eine Fehlantwort als Ladefehler', async () => {
		apiFetchMock.mockResolvedValueOnce(/** @type {any} */ ({ ok: false, status: 500 }));
		await etikettenStore.waehleTitel(titel);

		expect(etikettenStore.exemplareNichtGeladen).toBe(true);
		expect(aufDemBogen()).toEqual([]);
	});

	it('meldet einen Netzwerkfehler ebenso', async () => {
		const stumm = vi.spyOn(console, 'error').mockImplementation(() => {});
		apiFetchMock.mockRejectedValueOnce(new Error('Failed to fetch'));
		await etikettenStore.waehleTitel(titel);

		expect(etikettenStore.exemplareNichtGeladen).toBe(true);
		stumm.mockRestore();
	});

	it('lädt erneut und zeigt danach die Exemplare', async () => {
		apiFetchMock.mockResolvedValueOnce(/** @type {any} */ ({ ok: false, status: 500 }));
		await etikettenStore.waehleTitel(titel);
		apiFetchMock.mockResolvedValueOnce(exemplare(['B-1']));
		await etikettenStore.ladeExemplare();

		expect(etikettenStore.exemplareNichtGeladen).toBe(false);
		expect(aufDemBogen()).toEqual(['B-1']);
	});

	// Die Gegenprobe: Ein Titel ohne Exemplar ist kein Ladefehler.
	it('unterscheidet davon den Titel ohne Exemplar', async () => {
		apiFetchMock.mockResolvedValueOnce(exemplare([]));
		await etikettenStore.waehleTitel(titel);

		expect(etikettenStore.exemplareNichtGeladen).toBe(false);
		expect(etikettenStore.vorhandeneExemplare).toEqual([]);
	});
});

// Zwei Titel kurz nacheinander gewählt: Die langsamere Antwort des ersten darf die Exemplare des
// zweiten nicht ersetzen. Auf dem Bogen stünden sonst die Nummern des ersten unter dem Titel
// des zweiten.
describe('etikettenStore: zwei Titel kurz nacheinander', () => {
	it('zeigt die Exemplare des zuletzt gewählten Titels', async () => {
		vi.clearAllMocks();
		/** @type {(antwort: any) => void} */
		let liefereErsten = () => {};
		apiFetchMock.mockReturnValueOnce(new Promise((fertig) => (liefereErsten = fertig)));
		apiFetchMock.mockResolvedValueOnce(exemplare(['B-ZWEI']));

		const erster = etikettenStore.waehleTitel({ id: 'a', titel: 'Erster', autor: '' });
		await etikettenStore.waehleTitel({ id: 'b', titel: 'Zweiter', autor: '' });
		liefereErsten(exemplare(['B-EINS']));
		await erster;

		expect(etikettenStore.bogen).toEqual([{ barcode_id: 'B-ZWEI', titel: 'Zweiter', autor: '' }]);
		expect(etikettenStore.laedtExemplare).toBe(false);
	});

	// Dieselbe Lage nach dem Wechsel der Klasse: Die späte Antwort füllt keine geleerte Liste.
	it('lässt die Liste nach dem Wechsel der Klasse leer', async () => {
		vi.clearAllMocks();
		/** @type {(antwort: any) => void} */
		let liefere = () => {};
		apiFetchMock.mockReturnValueOnce(new Promise((fertig) => (liefere = fertig)));

		const laden = etikettenStore.waehleTitel({ id: 'a', titel: 'Erster', autor: '' });
		etikettenStore.wechsleKlasse();
		liefere(exemplare(['B-EINS']));
		await laden;

		expect(etikettenStore.vorhandeneExemplare).toEqual([]);
		expect(etikettenStore.laedtExemplare).toBe(false);
	});
});

// Dasselbe an der Titelsuche: Die langsamere Antwort auf einen älteren Suchtext darf die Treffer
// des jüngeren nicht ersetzen und nach der Wahl eines Titels keine Liste mehr öffnen.
describe('etikettenStore: Titelsuche, späte Antwort', () => {
	/** @param {string[]} titel @returns {any} */
	const treffer = (titel) => ({
		ok: true,
		json: async () => ({ books: titel.map((t) => ({ id: t, titel: t })) })
	});

	/** Tippt den Text und lässt die Wartezeit der Suche verstreichen. */
	async function tippe(/** @type {string} */ text) {
		etikettenStore.suche = text;
		etikettenStore.verarbeiteSuche();
		await vi.advanceTimersByTimeAsync(300);
	}

	beforeEach(() => {
		vi.clearAllMocks();
		vi.useFakeTimers();
	});

	it('zeigt die Treffer des jüngsten Suchtexts', async () => {
		/** @type {(antwort: any) => void} */
		let liefereErste = () => {};
		apiFetchMock.mockReturnValueOnce(new Promise((fertig) => (liefereErste = fertig)));
		apiFetchMock.mockResolvedValueOnce(treffer(['Faust II']));

		await tippe('Faust');
		await tippe('Faust II');
		liefereErste(treffer(['Faust', 'Faust II', 'Doktor Faustus']));
		await vi.advanceTimersByTimeAsync(0);

		expect(etikettenStore.treffer.map((b) => b.titel)).toEqual(['Faust II']);
		expect(etikettenStore.sucht).toBe(false);
		vi.useRealTimers();
	});

	it('öffnet nach der Wahl eines Titels keine Trefferliste mehr', async () => {
		/** @type {(antwort: any) => void} */
		let liefere = () => {};
		apiFetchMock.mockReturnValueOnce(new Promise((fertig) => (liefere = fertig)));
		apiFetchMock.mockResolvedValueOnce(exemplare(['B-1']));

		await tippe('Faust');
		await etikettenStore.waehleTitel({ id: 't1', titel: 'Faust', autor: '' });
		liefere(treffer(['Faust', 'Faust II']));
		await vi.advanceTimersByTimeAsync(0);

		expect(etikettenStore.treffer).toEqual([]);
		vi.useRealTimers();
	});

	it('nennt beim Fehlschlag den Grund, den der Server angibt', async () => {
		apiFetchMock.mockResolvedValueOnce(
			serverAntwort(403, { error: 'Keine Berechtigung für diese Aktion' })
		);

		await tippe('Faust');

		expect(snackbars.addToast).toHaveBeenCalledWith('Keine Berechtigung für diese Aktion', 'error');
		expect(etikettenStore.treffer).toEqual([]);
		vi.useRealTimers();
	});

	// Ohne Antwort blieb das Feld ohne Treffer und ohne Meldung, als gäbe es den Titel nicht
	// (OFFEN.md 5.106, in Chromium, Firefox und WebKit).
	it('nennt auch ohne Antwort den Grund', async () => {
		// mockImplementationOnce: Ein abgelehntes Promise aus dem Aufsetzen gälte bis zum Ablauf der
		// Wartezeit als unbehandelt.
		apiFetchMock.mockImplementationOnce(() =>
			Promise.reject(new Error('Keine Verbindung zum Server.'))
		);

		await tippe('Faust');

		expect(snackbars.addToast).toHaveBeenCalledWith('Keine Verbindung zum Server.', 'error');
		expect(etikettenStore.treffer).toEqual([]);
		expect(etikettenStore.sucht).toBe(false);
		vi.useRealTimers();
	});

	// Auch das Lesen des Grundes dauert: Hat inzwischen eine neue Suche begonnen, meldet die alte
	// nichts mehr.
	it('meldet den Fehlschlag eines älteren Suchtexts nicht', async () => {
		/** @type {(rumpf: string) => void} */
		let alterGrund = () => {};
		apiFetchMock.mockResolvedValueOnce({
			...serverAntwort(500),
			text: () => new Promise((ja) => (alterGrund = ja))
		});
		apiFetchMock.mockResolvedValueOnce(treffer(['Faust II']));

		await tippe('Faust');
		await tippe('Faust II');
		alterGrund(JSON.stringify({ error: 'Ein interner Serverfehler ist aufgetreten' }));
		await vi.advanceTimersByTimeAsync(0);

		expect(snackbars.addToast).not.toHaveBeenCalled();
		expect(etikettenStore.treffer.map((b) => b.titel)).toEqual(['Faust II']);
		vi.useRealTimers();
	});
});

// Über der Liste steht ein Kästchen für alle und ein Feld für die Nummer. Die Nummern stammen
// aus den gemeinsamen Prüffällen: 58968 ist ein Littera-Exemplar, dessen Etikett den
// EAN-13 5896800039556 trägt; B-100016 ist B-10001 mit dem Prüfzeichen früherer Etiketten.
describe('etikettenStore: alle, keine und einzelne Exemplare wählen', () => {
	beforeEach(async () => {
		vi.clearAllMocks();
		apiFetchMock.mockResolvedValueOnce(exemplare(['B-1', 'B-2', 'B-10001', '58968']));
		etikettenStore.exemplarSuche = '';
		await etikettenStore.waehleTitel({ id: 't3', titel: 'Titel', autor: '' });
	});

	it('zählt, wie viele Exemplare gewählt sind', () => {
		expect(etikettenStore.auswahl).toEqual({ gewaehlt: 4, gesamt: 4 });
		etikettenStore.vorhandeneExemplare[0].checked = false;
		expect(etikettenStore.auswahl).toEqual({ gewaehlt: 3, gesamt: 4 });
	});

	it('nimmt mit einem Griff alle vom Bogen und setzt alle wieder darauf', () => {
		expect(etikettenStore.auswahlSichtbar).toBe('alle');
		etikettenStore.setzeSichtbare(false);
		expect(aufDemBogen()).toEqual([]);
		expect(etikettenStore.auswahlSichtbar).toBe('keine');

		etikettenStore.vorhandeneExemplare[1].checked = true;
		expect(etikettenStore.auswahlSichtbar).toBe('teil');
		etikettenStore.setzeSichtbare(true);
		expect(aufDemBogen()).toEqual(['B-1', 'B-2', 'B-10001', '58968']);
	});

	it('zeigt beim Tippen nur die Exemplare, deren Nummer den Text enthält', () => {
		etikettenStore.exemplarSuche = ' b-1 ';
		expect(etikettenStore.sichtbareExemplare.map((c) => c.barcode_id)).toEqual(['B-1', 'B-10001']);
	});

	it('wählt mit dem Kästchen nur, was zu sehen ist; das Übrige bleibt, wie es war', () => {
		etikettenStore.exemplarSuche = 'B-';
		etikettenStore.setzeSichtbare(false);
		expect(aufDemBogen()).toEqual(['58968']);
		expect(etikettenStore.auswahlSichtbar).toBe('keine');
	});

	it('setzt die getippte Nummer mit der Eingabetaste auf den Bogen und leert das Feld', () => {
		etikettenStore.setzeSichtbare(false);
		etikettenStore.exemplarSuche = 'b-2';
		expect(etikettenStore.uebernimmNummer()).toBe(true);
		expect(aufDemBogen()).toEqual(['B-2']);
		expect(etikettenStore.exemplarSuche).toBe('');
	});

	// Das Feld sucht nur Nummern dieses Titels, kein Wort. Groß und klein zählen deshalb nicht,
	// auch hinter der Vorsilbe: Die Zeile steht beim Tippen ja schon da.
	it('nimmt eine von Hand getippte Nummer auch in anderer Schreibung', async () => {
		apiFetchMock.mockResolvedValueOnce(exemplare(['B-KL7', 'B-KL8']));
		await etikettenStore.waehleTitel({ id: 't5', titel: 'Titel', autor: '' });
		etikettenStore.setzeSichtbare(false);
		etikettenStore.exemplarSuche = 'b-kl7';
		expect(etikettenStore.uebernimmNummer()).toBe(true);
		expect(aufDemBogen()).toEqual(['B-KL7']);
	});

	it('liest ein gescanntes Littera-Etikett und ein Etikett mit Prüfzeichen wie die Theke', () => {
		etikettenStore.setzeSichtbare(false);
		etikettenStore.exemplarSuche = '5896800039556';
		expect(etikettenStore.uebernimmNummer()).toBe(true);
		etikettenStore.exemplarSuche = 'B-100016';
		expect(etikettenStore.uebernimmNummer()).toBe(true);
		expect(aufDemBogen()).toEqual(['B-10001', '58968']);
	});

	// Ein Lernmittel mit kurzer Nummer kann noch sein altes LMF-Etikett tragen; der Scan davon
	// setzt es mit seiner Nummer auf den Bogen, auch wenn der Scanner die Vorsilbe klein schickt.
	it('nimmt ein Exemplar über sein altes LMF-Etikett', async () => {
		apiFetchMock.mockResolvedValueOnce(
			/** @type {any} */ ({
				ok: true,
				json: async () => [
					{ barcode_id: 'B-20001', lmf_barcode: 'LMF-9783060158423-001' },
					{ barcode_id: 'B-20002', lmf_barcode: '' }
				]
			})
		);
		await etikettenStore.waehleTitel({ id: 't6', titel: 'Titel', autor: '' });
		etikettenStore.setzeSichtbare(false);
		etikettenStore.exemplarSuche = 'lmf-9783060158423-001';
		expect(etikettenStore.uebernimmNummer()).toBe(true);
		expect(aufDemBogen()).toEqual(['B-20001']);
	});

	it('nimmt nichts, wenn die Nummer nicht zu diesem Titel gehört oder nur ein Teil getippt ist', () => {
		etikettenStore.setzeSichtbare(false);
		for (const eingabe of ['B-9', 'B-', '1', 'A-1', '']) {
			etikettenStore.exemplarSuche = eingabe;
			expect(etikettenStore.uebernimmNummer(), `Eingabe „${eingabe}“`).toBe(false);
			expect(etikettenStore.exemplarSuche, 'die Eingabe bleibt stehen').toBe(eingabe);
		}
		expect(aufDemBogen()).toEqual([]);
	});

	it('beginnt bei einem anderen Titel wieder ohne Suchtext', async () => {
		etikettenStore.exemplarSuche = 'B-1';
		apiFetchMock.mockResolvedValueOnce(exemplare(['B-7']));
		await etikettenStore.waehleTitel({ id: 't4', titel: 'Anderer Titel', autor: '' });
		expect(etikettenStore.exemplarSuche).toBe('');
		expect(etikettenStore.sichtbareExemplare.map((c) => c.barcode_id)).toEqual(['B-7']);
	});
});

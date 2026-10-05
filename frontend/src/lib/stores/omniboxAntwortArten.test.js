import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';

// Was die Theke aus der Antwort eines Scans macht, je nach `type`: Leser geladen, Ausleihe,
// Hinweis, Zubehör-Anfrage, Suchtreffer. Die Rückkehr eines abgerechneten Buchs steht in
// omniboxRueckkehr.test.js, die gemischten Auflagen in omniboxAuflagenHinweis.test.js.

vi.mock('./toastStore.svelte.js', () => ({ toastStore: { addToast: vi.fn() } }));
vi.mock('../apiFetch.js', () => ({ apiFetch: vi.fn(), apiClient: { post: vi.fn() } }));
vi.mock('../audio.js', () => ({
	playSoundSuccess: vi.fn(),
	playSoundError: vi.fn(),
	playSuccessBeep: vi.fn(),
	playErrorBeep: vi.fn()
}));

import { apiClient } from '../apiFetch.js';
import { playSoundSuccess } from '../audio.js';
import { toastStore } from './toastStore.svelte.js';
import { createOmniboxStore } from './omnibox.svelte.js';

const IDA = { id: 'l-1', vorname: 'Ida', nachname: 'Berg', art: 'schueler' };
const OLE = { id: 'l-2', vorname: 'Ole', nachname: 'Kern', art: 'schueler' };

/** @type {ReturnType<typeof createOmniboxStore>[]} */
const stores = [];
afterEach(() => {
	while (stores.length) stores.pop()?.stoppeZeitgeber();
});

/** @param {any} [leser] der Leser, der schon an der Theke steht */
function neuerStore(leser = null) {
	const store = createOmniboxStore();
	stores.push(store);
	store.activeStudent = leser;
	return store;
}

/**
 * Scannt und liefert den Rückruf, mit dem die Theke die Akte neu lädt.
 * @param {ReturnType<typeof createOmniboxStore>} store
 * @param {any} antwort was die Buchungstür antwortet
 * @param {string} [scan]
 */
async function scanneMitAntwort(store, antwort, scan = 'B-518-1') {
	vi.mocked(apiClient.post).mockResolvedValue(antwort);
	const neuLaden = vi.fn();
	store.queryVal = scan;
	await store.submitAction(null, neuLaden);
	return neuLaden;
}

/**
 * Ein Scan, den der Server annimmt.
 * @param {ReturnType<typeof createOmniboxStore>} store
 * @param {any} data der Körper der Antwort
 * @param {string} [scan]
 */
function scanne(store, data, scan) {
	return scanneMitAntwort(store, { ok: true, json: async () => data }, scan);
}

beforeEach(() => vi.clearAllMocks());

describe('Theke: ein Leser wird geladen', () => {
	it('zeigt den Leser und sein Abholfach, blitzt grün und gibt den Ton', async () => {
		const store = neuerStore();

		await scanne(store, { type: 'student', student: IDA, abholbereit: [{ titel: 'Mathe 7' }] });

		expect(store.activeStudent).toEqual(IDA);
		expect(store.abholbereit).toEqual([{ titel: 'Mathe 7' }]);
		expect(store.screenFlash).toBe('success');
		expect(store.flashBorder).toBe('green');
		expect(playSoundSuccess).toHaveBeenCalledTimes(1);
	});

	// Bliebe das Abholfach des vorigen Lesers stehen, griffe die Theke für den falschen
	// Leser ins Fach.
	it('leert das Abholfach, wenn der nächste Leser nichts abzuholen hat', async () => {
		const store = neuerStore();
		await scanne(store, { type: 'student', student: IDA, abholbereit: [{ titel: 'Mathe 7' }] });

		await scanne(store, { type: 'student', student: OLE });

		expect(store.activeStudent).toEqual(OLE);
		expect(store.abholbereit).toEqual([]);
	});
});

describe('Theke: Ausleihe', () => {
	it('nennt Titel und Vornamen und lädt die Akte neu', async () => {
		const store = neuerStore(IDA);

		const neuLaden = await scanne(store, { type: 'ausleihe', book: { titel: 'Mathe 7' } });

		expect(toastStore.addToast).toHaveBeenCalledTimes(1);
		expect(toastStore.addToast).toHaveBeenCalledWith('„Mathe 7" ausgeliehen an Ida.', 'success');
		expect(neuLaden).toHaveBeenCalledTimes(1);
		expect(store.flashBorder).toBe('green');
		expect(playSoundSuccess).toHaveBeenCalledTimes(1);
	});

	// Kinder derselben Klasse haben eine andere Auflage: gebucht, aber als Warnung gezeigt.
	it('blitzt orange, wenn die Antwort einen Hinweis auf gemischte Auflagen trägt', async () => {
		const store = neuerStore(IDA);

		await scanne(store, {
			type: 'ausleihe',
			book: { titel: 'Mathe 7' },
			auflagen_hinweis: { klasse: '07B', andere: [] }
		});

		expect(store.flashBorder).toBe('orange');
		expect(store.screenFlash).toBe('warning');
	});

	it('nennt bei einem Gerät das Modell', async () => {
		const store = neuerStore(IDA);

		await scanne(store, { type: 'ausleihe', geraet: { modellname: 'Tablet 9' } });

		expect(toastStore.addToast).toHaveBeenCalledWith('„Tablet 9" ausgeliehen an Ida.', 'success');
	});

	// Der Leser hatte ein anderes Exemplar reserviert und eines aus dem Regal genommen.
	it('sagt, welches reservierte Exemplar zurück ins Regal muss', async () => {
		const store = neuerStore(IDA);

		await scanne(store, {
			type: 'ausleihe',
			book: { titel: 'Mathe 7' },
			regalfreigabe_barcode: 'B-518-2'
		});

		expect(toastStore.addToast).toHaveBeenCalledTimes(2);
		expect(toastStore.addToast).toHaveBeenCalledWith(
			'Hinweis: Reserviertes Exemplar B-518-2 zurück ins Regal räumen.',
			'warning'
		);
	});
});

describe('Theke: Hinweis des Servers', () => {
	it('blitzt grün, gibt den Ton und lädt die Akte neu', async () => {
		const store = neuerStore(IDA);

		const neuLaden = await scanne(store, { type: 'info', message: 'Buch reaktiviert' });

		expect(neuLaden).toHaveBeenCalledTimes(1);
		expect(store.screenFlash).toBe('success');
		expect(store.flashBorder).toBe('green');
		expect(playSoundSuccess).toHaveBeenCalledTimes(1);
	});
});

describe('Theke: Gerät mit Zubehör', () => {
	// Die Bestätigung der Liste schickt den Scan ein zweites Mal; dafür liegt er hier.
	it('hält Scan und Gerät fest, ohne Erfolg zu melden', async () => {
		const store = neuerStore(IDA);
		const geraet = { barcode_id: 'G-77', zubehoer: 'Netzteil' };

		const neuLaden = await scanne(store, { type: 'geraet_check', geraet }, 'g-77');

		expect(store.checklistAnfrage).toEqual({ query: 'G-77', geraet, overrideBlock: false });
		expect(toastStore.addToast).not.toHaveBeenCalled();
		expect(neuLaden).not.toHaveBeenCalled();
		expect(store.screenFlash).toBe('');
		expect(playSoundSuccess).not.toHaveBeenCalled();
	});
});

describe('Theke: Suchtreffer statt Buchung', () => {
	it('rüttelt und bittet um eine Auswahl', async () => {
		const store = neuerStore();

		const neuLaden = await scanne(store, { type: 'search_results' }, 'Mathe');

		expect(store.isShaking).toBe(true);
		expect(toastStore.addToast).toHaveBeenCalledTimes(1);
		expect(toastStore.addToast).toHaveBeenCalledWith(
			'Bitte wähle ein Ergebnis aus der Liste.',
			'warning'
		);
		expect(neuLaden).not.toHaveBeenCalled();
	});
});

describe('Theke: der Server lehnt den Scan ab', () => {
	it('zeigt seinen Grund im Fehlerbanner, ohne Toast', async () => {
		const store = neuerStore(IDA);

		const neuLaden = await scanneMitAntwort(store, {
			ok: false,
			status: 409,
			headers: new Headers(),
			text: async () => JSON.stringify({ error: 'Das Exemplar ist ausgesondert.' })
		});

		expect(store.errorMessage).toBe('Fehler: Das Exemplar ist ausgesondert.');
		expect(store.flashBorder).toBe('red');
		expect(toastStore.addToast).not.toHaveBeenCalled();
		expect(neuLaden).not.toHaveBeenCalled();
	});
});

describe('Theke: eine Antwort unbekannter Art', () => {
	it('ändert nichts und meldet nichts', async () => {
		const store = neuerStore(IDA);

		const neuLaden = await scanne(store, { type: 'kuenftige_art' });

		expect(store.activeStudent).toEqual(IDA);
		expect(toastStore.addToast).not.toHaveBeenCalled();
		expect(neuLaden).not.toHaveBeenCalled();
		expect(store.screenFlash).toBe('');
		expect(store.isShaking).toBe(false);
		expect(playSoundSuccess).not.toHaveBeenCalled();
	});
});

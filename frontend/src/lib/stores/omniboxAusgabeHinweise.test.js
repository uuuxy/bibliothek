import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { render } from '@testing-library/svelte';

// Hinweise zu ausgegebenen Schulbüchern an der Theke: Die Antwort einer Ausleihe trägt
// `doppelt_hinweis`, `klassen_hinweis` oder `auflagen_hinweis`. Der Store hält je Hinweis eine
// Zeile, bis ein anderer Leser kommt oder das Konto schließt; kommt das Buch zurück, fällt seine
// Zeile weg. Die Theke blitzt wie bei der Fremdrückgabe. Gebucht ist die Ausleihe trotzdem.

vi.mock('./snackbars.svelte.js', () => ({ snackbars: { addToast: vi.fn() } }));
vi.mock('../apiFetch.js', () => ({ apiFetch: vi.fn(), apiClient: { post: vi.fn() } }));
vi.mock('../utils/audio.js', () => ({
	spieleErfolgston: vi.fn(),
	spieleFehlerton: vi.fn()
}));

import { apiClient } from '../apiFetch.js';
import { erzeugeOmniboxStore } from './omnibox.svelte.js';
import OmniboxThekeHinweise from '../components/omnibox/OmniboxThekeHinweise.svelte';
import { omniboxStore } from './omnibox.svelte.js';

const AUFLAGEN = {
	klasse: '07B',
	auflage: '4. Aufl.',
	erscheinungsjahr: 2023,
	andere: [{ auflage: '3. Aufl.', erscheinungsjahr: 2019, kinder: 12 }]
};
const KLASSE = { klasse: '08G1', mit_buechern: 19, kinder: 28 };
const MATHE7 = { id: 'ex-7', titel: 'Mathe 7' };
const MATHE8 = { id: 'ex-8', titel: 'Mathe 8' };
const IDA = { id: 'ida', vorname: 'Ida', art: 'schueler' };

const SATZ_AUFLAGEN =
	'„Mathe 7" ist 4. Aufl. · 2023, eine andere Auflage als in der 07B: 12 Kinder haben 3. Aufl. · 2019';
const SATZ_KLASSE =
	'„Mathe 7" hat sonst niemand in der 08G1. 19 von 28 Kindern haben ihre Schulbücher schon.';

/** @type {ReturnType<typeof erzeugeOmniboxStore>[]} */
const stores = [];
afterEach(() => {
	while (stores.length) stores.pop()?.stoppeZeitgeber();
});

/** @param {ReturnType<typeof erzeugeOmniboxStore>} store @param {any} data */
async function scanne(store, data) {
	vi.mocked(apiClient.post).mockResolvedValue(
		/** @type {any} */ ({ ok: true, json: async () => data })
	);
	store.eingabe = 'B-518-1';
	await store.fuehreAktionAus(null, null);
}

function neuerStore() {
	const store = erzeugeOmniboxStore();
	stores.push(store);
	store.aktiverLeser = IDA;
	return store;
}

/** @param {ReturnType<typeof erzeugeOmniboxStore>} store */
const texte = (store) => store.ausgabeHinweise.map((/** @type {any} */ h) => h.text);

describe('Theke: Hinweise zu ausgegebenen Schulbüchern', () => {
	beforeEach(() => vi.clearAllMocks());

	it('hält den Hinweis der Ausleihe und blitzt als Warnung', async () => {
		const store = neuerStore();
		await scanne(store, { type: 'ausleihe', book: MATHE7, auflagen_hinweis: AUFLAGEN });
		expect(store.ausgabeHinweise).toEqual([
			{ exemplar: 'ex-7', art: 'auflagen', text: SATZ_AUFLAGEN }
		]);
		expect(store.rueckmeldung.bildschirmLeuchten).toBe('warning');
		expect(store.rueckmeldung.randLeuchten).toBe('orange');
	});

	it('bleibt bei einer gewöhnlichen Ausleihe leer und blitzt grün', async () => {
		const store = neuerStore();
		await scanne(store, { type: 'ausleihe', book: MATHE7 });
		expect(store.ausgabeHinweise).toEqual([]);
		expect(store.rueckmeldung.bildschirmLeuchten).toBe('success');
	});

	it('nennt das doppelte und das fremde Buch, das sichere Versehen zuerst', async () => {
		const store = neuerStore();
		await scanne(store, {
			type: 'ausleihe',
			book: MATHE7,
			doppelt_hinweis: true,
			klassen_hinweis: KLASSE
		});
		expect(texte(store)).toEqual(['Ida hat „Mathe 7" jetzt zweimal', SATZ_KLASSE]);
	});

	// Beim Bücherstapel kommt der nächste Scan nach einer Sekunde: Die Zeile muss ihn überstehen,
	// auch eine Rückgabe eines anderen Buchs.
	it('bleibt beim nächsten Scan stehen, und die Zeilen sammeln sich', async () => {
		const store = neuerStore();
		await scanne(store, { type: 'ausleihe', book: MATHE7, klassen_hinweis: KLASSE });
		await scanne(store, { type: 'ausleihe', book: MATHE8 });
		await scanne(store, { type: 'rueckgabe', book: { id: 'ex-d', titel: 'Deutsch 7' } });
		await scanne(store, {
			type: 'ausleihe',
			book: { id: 'ex-8b', titel: 'Mathe 8' },
			doppelt_hinweis: true
		});
		expect(texte(store)).toEqual([SATZ_KLASSE, 'Ida hat „Mathe 8" jetzt zweimal']);
	});

	it('nimmt die Zeile weg, wenn das Buch zurückkommt', async () => {
		const store = neuerStore();
		await scanne(store, { type: 'ausleihe', book: MATHE7, klassen_hinweis: KLASSE });
		await scanne(store, { type: 'ausleihe', book: MATHE8, doppelt_hinweis: true });
		await scanne(store, { type: 'rueckgabe', book: MATHE7 });
		expect(texte(store)).toEqual(['Ida hat „Mathe 8" jetzt zweimal']);
	});

	it('leert sich, wenn ein anderer Leser kommt, nicht beim selben', async () => {
		const store = neuerStore();
		await scanne(store, { type: 'ausleihe', book: MATHE7, klassen_hinweis: KLASSE });
		await scanne(store, { type: 'student', student: { ...IDA } });
		expect(texte(store)).toEqual([SATZ_KLASSE]);
		await scanne(store, {
			type: 'student',
			student: { id: 'ben', vorname: 'Ben', art: 'schueler' }
		});
		expect(store.ausgabeHinweise).toEqual([]);
	});

	it('leert sich, wenn das Konto geschlossen wird', async () => {
		const store = neuerStore();
		await scanne(store, { type: 'ausleihe', book: MATHE7, klassen_hinweis: KLASSE });
		store.aktiverLeser = null;
		expect(store.ausgabeHinweise).toEqual([]);
	});

	it('steht als Zeile je Hinweis über dem Konto', async () => {
		omniboxStore.aktiverLeser = { id: 'ida-z', vorname: 'Ida', art: 'schueler' };
		await scanne(omniboxStore, {
			type: 'ausleihe',
			book: MATHE7,
			klassen_hinweis: KLASSE,
			auflagen_hinweis: AUFLAGEN
		});
		const screen = render(OmniboxThekeHinweise);
		expect(screen.getAllByRole('status').map((z) => z.textContent?.trim())).toEqual([
			SATZ_KLASSE,
			SATZ_AUFLAGEN
		]);
		omniboxStore.aktiverLeser = null;
		omniboxStore.stoppeZeitgeber();
	});
});

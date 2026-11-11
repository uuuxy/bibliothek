import { describe, it, expect, vi, beforeEach } from 'vitest';

vi.mock('../../apiFetch.js', async (importOriginal) => ({
	.../** @type {any} */ (await importOriginal()),
	apiFetch: vi.fn()
}));
vi.mock('../../../inventur/lib/store.svelte.js', () => ({ melde: vi.fn() }));
vi.mock('../../stores/bestaetigung.svelte.js', () => ({
	bestaetigen: vi.fn(async () => true),
	verwerfenBestaetigen: vi.fn(async () => false)
}));

import { apiFetch } from '../../apiFetch.js';
import { verwerfenBestaetigen } from '../../stores/bestaetigung.svelte.js';
import { erzeugePlaner } from './lmfplanPlaner.svelte.js';

const STAND = {
	plan: null,
	zeilen: [],
	ausgelassen: [],
	vorbei: false,
	vorschlag: { quelle: 'regel', zeilen: [{ klassen: ['09H1'], vermerk: '' }], ausgelassen: [] },
	klassen: ['09H1', '09H2'],
	eingangsjahrgaenge: [5, 7],
	sommerferien: { jahr: 2027, von: '2027-06-28', bis: '2027-08-06', bekannt: true }
};

/** GETs der Art je Adresse; PUT (Vorschau) über `vorschauAntwort`. */
function antworte(vorschauAntwort = async () => new Response('{"zeilen":[]}')) {
	vi.mocked(apiFetch).mockImplementation(async (url, init) => {
		if (init?.method === 'PUT') return vorschauAntwort();
		return new Response(JSON.stringify(STAND), { status: 200 });
	});
}

const geholt = (/** @type {string} */ art) =>
	vi.mocked(apiFetch).mock.calls.filter(([url, o]) => !o?.method && url === `/api/lmf-plan/${art}`)
		.length;

beforeEach(() => {
	vi.mocked(apiFetch).mockReset();
	vi.mocked(verwerfenBestaetigen).mockClear();
});

// Die Umschaltleiste „Art des Plans" lud den Plan der anderen Art, ohne zu fragen: Eine
// umgeordnete, nicht gespeicherte Reihenfolge war danach weg. Beim Wechsel der Seite und bei
// einer fremden Änderung fragte der Planer schon.
describe('LMF-Planer: Art wechseln mit Ungespeichertem', () => {
	it('fragt und bleibt bei „Bleiben" auf der Art, mit dem Entwurf', async () => {
		antworte();
		const planer = erzeugePlaner();
		await planer.lade();
		planer.klasseHinein('09H2');
		const entwurf = JSON.stringify(planer.zustand.entwurf);

		await planer.waehleArt('ausgabe');

		expect(verwerfenBestaetigen).toHaveBeenCalledWith('Verwerfen und weiter');
		expect(planer.zustand.art).toBe('rueckgabe');
		expect(JSON.stringify(planer.zustand.entwurf)).toBe(entwurf);
		expect(geholt('ausgabe')).toBe(0);
	});

	it('wechselt nach „Verwerfen und weiter"', async () => {
		antworte();
		vi.mocked(verwerfenBestaetigen).mockResolvedValueOnce(true);
		const planer = erzeugePlaner();
		await planer.lade();
		planer.klasseHinein('09H2');

		await planer.waehleArt('ausgabe');

		expect(planer.zustand.art).toBe('ausgabe');
		await vi.waitFor(() => expect(geholt('ausgabe')).toBe(1));
	});

	it('fragt nicht, wenn nichts Ungespeichertes da ist (Gegenprobe)', async () => {
		antworte();
		const planer = erzeugePlaner();
		await planer.lade();

		await planer.waehleArt('ausgabe');

		expect(verwerfenBestaetigen).not.toHaveBeenCalled();
		expect(planer.zustand.art).toBe('ausgabe');
	});
});

// Eine Vorschau des vorigen Plans, die beim Umschalten noch unterwegs war, schrieb nach ihrer
// Antwort die Plätze dieses Plans in die gerade geleerte Tabelle.
describe('LMF-Planer: Vorschau beim Laden', () => {
	it('verwirft die Antwort einer Vorschau, die vor dem Laden losging', async () => {
		/** @type {(r: Response) => void} */
		let liefere = () => {};
		antworte(() => new Promise((fertig) => (liefere = fertig)));
		const planer = erzeugePlaner();
		await planer.lade();
		const entwurf = { ...planer.zustand.entwurf, letzter_tag: '2027-06-24' };

		const vorschau = planer.vorschau('rueckgabe', entwurf);
		await vi.waitFor(() =>
			expect(vi.mocked(apiFetch).mock.calls.some(([, o]) => o?.method === 'PUT')).toBe(true)
		);
		await planer.waehleArt('ausgabe');
		liefere(new Response(JSON.stringify({ zeilen: [{ datum: '2027-06-24', stunde: 4 }] })));
		await vorschau;

		expect(planer.zustand.plaetze).toEqual([]);
	});
});

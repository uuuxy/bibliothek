import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';

vi.mock('../../../../lib/apiFetch.js', () => ({
	apiFetch: vi.fn(),
	extractApiError: vi.fn(async (/** @type {any} */ res) => `Fehler ${res.status}`)
}));

import { apiFetch } from '../../../../lib/apiFetch.js';
import { erzeugeAuflagenSuche, kandidaten, AUFLAGEN_TREFFER_MAX } from './auflagenSuche.svelte.js';

// Die Suche des Dialogs „Andere Auflage zuordnen" (docs/OFFEN.md 4.18, Stufe 2). Sie fragt
// die Titel-Verwaltung zweimal — mit und ohne Exemplare —, weil eine alte Auflage oft keins
// mehr hat. Angeboten wird nur, was der Server zusammenfasst: Lernmittel, die noch nicht zu
// diesem Buch gehören.

const titel = (id, title, jahr, lernmittel = true) => ({
	id,
	title,
	erscheinungsjahr: jahr,
	istLernmittel: lernmittel
});
const antwort = (liste) => ({ ok: true, status: 200, json: async () => ({ data: liste }) });

describe('kandidaten', () => {
	it('nimmt beide Hälften, nur Lernmittel, ohne die bekannten, jeden Titel einmal', () => {
		const mit = { data: [titel('A', 'Mathe 7', 2019), titel('R', 'Roman', 2020, false)] };
		const ohne = { data: [titel('B', 'Mathe 7', 2014), titel('a', 'Mathe 7', 2019)] };
		const liste = kandidaten([mit, ohne], ['SELBST']);
		expect(liste.map((t) => t.id)).toEqual(['A', 'B']);
	});

	it('lässt die Titel weg, die schon zu diesem Buch gehören — auch in anderer Schreibweise', () => {
		const liste = kandidaten([{ data: [titel('abc', 'Mathe 7', 2019)] }, { data: [] }], ['ABC']);
		expect(liste).toEqual([]);
	});

	it('ordnet nach Titel und dann die jüngste Auflage zuerst, höchstens die Grenze', () => {
		const viele = Array.from({ length: AUFLAGEN_TREFFER_MAX + 5 }, (_, i) =>
			titel(`t${i}`, 'Deutschbuch 7', 2000 + i)
		);
		const liste = kandidaten([{ data: viele }, { data: [titel('x', 'Chemie 8', 2010)] }], []);
		expect(liste).toHaveLength(AUFLAGEN_TREFFER_MAX);
		expect(liste[0].id).toBe('x');
		expect(liste[1].erscheinungsjahr).toBe(2000 + AUFLAGEN_TREFFER_MAX + 4);
	});
});

describe('erzeugeAuflagenSuche', () => {
	beforeEach(() => {
		vi.useFakeTimers();
		vi.mocked(apiFetch).mockReset();
	});
	afterEach(() => {
		vi.useRealTimers();
	});

	it('fragt beide Hälften der Titel-Verwaltung ab', async () => {
		vi.mocked(apiFetch).mockResolvedValue(/** @type {any} */ (antwort([])));
		const s = erzeugeAuflagenSuche(() => []);
		s.suche = 'Mathe';
		s.tippen();
		await vi.advanceTimersByTimeAsync(250);
		expect(apiFetch).toHaveBeenCalledWith('/api/books?q=Mathe');
		expect(apiFetch).toHaveBeenCalledWith('/api/books?q=Mathe&bestand=ohne');
	});

	it('lässt eine späte Antwort auf die ältere Eingabe die Liste nicht überschreiben', async () => {
		/** @type {(v: any) => void} */
		let spaet = () => {};
		const langsam = new Promise((res) => (spaet = res));
		vi.mocked(apiFetch)
			.mockReturnValueOnce(/** @type {any} */ (langsam))
			.mockReturnValueOnce(/** @type {any} */ (langsam))
			.mockResolvedValue(/** @type {any} */ (antwort([titel('neu', 'Mathe 7', 2023)])));
		const s = erzeugeAuflagenSuche(() => []);
		s.suche = 'Ma';
		s.tippen();
		await vi.advanceTimersByTimeAsync(250);
		s.suche = 'Mathe 7';
		s.tippen();
		await vi.advanceTimersByTimeAsync(250);
		expect(s.treffer.map((t) => t.id)).toEqual(['neu']);

		spaet(antwort([titel('alt', 'Mathe 5', 2010)]));
		await vi.advanceTimersByTimeAsync(0);
		expect(s.treffer.map((t) => t.id)).toEqual(['neu']);
	});

	it('meldet einen Fehler als Fehler und nicht als leere Liste', async () => {
		vi.mocked(apiFetch)
			.mockResolvedValueOnce(/** @type {any} */ (antwort([titel('A', 'Mathe 7', 2019)])))
			.mockResolvedValueOnce(/** @type {any} */ ({ ok: false, status: 403 }));
		const s = erzeugeAuflagenSuche(() => []);
		s.suche = 'Mathe';
		s.tippen();
		await vi.advanceTimersByTimeAsync(250);
		expect(s.fehler).toBe('Fehler 403');
		expect(s.treffer).toEqual([]);
	});

	it('sucht erst ab zwei Zeichen', async () => {
		const s = erzeugeAuflagenSuche(() => []);
		s.suche = 'M';
		s.tippen();
		await vi.advanceTimersByTimeAsync(250);
		expect(apiFetch).not.toHaveBeenCalled();
	});
});

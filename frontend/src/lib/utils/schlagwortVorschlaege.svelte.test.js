import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { erzeugeSchlagwortVorschlaege } from './schlagwortVorschlaege.svelte.js';
import { apiFetch } from '../apiFetch.js';

vi.mock('../apiFetch.js', () => ({ apiFetch: vi.fn(), apiPut: vi.fn() }));

// Die Vorschläge der Schlagwort-Felder fragen seit dem 30.09.2026 beim Tippen den Server
// (Rasterdurchgang): Mit den Littera-Wörtern lagen 12.724 von 13.224 jenseits der 500
// häufigsten, die das Feld bis dahin nur kannte.

/** @param {string} wort @param {number} [titel] */
const antwort = (wort, titel = 1) =>
	/** @type {any} */ ({ ok: true, json: async () => [{ wort, titel }] });

describe('erzeugeSchlagwortVorschlaege', () => {
	beforeEach(() => {
		vi.useFakeTimers();
		vi.mocked(apiFetch).mockReset();
	});
	afterEach(() => vi.useRealTimers());

	it('lädt ohne Suche die häufigsten und mit Suche den kodierten Text', async () => {
		vi.mocked(apiFetch).mockResolvedValue(antwort('Fantasy', 12));
		const v = erzeugeSchlagwortVorschlaege();
		await v.lade();
		expect(apiFetch).toHaveBeenLastCalledWith('/api/schlagworte');
		expect(v.liste).toEqual([{ wert: 'Fantasy', beschreibung: '12 Titel' }]);

		await v.lade('  Pilze & Moose ');
		expect(apiFetch).toHaveBeenLastCalledWith('/api/schlagworte?suche=Pilze%20%26%20Moose');
	});

	it('fragt beim Tippen 300 ms nach dem letzten Zeichen einmal nach', async () => {
		vi.mocked(apiFetch).mockResolvedValue(antwort('Zwergpilze'));
		const v = erzeugeSchlagwortVorschlaege();
		for (const text of ['p', 'pi', 'pil', 'pilz']) {
			v.getippt(text);
			await vi.advanceTimersByTimeAsync(100);
		}
		expect(apiFetch).not.toHaveBeenCalled();
		await vi.advanceTimersByTimeAsync(300);
		expect(apiFetch).toHaveBeenCalledTimes(1);
		expect(apiFetch).toHaveBeenLastCalledWith('/api/schlagworte?suche=pilz');
		expect(v.liste[0].wert).toBe('Zwergpilze');
	});

	it('holt die häufigsten zurück, wenn das Feld wieder leer ist', async () => {
		vi.mocked(apiFetch).mockResolvedValue(antwort('Fantasy'));
		const v = erzeugeSchlagwortVorschlaege();
		v.getippt('  ');
		await vi.advanceTimersByTimeAsync(300);
		expect(apiFetch).toHaveBeenLastCalledWith('/api/schlagworte');
	});

	it('lässt eine ältere Antwort, die zuletzt ankommt, die jüngere nicht überschreiben', async () => {
		/** @type {(wert: any) => void} */
		let alteFertig = () => {};
		vi.mocked(apiFetch)
			.mockImplementationOnce(() => new Promise((fertig) => (alteFertig = fertig)))
			.mockResolvedValueOnce(antwort('Zwergpilze'));
		const v = erzeugeSchlagwortVorschlaege();
		const alt = v.lade();
		await v.lade('zwerg');
		alteFertig(antwort('Fantasy'));
		await alt;
		expect(v.liste.map((x) => x.wert)).toEqual(['Zwergpilze']);
	});

	it('behält die bisherigen Vorschläge, wenn der Server nicht antwortet', async () => {
		vi.mocked(apiFetch)
			.mockResolvedValueOnce(antwort('Fantasy'))
			.mockResolvedValueOnce(
				/** @type {any} */ ({ ok: false, status: 500, json: async () => ({}) })
			)
			.mockRejectedValueOnce(new Error('Netz weg'));
		const v = erzeugeSchlagwortVorschlaege();
		await v.lade();
		await v.lade('pilz');
		await v.lade('pilze');
		expect(v.liste.map((x) => x.wert)).toEqual(['Fantasy']);
	});
});

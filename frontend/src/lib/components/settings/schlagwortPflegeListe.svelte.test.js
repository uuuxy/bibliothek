import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { erzeugeSchlagwortPflegeListe } from './schlagwortPflegeListe.svelte.js';
import { apiGet } from '../../apiFetch.js';

vi.mock('../../apiFetch.js', () => ({ apiGet: vi.fn() }));

// Die Liste der Schlagwort-Pflege kommt seit dem 30.09.2026 gesucht vom Server (docs/OFFEN.md
// 4.20): Tippen wird entprellt, und nur die jüngste Anfrage schreibt die Liste.

/** @param {string} wort @param {number} treffer */
const antwort = (wort, treffer = 1) => ({
	zeilen: [{ id: wort, wort, titel: 1, verweise: [], ist_filter: false }],
	gesamt: 13207,
	verweise: 0,
	filter: 0,
	treffer
});

describe('erzeugeSchlagwortPflegeListe', () => {
	beforeEach(() => {
		vi.useFakeTimers();
		vi.mocked(apiGet).mockReset();
	});
	afterEach(() => vi.useRealTimers());

	it('fragt ohne Suche die ganze Liste an und mit Suche den kodierten Text', async () => {
		vi.mocked(apiGet).mockResolvedValue(antwort('Fantasy', 13207));
		const liste = erzeugeSchlagwortPflegeListe();
		await liste.lade();
		expect(apiGet).toHaveBeenLastCalledWith('/api/schlagworte/pflege');

		liste.suche = '  Zwerg & Pinguin ';
		await liste.lade();
		expect(apiGet).toHaveBeenLastCalledWith('/api/schlagworte/pflege?suche=Zwerg%20%26%20Pinguin');
	});

	it('wartet beim Tippen 300 ms und fragt dann einmal', async () => {
		vi.mocked(apiGet).mockResolvedValue(antwort('Zwergpinguin'));
		const liste = erzeugeSchlagwortPflegeListe();
		for (const text of ['z', 'zw', 'zwe']) {
			liste.suche = text;
			liste.angestossen();
			await vi.advanceTimersByTimeAsync(100);
		}
		expect(apiGet).not.toHaveBeenCalled();
		await vi.advanceTimersByTimeAsync(300);
		expect(apiGet).toHaveBeenCalledTimes(1);
		expect(apiGet).toHaveBeenLastCalledWith('/api/schlagworte/pflege?suche=zwe');
		expect(liste.liste?.zeilen[0].wort).toBe('Zwergpinguin');
	});

	it('lässt eine ältere Antwort, die zuletzt ankommt, die jüngere nicht überschreiben', async () => {
		/** @type {(wert: any) => void} */
		let alteFertig = () => {};
		vi.mocked(apiGet)
			.mockImplementationOnce(() => new Promise((fertig) => (alteFertig = fertig)))
			.mockResolvedValueOnce(antwort('Zwergpinguin'));
		const liste = erzeugeSchlagwortPflegeListe();
		const alt = liste.lade();
		liste.suche = 'zwerg';
		await liste.lade();
		alteFertig(antwort('Aal', 13207));
		await alt;
		expect(liste.liste?.zeilen[0].wort).toBe('Zwergpinguin');
	});

	it('meldet einen Ladefehler und nimmt ihn mit der nächsten Antwort zurück', async () => {
		vi.mocked(apiGet).mockRejectedValueOnce(new Error('500')).mockResolvedValueOnce(antwort('Aal'));
		const liste = erzeugeSchlagwortPflegeListe();
		await liste.lade();
		expect(liste.ladeFehler).toBe(true);
		await liste.lade();
		expect(liste.ladeFehler).toBe(false);
	});
});

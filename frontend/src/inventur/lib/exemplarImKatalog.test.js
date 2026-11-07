import { describe, it, expect, vi, afterEach } from 'vitest';

vi.mock('../../lib/apiFetch.js', () => ({
	apiFetch: vi.fn(),
	leseApiFehler: vi.fn()
}));

import { apiFetch } from '../../lib/apiFetch.js';
import { erzeugeExemplarFrage, hinweisOhneTreffer } from './exemplarImKatalog.svelte.js';

afterEach(() => {
	vi.useRealTimers();
});

// Ein Handscanner mit amerikanischer Tastatur tippt an einem deutschen Rechner den Bindestrich
// als ß (FACHKONZEPT §1). Die Katalogsuche fragt nach der Nummer, die die Theke nachschlägt, und
// ordnet die Antwort dem Scan zu, so wie er im Suchfeld steht.
describe('Katalogsuche nach einem Exemplar', () => {
	it('liest in einer gescannten Nummer ein ß hinter der Vorsilbe als Bindestrich', async () => {
		vi.useFakeTimers();
		vi.mocked(apiFetch).mockResolvedValue(
			/** @type {any} */ ({
				ok: true,
				json: async () => ({ treffer: { titel_id: 't1', barcode: 'B-00123' } })
			})
		);
		const exemplar = erzeugeExemplarFrage();

		exemplar.frage('Bß00123');
		await vi.advanceTimersByTimeAsync(250);

		expect(vi.mocked(apiFetch)).toHaveBeenCalledWith(
			'/api/buecher/exemplare/erkennen?code=B-00123'
		);
		expect(exemplar.fuer('Bß00123')).toMatchObject({ stand: 'fertig', titelId: 't1' });
	});
});

// Zeigt der Katalog zur Nummer keine Karte, sagt der Hinweis, warum: Das Exemplar ist ausgesondert,
// oder sein Titel fehlt in der Liste bis zum Neuladen. Ein Exemplar im Bestand heißt nie ausgesondert.
describe('Hinweis ohne Treffer zu einem gefundenen Exemplar', () => {
	/** @param {boolean} ausgesondert */
	const antwort = (ausgesondert) =>
		/** @type {const} */ ({
			code: 'B-7',
			stand: 'fertig',
			titelId: 't-1',
			barcode: 'B-7',
			ausgesondert
		});

	it('nennt ein ausgesondertes Exemplar ausgesondert', () => {
		expect(hinweisOhneTreffer('B-7', antwort(true))).toBe(
			'Exemplar B-7 ist ausgesondert. Sein Titel hat kein Exemplar mehr im Bestand und steht in der Titelliste unter „Ohne Exemplare“.'
		);
	});

	it('nennt bei einem Exemplar im Bestand den fehlenden Titel', () => {
		expect(hinweisOhneTreffer('B-7', antwort(false))).toBe(
			'Exemplar B-7 gehört zu einem Titel, der in dieser Liste noch fehlt. Neu laden zeigt ihn.'
		);
	});
});

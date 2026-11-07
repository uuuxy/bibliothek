import { describe, it, expect, vi, beforeEach } from 'vitest';

// leseApiFehler bleibt echt (ARCHITEKTUR.md 8.14).
vi.mock('../../apiFetch.js', async (importOriginal) => ({
	.../** @type {any} */ (await importOriginal()),
	apiFetch: vi.fn()
}));
vi.mock('../../stores/bestaetigung.svelte.js', () => ({ loeschenMitGrund: vi.fn() }));

import { apiFetch } from '../../apiFetch.js';
import { loeschenMitGrund } from '../../stores/bestaetigung.svelte.js';
import { baueLoeschfrage, loescheTitelMitRueckfrage, meldungZumErgebnis } from './titelLoeschen.js';

/** @param {number} status @param {any} koerper */
const antwort = (status, koerper) =>
	/** @type {any} */ ({
		ok: status < 400,
		status,
		json: async () => koerper,
		text: async () => JSON.stringify(koerper)
	});

/** @param {number} n */
const verliehene = (n) =>
	Array.from({ length: n }, (_, i) => ({
		exemplar_id: `ex-${i + 1}`,
		barcode: `B-${i + 1}`,
		titel: 'Mathematik 7',
		leser: `Leser ${i + 1}`,
		klasse: i === 0 ? '' : '7b'
	}));

describe('baueLoeschfrage', () => {
	it('ohne verliehene: Grund und ein Klick auf „Löschen"', () => {
		const frage = baueLoeschfrage(1, 'Mathematik 7', { titel: 1, im_haus: 9, verliehen: [] });
		expect(frage).toEqual({
			titel: '„Mathematik 7“ löschen?',
			text: 'Die 9 Exemplare im Haus kommen mit diesem Grund ins Abgangsbuch.'
		});
	});

	it('ein Exemplar im Haus, Einzahl', () => {
		expect(baueLoeschfrage(1, 'X', { titel: 1, im_haus: 1, verliehen: [] }).text).toBe(
			'Das Exemplar im Haus kommt mit diesem Grund ins Abgangsbuch.'
		);
	});

	it('ohne jedes Exemplar bleibt der Hinweis, dass es endgültig ist', () => {
		expect(baueLoeschfrage(1, 'X', { titel: 1, im_haus: 0, verliehen: [] })).toEqual({
			titel: '„X“ löschen?',
			text: 'Das lässt sich nicht rückgängig machen.'
		});
	});

	it('ohne Namen heißt die Frage nur „Titel löschen?"', () => {
		expect(baueLoeschfrage(1, undefined, { titel: 1, im_haus: 2, verliehen: [] }).titel).toBe(
			'Titel löschen?'
		);
	});

	it('ein verliehenes Exemplar: Es bleibt beim Leser, mit seinem Titel', () => {
		const nachsatz = (/** @type {number} */ anzahlTitel) =>
			baueLoeschfrage(anzahlTitel, undefined, {
				titel: anzahlTitel,
				im_haus: 2,
				verliehen: verliehene(1)
			}).zusatz?.nachsatz;
		expect(nachsatz(1)).toBe('Es bleibt beim Leser, und der Titel bleibt, bis es zurück ist.');
		expect(nachsatz(3)).toBe('Es bleibt beim Leser, und sein Titel bleibt, bis es zurück ist.');
	});

	it('genau fünf Namen stehen alle da, ohne Zählzeile', () => {
		const frage = baueLoeschfrage(1, 'X', { titel: 1, im_haus: 0, verliehen: verliehene(5) });
		expect(frage.zusatz?.liste).toEqual([
			'Leser 1',
			'Leser 2 (7b)',
			'Leser 3 (7b)',
			'Leser 4 (7b)',
			'Leser 5 (7b)'
		]);
	});

	it('mit verliehenen: Namen, eigenes Kästchen und der Knopf nennt die Zahl', () => {
		const frage = baueLoeschfrage(1, 'Mathematik 7', {
			titel: 1,
			im_haus: 9,
			verliehen: verliehene(3)
		});
		expect(frage.aktion).toBe('9 Exemplare löschen');
		expect(frage.zusatz).toEqual({
			hinweis: '3 Exemplare sind verliehen:',
			liste: ['Leser 1', 'Leser 2 (7b)', 'Leser 3 (7b)'],
			nachsatz: 'Sie bleiben bei den Lesern, und der Titel bleibt, bis sie zurück sind.',
			kaestchen: 'Auch diese 3 Exemplare löschen und aus den Leserkonten austragen',
			aktionMit: 'Alle 12 Exemplare löschen',
			noetig: false
		});
	});

	it('alle verliehen: ohne Kästchen gibt es nichts zu löschen', () => {
		const frage = baueLoeschfrage(1, 'X', { titel: 1, im_haus: 0, verliehen: verliehene(1) });
		expect(frage.text).toBe('Im Haus ist kein Exemplar.');
		expect(frage.aktion).toBe('Löschen');
		expect(frage.zusatz?.noetig).toBe(true);
		expect(frage.zusatz?.kaestchen).toBe(
			'Auch dieses Exemplar löschen und aus dem Leserkonto austragen'
		);
		expect(frage.zusatz?.aktionMit).toBe('Das Exemplar löschen');
	});

	it('mehrere Titel; mehr als fünf Namen werden gezählt', () => {
		const frage = baueLoeschfrage(4, undefined, {
			titel: 4,
			im_haus: 20,
			verliehen: verliehene(7)
		});
		expect(frage.titel).toBe('4 Titel löschen?');
		expect(frage.zusatz?.liste).toHaveLength(6);
		expect(frage.zusatz?.liste?.[5]).toBe('und 2 weitere');
		expect(frage.zusatz?.nachsatz).toBe(
			'Sie bleiben bei den Lesern, und ihre Titel bleiben, bis sie zurück sind.'
		);
	});
});

describe('meldungZumErgebnis', () => {
	it.each([
		[{ geloescht: ['a'], geblieben: [] }, 'Titel gelöscht'],
		[{ geloescht: ['a', 'b'], geblieben: [] }, '2 Titel gelöscht'],
		[
			{ geloescht: [], geblieben: ['a'] },
			'Der Titel bleibt, bis die verliehenen Exemplare zurück sind'
		],
		[
			{ geloescht: ['a'], geblieben: ['b'] },
			'1 Titel gelöscht. 1 Titel bleibt, bis die verliehenen Exemplare zurück sind'
		],
		[
			{ geloescht: [], geblieben: ['a', 'b'] },
			'2 Titel bleiben, bis die verliehenen Exemplare zurück sind'
		],
		[
			{ geloescht: ['a'], geblieben: ['b', 'c'] },
			'1 Titel gelöscht. 2 Titel bleiben, bis die verliehenen Exemplare zurück sind'
		]
	])('%o', (ergebnis, meldung) => {
		expect(meldungZumErgebnis(ergebnis)).toBe(meldung);
	});
});

describe('loescheTitelMitRueckfrage', () => {
	const folgen = { titel: 1, im_haus: 2, verliehen: verliehene(2) };
	beforeEach(() => {
		vi.mocked(apiFetch).mockReset();
		vi.mocked(loeschenMitGrund).mockReset();
	});

	it('Abbrechen schickt kein DELETE', async () => {
		vi.mocked(apiFetch).mockResolvedValueOnce(antwort(200, folgen));
		vi.mocked(loeschenMitGrund).mockResolvedValue(null);
		expect(await loescheTitelMitRueckfrage(['t-1'], 'X')).toBeNull();
		expect(apiFetch).toHaveBeenCalledTimes(1);
		expect(apiFetch).toHaveBeenCalledWith('/api/buecher/titel/loeschfolgen?ids=t-1', {
			credentials: 'include'
		});
	});

	it('ohne Kästchen gehen keine verliehenen mit', async () => {
		vi.mocked(apiFetch)
			.mockResolvedValueOnce(antwort(200, folgen))
			.mockResolvedValueOnce(antwort(200, { geloescht: [], geblieben: ['t-1'] }));
		vi.mocked(loeschenMitGrund).mockResolvedValue({ grund: 'AUSSORTIERT', zusatz: false });
		expect(await loescheTitelMitRueckfrage(['t-1'], 'X')).toEqual({
			geloescht: [],
			geblieben: ['t-1']
		});
		expect(apiFetch).toHaveBeenLastCalledWith(
			'/api/books',
			expect.objectContaining({
				method: 'DELETE',
				body: JSON.stringify({ ids: ['t-1'], grund: 'AUSSORTIERT', verliehene: [] })
			})
		);
	});

	it('mit Kästchen gehen genau die gezeigten verliehenen mit', async () => {
		vi.mocked(apiFetch)
			.mockResolvedValueOnce(antwort(200, folgen))
			.mockResolvedValueOnce(antwort(200, { geloescht: ['t-1'], geblieben: [] }));
		vi.mocked(loeschenMitGrund).mockResolvedValue({ grund: 'VERLUST', zusatz: true });
		await loescheTitelMitRueckfrage(['t-1'], 'X');
		expect(apiFetch).toHaveBeenLastCalledWith(
			'/api/books',
			expect.objectContaining({
				body: JSON.stringify({ ids: ['t-1'], grund: 'VERLUST', verliehene: ['ex-1', 'ex-2'] })
			})
		);
	});

	it('nennt die Meldung des Servers', async () => {
		vi.mocked(apiFetch)
			.mockResolvedValueOnce(antwort(200, folgen))
			.mockResolvedValueOnce(
				antwort(409, {
					error: 'Inzwischen wurde ein Exemplar ausgeliehen. Bitte noch einmal löschen.'
				})
			);
		vi.mocked(loeschenMitGrund).mockResolvedValue({ grund: 'AUSSORTIERT', zusatz: true });
		await expect(loescheTitelMitRueckfrage(['t-1'], 'X')).rejects.toThrow(
			'Inzwischen wurde ein Exemplar ausgeliehen. Bitte noch einmal löschen.'
		);
	});
});

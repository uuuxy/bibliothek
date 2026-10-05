import { describe, it, expect, vi, beforeEach } from 'vitest';
import { ladeTitelSchlagworte } from './schlagworte.js';
import { apiFetch } from '../apiFetch.js';

vi.mock('../apiFetch.js', () => ({ apiFetch: vi.fn(), apiPut: vi.fn() }));

// Der Bestellkorb ersetzt die Schlagworte eines Titels als Ganzes. Kennt er die vorhandenen
// nicht, sperrt er das Feld; dafür muss das Laden scheitern, statt eine leere Liste zu liefern.
describe('ladeTitelSchlagworte', () => {
	beforeEach(() => {
		vi.mocked(apiFetch).mockReset();
	});

	/** @param {any} antwort */
	const serverAntwortet = (antwort) => vi.mocked(apiFetch).mockResolvedValue(antwort);

	it('liefert die Liste des Titels', async () => {
		serverAntwortet({ ok: true, json: async () => ({ schlagworte: ['Fantasy', 'Drachen'] }) });

		await expect(ladeTitelSchlagworte('t 1')).resolves.toEqual(['Fantasy', 'Drachen']);
		expect(apiFetch).toHaveBeenCalledWith('/api/buecher/titel/t%201/schlagworte');
	});

	it.each([
		['ohne das Feld', {}],
		['mit null statt einer Liste', { schlagworte: null }],
		['mit einem Text statt einer Liste', { schlagworte: 'Fantasy' }],
		['ohne Körper', null]
	])('scheitert bei einer Antwort %s', async (_was, koerper) => {
		serverAntwortet({ ok: true, json: async () => koerper });

		await expect(ladeTitelSchlagworte('t1')).rejects.toThrow('Schlagworte: Antwort ohne Liste');
	});

	it('scheitert, wenn der Server ablehnt', async () => {
		serverAntwortet({ ok: false, status: 503 });

		await expect(ladeTitelSchlagworte('t1')).rejects.toThrow(
			'Schlagworte konnten nicht geladen werden (503)'
		);
	});
});

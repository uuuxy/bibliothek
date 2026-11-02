import { describe, it, expect, vi, afterEach } from 'vitest';

vi.mock('../../lib/apiFetch.js', () => ({
	apiFetch: vi.fn(),
	leseApiFehler: vi.fn()
}));

import { apiFetch } from '../../lib/apiFetch.js';
import { erzeugeExemplarFrage } from './exemplarImKatalog.svelte.js';

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

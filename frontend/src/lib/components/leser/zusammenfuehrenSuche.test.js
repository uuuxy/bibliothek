import { describe, it, expect, vi, afterEach } from 'vitest';

vi.mock('../../apiFetch.js', () => ({
	apiFetch: vi.fn(),
	leseApiFehler: vi.fn()
}));

import { apiFetch } from '../../apiFetch.js';
import { erzeugeKandidatenSuche } from './zusammenfuehrenSuche.svelte.js';

afterEach(() => {
	vi.useRealTimers();
});

// Ein Handscanner mit amerikanischer Tastatur tippt an einem deutschen Rechner den Bindestrich
// als ß (FACHKONZEPT §1); wer den Ausweis des zweiten Datensatzes scannt, sucht nach der Nummer.
describe('Kandidatensuche beim Zusammenführen', () => {
	it('liest in einer gescannten Nummer ein ß hinter der Vorsilbe als Bindestrich', async () => {
		vi.useFakeTimers();
		vi.mocked(apiFetch).mockResolvedValue(/** @type {any} */ ({ ok: true, json: async () => [] }));
		const suche = erzeugeKandidatenSuche(() => 'p1');

		suche.suche = 'Sß10001';
		suche.tippen();
		await vi.advanceTimersByTimeAsync(250);

		expect(vi.mocked(apiFetch)).toHaveBeenCalledWith(
			'/api/schueler/p1/zusammenfuehren-kandidaten?q=S-10001'
		);
	});
});

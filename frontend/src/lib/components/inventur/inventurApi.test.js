import { describe, it, expect, vi } from 'vitest';

vi.mock('../../apiFetch.js', async (importOriginal) => ({
	.../** @type {any} */ (await importOriginal()),
	apiFetch: vi.fn()
}));

import { apiFetch } from '../../apiFetch.js';
import { serverAntwort } from '../../serverAntwort.js';
import { ladeOffeneSessions, ladeAbgeschlosseneInventuren } from './inventurApi.js';

// Eine leere Liste hieße „keine Inventur läuft" und „keine frühere"; lehnt der Server ab,
// werfen die Abrufe deshalb mit seinem Grund.
describe('inventurApi: Listen der Seite', () => {
	it.each([
		['laufende', ladeOffeneSessions],
		['frühere', ladeAbgeschlosseneInventuren]
	])('wirft bei abgelehntem Abruf der %s mit dem Grund des Servers', async (_name, lade) => {
		vi.mocked(apiFetch).mockResolvedValue(
			serverAntwort(503, { error: 'Die Datenbank antwortet nicht' })
		);
		await expect(lade()).rejects.toThrow('Die Datenbank antwortet nicht');
	});
});

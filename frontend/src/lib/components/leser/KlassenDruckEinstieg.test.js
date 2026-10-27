import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, fireEvent, waitFor } from '@testing-library/svelte';

// leseApiFehler bleibt echt (ARCHITEKTUR.md 8.14).
vi.mock('../../apiFetch.js', async (importOriginal) => ({
	.../** @type {any} */ (await importOriginal()),
	apiFetch: vi.fn()
}));

import { apiFetch } from '../../apiFetch.js';
import { serverAntwort } from '../../serverAntwort.js';
import KlassenDruckEinstieg from './KlassenDruckEinstieg.svelte';

// Im Reiter „Klassenweise drucken" wird die Klasse gewählt, nicht getippt. Kam die Klassenliste
// nicht, blieb die Auswahl leer, ohne ein Wort, in Chromium, Firefox und WebKit (OFFEN.md 5.106).
describe('KlassenDruckEinstieg: die Klassenliste kommt nicht', () => {
	beforeEach(() => vi.clearAllMocks());

	it.each([
		[
			'bei einer Ablehnung',
			async () => serverAntwort(503, { error: 'Die Datenbank antwortet nicht' }),
			'Die Datenbank antwortet nicht'
		],
		[
			'ohne Antwort',
			() => Promise.reject(new Error('Keine Verbindung zum Server.')),
			'Keine Verbindung zum Server.'
		]
	])('nennt %s den Grund und lädt auf Wunsch neu', async (_, antwort, grund) => {
		vi.mocked(apiFetch).mockImplementationOnce(/** @type {any} */ (antwort));
		const k = render(KlassenDruckEinstieg);

		const kasten = await k.findByRole('alert');
		expect(kasten.textContent).toContain('Klassen nicht geladen');
		expect(kasten.textContent).toContain(grund);

		vi.mocked(apiFetch).mockResolvedValueOnce(serverAntwort(200, ['05A', '05B']));
		await fireEvent.click(k.getByRole('button', { name: 'Erneut versuchen' }));
		await waitFor(() => expect(k.queryByRole('alert')).toBeNull());
		expect(k.getByRole('button', { name: 'Klasse zum Druck markieren' })).toBeTruthy();
	});
});

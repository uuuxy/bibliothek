import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent, waitFor } from '@testing-library/svelte';

// leseApiFehler bleibt echt (ARCHITEKTUR.md 8.14).
vi.mock('../../apiFetch.js', async (importOriginal) => ({
	.../** @type {any} */ (await importOriginal()),
	apiFetch: vi.fn()
}));

import { apiFetch } from '../../apiFetch.js';
import { serverAntwort } from '../../serverAntwort.js';
import { authStore } from '../../stores/authStore.svelte.js';
import Leserakte from './Leserakte.svelte';

const apiFetchMock = vi.mocked(apiFetch);

beforeEach(() => {
	apiFetchMock.mockReset();
});

// Kam die Akte nicht, stand „Die Daten dieses Lesers konnten nicht abgerufen werden", gleich ob
// der Server ablehnte oder keine Verbindung bestand; sein Grund fehlte (OFFEN.md 5.106).
describe('Leserakte: Akte nicht geladen', () => {
	it('nennt den Grund des Servers und lädt mit „Erneut versuchen" nach', async () => {
		apiFetchMock.mockResolvedValue(serverAntwort(503, { error: 'Datenbank nicht erreichbar' }));
		render(Leserakte, { leser: { id: 'A' }, onDeselect: vi.fn() });

		await waitFor(() => expect(screen.getByText('Akte nicht geladen')).toBeTruthy());
		expect(screen.getByText('Datenbank nicht erreichbar')).toBeTruthy();

		apiFetchMock.mockImplementation(async (url) =>
			String(url) === '/api/schueler/A'
				? serverAntwort(200, { id: 'A', vorname: 'Anna', nachname: 'Probe', klasse: '05A' })
				: serverAntwort(200, [])
		);
		await fireEvent.click(screen.getByRole('button', { name: 'Erneut versuchen' }));
		await waitFor(() => expect(screen.queryByText('Akte nicht geladen')).toBeNull());
	});

	it('nennt ohne Verbindung den Satz der Verbindung', async () => {
		vi.spyOn(console, 'error').mockImplementation(() => {});
		apiFetchMock.mockRejectedValue(new Error('Keine Verbindung zum Server.'));
		render(Leserakte, { leser: { id: 'A' }, onDeselect: vi.fn() });

		await waitFor(() => expect(screen.getByText('Keine Verbindung zum Server.')).toBeTruthy());
	});
});

// Das getippte Ablaufjahr gilt für den Ausdruck dieses einen Lesers. Bliebe es beim Wechsel
// stehen, trüge der nächste Ausweis das Jahr eines anderen Schülers.
describe('Leserakte: Ablaufjahr beim Wechsel des Lesers', () => {
	it('fällt auf den Vorschlag des nächsten Lesers zurück', async () => {
		authStore.currentUser = /** @type {any} */ ({
			rolle: 'mitarbeiter',
			permissions: ['view_students']
		});
		apiFetchMock.mockImplementation(async (url) => {
			if (String(url) === '/api/schueler/A')
				return serverAntwort(200, {
					id: 'A',
					vorname: 'Anna',
					nachname: 'A',
					klasse: '05A',
					ausweis_gueltig_bis: 2030
				});
			if (String(url) === '/api/schueler/B')
				return serverAntwort(200, {
					id: 'B',
					vorname: 'Ben',
					nachname: 'B',
					klasse: '09A',
					ausweis_gueltig_bis: 2027
				});
			return serverAntwort(200, []);
		});
		const ansicht = render(Leserakte, { leser: { id: 'A' }, onDeselect: vi.fn() });
		const jahr = () =>
			/** @type {HTMLInputElement} */ (screen.getByLabelText('Ablaufjahr des Ausweises'));
		await waitFor(() => expect(jahr().value).toBe('2030'));

		await fireEvent.input(jahr(), { target: { value: '2041' } });
		await waitFor(() => expect(jahr().value).toBe('2041'));

		await ansicht.rerender({ leser: { id: 'B' } });
		await waitFor(() => expect(jahr().value).toBe('2027'));
	});
});

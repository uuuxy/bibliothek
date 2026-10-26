import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, fireEvent, screen } from '@testing-library/svelte';
import { serverAntwort } from '../../serverAntwort.js';

// Kommt die Liste der Meldungen nicht, steht an ihrer Stelle der Grund mit „Erneut versuchen".
// „Nichts offen." hieße, es sei alles erledigt, während das Band offene Meldungen zählt.
vi.mock('../../liveEvents.js', () => ({ abonniere: () => () => {} }));
const fetchMock = vi.fn();
vi.mock('../../apiFetch.js', async (importOriginal) => ({
	.../** @type {any} */ (await importOriginal()),
	apiFetch: (...args) => fetchMock(...args),
	apiClient: { post: vi.fn() }
}));

const { nachbuchMeldungen } = await import('../../stores/nachbuchMeldungen.svelte.js');
const { default: NachbuchMeldungen } = await import('./NachbuchMeldungen.svelte');

const zeile = {
	id: 'm1',
	barcode: 'B-1',
	titel: 'Momo',
	ergebnis: 'nicht_gebucht',
	grund: 'Ausweis unbekannt',
	gescannt_am: '2026-10-26T10:00:00Z'
};

describe('Meldungen aus dem Nachbuchen: die Liste kommt nicht', () => {
	beforeEach(() => {
		nachbuchMeldungen._zuruecksetzen();
		fetchMock.mockReset();
	});

	it('zeigt den Grund des Servers statt „Nichts offen." und lädt auf „Erneut versuchen"', async () => {
		fetchMock.mockResolvedValueOnce(serverAntwort(500, { error: 'Liste vom Server abgelehnt' }));
		render(NachbuchMeldungen);
		nachbuchMeldungen.oeffne();

		await vi.waitFor(() => expect(screen.getByText('Meldungen nicht geladen')).toBeTruthy());
		expect(screen.getByText('Liste vom Server abgelehnt')).toBeTruthy();
		expect(screen.queryByText('Nichts offen.')).toBeNull();

		fetchMock.mockResolvedValueOnce(serverAntwort(200, [zeile]));
		await fireEvent.click(screen.getByRole('button', { name: 'Erneut versuchen' }));

		await vi.waitFor(() => expect(screen.getByText('Ausweis unbekannt')).toBeTruthy());
		expect(screen.queryByText('Meldungen nicht geladen')).toBeNull();
	});

	it('zeigt nach einem Netzausfall keine alte Liste unter einem anderen Filter', async () => {
		fetchMock.mockResolvedValueOnce(serverAntwort(200, [zeile]));
		render(NachbuchMeldungen);
		nachbuchMeldungen.oeffne();
		await vi.waitFor(() => expect(screen.getByText('Ausweis unbekannt')).toBeTruthy());

		fetchMock.mockRejectedValueOnce(new Error('Keine Verbindung zum Server.'));
		await fireEvent.click(screen.getByRole('checkbox', { name: 'Auch erledigte zeigen' }));

		await vi.waitFor(() => expect(screen.getByText('Meldungen nicht geladen')).toBeTruthy());
		expect(screen.getByText('Keine Verbindung zum Server.')).toBeTruthy();
		expect(screen.queryByText('Ausweis unbekannt')).toBeNull();
	});
});

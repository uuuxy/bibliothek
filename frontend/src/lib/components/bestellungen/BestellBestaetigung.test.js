import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, fireEvent, screen } from '@testing-library/svelte';
import BestellBestaetigung from './BestellBestaetigung.svelte';
import { apiFetch } from '../../apiFetch.js';
import { serverAntwort } from '../../serverAntwort.js';

vi.mock('../../apiFetch.js', async (importOriginal) => ({
	.../** @type {any} */ (await importOriginal()),
	apiFetch: vi.fn()
}));

const BESTELLUNG = {
	schule_name: 'Testschule',
	bestelldatum: '2026-10-01T00:00:00Z',
	lieferant_name: 'Buchhandlung Lesezeichen',
	anzahl_exemplare: 2,
	positionen: [{ titel_name: 'Green Line 3', isbn: '', menge: 2 }],
	etiketten_vorhanden: false
};

beforeEach(() => {
	vi.mocked(apiFetch).mockReset();
	window.history.pushState({}, '', '/bestellung/probe-token');
});

// Den Satz „Dieser Link ist nicht mehr gültig" sagt der Server mit 404. Ist er nur kurz nicht
// erreichbar oder scheitert etwas anderes, hielte der Lieferant den Link sonst für abgelaufen.
describe('Bestellbestätigung des Lieferanten: laden', () => {
	it('sagt bei 404, dass der Link nicht mehr gültig ist', async () => {
		vi.mocked(apiFetch).mockResolvedValue(
			serverAntwort(404, { error: 'Dieser Link ist nicht mehr gültig' })
		);
		render(BestellBestaetigung);

		expect(
			await screen.findByRole('heading', { name: 'Dieser Link ist nicht mehr gültig' })
		).toBeTruthy();
	});

	it('nennt bei einer Störung den Grund und lädt auf „Erneut versuchen"', async () => {
		vi.mocked(apiFetch)
			.mockResolvedValueOnce(
				serverAntwort(500, { error: 'Ein interner Serverfehler ist aufgetreten' })
			)
			.mockResolvedValueOnce(serverAntwort(200, BESTELLUNG));
		render(BestellBestaetigung);

		expect(await screen.findByText('Bestellung nicht geladen')).toBeTruthy();
		expect(screen.getByText('Ein interner Serverfehler ist aufgetreten')).toBeTruthy();
		expect(screen.queryByText('Dieser Link ist nicht mehr gültig')).toBeNull();

		await fireEvent.click(screen.getByRole('button', { name: 'Erneut versuchen' }));

		expect(await screen.findByText('Green Line 3')).toBeTruthy();
	});

	it('sagt bei Netzausfall, dass keine Verbindung besteht', async () => {
		vi.mocked(apiFetch).mockRejectedValue(new Error('Keine Verbindung zum Server.'));
		render(BestellBestaetigung);

		expect(await screen.findByText('Keine Verbindung zum Server.')).toBeTruthy();
		expect(screen.queryByText('Dieser Link ist nicht mehr gültig')).toBeNull();
	});
});

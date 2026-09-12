import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render } from '@testing-library/svelte';
import KlassensatzReservierungen from './KlassensatzReservierungen.svelte';
import { apiFetch } from '../../apiFetch.js';

vi.mock('../../apiFetch.js', () => ({
	apiFetch: vi.fn(),
	extractApiError: vi.fn(async (/** @type {any} */ res) => JSON.parse(await res.text()).error)
}));
vi.mock('../../stores/toastStore.svelte.js', () => ({ toastStore: { addToast: vi.fn() } }));

// Dieselbe Klasse wie bei den Anliegen: Eine leere Warteschlange heißt „niemand wartet",
// und genau das stand bis zum 12.09.2026 auch dann da, wenn der Abruf gescheitert war.
// Klassensätze sind Unterrichtsvorbereitung — wer hier wartet, wartet auf Bücher für
// eine Stunde mit Termin.
describe('KlassensatzReservierungen', () => {
	beforeEach(() => vi.mocked(apiFetch).mockReset());

	it('unterscheidet „niemand wartet" von „nicht geladen"', async () => {
		vi.mocked(apiFetch).mockResolvedValue(
			/** @type {any} */ ({
				ok: false,
				status: 500,
				text: async () => JSON.stringify({ error: 'Datenbank nicht erreichbar' })
			})
		);
		const { findByText, queryByText } = render(KlassensatzReservierungen);

		expect(await findByText('Datenbank nicht erreichbar')).toBeTruthy();
		expect(queryByText('Keine offenen Klassensatz-Reservierungen.')).toBeNull();
	});

	it('leer bleibt leer', async () => {
		vi.mocked(apiFetch).mockResolvedValue(/** @type {any} */ ({ ok: true, json: async () => [] }));
		const { findByText } = render(KlassensatzReservierungen);
		expect(await findByText('Keine offenen Klassensatz-Reservierungen.')).toBeTruthy();
	});
});

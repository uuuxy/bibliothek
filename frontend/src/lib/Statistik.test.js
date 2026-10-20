import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, fireEvent } from '@testing-library/svelte';
import Statistik from './Statistik.svelte';
import { apiFetch } from './apiFetch.js';

vi.mock('./apiFetch.js', async (importOriginal) => ({
	.../** @type {any} */ (await importOriginal()),
	apiFetch: vi.fn()
}));

/** Eine Statistik ohne Ausleihen und ohne Ladenhüter. @param {string[]} nichtGelesen */
function antworte(nichtGelesen) {
	vi.mocked(apiFetch).mockImplementation(
		async (url) =>
			/** @type {any} */ ({
				ok: true,
				json: async () =>
					String(url).startsWith('/api/dashboard/summary')
						? { total_overdue: 0, overdue_buckets: [] }
						: {
								popular_titles: [],
								shelf_warmers: [],
								monats_trend: [],
								loss_stats: { gesamt_bestand: 0, verlorene_exemplare: 0, verlust_quote: 0 },
								nicht_gelesen: nichtGelesen
							}
			})
	);
}

describe('Statistik', () => {
	beforeEach(() => {
		vi.mocked(apiFetch).mockReset();
	});

	// Die Gegenprobe zum Test darunter: Ein leerer Bestand zeigt die Leerzustände.
	it('zeigt bei leerem Bestand die Leerzustände', async () => {
		antworte([]);
		const screen = render(Statistik);
		expect(await screen.findByText('Gesamtbestand')).toBeTruthy();
		expect(screen.getByText('Noch keine Ausleih-Aktivität im letzten Jahr.')).toBeTruthy();
		await fireEvent.click(screen.getByRole('button', { name: 'Ladenhüter' }));
		expect(screen.getByText(/Keine Ladenhüter/)).toBeTruthy();
	});

	// Ein Abschnitt, den der Server nicht lesen konnte, ist kein leerer Bestand: Statt vier
	// Nullen und „Keine Ladenhüter" in Grün steht dort, was nicht geladen wurde.
	it('zeigt statt des Leerzustands, welcher Abschnitt nicht geladen wurde', async () => {
		antworte(['renner', 'ladenhueter', 'kennzahlen', 'monats_trend']);
		const screen = render(Statistik);
		expect(await screen.findByText('Kennzahlen nicht geladen')).toBeTruthy();
		expect(screen.queryByText('Gesamtbestand')).toBeNull();
		expect(screen.getByText('Aktivität nicht geladen')).toBeTruthy();
		expect(screen.queryByText(/Noch keine Ausleih-Aktivität/)).toBeNull();
		expect(screen.getByText('Renner nicht geladen')).toBeTruthy();
		expect(screen.queryByText(/Noch keine Ausleihen registriert/)).toBeNull();

		await fireEvent.click(screen.getByRole('button', { name: 'Ladenhüter' }));
		expect(screen.getByText('Ladenhüter nicht geladen')).toBeTruthy();
		expect(screen.queryByText(/Keine Ladenhüter/)).toBeNull();
	});
});

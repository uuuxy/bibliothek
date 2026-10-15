import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, fireEvent, waitFor } from '@testing-library/svelte';
import SchlagworteKategorie from './SchlagworteKategorie.svelte';
import { apiGet } from '../../../apiFetch.js';

vi.mock('../../../apiFetch.js', () => ({ apiGet: vi.fn(), apiPut: vi.fn(), apiPost: vi.fn() }));

// Die Pflegeseite sucht seit dem 30.09.2026 am Server über alle Wörter (docs/OFFEN.md 4.20).
// Bis dahin lud sie die ersten 5.000 in alphabetischer Folge und suchte im Browser nur
// darunter; die 13.207 Wörter aus Littera hätten zu zwei Dritteln gefehlt.

/** @param {string} wort @param {object} [extra] */
const zeile = (wort, extra = {}) => ({
	id: wort,
	wort,
	titel: 1,
	verweise: [],
	ist_filter: false,
	...extra
});

describe('SchlagworteKategorie', () => {
	// Als Block: Gäbe der Schritt die Mock-Funktion zurück, riefe Vitest sie nach dem Test als
	// Aufräum-Schritt auf — ohne Pfad.
	beforeEach(() => {
		vi.mocked(apiGet).mockReset();
	});

	it('sucht am Server und sagt, wie viele Wörter passen', async () => {
		vi.mocked(apiGet).mockImplementation(async (/** @type {string} */ pfad) => {
			return pfad.includes('suche=')
				? {
						zeilen: [zeile('Zwergpinguin', { ist_filter: true })],
						gesamt: 13207,
						verweise: 0,
						filter: 1,
						treffer: 1
					}
				: {
						zeilen: Array.from({ length: 200 }, (_, i) => zeile(`Wort ${i}`)),
						gesamt: 13207,
						verweise: 0,
						filter: 1,
						treffer: 13207
					};
		});
		const screen = render(SchlagworteKategorie);
		await waitFor(() => expect(screen.container.textContent).toMatch(/200 von 13207 angezeigt/));
		expect(screen.container.textContent).toMatch(/13207 Schlagworte · 1 als Filter im Portal/);

		await fireEvent.input(screen.getByLabelText('Schlagwort suchen'), {
			target: { value: 'zwerg' }
		});
		await waitFor(() =>
			expect(apiGet).toHaveBeenLastCalledWith('/api/schlagworte/pflege?suche=zwerg')
		);
		await waitFor(() => expect(screen.container.textContent).toContain('Zwergpinguin'));
		expect(screen.container.textContent).not.toMatch(/angezeigt/);
	});
});

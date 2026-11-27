import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, cleanup } from '@testing-library/svelte';
import UeberfaelligQuote from './UeberfaelligQuote.svelte';
import { apiFetch } from '../../apiFetch.js';
import { authStore } from '../../stores/authStore.svelte.js';

vi.mock('../../apiFetch.js', async (importOriginal) => ({
	.../** @type {any} */ (await importOriginal()),
	apiFetch: vi.fn()
}));

/** @param {string[]} rechte */
function melde(rechte) {
	authStore.currentUser = /** @type {any} */ ({ rolle: 'mitarbeiter', permissions: rechte });
}

beforeEach(() => {
	vi.mocked(apiFetch).mockReset();
	vi.mocked(apiFetch).mockResolvedValue(
		/** @type {any} */ ({
			ok: true,
			status: 200,
			json: async () => ({ total_overdue: 0, overdue_buckets: [] })
		})
	);
});

afterEach(cleanup);

// Die Statistik öffnet mit view_stats, das Mahnwesen nicht: Der Knopf führte dann auf die Startseite.
describe('UeberfaelligQuote: der Weg ins Mahnwesen', () => {
	it('fehlt, wenn das Konto das Mahnwesen nicht öffnen darf', async () => {
		melde(['view_stats']);
		const screen = render(UeberfaelligQuote);

		await screen.findByText('Keine überfälligen Ausleihen');
		expect(screen.queryByRole('button', { name: 'Im Mahnwesen bearbeiten' })).toBeNull();
	});

	it('steht da, wenn das Konto das Mahnwesen öffnen darf', async () => {
		melde(['view_stats', 'view_students']);
		const screen = render(UeberfaelligQuote);

		await screen.findByText('Keine überfälligen Ausleihen');
		expect(screen.getByRole('button', { name: 'Im Mahnwesen bearbeiten' })).toBeTruthy();
	});
});

// Die Quote ist überfällig ÷ laufend. Ohne laufende Ausleihe gibt es keine, statt „NaN %“ oder „Infinity %“.
describe('UeberfaelligQuote: die Quote', () => {
	beforeEach(() => {
		melde(['view_stats']);
		vi.mocked(apiFetch).mockResolvedValue(
			/** @type {any} */ ({
				ok: true,
				status: 200,
				json: async () => ({ total_overdue: 3, laufende_ausleihen: 12, overdue_buckets: [] })
			})
		);
	});

	// Zähler und Nenner kommen aus derselben Abfrage, ohne Dauerleihen; „aktuell verliehen“ der
	// Übersicht zählt Dauerleihen mit und ist nicht der Nenner.
	it('rechnet gegen die laufenden Ausleihen der Mahnzahlen, nicht gegen „aktuell verliehen“', async () => {
		const screen = render(UeberfaelligQuote, /** @type {any} */ ({ aktuellVerliehen: 30 }));
		expect(await screen.findByText('≈ 25 % der laufenden Ausleihen')).toBeTruthy();
	});

	it('nennt ohne laufende Ausleihe keine Quote', async () => {
		vi.mocked(apiFetch).mockResolvedValue(
			/** @type {any} */ ({
				ok: true,
				status: 200,
				json: async () => ({ total_overdue: 3, laufende_ausleihen: 0, overdue_buckets: [] })
			})
		);
		const screen = render(UeberfaelligQuote);
		await vi.waitFor(() => expect(screen.container.textContent).toContain('3'));
		expect(screen.container.textContent).not.toContain('der laufenden Ausleihen');
	});
});

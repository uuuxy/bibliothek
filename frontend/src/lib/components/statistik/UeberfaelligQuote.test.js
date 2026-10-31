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

		await screen.findByText('Keine überfälligen Ausleihen.');
		expect(screen.queryByRole('button', { name: 'Im Mahnwesen bearbeiten' })).toBeNull();
	});

	it('steht da, wenn das Konto das Mahnwesen öffnen darf', async () => {
		melde(['view_stats', 'view_students']);
		const screen = render(UeberfaelligQuote);

		await screen.findByText('Keine überfälligen Ausleihen.');
		expect(screen.getByRole('button', { name: 'Im Mahnwesen bearbeiten' })).toBeTruthy();
	});
});

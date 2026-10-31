import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, cleanup, waitFor } from '@testing-library/svelte';
import LieferantenKategorie from './LieferantenKategorie.svelte';
import { apiGet } from '../../../apiFetch.js';
import { authStore } from '../../../stores/authStore.svelte.js';

vi.mock('../../../apiFetch.js', async (importOriginal) => ({
	.../** @type {any} */ (await importOriginal()),
	apiGet: vi.fn()
}));

const SATZ = /braucht zusätzlich das Recht „Bestellungen anzeigen“/;

/** @param {string[]} rechte */
function melde(rechte) {
	authStore.currentUser = /** @type {any} */ ({ rolle: 'mitarbeiter', permissions: rechte });
}

beforeEach(() => {
	vi.mocked(apiGet).mockReset();
	vi.mocked(apiGet).mockResolvedValue([]);
});

afterEach(cleanup);

// Die Kategorie öffnet mit create_orders, die Liste steht hinter view_orders: Ohne das Recht kam
// bei jedem Öffnen eine Ablehnung, und die Seite zeigte „Lieferanten nicht geladen".
describe('LieferantenKategorie nach Rechten', () => {
	it('fragt ohne view_orders nichts ab und nennt das fehlende Recht', () => {
		melde(['create_orders']);
		const screen = render(LieferantenKategorie);

		expect(screen.getByText(SATZ)).toBeTruthy();
		expect(apiGet).not.toHaveBeenCalled();
	});

	it('lädt mit view_orders die Liste', async () => {
		melde(['create_orders', 'view_orders']);
		const screen = render(LieferantenKategorie);

		expect(screen.queryByText(SATZ)).toBeNull();
		await waitFor(() => expect(apiGet).toHaveBeenCalledWith('/api/lieferanten'));
	});
});

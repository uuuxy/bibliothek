import { describe, it, expect, vi } from 'vitest';
import { render, fireEvent } from '@testing-library/svelte';

vi.mock('../../apiFetch.js', () => ({
	apiPost: vi.fn(async () => ({
		exists: false,
		titel_id: 't-9',
		titel: 'Dunkelnacht',
		autor: 'Boie, Kirsten',
		isbn: '9783751200530',
		ist_lernmittel: false,
		schlagwort_vorschlaege: ['Krieg']
	})),
	apiPut: vi.fn(async () => ({})),
	apiFetch: vi.fn(async (/** @type {string} */ url) =>
		url === '/api/buecher/titel/t-9/schlagworte'
			? { ok: true, json: async () => ({ id: 't-9', schlagworte: [] }) }
			: { ok: true, json: async () => [] }
	),
	apiClient: { post: vi.fn() }
}));
vi.mock('../../stores/toastStore.svelte.js', () => ({ toastStore: { addToast: vi.fn() } }));
vi.mock('../../stores/orderStore.svelte.js', () => ({
	orderStore: {
		searchResults: [
			{ source: 'dnb', isbn: '9783751200530', titel: 'Dunkelnacht', autor: 'Boie, Kirsten' }
		],
		showDropdown: true,
		searchLoading: false,
		searchQuery: '',
		suppliers: [],
		selectedSupplierId: '',
		handleSearchInput: vi.fn(),
		resetSearch: vi.fn(),
		addToCart: vi.fn(),
		preiseErfassen: true
	}
}));

import { apiPost } from '../../apiFetch.js';
import OrderSearch from './OrderSearch.svelte';

// Ein DNB-Treffer wird über POST /api/buecher/aus-isbn angelegt, und das Fenster öffnet sich
// mit dem, was die Tür zurückgibt. Seit dem 23.09.2026 gehört der Schlagwort-Vorschlag dazu
// (docs/OFFEN.md 4.20) — fällt er bei der Übergabe ans Fenster weg, ist der ganze Vorschlag
// unsichtbar, ohne dass irgendetwas scheitert.
describe('OrderSearch: ein DNB-Treffer bringt seinen Schlagwort-Vorschlag ins Fenster', () => {
	it('zeigt den Vorschlag der Tür im Fenster zum Übernehmen', async () => {
		const screen = render(OrderSearch);
		await fireEvent.click(screen.getByRole('button', { name: /Dunkelnacht/ }));

		expect(apiPost).toHaveBeenCalledWith('/api/buecher/aus-isbn', { isbn: '9783751200530' });
		const angebot = await screen.findByRole('button', { name: '„Krieg“ übernehmen' });
		await vi.waitFor(() => expect(/** @type {HTMLButtonElement} */ (angebot).disabled).toBe(false));
	});
});

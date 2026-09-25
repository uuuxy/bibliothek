import { describe, it, expect, vi, beforeEach } from 'vitest';
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
import { orderStore } from '../../stores/orderStore.svelte.js';
import OrderSearch from './OrderSearch.svelte';

// Die Frage schließt die Trefferliste; der nächste Test braucht sie wieder offen.
beforeEach(() => {
	orderStore.showDropdown = true;
	vi.mocked(apiPost).mockClear();
});

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

// Steht die ISBN des DNB-Treffers nur in der anderen Länge im Katalog (ISBN-10 ↔ ISBN-13), legt
// die Tür nichts an und antwortet mit andere_form (docs/OFFEN.md 4.18, Stufe 4). Die Titelsuche
// fragt dann; das Fenster öffnet sich erst mit der Wahl.
describe('OrderSearch: dieselbe ISBN in der anderen Länge', () => {
	const frage = {
		exists: false,
		titel_id: '',
		isbn: '9783751200530',
		andere_form: {
			exists: true,
			titel_id: 't-10',
			titel: 'Dunkelnacht (Altbestand)',
			isbn: '3751200533',
			ist_lernmittel: false
		}
	};

	async function bisZurFrage() {
		vi.mocked(apiPost).mockResolvedValueOnce(frage);
		const screen = render(OrderSearch);
		await fireEvent.click(screen.getByRole('button', { name: /Dunkelnacht/ }));
		await screen.findByText(/in zehnstelliger Form/);
		return screen;
	}

	it('fragt, statt das Fenster zu öffnen', async () => {
		const screen = await bisZurFrage();
		expect(screen.queryByRole('button', { name: 'In den Warenkorb' })).toBeNull();
		expect(screen.getByRole('button', { name: /Diesen Titel nehmen/ }).textContent).toContain(
			'Dunkelnacht (Altbestand)'
		);
		const neu = screen.getByRole('button', { name: /Neu anlegen/ }).textContent;
		expect(neu).toContain('„Dunkelnacht"');
		expect(neu).toContain('9783751200530 aus der DNB');
	});

	it('„Diesen Titel nehmen" öffnet das Fenster mit dem Titel aus dem Katalog', async () => {
		const screen = await bisZurFrage();
		await fireEvent.click(screen.getByRole('button', { name: /Diesen Titel nehmen/ }));
		await screen.findByRole('button', { name: 'In den Warenkorb' });
		expect(screen.getByText('Dunkelnacht (Altbestand)')).toBeTruthy();
		expect(screen.queryByText(/in zehnstelliger Form/)).toBeNull();
		expect(apiPost).toHaveBeenCalledTimes(1);
	});

	it('„Neu anlegen" fragt die Tür erneut, mit neu_anlegen', async () => {
		const screen = await bisZurFrage();
		await fireEvent.click(screen.getByRole('button', { name: /Neu anlegen/ }));
		await screen.findByRole('button', { name: 'In den Warenkorb' });
		expect(apiPost).toHaveBeenLastCalledWith('/api/buecher/aus-isbn', {
			isbn: '9783751200530',
			neu_anlegen: true
		});
	});
});

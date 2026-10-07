import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, fireEvent, waitFor } from '@testing-library/svelte';

vi.mock('../inventur/lib/store.svelte.js', () => ({ showToast: vi.fn() }));
vi.mock('./stores/bestaetigung.svelte.js', () => ({
	loeschenBestaetigen: vi.fn(async () => true)
}));
vi.mock('./apiFetch.js', async (original) => ({
	.../** @type {any} */ (await original()),
	apiFetch: vi.fn()
}));

import { apiFetch } from './apiFetch.js';
import BookExemplareTab from './BookExemplareTab.svelte';
import { authStore } from './stores/authStore.svelte.js';

// Der Kopf der Buchakte nennt „1 von 2 verfügbar". Nach „Exemplar löschen" führt der Reiter
// die zwei Zahlen nach: Ein bestelltes Exemplar, das nie eintraf, stand nie im Bestand, sein
// Löschen ändert ihn nicht.
describe('Buchakte: Exemplar löschen führt den Bestand im Kopf nach', () => {
	const exemplare = () => [
		{ id: 'e1', barcode_id: 'B-1', ist_ausleihbar: true, ist_verfuegbar: true, im_bestand: true },
		{ id: 'e2', barcode_id: 'B-2', ist_ausleihbar: true, ist_verfuegbar: false, im_bestand: true },
		{ id: 'e3', barcode_id: 'B-3', ist_ausleihbar: false, ist_verfuegbar: true, im_bestand: false }
	];

	beforeEach(() => {
		vi.mocked(apiFetch).mockReset();
		vi.mocked(apiFetch).mockResolvedValue(/** @type {any} */ ({ ok: true, status: 200 }));
		authStore.currentUser = /** @type {any} */ ({ rolle: 'admin', permissions: ['*'] });
	});

	/** @param {number} karte Stelle der Karte in der Liste */
	async function loesche(karte) {
		const book = { id: 't-1', title: 'Momo', gesamt: 2, verfuegbar: 1 };
		const screen = render(BookExemplareTab, {
			props: { exemplare: exemplare(), book, loadAll: () => {} }
		});
		await fireEvent.click(screen.getAllByRole('button', { name: 'Exemplar löschen' })[karte]);
		await waitFor(() => expect(apiFetch).toHaveBeenCalledTimes(1));
		await waitFor(() =>
			expect(screen.getAllByRole('button', { name: 'Exemplar löschen' })).toHaveLength(2)
		);
		return book;
	}

	it('lässt den Bestand stehen, wenn ein bestelltes Exemplar gelöscht wird', async () => {
		const book = await loesche(2);
		expect({ gesamt: book.gesamt, verfuegbar: book.verfuegbar }).toEqual({
			gesamt: 2,
			verfuegbar: 1
		});
	});

	it('zieht ein verfügbares Exemplar des Bestands von beiden Zahlen ab', async () => {
		const book = await loesche(0);
		expect({ gesamt: book.gesamt, verfuegbar: book.verfuegbar }).toEqual({
			gesamt: 1,
			verfuegbar: 0
		});
	});
});

import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, fireEvent } from '@testing-library/svelte';
import BookExemplareTab from './BookExemplareTab.svelte';
import { authStore } from './stores/authStore.svelte.js';
import { printQueue, clearPrintQueue } from './stores/printQueue.svelte.js';

vi.mock('../inventur/lib/store.svelte.js', () => ({ showToast: vi.fn() }));

/**
 * Das Etikett eines Exemplars entsteht im Druck-Center, auf dem Bogen nach der Vorlage
 * (OFFEN.md 5.5). Die Buchakte übergibt nur — über dieselbe printQueue wie Wareneingang und
 * Nachdruck. Den Knopf bekommt nur, wer das Druck-Center öffnen darf: Sonst stellte der
 * Router den gesperrten Reiter zurück, und der Klick endete auf dem ersten erlaubten.
 */

const book = { id: 't-1', title: 'Seydlitz Geographie', author: 'Anna Autorin' };

/** Zeigt ein Exemplar mit dieser Nummer und liefert den Etikett-Knopf der Karte (oder null). */
function etikettKnopf(nummer = 'B-10001') {
	const exemplare = [
		{ id: 'ex-1', barcode_id: nummer, ist_ausleihbar: true, ist_verfuegbar: true }
	];
	const screen = render(BookExemplareTab, { props: { exemplare, book, loadAll: () => {} } });
	return screen.queryByRole('button', { name: `Etikett für ${nummer} im Druck-Center drucken` });
}

beforeEach(() => {
	clearPrintQueue();
});

describe('Etikett aus der Buchakte', () => {
	it('übergibt das Exemplar mit Titel und Autor an das Druck-Center', async () => {
		authStore.currentUser = { rolle: 'mitarbeiter', permissions: ['edit_books', 'view_students'] };

		await fireEvent.click(/** @type {HTMLElement} */ (etikettKnopf()));

		expect(printQueue.copies).toEqual([
			{ barcode_id: 'B-10001', titel: 'Seydlitz Geographie', autor: 'Anna Autorin' }
		]);
	});

	it('bietet den Knopf nicht an, wer das Druck-Center nicht öffnen darf', () => {
		authStore.currentUser = { rolle: 'mitarbeiter', permissions: ['edit_books'] };

		expect(etikettKnopf()).toBeNull();
	});

	// Ein Buch aus Littera trägt die 13 Ziffern seines alten Etiketts als Nummer. Bis zum
	// 30.09.2026 gab es den Knopf nur bei „B-", also für keines dieser Bücher.
	it('bietet den Knopf auch bei einem Buch aus Littera an', () => {
		authStore.currentUser = { rolle: 'mitarbeiter', permissions: ['edit_books', 'view_students'] };

		expect(etikettKnopf('5896800039556')).not.toBeNull();
	});

	it.each(['AUTO-7', 'SYS-100001'])(
		'bietet bei der Platzhalternummer %s keinen Knopf an',
		(nummer) => {
			authStore.currentUser = {
				rolle: 'mitarbeiter',
				permissions: ['edit_books', 'view_students']
			};

			expect(etikettKnopf(nummer)).toBeNull();
		}
	);
});

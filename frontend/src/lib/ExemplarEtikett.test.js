import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, fireEvent } from '@testing-library/svelte';
import BuchakteExemplare from './BuchakteExemplare.svelte';
import { authStore } from './stores/authStore.svelte.js';
import { druckWarteschlange, leereDruckwarteschlange } from './stores/druckWarteschlange.svelte.js';

vi.mock('../inventur/lib/store.svelte.js', () => ({ melde: vi.fn() }));

/**
 * Das Etikett eines Exemplars entsteht im Druck-Center, auf dem Bogen nach der Vorlage
 * (OFFEN.md 5.5). Die Buchakte übergibt nur — über dieselbe druckWarteschlange wie Wareneingang und
 * Nachdruck. Den Knopf bekommt nur, wer im Druck-Center den Reiter „Buch-Etiketten" öffnen
 * darf (view_books, POST /api/print/labels): Sonst endete der Klick auf einem anderen Reiter.
 */

const book = { id: 't-1', title: 'Seydlitz Geographie', author: 'Anna Autorin' };

/** Zeigt ein Exemplar mit dieser Nummer und liefert den Etikett-Knopf der Karte (oder null). */
function etikettKnopf(nummer = 'B-10001') {
	const exemplare = [
		{ id: 'ex-1', barcode_id: nummer, ist_ausleihbar: true, ist_verfuegbar: true }
	];
	const screen = render(BuchakteExemplare, {
		props: { exemplare, titel: book, ladeAlles: () => {} }
	});
	return screen.queryByRole('button', { name: `Etikett für ${nummer} im Druck-Center drucken` });
}

beforeEach(() => {
	leereDruckwarteschlange();
});

describe('Etikett aus der Buchakte', () => {
	it('übergibt das Exemplar mit Titel und Autor an das Druck-Center', async () => {
		authStore.currentUser = { rolle: 'mitarbeiter', permissions: ['edit_books', 'view_books'] };

		await fireEvent.click(/** @type {HTMLElement} */ (etikettKnopf()));

		expect(druckWarteschlange.copies).toEqual([
			{ barcode_id: 'B-10001', titel: 'Seydlitz Geographie', autor: 'Anna Autorin' }
		]);
	});

	it('bietet den Knopf nicht an, wer die Buch-Etiketten nicht öffnen darf', () => {
		// edit_books öffnet das Druck-Center (Reiter „Fehlende Etiketten"), view_students die
		// Ausweise; gedruckt wird das Etikett aber im Reiter „Buch-Etiketten".
		for (const rechte of [['edit_books'], ['edit_books', 'view_students']]) {
			authStore.currentUser = { rolle: 'mitarbeiter', permissions: rechte };
			expect(etikettKnopf(), rechte.join(', ')).toBeNull();
		}
	});

	// Ein Buch aus Littera trägt die 13 Ziffern seines alten Etiketts als Nummer. Bis zum
	// 30.09.2026 gab es den Knopf nur bei „B-", also für keines dieser Bücher.
	it('bietet den Knopf auch bei einem Buch aus Littera an', () => {
		authStore.currentUser = { rolle: 'mitarbeiter', permissions: ['edit_books', 'view_books'] };

		expect(etikettKnopf('5896800039556')).not.toBeNull();
	});

	it.each(['AUTO-7', 'SYS-100001'])(
		'bietet bei der Platzhalternummer %s keinen Knopf an',
		(nummer) => {
			authStore.currentUser = {
				rolle: 'mitarbeiter',
				permissions: ['edit_books', 'view_books']
			};

			expect(etikettKnopf(nummer)).toBeNull();
		}
	);
});

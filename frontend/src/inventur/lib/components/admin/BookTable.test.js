import { describe, it, expect, vi } from 'vitest';
import { render, fireEvent } from '@testing-library/svelte';
import BookTable from './BookTable.svelte';

// Die Auswahl für Massenaktionen darf nur enthalten, was in der Liste steht
// (Rasterdurchgang 06.09.2026).
//
// Vorher überlebte sie jeden Such- und Filterwechsel: „Alle auswählen" bei 312 Titeln,
// dann „Mathe" tippen — vier Zeilen sichtbar, keine angehakt, und die Werkzeugleiste
// sagte weiter „Löschen (312)". Die Rückfrage nannte dieselbe Zahl, gelöscht worden wären
// die 312 unsichtbaren Titel. Derselbe Weg führt über „Zum Klassensatz hinzufügen".
const buch = (/** @type {string} */ id, /** @type {string} */ titel) => ({
	id,
	titel,
	autor: 'Autor',
	isbn: '',
	verfuegbar: 1,
	gesamt: 1,
	exemplare: []
});

/** @param {any[]} books */
function zeige(books, onDelete = vi.fn()) {
	return render(BookTable, {
		books,
		onEdit: vi.fn(),
		onDelete,
		onAssignClass: vi.fn(),
		onRetryCovers: vi.fn()
	});
}

describe('BookTable: Auswahl und Massenaktion', () => {
	it('nimmt beim Filtern die unsichtbaren Zeilen aus der Auswahl', async () => {
		const geloescht = vi.fn();
		const { rerender, getByLabelText, getByRole } = zeige(
			[buch('1', 'Mathe 5'), buch('2', 'Deutsch 5'), buch('3', 'Physik 7')],
			geloescht
		);
		await fireEvent.click(getByLabelText('Alle Bücher auswählen'));
		expect(getByRole('button', { name: /Löschen \(3\)/ })).toBeTruthy();

		// Suche eingegrenzt: nur noch ein Titel in der Liste.
		await rerender({ books: [buch('1', 'Mathe 5')] });

		// Die Werkzeugleiste zählt jetzt nur noch die sichtbare Zeile …
		expect(
			getByRole('button', { name: /Löschen \(1\)/ }),
			'die Leiste zählt weiter die unsichtbaren Zeilen'
		).toBeTruthy();
		// … und wer löscht, trifft auch nur sie.
		await fireEvent.click(getByRole('button', { name: /Löschen \(1\)/ }));
		expect(geloescht.mock.calls[0][0], 'die Massenaktion traf unsichtbare Zeilen').toEqual(['1']);
	});
});

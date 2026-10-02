import { describe, it, expect } from 'vitest';
import { render } from '@testing-library/svelte';
import BookBorrowersTab from './BookBorrowersTab.svelte';

// Der Knopf druckt alle, die den Titel gerade haben, auch die ohne Verzug. Die Mahnliste
// mit nur den Überfälligen steht im Mahnwesen.
describe('Buchakte, Reiter „Ausleiher“: Druckknopf', () => {
	const ausleiher = {
		schueler_name: 'Lena',
		schueler_nachname: 'Groß',
		klasse: '7b',
		schueler_barcode: 'A-1',
		exemplar_barcode: 'B-1',
		ausgeliehen_am: '2026-01-02T10:00:00Z',
		rueckgabe_frist: '2026-02-01T10:00:00Z'
	};

	it('heißt „Liste drucken“ und nennt keine Mahnliste', () => {
		const screen = render(BookBorrowersTab, {
			borrowers: [ausleiher],
			book: { title: 'Die Räuber' },
			onBack: () => {}
		});
		expect(screen.getByRole('button', { name: 'Liste drucken' })).toBeTruthy();
		expect(screen.container.textContent ?? '').not.toContain('Mahnliste');
	});
});

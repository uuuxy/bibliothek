import { describe, it, expect } from 'vitest';
import { render } from '@testing-library/svelte';
import BorrowedBooksList from './BorrowedBooksList.svelte';

// Eine Dauerleihe zeigt keine Frist und wird nie überfällig.
//
// Entschieden am 16.09.2026: „kollegen haben keine frist bzw werden einfach nie gesperrt!" Eine
// Ausleihe an jemanden, der kein Schüler ist, ist eine Dauerleihe
// (`ausleihen.ist_handapparat`), und die Sperr-Automatik zählt seit jeher nur Ausleihen
// ohne dieses Merkmal. Diese Liste rechnete daneben ihr eigenes „überfällig" aus dem
// blossen Datum: Nach einem Jahr stand die Zeile eines Kollegen rot da, während die Theke
// ihn anstandslos bediente — zwei Wahrheiten über dieselbe Ausleihe.
describe('Ausleihliste: Dauerleihe', () => {
	/** @param {boolean} dauerleihe */
	const buch = (dauerleihe) => ({
		id: 'ex-1',
		ausleihe_id: 'a-1',
		barcode_id: 'B-1',
		titel: 'Der Zauberberg',
		autor: 'Mann',
		ist_dauerleihe: dauerleihe,
		ausgeliehen_am: '2025-01-01T10:00:00Z',
		// Ein Jahr alt: Als Frist gelesen wäre das lange überfällig.
		rueckgabe_frist: '2025-02-01T10:00:00Z'
	});

	it('nennt statt des Datums „ohne Frist“ und meldet Dauerleihe', () => {
		const screen = render(BorrowedBooksList, { books: [buch(true)], mode: 'loans' });
		const text = screen.container.textContent ?? '';
		expect(text).toContain('ohne Frist');
		expect(text, 'das Fristdatum steht trotzdem da').not.toContain('1.2.2025');
		expect(text).not.toContain('Überfällig');
		// „In Frist" wäre bei etwas, das keine Frist hat, genauso falsch wie „Überfällig".
		expect(text).toContain('Dauerleihe');
		expect(text).not.toContain('In Frist');
	});

	// Die Gegenprobe: Ohne sie misst der Test nur, dass niemand überfällig wird — auch
	// dann, wenn die Regel zu weit greift und Schüler mitnimmt.
	it('lässt die befristete Ausleihe überfällig werden', () => {
		const screen = render(BorrowedBooksList, { books: [buch(false)], mode: 'loans' });
		const text = screen.container.textContent ?? '';
		expect(text).toContain('Überfällig');
		expect(text).toContain('1.2.2025');
	});
});

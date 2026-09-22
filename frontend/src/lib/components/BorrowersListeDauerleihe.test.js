import { describe, it, expect } from 'vitest';
import { render } from '@testing-library/svelte';
import BorrowersListe from './BorrowersListe.svelte';

// Eine Dauerleihe zeigt keine Frist und wird nie überfällig — dieselbe Regel wie in der
// Akte (BorrowedBooksListDauerleihe.test.js). Diese Liste („wer hat den Titel gerade?")
// färbte bis zum 22.09.2026 nach dem blossen Datum: Nach einem Jahr stand der Kollege rot
// da, während die Theke ihn anstandslos bediente (OFFEN.md 5.18).
describe('Ausleiher-Liste: Dauerleihe', () => {
	/** @param {boolean} dauerleihe */
	const zeile = (dauerleihe) => ({
		schueler_name: 'Kim',
		schueler_nachname: 'Kollegin',
		klasse: 'Lehrer',
		schueler_barcode: 'A-1',
		exemplar_barcode: 'B-1',
		ausgeliehen_am: '2025-01-01T10:00:00Z',
		// Ein Jahr alt: Als Frist gelesen wäre das lange überfällig.
		rueckgabe_frist: '2025-02-01T10:00:00Z',
		ist_dauerleihe: dauerleihe
	});
	const fmtDate = (/** @type {string} */ d) => new Date(d).toLocaleDateString('de-DE');

	it('nennt statt des Datums „ohne Frist“ und färbt nicht', () => {
		const screen = render(BorrowersListe, { zeilen: [zeile(true)], onBack: () => {}, fmtDate });
		const text = screen.container.textContent ?? '';
		expect(text).toContain('ohne Frist');
		expect(text).not.toContain('1.2.2025');
		expect(screen.container.querySelector('.text-rose-600')).toBeNull();
	});

	// Die Gegenprobe: Ohne sie misst der Test nur, dass niemand rot wird — auch dann,
	// wenn die Regel zu weit greift und Schüler mitnimmt.
	it('lässt die befristete Ausleihe überfällig werden', () => {
		const screen = render(BorrowersListe, { zeilen: [zeile(false)], onBack: () => {}, fmtDate });
		expect(screen.container.textContent ?? '').toContain('1.2.2025');
		expect(screen.container.querySelector('.text-rose-600')).not.toBeNull();
	});
});

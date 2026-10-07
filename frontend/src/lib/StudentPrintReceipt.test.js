import { describe, it, expect } from 'vitest';
import { render } from '@testing-library/svelte';
import StudentPrintReceipt from './StudentPrintReceipt.svelte';

// Die Quittung liest die Ausleihen so, wie die Leserakte sie vom Server bekommt
// (repository.BorrowedBook: barcode_id, ausgeliehen_am, rueckgabe_frist). Mit anderen
// Feldnamen stand je Buch „-" und zweimal „Keine Angabe" auf dem Ausdruck.
describe('StudentPrintReceipt', () => {
	it('nennt je Ausleihe Nummer, Ausleihdatum und Frist', () => {
		const { container } = render(StudentPrintReceipt, {
			profile: {
				vorname: 'Emil',
				nachname: 'Tischbein',
				klasse: '05F',
				entliehene_buecher: [
					{
						titel: 'Emil und die Detektive',
						autor: 'Kästner',
						barcode_id: '1057850039567',
						ausgeliehen_am: '2026-09-04T08:00:00Z',
						rueckgabe_frist: '2026-09-24T12:00:00Z'
					}
				]
			}
		});
		const text = container.textContent ?? '';
		expect(text).toContain('1057850039567');
		expect(text).toContain('04.09.2026');
		expect(text).toContain('24.09.2026');
		expect(text).not.toContain('Keine Angabe');
	});

	// Eine Dauerleihe (Kollegium) hat keine Frist und wird nie überfällig, wie in der Leserakte.
	// Ihr Datum gehört deshalb nicht auf die Quittung.
	it('nennt eine Dauerleihe „ohne Frist“ und färbt nur die abgelaufene Frist', () => {
		/** @param {boolean} dauerleihe */
		const quittung = (dauerleihe) =>
			render(StudentPrintReceipt, {
				profile: {
					vorname: 'Kim',
					nachname: 'Kollegin',
					entliehene_buecher: [
						{
							titel: 'Handapparat',
							barcode_id: 'B-1',
							ausgeliehen_am: '2025-01-01T10:00:00Z',
							rueckgabe_frist: '2025-02-01T10:00:00Z',
							ist_dauerleihe: dauerleihe
						}
					]
				}
			}).container;

		const dauer = quittung(true);
		expect(dauer.textContent ?? '').toContain('ohne Frist');
		expect(dauer.textContent ?? '').not.toContain('01.02.2025');
		expect(dauer.querySelector('.text-error'), 'die Dauerleihe steht als überfällig da').toBeNull();

		const befristet = quittung(false);
		expect(befristet.textContent ?? '').toContain('01.02.2025');
		expect(befristet.querySelector('.text-error')).not.toBeNull();
	});
});

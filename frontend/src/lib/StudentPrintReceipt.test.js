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
});

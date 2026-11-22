import { describe, it, expect, vi } from 'vitest';
import { render } from '@testing-library/svelte';
import LeserakteStammdaten from './LeserakteStammdaten.svelte';

// Ein Link „mailto:" deutet ? und & als Felder der Mail. Eine Adresse aus einer Importdatei
// setzte sonst beim Klick verdeckte Empfänger, Betreff und Text.
describe('Eltern-E-Mail in der Akte', () => {
	it('gibt dem Mailprogramm nur die Adresse', () => {
		const adresse = 'eltern@example.org?cc=fremd@example.org&subject=Mahnung';
		const { container } = render(LeserakteStammdaten, {
			leser: {
				id: 'l1',
				vorname: 'Katrin',
				nachname: 'Wendlandt',
				art: 'schueler',
				barcode_id: 'S-0007',
				klasse: '07A',
				geburtsdatum: null,
				lusd_id: null,
				strasse: '',
				eltern_email: adresse
			},
			onEdit: vi.fn()
		});
		const link = container.querySelector('a[href^="mailto:"]');

		expect(link?.textContent).toBe(adresse);
		expect(link?.getAttribute('href')).toBe(
			'mailto:eltern@example.org%3Fcc%3Dfremd@example.org%26subject%3DMahnung'
		);
	});
});

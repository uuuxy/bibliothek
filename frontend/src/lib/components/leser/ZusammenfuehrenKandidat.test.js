import { describe, it, expect } from 'vitest';
import { render } from '@testing-library/svelte';
import ZusammenfuehrenKandidat from './ZusammenfuehrenKandidat.svelte';

const LESER = { vorname: 'Mia', nachname: 'Beispiel', klasse: '7a', barcode_id: 'S-1001' };

// Vor dem Zusammenführen zählt, wer noch Bücher hat. Die Suche liefert die Zahl, die Akte die
// Liste der Ausleihen; beide Formen zeigen dieselbe Marke.
describe('Zusammenführen: offene Ausleihen eines Kandidaten', () => {
	it('nennt die Zahl, die die Suche liefert', () => {
		const screen = render(ZusammenfuehrenKandidat, { kandidat: { ...LESER, offene_buecher: 3 } });

		expect(screen.getByText('3 offene Ausleihen')).toBeTruthy();
	});

	it('zählt sonst die Ausleihen der Akte', () => {
		const screen = render(ZusammenfuehrenKandidat, {
			kandidat: { ...LESER, entliehene_buecher: [{ id: 'a1' }] }
		});

		expect(screen.getByText('1 offene Ausleihe')).toBeTruthy();
	});
});

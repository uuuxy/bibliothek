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

// Der Vermerk folgt der Regel der Theke (sperrStatus.ausleiheGesperrt): beide Sperren, ein
// Kollege nie. Beim Zusammenführen gehen beide Sperren auf den bleibenden Datensatz über.
describe('Zusammenführen: Vermerk gesperrt', () => {
	it('nennt auch die Sperre von Hand', () => {
		const screen = render(ZusammenfuehrenKandidat, {
			kandidat: { ...LESER, art: 'schueler', is_manually_blocked: true }
		});

		expect(screen.getByText('gesperrt')).toBeTruthy();
	});

	it('nennt bei einem Kollegen keine Sperre, wie die Theke', () => {
		const screen = render(ZusammenfuehrenKandidat, {
			kandidat: { ...LESER, art: 'lehrkraft', is_manually_blocked: true }
		});

		expect(screen.queryByText('gesperrt')).toBeNull();
	});
});

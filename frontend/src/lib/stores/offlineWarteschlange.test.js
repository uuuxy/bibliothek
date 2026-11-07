import { describe, it, expect } from 'vitest';
import { normalisiereEintrag } from './offlineWarteschlange.js';

// Die Warteschlange liest zwei Formate: das heutige (`art`, `barcode`) und das frühere
// (`action_type`, `barcode_id`) aus alten Sicherungsdateien.
describe('normalisiereEintrag: die Absicht', () => {
	it('liest art, sonst die frühere Schreibweise action_type', () => {
		expect(normalisiereEintrag({ barcode: 'B-1', art: 'ausleihe' })?.art).toBe('ausleihe');
		expect(normalisiereEintrag({ barcode: 'B-1', art: 'rueckgabe' })?.art).toBe('rueckgabe');
		expect(normalisiereEintrag({ barcode_id: 'B-1', action_type: 'checkout' })?.art).toBe(
			'ausleihe'
		);
		expect(normalisiereEintrag({ barcode_id: 'B-1', action_type: 'checkin' })?.art).toBe(
			'rueckgabe'
		);
	});

	it('art geht vor action_type', () => {
		const eintrag = normalisiereEintrag({
			barcode: 'B-1',
			art: 'rueckgabe',
			action_type: 'checkout'
		});
		expect(eintrag?.art).toBe('rueckgabe');
	});

	it('verwirft einen Eintrag ohne lesbare Absicht', () => {
		for (const roh of [
			{ barcode: 'B-1' },
			{ barcode: 'B-1', action_type: 'renew' },
			{ barcode: 'B-1', art: 'verlaengern' },
			{ barcode: 'B-1', art: 'verlaengern', action_type: 'checkout' },
			// Namen, die jedes Objekt von Haus aus kennt, sind keine Absicht.
			{ barcode: 'B-1', action_type: 'constructor' },
			{ barcode: 'B-1', action_type: 'toString' }
		]) {
			expect(normalisiereEintrag(roh), JSON.stringify(roh)).toBeNull();
		}
	});
});

// Eine eingespielte Sicherungsdatei kann fremd oder von Hand gebaut sein. Ein `null` in ihrer
// Liste wird übergangen, statt das Einspielen abzubrechen. Ein Ausweis, der kein Text ist, bleibt
// weg: Mit ihm lehnte der Server die ganze Portion ab (`tuer.LeseAnfrage` in api/theke/nachbuchen.go).
describe('normalisiereEintrag: eine fremde Sicherungsdatei', () => {
	it('übergeht einen leeren Eintrag', () => {
		expect(normalisiereEintrag(null)).toBeNull();
	});

	it('übernimmt den Ausweis nur als Text', () => {
		const mit = (/** @type {any} */ ausweis_barcode) =>
			normalisiereEintrag({ barcode: 'B-1', art: 'ausleihe', ausweis_barcode });
		expect(mit('S-1')?.ausweis_barcode).toBe('S-1');
		for (const wert of [undefined, '', 4711, true]) {
			expect(mit(wert), String(wert)).not.toHaveProperty('ausweis_barcode');
		}
	});
});

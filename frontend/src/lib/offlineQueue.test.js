import { describe, it, expect } from 'vitest';
import { normalisiereEintrag } from './offlineQueue.js';

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

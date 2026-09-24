import { describe, it, expect } from 'vitest';
import { ausleiheGesperrt } from './sperrStatus.js';

// Das eine Anzeige-Prädikat folgt der Regel der Theke (pruefeAusleihSperren): die zwei
// Sperren am Leser zählen, ein Kollege wird nie gesperrt (16.09. und 24.09.2026). Zeigte die
// Akte eine alte Sperre an einem Kollegen als „Gesperrt", obwohl die Theke ihm alles
// ausgibt, wären es wieder zwei Wahrheiten — der Befund vom 31.08.2026.
describe('ausleiheGesperrt', () => {
	it('zählt beide Sperren am Schüler', () => {
		expect(ausleiheGesperrt({ art: 'schueler', is_manually_blocked: true })).toBe(true);
		expect(ausleiheGesperrt({ art: 'schueler', ist_gesperrt: true })).toBe(true);
		expect(ausleiheGesperrt({ art: 'schueler' })).toBe(false);
	});

	it('nennt einen Kollegen nie gesperrt, auch mit alter Sperre', () => {
		for (const art of ['lehrkraft', 'liv']) {
			expect(ausleiheGesperrt({ art, is_manually_blocked: true, ist_gesperrt: true })).toBe(false);
		}
	});

	it('liest eine Zeile ohne Art als Schüler — so liefert sie die Sicht schueler', () => {
		expect(ausleiheGesperrt({ is_manually_blocked: true })).toBe(true);
	});
});

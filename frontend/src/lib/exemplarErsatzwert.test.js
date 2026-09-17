import { describe, it, expect } from 'vitest';
import { ersatzwertBekannt } from './components/exemplarErsatzwert.js';

/**
 * 0,00 € heißt zweierlei — und das ist der ganze Sinn dieser Regel.
 *
 * Bis zum 17.09.2026 hing die Anzeige am Betrag (`ersatzwert > 0`). Ausgerechnet beim
 * Totalschaden verschwand die Zeile damit: 100 % Wertverlust ergeben 0,00 €, und genau
 * dort ist die Null das Ergebnis und gehört auf den Bildschirm.
 */
describe('ersatzwertBekannt', () => {
	it('zeigt die 0 beim Totalschaden — dort ist sie das Ergebnis', () => {
		expect(
			ersatzwertBekannt({
				ersatzwert: 0,
				ersatzwert_herleitung: '3. Verleihjahr → 60 % von 41,50 € (Listenpreis), abzüglich 100 %'
			})
		).toBe(true);
	});

	it('verschweigt die 0, wenn kein Preis erfasst ist — dort ist sie keine Aussage', () => {
		expect(
			ersatzwertBekannt({
				ersatzwert: 0,
				ersatzwert_herleitung: 'kein Preis hinterlegt — Betrag bitte eintragen'
			})
		).toBe(false);
	});

	it('verschweigt sie auch ohne jede Herleitung (alte Antwort, andere Lese-Tür)', () => {
		expect(ersatzwertBekannt({})).toBe(false);
		expect(ersatzwertBekannt({ ersatzwert: 0 })).toBe(false);
	});

	it('zeigt den gewöhnlichen Fall', () => {
		expect(
			ersatzwertBekannt({ ersatzwert: 24.9, ersatzwert_herleitung: '3. Verleihjahr → 60 % …' })
		).toBe(true);
	});
});

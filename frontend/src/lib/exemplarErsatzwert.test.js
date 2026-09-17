import { describe, it, expect } from 'vitest';
import { ersatzwertBekannt } from './components/exemplarErsatzwert.js';

/**
 * 0,00 € heißt zweierlei — und das ist der ganze Sinn dieser Regel.
 *
 * Bis zum 17.09.2026 hing die Anzeige am Betrag (`ersatzwert > 0`). Ausgerechnet beim
 * Totalschaden verschwand die Zeile damit: 100 % Wertverlust ergeben 0,00 €, und genau
 * dort ist die Null das Ergebnis und gehört auf den Bildschirm.
 *
 * Danach las die Regel den deutschen Herleitungssatz des Servers. Seit dem Rasterdurchgang
 * vom 17.09.2026 tut sie das NICHT mehr: Der Server schickt die Auskunft als eigenes Feld.
 * Diese Tests messen deshalb am Feld — und der Satz darf sich ändern, ohne dass die Karte
 * still eine falsche Zahl zeigt.
 */
describe('ersatzwertBekannt', () => {
	it('zeigt die 0 beim Totalschaden — dort ist sie das Ergebnis', () => {
		expect(ersatzwertBekannt({ ersatzwert: 0, ersatzwert_bekannt: true })).toBe(true);
	});

	it('verschweigt die 0, wenn kein Preis erfasst ist — dort ist sie keine Aussage', () => {
		expect(ersatzwertBekannt({ ersatzwert: 0, ersatzwert_bekannt: false })).toBe(false);
	});

	it('verschweigt sie ohne das Feld (alte Antwort, andere Lese-Tür)', () => {
		expect(ersatzwertBekannt({})).toBe(false);
		expect(ersatzwertBekannt({ ersatzwert: 0 })).toBe(false);
		expect(ersatzwertBekannt(/** @type {any} */ (undefined))).toBe(false);
	});

	it('zeigt den gewöhnlichen Fall', () => {
		expect(ersatzwertBekannt({ ersatzwert: 24.9, ersatzwert_bekannt: true })).toBe(true);
	});

	// Die Regel darf NICHT wieder am Betrag hängen: Ein Buch ohne erfassten Preis hat
	// keinen Ersatzwert, auch wenn irgendwo eine Zahl steht.
	it('richtet sich nach dem Feld, nicht nach dem Betrag', () => {
		expect(ersatzwertBekannt({ ersatzwert: 24.9, ersatzwert_bekannt: false })).toBe(false);
	});
});

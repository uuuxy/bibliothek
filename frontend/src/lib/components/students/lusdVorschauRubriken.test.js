import { describe, it, expect } from 'vitest';
import { rubriken, abgaengerHinweis } from './lusdVorschauRubriken.js';

// Raster-Fund 02.09.2026 (Frontend-Prüfer): Fehlte `karenz_tage` in der Antwort, fiel die
// Ansicht auf 0 zurück und versprach „sofort anonymisiert (Karenzzeit 0)" — das Backend
// arbeitet in diesem Fall aber mit seiner Vorgabe von 90 Tagen (StandardAbgaengerKarenzTage).
// Ein Rückfall, der etwas anderes sagt als der Server, ist keine Vorgabe, sondern eine
// falsche Auskunft an genau der Stelle, an der der Admin die Folgen abschätzt.

/** @returns {any} Leere Vorschau ohne Karenz-Feld — so kam es von älteren Servern. */
const ohneKarenz = () => ({ modus: 'lusd_id' });

describe('lusdVorschauRubriken: Karenzzeit-Rückfall', () => {
	it('fällt ohne karenz_tage auf die Server-Vorgabe 90 zurück, nicht auf 0', () => {
		const abgaenger = rubriken(ohneKarenz()).find((r) => r.key === 'graduates');
		expect(abgaenger?.hint).toContain('90 Tagen');
		expect(abgaenger?.hint).not.toContain('sofort anonymisiert');
	});

	it('zeigt eine gelieferte Karenzzeit unverändert — auch die 0', () => {
		expect(
			rubriken({ ...ohneKarenz(), karenz_tage: 30 }).find((r) => r.key === 'graduates')?.hint
		).toContain('30 Tagen');
		expect(
			rubriken({ ...ohneKarenz(), karenz_tage: 0 }).find((r) => r.key === 'graduates')?.hint
		).toBe(abgaengerHinweis(0));
	});
});

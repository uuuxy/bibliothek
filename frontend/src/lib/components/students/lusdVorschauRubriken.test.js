import { describe, it, expect } from 'vitest';
import { rubriken, abgaengerHinweis, dublettenHinweis } from './lusdVorschauRubriken.js';

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

// OFFEN.md 5.6 (22.09.2026): Zwei Zeilen mit gleichem Namen und Geburtsdatum, aber
// verschiedenen Klassen, fielen still zu einer Person zusammen — die Vorschau zeigte nur
// „1 doppelte Zeile zusammengelegt". Jetzt steht die Person mit beiden Klassen in einer
// eigenen Rubrik; der Hinweis hängt am Modus, weil mit LUSD-ID die Identität sicher ist.
describe('lusdVorschauRubriken: Dubletten mit abweichender Klasse', () => {
	it('listet die zusammengelegten Zeilen mit beiden Klassen', () => {
		const eintrag = {
			id: 'zeile-4',
			vorname: 'Max',
			nachname: 'Mustermann',
			alte_klasse: '5a',
			neue_klasse: '6a'
		};
		const rubrik = rubriken(
			/** @type {any} */ ({ modus: 'name_geburtsdatum', dubletten_abweichend: [eintrag] })
		).find((r) => r.key === 'mergedRows');
		expect(rubrik?.items).toEqual([eintrag]);
		expect(rubrik?.hint).toBe(dublettenHinweis('name_geburtsdatum'));
		expect(rubrik?.hint).toContain('Geburtsdatum in der Datei prüfen');
	});

	it('warnt mit LUSD-ID nicht vor zwei Schülern — die ID macht die Person sicher', () => {
		expect(dublettenHinweis('lusd_id')).not.toContain('zwei Schüler');
	});

	it('bleibt leer, wenn ein älterer Server das Feld nicht liefert', () => {
		expect(rubriken(ohneKarenz()).find((r) => r.key === 'mergedRows')?.items).toEqual([]);
	});
});

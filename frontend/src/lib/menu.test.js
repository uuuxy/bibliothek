import { describe, it, expect } from 'vitest';
import { canSeeItem, menuGroups } from './menu.js';

/**
 * Die Sichtbarkeitsregeln der Navigation — hier stand ein Widerspruch zwischen
 * Oberfläche und Rechtelage.
 *
 * Das Backend hat den Admin-Vorrang fest eingebaut („Admins dürfen immer",
 * api/permission_middleware.go). Im Menü stand die Portal-Ausnahme aber ÜBER dem
 * Admin-Vorrang. Ergebnis: Ein Admin durfte die Klassensatz-Reservierung aufrufen, sah
 * den Menüpunkt aber nicht — und fand die Funktion deshalb schlicht nicht. Genau so ist
 * es passiert (Peter, 09.08.2026: „Klassensatz reservieren wo und wie?").
 *
 * Deshalb prüft der erste Test nicht eine Regel, sondern die Zusage: Der Admin sieht
 * ALLES, ausnahmslos.
 */
const admin = { rolle: 'admin' };
const kollegium = { rolle: 'kollegium', permissions: [] };
const helfer = { rolle: 'helfer', permissions: ['view_books'] };

/** Alle Menüpunkte flach, unabhängig von der Gruppierung. */
const allePunkte = menuGroups.flatMap((g) => g.items);

describe('Menü-Sichtbarkeit', () => {
	it('zeigt dem Admin ausnahmslos jeden Menüpunkt', () => {
		const unsichtbar = allePunkte.filter((item) => !canSeeItem(item, admin)).map((i) => i.id);

		expect(
			unsichtbar,
			`Diese Punkte sieht der Admin nicht. Das Backend lässt ihn überall hinein — eine\n` +
				`Ausnahme im Menü macht die Funktion damit nur unauffindbar, nicht sicherer:\n  ${unsichtbar.join('\n  ')}`
		).toEqual([]);
	});

	it('zeigt dem Kollegium nur sein Portal', () => {
		const sichtbar = allePunkte.filter((item) => canSeeItem(item, kollegium)).map((i) => i.id);
		expect(sichtbar).toEqual(['kollegium_portal']);
	});

	it('hält das Portal von allen anderen Rollen fern', () => {
		const portal = allePunkte.find((i) => i.id === 'kollegium_portal');
		// Kein expect(...).toBeTruthy(): Das verengt den Typ nicht, und ohne den Punkt
		// prüfte der Test unten stillschweigend gegen undefined.
		if (!portal) throw new Error('Menüpunkt kollegium_portal fehlt — Test läuft ins Leere');
		expect(canSeeItem(portal, helfer)).toBe(false);
		expect(canSeeItem(portal, { rolle: 'mitarbeiter', permissions: ['*'] })).toBe(false);
	});

	it('lässt ohne Anmeldung nichts durch', () => {
		expect(allePunkte.every((item) => canSeeItem(item, null))).toBe(false);
		expect(allePunkte.some((item) => canSeeItem(item, null))).toBe(false);
	});
});

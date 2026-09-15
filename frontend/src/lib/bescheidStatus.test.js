import { describe, it, expect } from 'vitest';
import { bescheidStatus, liegtBeiDerSchule } from './bescheidStatus.js';

// Fünf Zustände, eine Stelle: Arbeitsliste (Mahnwesen) und Schülerakte lesen dieselbe
// Funktion. Die Reihenfolge ist der Kern — „Rückgabe nach Übergabe" schlägt alles, weil
// sie eine Handlung verlangt (die Aufsicht ist zu informieren).
const datum = () => '01.09.2026';

describe('bescheidStatus', () => {
	it('nennt die Rückgabe nach der Übergabe zuerst — auch bei erledigtem Brief', () => {
		const s = bescheidStatus(
			{ rueckgabe_nach_uebergabe: true, status: 'erledigt', frist_abgelaufen: true },
			datum
		);
		expect(s.text).toBe('Rückgabe nach Übergabe');
		expect(s.tip).toContain('Aufsicht');
	});

	it('meldet den übergebenen Brief mit dem Datum der Übergabe', () => {
		const s = bescheidStatus({ status: 'uebergeben', uebergeben_am: '2026-09-01' }, datum);
		expect(s.text).toBe('übergeben');
		expect(s.detail).toBe('01.09.2026');
	});

	it('meldet erledigt', () => {
		expect(bescheidStatus({ status: 'erledigt' }, datum).text).toBe('erledigt');
	});

	it('meldet die abgelaufene Frist als Fehler-Ton — das ist die Arbeitsliste', () => {
		const s = bescheidStatus({ status: 'offen', frist_abgelaufen: true }, datum);
		expect(s.text).toBe('Frist abgelaufen');
		expect(s.ton).toBe('fehler');
	});

	// Ein frischer Brief sagt, was als Nächstes kommt — bis zum 15.09.2026 stand nur „offen",
	// und der nächste Schritt stand nirgends auf dem Bildschirm.
	it('nennt bei laufender Frist das Abwarten und die Übergabe als nächsten Schritt', () => {
		const s = bescheidStatus({ status: 'offen', frist_abgelaufen: false }, datum);
		expect(s.text).toBe('Frist läuft');
		expect(s.tip).toContain('übergeben');
	});

	// Ein übergebener Brief OHNE Datum darf keinen leeren Zusatz zeigen („übergeben · ").
	it('lässt das Detail weg, wenn kein Übergabedatum da ist', () => {
		expect(bescheidStatus({ status: 'uebergeben' }, datum).detail).toBeUndefined();
	});
});

// Die Zahl am Reiter „Schadensersatz" zählt, was bei der Schule liegt: offen (Frist läuft
// oder abgelaufen) und die Rückgabe nach der Übergabe. Übergeben liegt bei der Aufsicht,
// erledigt ist bezahlt oder storniert — beides zählt nicht.
describe('liegtBeiDerSchule', () => {
	it('zählt offene Briefe, ob die Frist läuft oder abgelaufen ist', () => {
		expect(liegtBeiDerSchule({ status: 'offen', frist_abgelaufen: false })).toBe(true);
		expect(liegtBeiDerSchule({ status: 'offen', frist_abgelaufen: true })).toBe(true);
	});
	it('zählt übergebene und erledigte Briefe nicht', () => {
		expect(liegtBeiDerSchule({ status: 'uebergeben' })).toBe(false);
		expect(liegtBeiDerSchule({ status: 'erledigt' })).toBe(false);
	});
	it('zählt die Rückgabe nach der Übergabe — die Aufsicht ist zu informieren', () => {
		expect(liegtBeiDerSchule({ status: 'uebergeben', rueckgabe_nach_uebergabe: true })).toBe(true);
	});
});

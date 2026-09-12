import { describe, it, expect } from 'vitest';
import { bescheidStatus } from './bescheidStatus.js';

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

	it('meldet sonst offen', () => {
		expect(bescheidStatus({ status: 'offen', frist_abgelaufen: false }, datum).text).toBe('offen');
	});

	// Ein übergebener Brief OHNE Datum darf keinen leeren Zusatz zeigen („übergeben · ").
	it('lässt das Detail weg, wenn kein Übergabedatum da ist', () => {
		expect(bescheidStatus({ status: 'uebergeben' }, datum).detail).toBeUndefined();
	});
});

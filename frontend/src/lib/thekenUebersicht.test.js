import { describe, it, expect } from 'vitest';
import {
	KACHELN,
	zaehleAbholbereit,
	zaehleUeberfaellig,
	sichtbareKacheln
} from './thekenUebersicht.js';

describe('Theken-Übersicht: Zahlen, nie Namen', () => {
	it('zählt nur Vormerkungen im Abholfach; Unsinn ergibt 0', () => {
		expect(
			zaehleAbholbereit([
				{ status: 'abholbereit', schueler_name: 'Max' },
				{ status: 'wartend' },
				null,
				{ status: 'abholbereit' }
			])
		).toBe(2);
		expect(zaehleAbholbereit(undefined)).toBe(0);
		expect(zaehleAbholbereit({ data: [] })).toBe(0);
	});

	it('liest die Überfälligen aus der anonymen Dashboard-Zusammenfassung', () => {
		expect(zaehleUeberfaellig({ total_overdue: 38 })).toBe(38);
		expect(zaehleUeberfaellig({ total_overdue: -1 })).toBe(0);
		expect(zaehleUeberfaellig(null)).toBe(0);
	});

	it('zeigt nur Kacheln, deren Route der Benutzer lesen darf', () => {
		const helfer = new Set(['view_students']);
		expect(sichtbareKacheln((r) => helfer.has(r)).map((k) => k.id)).toEqual(['ueberfaellig']);
		expect(sichtbareKacheln(() => true)).toHaveLength(KACHELN.length);
		expect(sichtbareKacheln(() => false)).toEqual([]);
	});

	it('jede Kachel nennt ein Recht — sonst gäbe es einen 403-Toast am Tresen', () => {
		for (const k of KACHELN) expect(k.recht, k.id).toMatch(/^[a-z_]+$/);
	});
});

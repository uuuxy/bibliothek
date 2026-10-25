import { describe, it, expect } from 'vitest';
import { gemahntSatz, mahnungenJeKind } from './mahnungen.js';

describe('Mahnungen: wie oft und wann zuletzt', () => {
	// „Noch nicht gemahnt" ist eine Aussage, kein leeres Feld, und eine Zahl ohne Tag (aus
	// Littera übernommen) erfindet kein Datum.
	it('sagt Zahl und Tag des letzten Briefs, „noch nicht gemahnt" bei 0', () => {
		expect(gemahntSatz(0, '')).toBe('noch nicht gemahnt');
		expect(gemahntSatz(undefined, undefined)).toBe('noch nicht gemahnt');
		expect(gemahntSatz(1, '2026-09-26')).toBe('1× gemahnt, zuletzt 26.09.2026');
		expect(gemahntSatz(2, '2026-10-01')).toBe('2× gemahnt, zuletzt 01.10.2026');
		expect(gemahntSatz(3, '')).toBe('3× gemahnt');
		expect(gemahntSatz(3, undefined)).toBe('3× gemahnt');
	});

	// Gezählt wird je Buch. Für das Kind gilt, was am weitesten ging: Zum ersten Buch sind
	// zwei Briefe hinaus, das zweite wurde erst danach fällig.
	it('nimmt je Kind die höchste Zahl und den jüngsten Tag über seine Bücher', () => {
		expect(
			mahnungenJeKind([
				{ mahnstufe: 2, letztes_mahndatum: '2026-09-26' },
				{ mahnstufe: 0 },
				{ mahnstufe: 1, letztes_mahndatum: '2026-10-01' }
			])
		).toEqual({ gemahnt: 2, zuletztGemahnt: '2026-10-01' });
	});

	it('kommt ohne Angaben aus: nichts gemahnt, kein Tag', () => {
		expect(mahnungenJeKind([{ titel: 'ohne Zahl' }])).toEqual({ gemahnt: 0, zuletztGemahnt: '' });
		expect(mahnungenJeKind([])).toEqual({ gemahnt: 0, zuletztGemahnt: '' });
		expect(mahnungenJeKind(/** @type {any} */ (undefined))).toEqual({
			gemahnt: 0,
			zuletztGemahnt: ''
		});
		// Zahl aus Littera ohne Tag.
		expect(mahnungenJeKind([{ mahnstufe: 3 }])).toEqual({ gemahnt: 3, zuletztGemahnt: '' });
	});
});

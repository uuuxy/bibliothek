import { describe, it, expect } from 'vitest';
import { avatarVerlauf, initialen } from './avatarKachel.js';

// Die Initialen-Kachel steht in der Schülerakte, wenn kein Foto da ist (StudentProfileCard).
describe('avatarKachel', () => {
	describe('initialen', () => {
		it('nimmt je den ersten Buchstaben von Vor- und Nachname, groß geschrieben', () => {
			expect(initialen({ vorname: 'Max', nachname: 'Mustermann' })).toBe('MM');
			expect(initialen({ vorname: 'max', nachname: 'mustermann' })).toBe('MM');
		});

		it('kommt mit einem fehlenden Namensteil aus und zeigt ohne Namen „?"', () => {
			expect(initialen({ vorname: 'Max' })).toBe('M');
			expect(initialen({ nachname: 'Schmidt' })).toBe('S');
			expect(initialen({})).toBe('?');
		});
	});

	describe('avatarVerlauf', () => {
		const namen = [
			{ vorname: 'Max', nachname: 'Mustermann' },
			{ vorname: 'Anna', nachname: 'Schmidt' },
			{ vorname: 'Hans', nachname: 'Meier' },
			{ vorname: 'Lea', nachname: 'Wagner' },
			{ vorname: 'Tom', nachname: 'Becker' },
			{ vorname: 'Mia', nachname: 'Hoffmann' }
		];

		it('gibt demselben Namen immer dieselbe Kachel', () => {
			for (const p of namen) expect(avatarVerlauf({ ...p })).toBe(avatarVerlauf(p));
		});

		it('verteilt verschiedene Namen auf verschiedene Verläufe', () => {
			expect(new Set(namen.map(avatarVerlauf)).size).toBeGreaterThan(1);
		});

		it('liefert auch ohne Namen einen Verlauf', () => {
			expect(avatarVerlauf({})).toMatch(/^from-\S+ to-\S+$/);
		});
	});
});

import { describe, it, expect } from 'vitest';
import { initialen } from './avatarKachel.js';

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
});

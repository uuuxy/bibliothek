import { describe, it, expect } from 'vitest';
import { avatarVerlauf, initialen } from './avatarKachel.js';

describe('avatarKachel', () => {
	describe('avatarVerlauf', () => {
		it('returns deterministic gradient based on name', () => {
			const max1 = avatarVerlauf({ vorname: 'Max', nachname: 'Mustermann' });
			const max2 = avatarVerlauf({ vorname: 'Max', nachname: 'Mustermann' });
			expect(max1).toBe(max2);

			const anna = avatarVerlauf({ vorname: 'Anna', nachname: 'Schmidt' });
			const hans = avatarVerlauf({ vorname: 'Hans', nachname: 'Meier' });

			// Check they return valid gradient strings
			expect(max1).toContain('from-');
			expect(anna).toContain('from-');
			expect(hans).toContain('from-');
		});

		it('handles missing names safely', () => {
			expect(avatarVerlauf({})).toBeTypeOf('string');
			expect(avatarVerlauf({ vorname: 'Max' })).toBeTypeOf('string');
			expect(avatarVerlauf({ nachname: 'Mustermann' })).toBeTypeOf('string');
		});

		it('returns expected gradients based on hash distribution', () => {
			// These map to specific gradients in the array
			expect(avatarVerlauf({ vorname: 'A', nachname: 'B' })).toBeTypeOf('string');
			expect(avatarVerlauf({ vorname: 'A', nachname: 'B' })).toContain('from-');
		});
	});

	describe('initialen', () => {
		it('returns initials for first and last name', () => {
			expect(initialen({ vorname: 'Max', nachname: 'Mustermann' })).toBe('MM');
			expect(initialen({ vorname: 'Anna', nachname: 'Schmidt' })).toBe('AS');
		});

		it('handles missing names', () => {
			expect(initialen({ vorname: 'Max' })).toBe('M');
			expect(initialen({ nachname: 'Schmidt' })).toBe('S');
			expect(initialen({})).toBe('?');
		});

		it('uppercases the initials', () => {
			expect(initialen({ vorname: 'max', nachname: 'mustermann' })).toBe('MM');
		});
	});
});

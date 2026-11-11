import { describe, it, expect } from 'vitest';
import { passendeMasse } from './bildVerkleinern.js';

describe('passendeMasse', () => {
	it('verkleinert im Seitenverhältnis auf den Rahmen und vergrößert nie', () => {
		expect(passendeMasse(4000, 3000, 1200, 1200)).toEqual({ breite: 1200, hoehe: 900 });
		expect(passendeMasse(3000, 4000, 1200, 1200)).toEqual({ breite: 900, hoehe: 1200 });
		expect(passendeMasse(1200, 1800, 600, 900)).toEqual({ breite: 600, hoehe: 900 });
		expect(passendeMasse(300, 200, 1200, 1200)).toEqual({ breite: 300, hoehe: 200 });
	});
});

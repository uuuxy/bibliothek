import { describe, it, expect } from 'vitest';
import { hinweisZumRand, linienAbstand, linienStil } from './randLinie.js';

// Wie der Rand eines Textfelds in Material 3: Outline, aktiv Primary und doppelt so stark, Error.
describe('linienStil', () => {
	it('ruhig: Outline, einfache Stärke', () => {
		expect(linienStil({ zuNah: false, eingerastet: false, aktiv: false })).toBe(
			'border-outline border'
		);
	});
	it('eingerastet: Primary, doppelte Stärke', () => {
		expect(linienStil({ zuNah: false, eingerastet: true, aktiv: true })).toBe(
			'border-primary border-2'
		);
	});
	it('zu nah am Rand: Error, beim Ziehen doppelt so stark', () => {
		expect(linienStil({ zuNah: true, eingerastet: false, aktiv: false })).toBe(
			'border-error border'
		);
		expect(linienStil({ zuNah: true, eingerastet: false, aktiv: true })).toBe(
			'border-error border-2'
		);
	});
});

describe('linienAbstand', () => {
	it('ohne gezogenes Element der sichere Rand', () => {
		expect(linienAbstand(null)).toBe('3mm');
	});
	it('beim Ziehen die Grenze des Elements, oben rechts unten links', () => {
		expect(linienAbstand({ links: 6.3, oben: 6.3, rechts: 6.3, unten: 3.3 })).toBe(
			'6.3mm 6.3mm 3.3mm 6.3mm'
		);
	});
});

describe('hinweisZumRand: der Satz sagt, warum die Linie rot ist', () => {
	it('nichts zu nah', () => {
		expect(hinweisZumRand([])).toBe('Gestrichelt: sicherer Druckbereich');
	});
	it('nur der Strichcode', () => {
		expect(hinweisZumRand([{ type: 'barcode' }])).toBe(
			'Rot: Die Striche des Strichcodes brauchen 6,3 mm Abstand zum Rand'
		);
	});
	it('ein anderes Element', () => {
		expect(hinweisZumRand([{ type: 'barcode' }, { type: 'text' }])).toBe(
			'Rot: Ein Element steht zu nah am Rand'
		);
	});
});

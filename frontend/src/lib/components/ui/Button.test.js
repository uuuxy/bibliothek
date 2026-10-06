import { describe, it, expect } from 'vitest';
import { render } from '@testing-library/svelte';
import Button from './Button.svelte';

/**
 * Tailwind-Utilities haben alle dieselbe Spezifität. Welche gewinnt, entscheidet die
 * Reihenfolge im Stylesheet — nicht die im class-Attribut. Steht die Fläche der Variante im
 * Bundle hinter der des Aufrufers, bleibt ein getönter Button in der Farbe der Variante,
 * obwohl beide Klassen am Element hängen. Button.svelte entfernt die kollidierende Farbe der
 * Variante deshalb, statt sie mitzuschicken.
 */
const klassen = (el) => (el.getAttribute('class') || '').split(/\s+/);

describe('Button — Farb-Overrides des Aufrufers', () => {
	it('entfernt die Hintergrundfarbe der Variante, wenn der Aufrufer eine eigene mitgibt', () => {
		const { getByRole } = render(Button, { variant: 'secondary', class: 'bg-primary-container' });
		const k = klassen(getByRole('button'));
		expect(k).toContain('bg-primary-container');
		expect(k).not.toContain('bg-surface-container-lowest');
	});

	it('entfernt Rahmen- und Textfarbe der Variante gleichermaßen', () => {
		const { getByRole } = render(Button, {
			variant: 'secondary',
			class: 'border-outline text-primary'
		});
		const k = klassen(getByRole('button'));
		expect(k).toEqual(expect.arrayContaining(['border-outline', 'text-primary']));
		expect(k).not.toContain('border-outline-variant');
		expect(k).not.toContain('text-on-surface-variant');
	});

	it('lässt die Variante unangetastet, wenn keine Farbe überschrieben wird', () => {
		const { getByRole } = render(Button, { variant: 'secondary', class: 'w-full px-6' });
		const k = klassen(getByRole('button'));
		expect(k).toEqual(
			expect.arrayContaining([
				'bg-surface-container-lowest',
				'border-outline-variant',
				'text-on-surface-variant'
			])
		);
	});

	it('ersetzt nur die Familie, die der Aufrufer anfasst', () => {
		// Nur bg wird überschrieben — Rahmen und Text der Variante müssen bleiben,
		// sonst verliert ein „danger"-Button seine rote Schrift.
		const { getByRole } = render(Button, { variant: 'danger', class: 'bg-surface' });
		const k = klassen(getByRole('button'));
		expect(k).toContain('bg-surface');
		expect(k).not.toContain('bg-error-container');
		expect(k).toEqual(expect.arrayContaining(['border-transparent', 'text-on-error-container']));
	});

	it('erkennt auch die Rollen für Erfolg und Warnung als Farbe des Aufrufers', () => {
		const { getByRole } = render(Button, {
			variant: 'secondary',
			class: 'bg-warning-container text-on-warning-container'
		});
		const k = klassen(getByRole('button'));
		expect(k).toEqual(
			expect.arrayContaining(['bg-warning-container', 'text-on-warning-container'])
		);
		expect(k).not.toContain('bg-surface-container-lowest');
		expect(k).not.toContain('text-on-surface-variant');
	});

	it('hält Größenangaben aus der Farblogik heraus', () => {
		// text-[10px] ist eine Größe, keine Farbe — die Textfarbe der Variante bleibt.
		const { getByRole } = render(Button, { variant: 'secondary', class: 'text-[10px]' });
		expect(klassen(getByRole('button'))).toContain('text-on-surface-variant');
	});

	it('rührt Zustandsvarianten nicht an', () => {
		// hover:/disabled: beschreiben andere Zustände und kollidieren nicht mit der Grundfarbe.
		const { getByRole } = render(Button, {
			variant: 'primary',
			class: 'disabled:bg-surface hover:bg-success'
		});
		const k = klassen(getByRole('button'));
		expect(k).toContain('bg-primary');
		expect(k).toEqual(expect.arrayContaining(['disabled:bg-surface', 'hover:bg-success']));
	});

	it('behält die gemeinsame Control-Höhe je Größe', () => {
		/** @type {[('sm'|'md'|'lg'), string][]} */
		const groessen = [
			// sm = 32/14 seit dem Typografie-Audit (M3 kennt keinen 12-px-Knopf).
			['sm', 'h-8'],
			['md', 'h-9'],
			['lg', 'h-10']
		];
		for (const [size, h] of groessen) {
			const { getByRole, unmount } = render(Button, { size });
			expect(klassen(getByRole('button'))).toContain(h);
			unmount();
		}
	});
});

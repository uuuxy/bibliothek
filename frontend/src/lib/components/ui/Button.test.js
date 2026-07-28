import { describe, it, expect } from 'vitest';
import { render } from '@testing-library/svelte';
import Button from './Button.svelte';

/**
 * Hintergrund: Tailwind-Utilities haben alle dieselbe Spezifität. Welche gewinnt,
 * entscheidet die Reihenfolge IM STYLESHEET — nicht die im class-Attribut. Steht
 * `bg-white` der Variante im Bundle hinter `bg-blue-50` des Aufrufers, bleibt ein
 * getönter Button stumm weiß, obwohl beide Klassen am Element hängen.
 *
 * Das ist genau so passiert: sechs getönte Buttons (Medienkatalog-Toolbar,
 * Klassenkarte, Stammdaten, Buch-Akte, Offline-Banner, Mahnwesen) rendelten
 * unbemerkt im neutralen Sekundär-Look. Button.svelte entfernt die kollidierende
 * Farbe der Variante deshalb, statt sie mitzuschicken.
 *
 * Ein Screenshot hätte den Rückfall nicht verhindert — diese Tests schon.
 */
const klassen = (el) => (el.getAttribute('class') || '').split(/\s+/);

describe('Button — Farb-Overrides des Aufrufers', () => {
	it('entfernt die Hintergrundfarbe der Variante, wenn der Aufrufer eine eigene mitgibt', () => {
		const { getByRole } = render(Button, { variant: 'secondary', class: 'bg-blue-50' });
		const k = klassen(getByRole('button'));
		expect(k).toContain('bg-blue-50');
		expect(k).not.toContain('bg-white');
	});

	it('entfernt Rahmen- und Textfarbe der Variante gleichermaßen', () => {
		const { getByRole } = render(Button, {
			variant: 'secondary',
			class: 'border-blue-100 text-blue-600'
		});
		const k = klassen(getByRole('button'));
		expect(k).toEqual(expect.arrayContaining(['border-blue-100', 'text-blue-600']));
		expect(k).not.toContain('border-slate-200');
		expect(k).not.toContain('text-slate-700');
	});

	it('lässt die Variante unangetastet, wenn keine Farbe überschrieben wird', () => {
		const { getByRole } = render(Button, { variant: 'secondary', class: 'w-full px-6' });
		const k = klassen(getByRole('button'));
		expect(k).toEqual(expect.arrayContaining(['bg-white', 'border-slate-200', 'text-slate-700']));
	});

	it('ersetzt nur die Familie, die der Aufrufer anfasst', () => {
		// Nur bg wird überschrieben — Rahmen und Text der Variante müssen bleiben,
		// sonst verliert ein „danger"-Button seine rote Schrift.
		const { getByRole } = render(Button, { variant: 'danger', class: 'bg-white' });
		const k = klassen(getByRole('button'));
		expect(k).toContain('bg-white');
		expect(k).not.toContain('bg-rose-50');
		expect(k).toEqual(expect.arrayContaining(['border-rose-200', 'text-rose-700']));
	});

	it('hält Größenangaben aus der Farblogik heraus', () => {
		// text-[10px] ist eine Größe, keine Farbe — die Textfarbe der Variante bleibt.
		const { getByRole } = render(Button, { variant: 'secondary', class: 'text-[10px]' });
		expect(klassen(getByRole('button'))).toContain('text-slate-700');
	});

	it('rührt Zustandsvarianten nicht an', () => {
		// hover:/disabled: beschreiben andere Zustände und kollidieren nicht mit der Grundfarbe.
		const { getByRole } = render(Button, {
			variant: 'primary',
			class: 'disabled:bg-slate-200 hover:bg-emerald-700'
		});
		const k = klassen(getByRole('button'));
		expect(k).toContain('bg-blue-600');
		expect(k).toEqual(expect.arrayContaining(['disabled:bg-slate-200', 'hover:bg-emerald-700']));
	});

	it('behält die gemeinsame Control-Höhe je Größe', () => {
		/** @type {[('sm'|'md'|'lg'), string][]} */
		const groessen = [
			['sm', 'h-7'],
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

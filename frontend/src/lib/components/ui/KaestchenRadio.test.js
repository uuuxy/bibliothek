import { describe, it, expect } from 'vitest';
import { render } from '@testing-library/svelte';
import Kaestchen from './Kaestchen.svelte';
import Radio from './Radio.svelte';

// Kästchen und Radio nach Googles Token (material-web, md.comp.checkbox und md.comp.radio-button):
// Kasten 18 dp mit 2 dp Ecke, Kreis 20 dp, Schicht 40 dp, die Welle beim Drücken in der Farbe
// des kommenden Zustands: ohne Auswahl primary, ausgewählt on-surface
// („unselected-pressed-state-layer-color: primary", „selected-pressed-…: on-surface").

/** @param {Element | null} el */
const klassen = (el) => (el?.getAttribute('class') || '').split(/\s+/);

describe('Kästchen nach M3', () => {
	it('hat Kasten 18 px mit 2 px Ecke in einer runden Schicht von 40 px, die die Welle trägt', () => {
		const { container } = render(Kaestchen, { etikett: 'Mehrjahresband' });
		const schicht = container.querySelector('[data-welle]');
		expect(klassen(schicht)).toEqual(expect.arrayContaining(['h-10', 'w-10', 'rounded-full']));
		const eingabe = container.querySelector('input[type=checkbox]');
		expect(klassen(eingabe)).toEqual(expect.arrayContaining(['h-10', 'w-10', 'rounded-full']));
		const kasten = eingabe?.nextElementSibling ?? null;
		expect(klassen(kasten)).toEqual(expect.arrayContaining(['h-4.5', 'w-4.5', 'rounded-xs']));
	});

	it('wellt ohne Auswahl in primary und ausgewählt in on-surface', () => {
		const leer = render(Kaestchen, { etikett: 'A' });
		const schichtLeer = /** @type {HTMLElement} */ (leer.container.querySelector('[data-welle]'));
		expect(schichtLeer.style.getPropertyValue('--welle-farbe')).toBe('var(--color-primary)');
		const an = render(Kaestchen, { etikett: 'B', checked: true });
		const schichtAn = /** @type {HTMLElement} */ (an.container.querySelector('[data-welle]'));
		expect(schichtAn.style.getPropertyValue('--welle-farbe')).toBe('var(--color-on-surface)');
	});
});

describe('Radio nach M3', () => {
	it('hat einen Kreis von 20 px in einer runden Schicht von 40 px und wellt nach dem Zustand', () => {
		const { container } = render(Radio, { group: 'a', value: 'b' });
		const schicht = /** @type {HTMLElement} */ (container.querySelector('[data-welle]'));
		expect(klassen(schicht)).toEqual(expect.arrayContaining(['h-10', 'w-10', 'rounded-full']));
		expect(schicht.style.getPropertyValue('--welle-farbe')).toBe('var(--color-primary)');
		const kreis = container.querySelector('input[type=radio]')?.nextElementSibling ?? null;
		expect(klassen(kreis)).toEqual(expect.arrayContaining(['h-5', 'w-5', 'rounded-full']));
		const gewaehlt = render(Radio, { group: 'a', value: 'a' });
		const schichtGewaehlt = /** @type {HTMLElement} */ (
			gewaehlt.container.querySelector('[data-welle]')
		);
		expect(schichtGewaehlt.style.getPropertyValue('--welle-farbe')).toBe('var(--color-on-surface)');
	});
});

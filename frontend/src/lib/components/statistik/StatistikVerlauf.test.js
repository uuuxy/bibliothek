import { describe, it, expect, vi } from 'vitest';
import { render, fireEvent } from '@testing-library/svelte';
import StatistikVerlauf from './StatistikVerlauf.svelte';

// Der Zeiger markiert den Monat unter sich; außerhalb der Zeichenfläche markiert er keinen.
const DATEN = [
	{ monat: '2026-09', ausleihen: 4, rueckgaben: 2 },
	{ monat: '2026-10', ausleihen: 6, rueckgaben: 3 }
];

describe('StatistikVerlauf: der Monat unter dem Zeiger', () => {
	it('markiert nur innerhalb der Zeichenfläche', async () => {
		const { container } = render(StatistikVerlauf, { daten: DATEN });
		const flaeche = /** @type {SVGSVGElement} */ (container.querySelector('svg'));
		vi.spyOn(flaeche, 'getBoundingClientRect').mockReturnValue(
			/** @type {any} */ ({ left: 0, width: 640 })
		);
		const markiert = () => container.querySelectorAll('rect.fill-on-surface\\/8').length;

		await fireEvent.mouseMove(flaeche, { clientX: 100 });
		expect(markiert()).toBe(1);

		await fireEvent.mouseMove(flaeche, { clientX: 5 });
		expect(markiert()).toBe(0);

		await fireEvent.mouseMove(flaeche, { clientX: 639 });
		expect(markiert()).toBe(0);
	});
});

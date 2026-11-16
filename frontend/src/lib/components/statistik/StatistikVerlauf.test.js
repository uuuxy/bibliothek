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

	it('nennt auch zum ersten Monat die Zahlen', async () => {
		const { container } = render(StatistikVerlauf, { daten: DATEN });
		const flaeche = /** @type {SVGSVGElement} */ (container.querySelector('svg'));
		vi.spyOn(flaeche, 'getBoundingClientRect').mockReturnValue(
			/** @type {any} */ ({ left: 0, width: 640 })
		);

		await fireEvent.mouseMove(flaeche, { clientX: 100 });

		const einblendung = container.querySelector('.bg-inverse-surface')?.textContent ?? '';
		expect(einblendung).toContain('Ausleihen: 4');
		expect(einblendung).toContain('Rückgaben: 2');
	});
});

// Ausleihen sind ganze Zahlen; die Achse zählt deshalb in ganzen Schritten, auch wenn kein
// Monat mehr als ein oder zwei Ausleihen hat.
describe('StatistikVerlauf: die Achse', () => {
	const achse = (/** @type {number} */ hoechstwert) => {
		const { container } = render(StatistikVerlauf, {
			daten: [{ monat: '2026-10', ausleihen: hoechstwert, rueckgaben: 0 }]
		});
		return [...container.querySelectorAll('text[text-anchor="end"]')].map((t) =>
			(t.textContent ?? '').trim()
		);
	};

	it('zählt bei wenigen Ausleihen in ganzen Schritten', () => {
		expect(achse(1)).toEqual(['0', '1']);
		expect(achse(2)).toEqual(['0', '1', '2']);
	});

	it('behält die Schritte bei mehr Ausleihen', () => {
		expect(achse(8)).toEqual(['0', '2', '4', '6', '8']);
		expect(achse(37)).toEqual(['0', '10', '20', '30', '40']);
	});
});

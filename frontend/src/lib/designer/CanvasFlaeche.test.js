import { describe, it, expect, beforeEach, vi } from 'vitest';
import { render } from '@testing-library/svelte';
import CanvasFlaeche from './CanvasFlaeche.svelte';
import { idStore, resetDesign } from './idDesignerStore.svelte.js';
import { SICHERHEITSRAND_MM } from './kartenRand.js';

// Die Leinwand zeigt den sicheren Rand als gestrichelte Linie über den Elementen. Sie fängt
// keinen Zeiger ab: Ein Element unter ihr lässt sich weiter greifen und bis an den Rand ziehen.
const LESER = { id: 's-1', vorname: 'Max', nachname: 'Mustermann', barcode_id: 'DEMO-S-001' };

describe('CanvasFlaeche: der sichere Rand', () => {
	beforeEach(() => resetDesign());

	it.each(/** @type {('front' | 'back')[]} */ (['front', 'back']))(
		'zeigt die Linie auf der Seite %s',
		(side) => {
			const { container } = render(CanvasFlaeche, {
				props: {
					side,
					gewaehlteKennung: null,
					onSelect: () => {},
					leser: LESER,
					zoom: 100,
					barcodeType: 'code39'
				}
			});
			const linie = container.querySelector('[data-sicherheitsrand]');
			expect(linie, 'Die gestrichelte Linie fehlt.').not.toBeNull();
			expect(linie?.getAttribute('style')).toContain(`inset: ${SICHERHEITSRAND_MM}mm`);
			expect(linie?.classList.contains('border-dashed')).toBe(true);
			expect(linie?.classList.contains('pointer-events-none')).toBe(true);
			expect(linie?.getAttribute('aria-hidden')).toBe('true');
		}
	);
});

// Die vier Griffe ziehen je ihre Ecke; die gegenüberliegende bleibt stehen.
describe('CanvasFlaeche: Größe ändern', () => {
	beforeEach(() => resetDesign());

	it('der Griff unten rechts vergrößert, ohne das Element zu verschieben', async () => {
		const element = idStore.front.elements.find((e) => e.type === 'text' || e.id === 'header');
		const el = /** @type {any} */ (element);
		el.proportional = false;
		const { x, y, width, height } = el;
		const { container } = render(CanvasFlaeche, {
			props: {
				side: 'front',
				gewaehlteKennung: el.id,
				onSelect: () => {},
				leser: LESER,
				zoom: 100,
				barcodeType: 'code39'
			}
		});
		// 856 Pixel Breite: 10 Pixel sind 1 mm.
		vi.spyOn(Element.prototype, 'getBoundingClientRect').mockReturnValue(
			/** @type {any} */ ({ left: 0, top: 0, width: 856, height: 540 })
		);
		const griff = /** @type {HTMLElement} */ (container.querySelector('.cursor-se-resize'));

		griff.dispatchEvent(new MouseEvent('pointerdown', { bubbles: true, clientX: 0, clientY: 0 }));
		window.dispatchEvent(new MouseEvent('pointermove', { clientX: 40, clientY: 40 }));

		expect(el.width).toBeCloseTo(width + 4);
		expect(el.height).toBeCloseTo(height + 4);
		expect([el.x, el.y]).toEqual([x, y]);
		window.dispatchEvent(new MouseEvent('pointerup'));
		vi.restoreAllMocks();
	});
});

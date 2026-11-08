import { describe, it, expect, beforeEach, vi } from 'vitest';
import { tick } from 'svelte';
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

// Nah an der Linie rastet die Kante ein, und die Linie wird Primary; weiter gezogen läuft das
// Element frei über die Linie, sie wird Error, und der Satz darunter sagt warum.
describe('CanvasFlaeche: Einrasten an der Linie', () => {
	beforeEach(() => resetDesign());

	/** @param {any} el  das einzige Element der Vorderseite */
	function zeichne(el) {
		idStore.front.elements = [el];
		const ansicht = render(CanvasFlaeche, {
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
		const griff = /** @type {HTMLElement} */ (
			ansicht.container.querySelector('[data-ausweis-vorschau] [role="presentation"]')
		);
		griff.dispatchEvent(new MouseEvent('pointerdown', { bubbles: true, clientX: 0, clientY: 0 }));
		return ansicht;
	}
	const linie = (/** @type {HTMLElement} */ c) =>
		/** @type {HTMLElement} */ (c.querySelector('[data-sicherheitsrand]'));
	/** @param {number} x  Zeigerweg in Pixeln */
	const ziehe = async (x) => {
		window.dispatchEvent(new MouseEvent('pointermove', { clientX: x, clientY: 0 }));
		await tick();
	};

	it('die linke Kante rastet ein, weiter gezogen läuft sie über die Linie', async () => {
		const el = { id: 't', type: 'text', content: 'A', x: 10, y: 20, width: 20, height: 5 };
		const { container, getByText } = zeichne({ ...el, zIndex: 1, show: true, style: {} });
		const gezeichnet = /** @type {any} */ (idStore.front.elements[0]);

		await ziehe(-64); // 3,6 mm: 0,6 mm vor der Linie
		expect(gezeichnet.x).toBe(3);
		expect(linie(container).className).toContain('border-primary border-2');

		await ziehe(-85); // 1,5 mm: über die Linie hinaus
		expect(gezeichnet.x).toBe(1.5);
		expect(linie(container).className).toContain('border-error border-2');
		getByText(/Rot: Ein Element steht zu nah am Rand/);

		window.dispatchEvent(new MouseEvent('pointerup'));
		await tick();
		expect(linie(container).className).toContain('border-error border');
		expect(linie(container).className).not.toContain('border-2');
		vi.restoreAllMocks();
	});

	it('eine Farbfläche rastet nicht ein und darf in den Randstreifen', async () => {
		const el = { id: 'f', type: 'box', content: '', x: 10, y: 20, width: 20, height: 5 };
		const { container } = zeichne({ ...el, zIndex: 0, show: true, style: { color: '#dbe3ea' } });
		await ziehe(-65);
		expect(/** @type {any} */ (idStore.front.elements[0]).x).toBe(3.5);
		await ziehe(-100);
		expect(linie(container).className).toContain('border-outline border');
		window.dispatchEvent(new MouseEvent('pointerup'));
		vi.restoreAllMocks();
	});

	it('beim Ziehen des Strichcodes zeigt die Linie die Grenze seiner Striche', async () => {
		const el = { id: 'b', type: 'barcode', content: '', x: 20, y: 20, width: 30, height: 11 };
		const { container } = zeichne({ ...el, zIndex: 1, show: true, proportional: true });
		await ziehe(-142); // 5,8 mm: 0,5 mm vor der Grenze der Striche
		expect(/** @type {any} */ (idStore.front.elements[0]).x).toBe(6.3);
		expect(linie(container).getAttribute('style')).toContain('inset: 6.3mm 6.3mm 3.3mm 6.3mm');
		window.dispatchEvent(new MouseEvent('pointerup'));
		await tick();
		expect(linie(container).getAttribute('style')).toContain('inset: 3mm');
		vi.restoreAllMocks();
	});
});

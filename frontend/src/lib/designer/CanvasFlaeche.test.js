import { describe, it, expect, beforeEach } from 'vitest';
import { render } from '@testing-library/svelte';
import CanvasFlaeche from './CanvasFlaeche.svelte';
import { resetDesign } from './idDesignerStore.svelte.js';
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

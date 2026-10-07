import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { render } from '@testing-library/svelte';
import { flushSync } from 'svelte';

vi.mock('../../stores/toastStore.svelte.js', () => ({
	toastStore: { addToast: vi.fn() }
}));

vi.mock('../../apiFetch.js', () => ({
	apiFetch: vi.fn()
}));

import { apiFetch } from '../../apiFetch.js';
import { labelStore } from '../../stores/labels.svelte.js';
import LabelPreview from './LabelPreview.svelte';

// Die Vorschau zeichnet den gewählten Bogen in zwei Dritteln der Größe: Spalten, Etikett und
// Ränder kommen aus dem Format (etikettformate.js, gehalten gegen api/label_formats.go).
describe('Druck-Center: Vorschau je Etikettenformat', () => {
	beforeEach(async () => {
		vi.clearAllMocks();
		// Die Vorschau misst ihre Spalte (bind:clientWidth); jsdom kennt den Beobachter nicht.
		vi.stubGlobal(
			'ResizeObserver',
			class {
				observe() {}
				unobserve() {}
				disconnect() {}
			}
		);
		vi.mocked(apiFetch).mockResolvedValueOnce(
			/** @type {any} */ ({
				ok: true,
				json: async () => [{ barcode_id: 'B-0041873' }, { barcode_id: 'B-0041874' }]
			})
		);
		labelStore.generationMode = 'existing';
		labelStore.startPosition = 1;
		await labelStore.selectBookTitle({ id: 't1', titel: 'Titel', autor: 'Autorin' });
	});

	afterEach(() => {
		vi.unstubAllGlobals();
	});

	/** @param {string} formatId */
	function gezeichnet(formatId) {
		labelStore.formatId = formatId;
		flushSync();
		const blatt = /** @type {HTMLElement} */ (
			document.querySelector('[data-testid="etiketten-blatt"]')
		);
		const raster = /** @type {HTMLElement} */ (blatt.firstElementChild);
		const etikett = /** @type {HTMLElement} */ (raster.firstElementChild);
		return {
			spalten: raster.style.gridTemplateColumns,
			rand: blatt.style.padding,
			etikett: [etikett.style.width, etikett.style.height]
		};
	}

	it('zeichnet Raster, Etikett und Ränder des gewählten Bogens', () => {
		render(LabelPreview);

		expect(gezeichnet('zweckform_l4760')).toEqual({
			spalten: 'repeat(3, minmax(0, 42.3mm))',
			rand: '10mm 4.8mm 0px',
			etikett: ['42.3mm', '25.4mm']
		});
		expect(gezeichnet('avery_3475')).toEqual({
			spalten: 'repeat(3, minmax(0, 46.6mm))',
			rand: '0.3mm 0mm 0px',
			etikett: ['46.6mm', '24.6mm']
		});
		expect(gezeichnet('standard_52')).toEqual({
			spalten: 'repeat(4, minmax(0, 32.1mm))',
			rand: '7.1mm 2.2mm 0px',
			etikett: ['32.1mm', '14.1mm']
		});
		labelStore.formatId = 'zweckform_l4760';
	});
});

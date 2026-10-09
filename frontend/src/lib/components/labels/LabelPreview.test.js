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
// Ränder kommen aus dem Format (etikettformate.js, gehalten gegen pdf/etikett_formate.go).
describe('Druck-Center: Vorschau je Etikettenformat', () => {
	beforeEach(async () => {
		vi.clearAllMocks();
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

// Bei einem Titel mit 409 Exemplaren zeichnete die Vorschau alle Bogen untereinander, die
// Seite war rund 13.000 px hoch. Alle Bogen zeigt die PDF, die „A4-Bogen drucken" öffnet.
describe('Druck-Center: Die Vorschau zeichnet den ersten Bogen', () => {
	beforeEach(() => {
		vi.clearAllMocks();
		labelStore.formatId = 'zweckform_l4760';
		labelStore.generationMode = 'existing';
		labelStore.startPosition = 1;
	});

	afterEach(() => {
		labelStore.startPosition = 1;
	});

	/** @param {number} anzahl */
	async function waehleTitelMit(anzahl) {
		vi.mocked(apiFetch).mockResolvedValueOnce(
			/** @type {any} */ ({
				ok: true,
				json: async () =>
					Array.from({ length: anzahl }, (_, i) => ({ barcode_id: `B-${1000 + i}` }))
			})
		);
		await labelStore.selectBookTitle({ id: `t${anzahl}`, titel: 'Titel', autor: 'Autorin' });
		flushSync();
	}

	const felder = () =>
		/** @type {HTMLElement} */ (
			document.querySelector('[data-testid="etiketten-blatt"]')?.firstElementChild
		).children.length;

	it('nennt bei mehr Etiketten, als auf einen Bogen passen, die Zahl der Bogen', async () => {
		const screen = render(LabelPreview);
		await waehleTitelMit(50);

		expect(felder()).toBe(21);
		expect(screen.getByText('Bogen 1 von 3 · 50 Etiketten')).toBeTruthy();
	});

	it('zählt freigelassene Felder zum Bogen und nicht zu den Etiketten', async () => {
		const screen = render(LabelPreview);
		await waehleTitelMit(41);
		labelStore.startPosition = 3;
		flushSync();

		expect(felder()).toBe(21);
		expect(screen.getByText('Bogen 1 von 3 · 41 Etiketten')).toBeTruthy();
	});

	it('zeigt bei einem einzigen Bogen keine Zeile darunter', async () => {
		const screen = render(LabelPreview);
		await waehleTitelMit(21);

		expect(felder()).toBe(21);
		expect(screen.queryByText(/^Bogen 1 von/)).toBeNull();
	});
});

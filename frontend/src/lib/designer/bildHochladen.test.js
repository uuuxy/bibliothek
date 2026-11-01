import { describe, it, expect, beforeEach, vi } from 'vitest';
import { render, fireEvent, waitFor } from '@testing-library/svelte';
import Eigenschaften from './Eigenschaften.svelte';
import Werkzeugleiste from './Werkzeugleiste.svelte';
import { fuegeBilderHinzu, idStore, resetDesign } from './idDesignerStore.svelte.js';

vi.mock('../apiFetch.js', () => ({ apiFetch: vi.fn() }));

// Ein Bild für die Karte kommt nur über das Dateifeld: Ein Schul-Logo lässt sich sonst nicht
// auf den Ausweis bringen.
const datei = () => new File(['png'], 'logo.png', { type: 'image/png' });

/** @param {string} kennung */
const inhalt = (kennung) => idStore.front.elements.find((e) => e.id === kennung)?.content;

function fuegeZweiBilderHinzu() {
	fuegeBilderHinzu('front', ['data:image/gif;base64,erstes', 'data:image/gif;base64,zweites']);
	return idStore.front.elements.slice(-2).map((e) => e.id);
}

const LEISTE = {
	zoom: 100,
	onZoom: vi.fn(),
	side: /** @type {'front'} */ ('front'),
	onSide: vi.fn(),
	druckArt: /** @type {'card'} */ ('card'),
	onDruckArt: vi.fn(),
	onPrint: vi.fn(),
	barcodeType: /** @type {'code39'} */ ('code39'),
	onBarcodeType: vi.fn(),
	onVorlage: vi.fn(),
	muster: null
};

describe('Ausweis-Designer: Bild hochladen', () => {
	beforeEach(() => resetDesign());

	it('die Werkzeugleiste legt je Datei ein Bild auf die Karte', async () => {
		const vorher = idStore.front.elements.length;
		const { container } = render(Werkzeugleiste, LEISTE);
		const feld = /** @type {HTMLInputElement} */ (container.querySelector('input[type="file"]'));

		await fireEvent.change(feld, { target: { files: [datei(), datei()] } });

		await waitFor(() => expect(idStore.front.elements.length).toBe(vorher + 2));
	});

	// Bliebe die Datei im Feld stehen, meldete der Browser beim zweiten Wählen derselben Datei
	// keine Änderung, und ein gelöschtes Logo käme nicht mehr auf die Karte.
	it('die Werkzeugleiste leert das Feld nach dem Einlesen', async () => {
		const { container } = render(Werkzeugleiste, LEISTE);
		const feld = /** @type {HTMLInputElement} */ (container.querySelector('input[type="file"]'));
		const geleert = vi.fn();
		Object.defineProperty(feld, 'value', { configurable: true, set: geleert });

		await fireEvent.change(feld, { target: { files: [datei()] } });

		await waitFor(() => expect(geleert).toHaveBeenCalledWith(''));
	});

	// Ohne Leeren käme dasselbe Bild für ein zweites Element nicht an: Das Feld bleibt beim Wechsel
	// des Elements dasselbe, und der Browser meldet keine Änderung.
	it('die Eigenschaften leeren das Feld nach dem Wählen', async () => {
		const element = idStore.front.elements.find((e) => e.type === 'logo' || e.type === 'image');
		const kennung = /** @type {any} */ (element).id;
		const { container } = render(Eigenschaften, { gewaehlteKennung: kennung, side: 'front' });
		const feld = /** @type {HTMLInputElement} */ (container.querySelector('input[type="file"]'));
		const geleert = vi.fn();
		Object.defineProperty(feld, 'value', { configurable: true, set: geleert });

		await fireEvent.change(feld, { target: { files: [datei()] } });

		expect(geleert).toHaveBeenCalledWith('');
	});

	it('das Bild geht an das Element, das beim Wählen gewählt war', async () => {
		const [erstes, zweites] = fuegeZweiBilderHinzu();
		const screen = render(Eigenschaften, { gewaehlteKennung: erstes, side: 'front' });
		const feld = /** @type {HTMLInputElement} */ (
			screen.container.querySelector('input[type="file"]')
		);

		await fireEvent.change(feld, { target: { files: [datei()] } });
		await screen.rerender({ gewaehlteKennung: zweites });

		await waitFor(() => expect(inhalt(erstes)).toMatch(/^data:image\/png/));
		expect(inhalt(zweites)).toBe('data:image/gif;base64,zweites');
	});

	it('die Eigenschaften tauschen das Bild des gewählten Elements', async () => {
		const element = idStore.front.elements.find((e) => e.type === 'logo' || e.type === 'image');
		const kennung = /** @type {any} */ (element).id;
		const { container } = render(Eigenschaften, { gewaehlteKennung: kennung, side: 'front' });
		const feld = /** @type {HTMLInputElement} */ (container.querySelector('input[type="file"]'));

		await fireEvent.change(feld, { target: { files: [datei()] } });

		await waitFor(() =>
			expect(idStore.front.elements.find((e) => e.id === kennung)?.content).toMatch(
				/^data:image\/png/
			)
		);
	});
});

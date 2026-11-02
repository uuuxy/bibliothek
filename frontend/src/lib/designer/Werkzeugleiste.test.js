import { describe, it, expect, beforeEach } from 'vitest';
import { render, fireEvent } from '@testing-library/svelte';
import Werkzeugleiste from './Werkzeugleiste.svelte';
import { idStore, resetDesign } from './idDesignerStore.svelte.js';

const leer = () => {};

/** @param {'front'|'back'} side */
function leiste(side) {
	return render(Werkzeugleiste, {
		zoom: 1,
		onZoom: leer,
		side,
		onSide: leer,
		druckArt: 'card',
		onDruckArt: leer,
		onPrint: leer,
		barcodeType: 'code39',
		onBarcodeType: leer,
		onVorlage: leer,
		muster: null
	});
}

// Vorder- und Rückseite tragen je einen eigenen Hintergrund; die Auswahl gilt der Seite, die
// gerade offen ist.
describe('Werkzeugleiste: Karten-Hintergrund', () => {
	beforeEach(() => resetDesign());

	it('färbt die offene Seite, die andere bleibt', async () => {
		const vorn = idStore.front.theme;
		const screen = leiste('back');

		await fireEvent.click(screen.getByRole('combobox', { name: 'Karten-Hintergrund' }));
		await fireEvent.click(await screen.findByRole('option', { name: 'Smaragd' }));

		expect(idStore.back.theme).toBe('smaragd');
		expect(idStore.front.theme).toBe(vorn);
	});
});

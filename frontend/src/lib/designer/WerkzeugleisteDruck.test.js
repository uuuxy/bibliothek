import { describe, it, expect, vi } from 'vitest';
import { render, fireEvent, waitFor } from '@testing-library/svelte';
import WerkzeugleisteDruck from './WerkzeugleisteDruck.svelte';

// Das Muster-Etikett holt der Server; ein zweiter Klick vor der Antwort öffnete einen zweiten Tab
// mit demselben Bogen, wie beim Stapeldruck der Schülerdatei.
describe('WerkzeugleisteDruck: Testdruck', () => {
	it('nimmt bis zum Ende des Drucks keinen zweiten Klick an', async () => {
		/** @type {() => void} */
		let fertig = () => {};
		const onPrint = vi.fn(() => new Promise((weiter) => (fertig = () => weiter(undefined))));
		const screen = render(WerkzeugleisteDruck, {
			druckArt: 'etikett',
			onDruckArt: vi.fn(),
			side: 'front',
			onPrint
		});
		const knopf = /** @type {HTMLButtonElement} */ (
			screen.getByRole('button', { name: /Muster-Etikett drucken/ })
		);

		await fireEvent.click(knopf);
		await fireEvent.click(knopf);

		expect(onPrint).toHaveBeenCalledTimes(1);
		expect(knopf.disabled, 'gesperrt, solange der Druck läuft').toBe(true);
		fertig();
		await waitFor(() => expect(knopf.disabled, 'nach dem Druck wieder frei').toBe(false));
	});
});

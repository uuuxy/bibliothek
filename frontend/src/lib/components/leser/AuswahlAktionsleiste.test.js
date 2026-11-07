import { describe, it, expect, vi } from 'vitest';
import { render, fireEvent, waitFor } from '@testing-library/svelte';
import AuswahlAktionsleiste from './AuswahlAktionsleiste.svelte';

// Ein zweiter Klick vor dem Ende des Drucks schickte den Etikettenbogen noch einmal und öffnete ihn
// in einem zweiten Tab, bei einer um 2,5 s verzögerten Antwort gleich in Chromium, Firefox und
// WebKit; wie „A4-Bogen drucken" in Etikettendruck.svelte.
describe('AuswahlAktionsleiste: Drucken', () => {
	for (const etikettModus of [true, false]) {
		const name = etikettModus ? 'Etiketten drucken' : 'Ausweise drucken';
		it(`${name}: nimmt bis zum Ende des Drucks keinen zweiten Klick an`, async () => {
			/** @type {() => void} */
			let fertig = () => {};
			const onDrucken = vi.fn(() => new Promise((weiter) => (fertig = () => weiter(undefined))));
			const screen = render(AuswahlAktionsleiste, {
				anzahl: 2,
				ohneDatum: 0,
				etikettModus,
				onDrucken,
				onLeeren: vi.fn()
			});
			const knopf = /** @type {HTMLButtonElement} */ (
				screen.getByRole('button', { name: new RegExp(name) })
			);

			await fireEvent.click(knopf);
			await fireEvent.click(knopf);

			expect(onDrucken).toHaveBeenCalledTimes(1);
			expect(knopf.disabled, 'gesperrt, solange der Druck läuft').toBe(true);
			fertig();
			await waitFor(() => expect(knopf.disabled, 'nach dem Druck wieder frei').toBe(false));
			await fireEvent.click(knopf);
			expect(onDrucken).toHaveBeenCalledTimes(2);
		});
	}
});

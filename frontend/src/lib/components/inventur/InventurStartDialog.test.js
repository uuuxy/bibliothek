import { describe, it, expect, vi } from 'vitest';
import { render, fireEvent, waitFor } from '@testing-library/svelte';
import InventurStartDialog from './InventurStartDialog.svelte';

// Ein zweiter Klick auf „Inventur Starten" vor der Antwort schickte einen zweiten Start. Der
// Server lehnt ihn ab, weil der Bereich schon läuft; nach dem Abschluss stand dann „Für diesen
// Bereich läuft bereits eine Inventur", obwohl keine lief. Gleich in Chromium, Firefox und WebKit
// (OFFEN.md 5.106).
describe('InventurStartDialog: Start', () => {
	it('schickt keinen zweiten Start, solange der erste läuft', async () => {
		/** @type {() => void} */
		let fertig = () => {};
		const onStart = vi.fn(() => new Promise((weiter) => (fertig = () => weiter(undefined))));
		const k = render(InventurStartDialog, {
			props: {
				open: true,
				ablauf: {
					bereichArt: 'global',
					gewaehlteSignatur: '',
					gewaehltesFach: '',
					gewaehlterJahrgang: '',
					signaturen: [],
					faecher: [],
					fehler: ''
				},
				onClose: vi.fn(),
				onStart
			}
		});
		const knopf = /** @type {HTMLButtonElement} */ (
			k.getByRole('button', { name: /Inventur Starten/ })
		);

		await fireEvent.click(knopf);
		await fireEvent.click(knopf);

		expect(onStart, 'ein zweiter Start vor der Antwort').toHaveBeenCalledTimes(1);
		expect(knopf.disabled, 'gesperrt, solange der Start läuft').toBe(true);

		fertig();
		await waitFor(() => expect(knopf.disabled).toBe(false));
	});
});

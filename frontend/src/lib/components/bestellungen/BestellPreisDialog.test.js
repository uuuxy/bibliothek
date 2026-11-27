import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, fireEvent, waitFor } from '@testing-library/svelte';

vi.mock('../../apiFetch.js', async (original) => ({
	.../** @type {any} */ (await original()),
	apiPut: vi.fn()
}));
vi.mock('../../stores/snackbars.svelte.js', () => ({ snackbars: { addToast: vi.fn() } }));

import { apiPut } from '../../apiFetch.js';
import BestellPreisDialog from './BestellPreisDialog.svelte';

const position = { id: 'pos-1', titel_name: 'Mathe 7', menge: 2, einzelpreis: 9.5 };

// Der Preis laut Rechnung: Das Feld steht auf dem gespeicherten Preis, ein neuer Preis geht an die
// Position dieser Bestellung, und ohne Änderung bleibt der Knopf gesperrt.
describe('BestellPreisDialog', () => {
	beforeEach(() => vi.clearAllMocks());

	it('schickt den neuen Preis an die Position und lädt danach neu', async () => {
		vi.mocked(apiPut).mockResolvedValueOnce({ einzelpreis: 12.4, von: 9.5 });
		const onAktualisieren = vi.fn();
		const onclose = vi.fn();
		const screen = render(BestellPreisDialog, {
			props: { bestellungId: 'best-1', position, onclose, onAktualisieren }
		});

		const feld = /** @type {HTMLInputElement} */ (await screen.findByLabelText('Einzelpreis*'));
		expect(feld.value).toBe('9,50');
		const knopf = /** @type {HTMLButtonElement} */ (
			screen.getByRole('button', { name: 'Preis ändern' })
		);
		// Ein unveränderter Preis ist noch keine Wahl: gesperrt (M3 Dialogs).
		expect(knopf.disabled).toBe(true);
		expect(
			screen.getByText('Gilt für 2 Exemplare von „Mathe 7“ aus dieser Bestellung')
		).toBeTruthy();

		await fireEvent.input(feld, { target: { value: '12.4' } });
		expect(knopf.disabled).toBe(false);
		await fireEvent.click(knopf);

		await waitFor(() =>
			expect(apiPut).toHaveBeenCalledWith('/api/bestellungen/best-1/positionen/pos-1/einzelpreis', {
				einzelpreis: 12.4
			})
		);
		await waitFor(() => expect(onclose).toHaveBeenCalled());
		expect(onAktualisieren).toHaveBeenCalled();
	});

	// Ein leerer, falscher, negativer oder zu großer Preis sperrt den Knopf nicht: Der Klick nennt
	// am Feld den Grund und speichert nichts, statt 0 € zu schreiben.
	it('nennt bei leerem, falschem, negativem oder zu großem Preis das Feld und speichert nicht', async () => {
		const screen = render(BestellPreisDialog, {
			props: { bestellungId: 'best-1', position, onclose: vi.fn(), onAktualisieren: vi.fn() }
		});
		const feld = /** @type {HTMLInputElement} */ (await screen.findByLabelText('Einzelpreis*'));
		const knopf = /** @type {HTMLButtonElement} */ (
			screen.getByRole('button', { name: 'Preis ändern' })
		);
		for (const getippt of ['', 'zwölf', '-3', '100000']) {
			await fireEvent.input(feld, { target: { value: getippt } });
			expect(knopf.disabled, getippt).toBe(false);
			await fireEvent.click(knopf);
			expect(screen.getByText('Bitte einen Preis von 0 bis 99.999,99 € eintragen')).toBeTruthy();
			expect(feld.getAttribute('aria-invalid')).toBe('true');
			await waitFor(() => expect(document.activeElement).toBe(feld));
		}
		expect(apiPut).not.toHaveBeenCalled();
	});
});

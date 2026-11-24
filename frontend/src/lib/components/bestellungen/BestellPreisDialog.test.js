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

		const feld = /** @type {HTMLInputElement} */ (await screen.findByLabelText('Einzelpreis'));
		expect(Number(feld.value)).toBe(9.5);
		const knopf = /** @type {HTMLButtonElement} */ (
			screen.getByRole('button', { name: 'Preis ändern' })
		);
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

	it('sperrt einen negativen Preis', async () => {
		const screen = render(BestellPreisDialog, {
			props: { bestellungId: 'best-1', position, onclose: vi.fn(), onAktualisieren: vi.fn() }
		});
		const feld = await screen.findByLabelText('Einzelpreis');
		await fireEvent.input(feld, { target: { value: '-3' } });
		const knopf = /** @type {HTMLButtonElement} */ (
			screen.getByRole('button', { name: 'Preis ändern' })
		);
		expect(knopf.disabled).toBe(true);
		expect(apiPut).not.toHaveBeenCalled();
	});
});

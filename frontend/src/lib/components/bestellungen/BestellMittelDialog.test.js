import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, fireEvent, waitFor } from '@testing-library/svelte';

vi.mock('../../apiFetch.js', () => ({ apiPut: vi.fn(async () => ({})) }));
vi.mock('../../stores/snackbars.svelte.js', () => ({ snackbars: { addToast: vi.fn() } }));

import { apiPut } from '../../apiFetch.js';
import BestellMittelDialog from './BestellMittelDialog.svelte';

// Ein unveränderter Topf ist noch keine Wahl: gesperrt. Ein fehlender Grund sperrt nicht; der
// Klick nennt ihn am Feld, führt dorthin und schickt nichts.
describe('BestellMittelDialog', () => {
	beforeEach(() => vi.clearAllMocks());

	const dialog = () =>
		render(BestellMittelDialog, {
			open: true,
			bestellungId: 'b1',
			aktuell: 'land',
			onclose: () => {},
			onAktualisieren: () => {}
		});

	it('sperrt den unveränderten Topf und nennt einen fehlenden Grund beim Klick', async () => {
		const { getByRole, getByLabelText, getByText } = dialog();
		const knopf = /** @type {HTMLButtonElement} */ (getByRole('button', { name: 'Topf ändern' }));
		expect(knopf.disabled).toBe(true);

		await fireEvent.click(getByLabelText('Spende · ohne Topf'));
		expect(knopf.disabled).toBe(false);
		await fireEvent.click(knopf);
		const feld = getByLabelText(/Grund der Korrektur/);
		expect(getByText('Bitte den Grund der Korrektur eintragen')).toBeTruthy();
		expect(feld.getAttribute('aria-invalid')).toBe('true');
		await waitFor(() => expect(document.activeElement).toBe(feld));
		expect(apiPut).not.toHaveBeenCalled();
	});

	it('schickt den neuen Topf mit getrimmtem Grund', async () => {
		const { getByRole, getByLabelText } = dialog();
		await fireEvent.click(getByLabelText('Spende · ohne Topf'));
		await fireEvent.input(getByLabelText(/Grund der Korrektur/), {
			target: { value: '  Spende des Fördervereins ' }
		});
		await fireEvent.click(getByRole('button', { name: 'Topf ändern' }));
		await waitFor(() =>
			expect(apiPut).toHaveBeenCalledWith('/api/bestellungen/b1/mittel', {
				mittel: 'spende',
				grund: 'Spende des Fördervereins'
			})
		);
	});
});

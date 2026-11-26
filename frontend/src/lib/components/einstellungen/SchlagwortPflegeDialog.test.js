import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, fireEvent, waitFor } from '@testing-library/svelte';

vi.mock('../../apiFetch.js', () => ({
	apiGet: vi.fn(async () => ({ zeilen: [] })),
	apiPost: vi.fn(async () => ({})),
	apiPut: vi.fn(async () => ({ wort: 'Neu' }))
}));
vi.mock('../../stores/snackbars.svelte.js', () => ({ snackbars: { addToast: vi.fn() } }));

import { apiPost, apiPut } from '../../apiFetch.js';
import SchlagwortPflegeDialog from './SchlagwortPflegeDialog.svelte';

// Jede Art des Dialogs geht an ihre Tür: Umbenennen ändert das Wort, ein Verweis legt eine neue
// Schreibweise an. Vertauscht, legte ein Umbenennen still einen Verweis an.
/** @param {'umbenennen' | 'verweis'} art */
const dialog = (art) =>
	render(SchlagwortPflegeDialog, {
		auftrag: { art, zeile: { id: 's1', wort: 'Alt', titel: 3, verweise: [] } },
		onclose: () => {},
		onfertig: async () => {}
	});

describe('SchlagwortPflegeDialog', () => {
	beforeEach(() => vi.clearAllMocks());

	it('benennt über die Tür des Worts um', async () => {
		const { getByLabelText, getByRole } = dialog('umbenennen');
		await fireEvent.input(getByLabelText('Neue Schreibweise'), { target: { value: 'Neu' } });
		await fireEvent.click(getByRole('button', { name: 'Umbenennen' }));

		await waitFor(() =>
			expect(apiPut).toHaveBeenCalledWith(
				'/api/schlagworte/s1/wort',
				expect.objectContaining({ wort: 'Neu' })
			)
		);
		expect(apiPost).not.toHaveBeenCalled();
	});

	it('legt einen Verweis über die Tür der Verweise an', async () => {
		const { getByLabelText, getByRole } = dialog('verweis');
		await fireEvent.input(getByLabelText('Schreibweise'), { target: { value: 'Ald' } });
		await fireEvent.click(getByRole('button', { name: 'Verweis anlegen' }));

		await waitFor(() =>
			expect(apiPost).toHaveBeenCalledWith('/api/schlagworte/verweise', {
				wort: 'Ald',
				ziel_id: 's1'
			})
		);
		expect(apiPut).not.toHaveBeenCalled();
	});
});

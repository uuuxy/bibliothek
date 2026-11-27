import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, fireEvent, waitFor } from '@testing-library/svelte';

vi.mock('../../apiFetch.js', () => ({
	apiGet: vi.fn(async () => ({ zeilen: [] })),
	apiPost: vi.fn(async () => ({})),
	apiPut: vi.fn(async () => ({ wort: 'Neu' }))
}));
vi.mock('../../stores/snackbars.svelte.js', () => ({ snackbars: { addToast: vi.fn() } }));

import { apiGet, apiPost, apiPut } from '../../apiFetch.js';
import SchlagwortPflegeDialog from './SchlagwortPflegeDialog.svelte';

// Jede Art des Dialogs geht an ihre Tür: Umbenennen ändert das Wort, ein Verweis legt eine neue
// Schreibweise an. Vertauscht, legte ein Umbenennen still einen Verweis an.
/** @param {'umbenennen' | 'verweis' | 'zusammenfuehren'} art */
const dialog = (art) =>
	render(SchlagwortPflegeDialog, {
		auftrag: { art, zeile: { id: 's1', wort: 'Alt', titel: 3, verweise: [] } },
		onclose: () => {},
		onfertig: async () => {}
	});

describe('SchlagwortPflegeDialog', () => {
	beforeEach(() => vi.clearAllMocks());

	// Eine unveränderte Schreibweise ist noch keine Wahl: gesperrt. Ein leeres Feld oder ein
	// Ziel, das es nicht gibt, nennt beim Klick das Feld und geht an keine Tür.
	it('sperrt eine unveränderte Schreibweise und nennt ein leeres Feld beim Klick', async () => {
		const { getByLabelText, getByRole, getByText } = dialog('umbenennen');
		const feld = /** @type {HTMLInputElement} */ (getByLabelText('Neue Schreibweise*'));
		const knopf = /** @type {HTMLButtonElement} */ (getByRole('button', { name: 'Umbenennen' }));
		expect(feld.value).toBe('Alt');
		expect(knopf.disabled).toBe(true);
		await fireEvent.input(feld, { target: { value: ' ' } });
		expect(knopf.disabled).toBe(false);
		await fireEvent.click(knopf);
		expect(getByText('Bitte die neue Schreibweise eintragen')).toBeTruthy();
		expect(feld.getAttribute('aria-invalid')).toBe('true');
		expect(apiPut).not.toHaveBeenCalled();
	});

	it('nennt beim Zusammenführen ein Ziel, das es nicht gibt', async () => {
		const { getByLabelText, getByRole, getByText } = dialog('zusammenfuehren');
		await fireEvent.input(getByLabelText('Mit Schlagwort*'), { target: { value: 'Gibtsnicht' } });
		const knopf = /** @type {HTMLButtonElement} */ (
			getByRole('button', { name: 'Zusammenführen' })
		);
		await waitFor(() => expect(knopf.disabled).toBe(false));
		await fireEvent.click(knopf);
		expect(
			getByText('Kein Schlagwort mit dieser Schreibweise, bitte eines aus der Liste wählen')
		).toBeTruthy();
		expect(apiPost).not.toHaveBeenCalled();
	});

	// Das Ziel steht erst mit der Antwort der Suche fest. Ein Klick davor fand keins, meldete
	// fälschlich „Kein Schlagwort …“ und schickte nichts; jetzt wartet der gesperrte Knopf.
	it('sperrt beim Zusammenführen, bis die Suche das Ziel kennt', async () => {
		vi.mocked(apiGet).mockResolvedValueOnce({
			zeilen: [{ id: 's2', wort: 'Kriminalroman', titel: 4, verweise: [] }]
		});
		const { getByLabelText, getByRole, queryByText } = dialog('zusammenfuehren');
		await fireEvent.input(getByLabelText('Mit Schlagwort*'), {
			target: { value: 'Kriminalroman' }
		});
		const knopf = /** @type {HTMLButtonElement} */ (
			getByRole('button', { name: 'Zusammenführen' })
		);
		expect(knopf.disabled).toBe(true);
		await fireEvent.click(knopf);
		expect(queryByText(/Kein Schlagwort mit dieser Schreibweise/)).toBeNull();
		await waitFor(() => expect(knopf.disabled).toBe(false));
		await fireEvent.click(knopf);
		await waitFor(() =>
			expect(apiPost).toHaveBeenCalledWith('/api/schlagworte/s1/zusammenfuehren', {
				ziel_id: 's2',
				alte_als_verweis: true
			})
		);
	});

	it('benennt über die Tür des Worts um', async () => {
		const { getByLabelText, getByRole } = dialog('umbenennen');
		await fireEvent.input(getByLabelText('Neue Schreibweise*'), { target: { value: 'Neu' } });
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
		await fireEvent.input(getByLabelText('Schreibweise*'), { target: { value: 'Ald' } });
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

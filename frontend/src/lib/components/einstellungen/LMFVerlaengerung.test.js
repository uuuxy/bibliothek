import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, fireEvent } from '@testing-library/svelte';

vi.mock('../../stores/bestaetigung.svelte.js', () => ({ bestaetigen: vi.fn() }));
vi.mock('../../apiFetch.js', () => ({
	apiFetch: vi.fn(async () => ({ ok: true, json: async () => ['7a', '7b'] })),
	apiClient: { post: vi.fn() },
	leseApiFehler: vi.fn()
}));

import { bestaetigen } from '../../stores/bestaetigung.svelte.js';
import { apiClient } from '../../apiFetch.js';
import LMFVerlaengerung from './LMFVerlaengerung.svelte';

/** Wählt die Klasse 7a, trägt den 27.07.2027 ein und drückt den Knopf. */
async function verlaengere() {
	const screen = render(LMFVerlaengerung);
	await fireEvent.click(screen.getByRole('combobox'));
	await fireEvent.click(await screen.findByRole('option', { name: '7a' }));
	await fireEvent.input(screen.getByLabelText('Neues Rückgabedatum'), {
		target: { value: '2027-07-27' }
	});
	await fireEvent.click(screen.getByRole('button', { name: 'Klassen-LMF global verlängern' }));
}

// Das Datumsfeld liefert seinen Wert als JJJJ-MM-TT. Die Rückfrage nennt das Datum, wie es
// überall im Programm steht; an den Server geht der Wert des Felds.
describe('LMF-Massenverlängerung: das neue Rückgabedatum', () => {
	beforeEach(() => {
		vi.mocked(bestaetigen).mockReset();
		vi.mocked(apiClient.post).mockReset();
	});

	it('steht in der Rückfrage als TT.MM.JJJJ', async () => {
		vi.mocked(bestaetigen).mockResolvedValueOnce(false);
		await verlaengere();

		await vi.waitFor(() =>
			expect(bestaetigen).toHaveBeenCalledWith(
				expect.objectContaining({
					titel: 'Alle LMF-Ausleihen der Klasse 7a verlängern?',
					text: expect.stringMatching(/^Neues Rückgabedatum 27\.07\.2027\. /)
				})
			)
		);
		expect(apiClient.post).not.toHaveBeenCalled();
	});

	it('geht nach dem Ja unverändert an den Server', async () => {
		vi.mocked(bestaetigen).mockResolvedValueOnce(true);
		vi.mocked(apiClient.post).mockResolvedValueOnce(
			/** @type {any} */ ({ ok: true, json: async () => ({ updated_count: 1 }) })
		);
		await verlaengere();

		await vi.waitFor(() =>
			expect(apiClient.post).toHaveBeenCalledWith('/api/ausleihen/global-extend-lmf', {
				klasse: '7a',
				neues_rueckgabe_datum: '2027-07-27'
			})
		);
	});
});

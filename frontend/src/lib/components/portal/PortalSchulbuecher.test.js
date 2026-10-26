import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, fireEvent, waitFor } from '@testing-library/svelte';
import PortalSchulbuecher from './PortalSchulbuecher.svelte';
import { apiFetch } from '../../apiFetch.js';
import { serverAntwort } from '../../serverAntwort.js';

vi.mock('../../apiFetch.js', async (importOriginal) => ({
	.../** @type {any} */ (await importOriginal()),
	apiFetch: vi.fn()
}));

const LISTE = {
	faecher: [{ fach: 'Mathematik', titel: 1, gesamt: 30, verliehen: 0, verfuegbar: 30 }],
	titel: []
};

beforeEach(() => {
	vi.mocked(apiFetch).mockReset();
});

// Lehnt der Server ab, steht sein Grund an der Stelle der Liste. Auch das Lesen des Grundes
// dauert: Hat inzwischen ein neuer Abruf begonnen, verdeckt der alte die neue Liste nicht.
describe('PortalSchulbuecher: der Server lehnt ab', () => {
	it('nennt den Grund, den der Server angibt', async () => {
		vi.mocked(apiFetch).mockResolvedValue(
			serverAntwort(403, { error: 'Keine Berechtigung für diese Aktion' })
		);
		const screen = render(PortalSchulbuecher);

		expect(await screen.findByText('Keine Berechtigung für diese Aktion')).toBeTruthy();
	});

	it('lässt die Ablehnung eines überholten Abrufs nicht über der neuen Liste stehen', async () => {
		/** @type {(rumpf: string) => void} */
		let alterGrund = () => {};
		vi.mocked(apiFetch)
			.mockResolvedValueOnce({
				...serverAntwort(500),
				text: () => new Promise((ja) => (alterGrund = ja))
			})
			.mockResolvedValue(serverAntwort(200, LISTE));
		const screen = render(PortalSchulbuecher);
		await waitFor(() => expect(apiFetch).toHaveBeenCalledTimes(1));

		await fireEvent.click(screen.getByRole('combobox', { name: 'Nach Jahrgang filtern' }));
		await fireEvent.click(screen.getByRole('option', { name: 'Jahrgang 6' }));
		await screen.findByTestId('schulbuecher-faecher');

		alterGrund(JSON.stringify({ error: 'Ein interner Serverfehler ist aufgetreten' }));
		await new Promise((weiter) => setTimeout(weiter, 0));

		expect(screen.queryByText('Ein interner Serverfehler ist aufgetreten')).toBeNull();
		expect(screen.getByTestId('schulbuecher-faecher')).toBeTruthy();
	});
});

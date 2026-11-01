import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, fireEvent, waitFor } from '@testing-library/svelte';
import TitelVerwaltung from './TitelVerwaltung.svelte';
import {
	holeBuecherListe,
	loescheBuecher,
	holeExterneCover,
	retryExterneCover
} from '$lib/admin_api.js';
import { appState } from '$lib/store.svelte.js';
import { bestaetigen, loeschenBestaetigen } from '../../../lib/stores/bestaetigung.svelte.js';
import { snackbars } from '../../../lib/stores/snackbars.svelte.js';

vi.mock('$lib/admin_api.js', async (importOriginal) => ({
	.../** @type {any} */ (await importOriginal()),
	holeBuecherListe: vi.fn(),
	loescheBuecher: vi.fn(),
	holeExterneCover: vi.fn(),
	retryExterneCover: vi.fn()
}));
vi.mock('../../../lib/stores/bestaetigung.svelte.js', async (importOriginal) => ({
	.../** @type {any} */ (await importOriginal()),
	bestaetigen: vi.fn(),
	loeschenBestaetigen: vi.fn()
}));

const TITEL = { id: 't1', title: 'Mathe 9', author: 'Autor', isbn: '', verfuegbar: 1, gesamt: 1 };

// Scheitert das Laden der Liste, nennt die Seite den Grund. Eine leere Tabelle sagte sonst
// „Keine Bücher gefunden", und weil der Fehler die Anmeldung zurücknahm, lud danach auch die
// Suche nicht mehr.
describe('TitelVerwaltung: die Liste lädt nicht', () => {
	beforeEach(() => {
		vi.mocked(holeBuecherListe).mockReset();
		appState.adminAuthenticated = true;
		appState.searchQuery = '';
	});

	it('nennt den Grund und lädt auf „Erneut versuchen"', async () => {
		vi.mocked(holeBuecherListe)
			.mockRejectedValueOnce(new Error('Ein interner Serverfehler ist aufgetreten'))
			.mockResolvedValueOnce([TITEL]);
		const screen = render(TitelVerwaltung);

		expect(await screen.findByText('Titel nicht geladen')).toBeTruthy();
		expect(screen.getByText('Ein interner Serverfehler ist aufgetreten')).toBeTruthy();
		expect(screen.queryByText('Keine Bücher gefunden.')).toBeNull();

		await fireEvent.click(screen.getByRole('button', { name: 'Erneut versuchen' }));

		expect(await screen.findByText('Mathe 9')).toBeTruthy();
		expect(screen.queryByText('Titel nicht geladen')).toBeNull();
	});

	it('lädt nach dem Fehler auf eine neue Suche hin', async () => {
		vi.mocked(holeBuecherListe)
			.mockRejectedValueOnce(new Error('Ein interner Serverfehler ist aufgetreten'))
			.mockResolvedValueOnce([TITEL]);
		const screen = render(TitelVerwaltung);
		await screen.findByText('Titel nicht geladen');

		appState.searchQuery = 'Mathe';

		expect(await screen.findByText('Mathe 9')).toBeTruthy();
		await waitFor(() => expect(holeBuecherListe).toHaveBeenCalledTimes(2));
	});
});

// Löschen und „Cover lokalisieren“ gehen beide erst nach einer Rückfrage los und nur, wenn es
// etwas zu tun gibt.
describe('TitelVerwaltung: Aktionen der Leiste', () => {
	beforeEach(() => {
		vi.mocked(holeBuecherListe).mockReset();
		vi.mocked(holeBuecherListe).mockResolvedValue([TITEL]);
		vi.mocked(loescheBuecher).mockReset();
		vi.mocked(loescheBuecher).mockResolvedValue(/** @type {any} */ ({}));
		vi.mocked(holeExterneCover).mockReset();
		vi.mocked(retryExterneCover).mockReset();
		vi.mocked(retryExterneCover).mockResolvedValue(
			/** @type {any} */ ({ updated: 1, skipped: 0, failed: 0 })
		);
		vi.mocked(bestaetigen).mockReset();
		vi.mocked(bestaetigen).mockResolvedValue(true);
		vi.mocked(loeschenBestaetigen).mockReset();
		vi.mocked(loeschenBestaetigen).mockResolvedValue(true);
		appState.adminAuthenticated = true;
		appState.searchQuery = '';
		for (const t of [...snackbars.toasts]) snackbars.removeToast(t.id);
	});

	it('löscht die gewählten Titel nach der Rückfrage', async () => {
		const screen = render(TitelVerwaltung);
		await screen.findByText('Mathe 9');

		await fireEvent.click(screen.getByLabelText('Buch auswählen'));
		await fireEvent.click(screen.getByRole('button', { name: /Löschen \(1\)/ }));

		await waitFor(() => expect(loescheBuecher).toHaveBeenCalledWith(['t1']));
	});

	it('lokalisiert externe Cover nach der Rückfrage', async () => {
		vi.mocked(holeExterneCover).mockResolvedValue(/** @type {any} */ ([{ id: 't1' }]));
		const screen = render(TitelVerwaltung);
		await screen.findByText('Mathe 9');

		await fireEvent.click(screen.getByRole('button', { name: 'Cover lokalisieren' }));

		await waitFor(() => expect(retryExterneCover).toHaveBeenCalledWith(['t1']));
	});

	it('sagt ohne externes Cover, dass es nichts zu tun gibt, und fragt nicht', async () => {
		vi.mocked(holeExterneCover).mockResolvedValue([]);
		const screen = render(TitelVerwaltung);
		await screen.findByText('Mathe 9');

		await fireEvent.click(screen.getByRole('button', { name: 'Cover lokalisieren' }));

		await waitFor(() =>
			expect(snackbars.toasts.map((t) => t.message)).toContain(
				'Keine externen Cover mehr vorhanden.'
			)
		);
		expect(bestaetigen).not.toHaveBeenCalled();
	});
});

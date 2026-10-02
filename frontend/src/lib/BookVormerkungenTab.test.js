import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, fireEvent, waitFor, within } from '@testing-library/svelte';
import BookVormerkungenTab from './BookVormerkungenTab.svelte';
import { apiFetch, apiClient } from './apiFetch.js';
import { loeschenBestaetigen } from './stores/bestaetigung.svelte.js';
import { authStore } from './stores/authStore.svelte.js';
import { toastStore } from './stores/toastStore.svelte.js';

vi.mock('./apiFetch.js', async (importOriginal) => ({
	.../** @type {any} */ (await importOriginal()),
	apiFetch: vi.fn(),
	apiClient: { post: vi.fn() }
}));
vi.mock('./stores/bestaetigung.svelte.js', async (importOriginal) => ({
	.../** @type {any} */ (await importOriginal()),
	loeschenBestaetigen: vi.fn(async () => true)
}));

const antwort = (/** @type {any} */ daten, ok = true) =>
	/** @type {any} */ ({ ok, json: async () => daten });

const VORMERKUNG = {
	id: 'v1',
	erstellt_am: '2026-09-20T10:00:00Z',
	schueler_name: 'Ida Imker',
	notiz: 'Referat'
};

beforeEach(() => {
	vi.mocked(apiFetch).mockReset();
	vi.mocked(apiClient.post).mockReset();
	vi.mocked(loeschenBestaetigen).mockClear();
	authStore.currentUser = /** @type {any} */ ({
		rolle: 'mitarbeiter',
		permissions: ['view_students']
	});
	for (const t of [...toastStore.toasts]) toastStore.removeToast(t.id);
});

// Der Reiter hält zwei Abläufe: einen Schüler vormerken und eine Vormerkung löschen. Beide
// stehen hier, damit eine Umstellung der Oberfläche keinen von ihnen still bricht.
describe('BookVormerkungenTab', () => {
	it('merkt einen gesuchten Schüler vor und lädt die Liste neu', async () => {
		vi.mocked(apiFetch).mockImplementation(async (adresse) =>
			String(adresse).startsWith('/api/schueler?q=')
				? antwort([
						{ id: 's1', vorname: 'Ida', nachname: 'Imker', klasse: '07A', barcode_id: 'A-1' }
					])
				: antwort([VORMERKUNG])
		);
		vi.mocked(apiClient.post).mockResolvedValue(antwort({}));
		const screen = render(BookVormerkungenTab, { vormerkungen: [], book: { id: 't1' } });

		await fireEvent.click(screen.getByRole('button', { name: '+ Schüler vormerken' }));
		await fireEvent.input(screen.getByLabelText('Schüler suchen (Name oder Barcode)'), {
			target: { value: 'Ida' }
		});
		await fireEvent.click(screen.getByRole('button', { name: 'Suchen' }));
		await fireEvent.click(await screen.findByRole('button', { name: 'Auswählen' }));

		await waitFor(() =>
			expect(apiClient.post).toHaveBeenCalledWith('/api/vormerkungen', {
				titel_id: 't1',
				schueler_id: 's1',
				notiz: ''
			})
		);
		const tabelle = await screen.findByRole('table', { name: 'Vormerkungen' });
		expect(within(tabelle).getByText('Ida Imker')).toBeTruthy();
		expect(toastStore.toasts.map((t) => t.message)).toContain('Erfolgreich vorgemerkt');
	});

	it('löscht eine Vormerkung nach der Rückfrage', async () => {
		vi.mocked(apiFetch).mockResolvedValue(antwort({}));
		const screen = render(BookVormerkungenTab, { vormerkungen: [VORMERKUNG], book: { id: 't1' } });

		await fireEvent.click(screen.getByRole('button', { name: 'Vormerkung löschen' }));

		await waitFor(() =>
			expect(apiFetch).toHaveBeenCalledWith('/api/vormerkungen/v1', { method: 'DELETE' })
		);
		expect(loeschenBestaetigen).toHaveBeenCalledWith('Vormerkung löschen?');
		expect(
			await screen.findByText('Keine ausstehenden Vormerkungen für diesen Titel.')
		).toBeTruthy();
	});
});

import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, fireEvent, waitFor, within } from '@testing-library/svelte';
import BuchakteVormerkungen from './BuchakteVormerkungen.svelte';
import { apiFetch, apiClient } from '../../apiFetch.js';
import { loeschenBestaetigen } from '../../stores/bestaetigung.svelte.js';
import { authStore } from '../../stores/authStore.svelte.js';
import { snackbars } from '../../stores/snackbars.svelte.js';
import { serverAntwort } from '../../serverAntwort.js';

vi.mock('../../apiFetch.js', async (importOriginal) => ({
	.../** @type {any} */ (await importOriginal()),
	apiFetch: vi.fn(),
	apiClient: { post: vi.fn() }
}));
vi.mock('../../stores/bestaetigung.svelte.js', async (importOriginal) => ({
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
	for (const t of [...snackbars.toasts]) snackbars.removeToast(t.id);
});

// Der Reiter hält zwei Abläufe: einen Schüler vormerken und eine Vormerkung löschen. Beide
// stehen hier, damit eine Umstellung der Oberfläche keinen von ihnen still bricht.
describe('BuchakteVormerkungen', () => {
	it('merkt einen gesuchten Schüler vor und lädt die Liste neu', async () => {
		vi.mocked(apiFetch).mockImplementation(async (adresse) =>
			String(adresse).startsWith('/api/schueler?q=')
				? antwort([
						{ id: 's1', vorname: 'Ida', nachname: 'Imker', klasse: '07A', barcode_id: 'A-1' }
					])
				: antwort([VORMERKUNG])
		);
		vi.mocked(apiClient.post).mockResolvedValue(antwort({}));
		const screen = render(BuchakteVormerkungen, { vormerkungen: [], titel: { id: 't1' } });

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
		expect(snackbars.toasts.map((t) => t.message)).toContain('Erfolgreich vorgemerkt');
	});

	// Angelegt ist angelegt: Scheitert danach das Nachladen der Liste, sagt die Meldung das zuerst
	// und nennt dann den Grund des Servers.
	it('nennt beim gescheiterten Nachladen die angelegte Vormerkung und den Grund', async () => {
		vi.mocked(apiFetch).mockImplementation(async (adresse) =>
			String(adresse).startsWith('/api/schueler?q=')
				? antwort([
						{ id: 's1', vorname: 'Ida', nachname: 'Imker', klasse: '07A', barcode_id: 'A-1' }
					])
				: serverAntwort(500, { error: 'Ein interner Serverfehler ist aufgetreten' })
		);
		vi.mocked(apiClient.post).mockResolvedValue(antwort({}));
		const screen = render(BuchakteVormerkungen, { vormerkungen: [], titel: { id: 't1' } });

		await fireEvent.click(screen.getByRole('button', { name: '+ Schüler vormerken' }));
		await fireEvent.input(screen.getByLabelText('Schüler suchen (Name oder Barcode)'), {
			target: { value: 'Ida' }
		});
		await fireEvent.click(screen.getByRole('button', { name: 'Suchen' }));
		await fireEvent.click(await screen.findByRole('button', { name: 'Auswählen' }));

		await waitFor(() =>
			expect(snackbars.toasts.map((t) => t.message)).toContain(
				'Vormerkung angelegt — die Liste konnte nicht neu geladen werden: Ein interner Serverfehler ist aufgetreten'
			)
		);
	});

	// M3, Tabs: „Text labels should clearly and succinctly describe the content within the
	// tab." Eine Überschrift darunter sagte dasselbe ein zweites Mal; die anderen Reiter der
	// Akte tragen keine.
	it('wiederholt den Namen des Reiters nicht als Überschrift', () => {
		const screen = render(BuchakteVormerkungen, {
			vormerkungen: [VORMERKUNG],
			titel: { id: 't1' }
		});
		expect(screen.queryAllByRole('heading')).toEqual([]);
		expect(screen.getByRole('button', { name: '+ Schüler vormerken' })).toBeTruthy();
	});

	it('löscht eine Vormerkung nach der Rückfrage', async () => {
		vi.mocked(apiFetch).mockResolvedValue(antwort({}));
		const screen = render(BuchakteVormerkungen, {
			vormerkungen: [VORMERKUNG],
			titel: { id: 't1' }
		});

		await fireEvent.click(screen.getByRole('button', { name: 'Vormerkung löschen' }));

		await waitFor(() =>
			expect(apiFetch).toHaveBeenCalledWith('/api/vormerkungen/v1', { method: 'DELETE' })
		);
		expect(loeschenBestaetigen).toHaveBeenCalledWith('Vormerkung löschen?');
		expect(
			await screen.findByText('Keine ausstehenden Vormerkungen für diesen Titel.')
		).toBeTruthy();
	});

	// Lehnt der Server die Suche ab, steht sein Grund da. „Keine Schüler gefunden" hieße, es gebe
	// den Schüler nicht.
	it('nennt bei einer abgelehnten Suche den Grund statt „Keine Schüler gefunden"', async () => {
		vi.mocked(apiFetch).mockResolvedValue(
			serverAntwort(403, { error: 'Keine Berechtigung für diese Aktion' })
		);
		const screen = render(BuchakteVormerkungen, { vormerkungen: [], titel: { id: 't1' } });

		await fireEvent.click(screen.getByRole('button', { name: '+ Schüler vormerken' }));
		await fireEvent.input(screen.getByLabelText('Schüler suchen (Name oder Barcode)'), {
			target: { value: 'Ida' }
		});
		await fireEvent.click(screen.getByRole('button', { name: 'Suchen' }));

		expect(await screen.findByText('Keine Berechtigung für diese Aktion')).toBeTruthy();
		expect(screen.queryByText('Keine Schüler gefunden.')).toBeNull();
	});

	it('schreibt das Datum der Vormerkung als TT.MM.JJJJ', () => {
		const screen = render(BuchakteVormerkungen, {
			vormerkungen: [VORMERKUNG],
			titel: { id: 't1' }
		});
		expect(screen.getByText('20.09.2026')).toBeTruthy();
	});
});

// Ein zweiter Auftrag vor der Antwort schickte eine zweite Anfrage: Nach „Erfolgreich vorgemerkt"
// stand „Dieser Schüler hat den Titel bereits vorgemerkt", nach „Vormerkung gelöscht" stand
// „Vormerkung nicht gefunden", in Chromium, Firefox und WebKit (OFFEN.md 5.106).
describe('BuchakteVormerkungen: ein Auftrag zur Zeit', () => {
	const leereWarteschlange = () => new Promise((weiter) => setTimeout(weiter));

	it('schickt bei einem zweiten Klick auf „Auswählen" vor der Antwort keine zweite Vormerkung', async () => {
		/** @type {(antwort: any) => void} */
		let liefere = () => {};
		vi.mocked(apiFetch).mockImplementation(async () =>
			antwort([{ id: 's1', vorname: 'Ida', nachname: 'Imker', klasse: '07A', barcode_id: 'A-1' }])
		);
		vi.mocked(apiClient.post).mockImplementation(() => new Promise((fertig) => (liefere = fertig)));
		const screen = render(BuchakteVormerkungen, { vormerkungen: [], titel: { id: 't1' } });
		await fireEvent.click(screen.getByRole('button', { name: '+ Schüler vormerken' }));
		await fireEvent.input(screen.getByLabelText('Schüler suchen (Name oder Barcode)'), {
			target: { value: 'Ida' }
		});
		await fireEvent.click(screen.getByRole('button', { name: 'Suchen' }));
		const knopf = /** @type {HTMLButtonElement} */ (
			await screen.findByRole('button', { name: 'Auswählen' })
		);

		await fireEvent.click(knopf);
		await fireEvent.click(knopf);
		await leereWarteschlange();

		expect(apiClient.post, 'eine zweite Vormerkung vor der Antwort').toHaveBeenCalledTimes(1);
		expect(knopf.disabled, 'gesperrt, solange das Vormerken läuft').toBe(true);

		liefere(serverAntwort(503, { error: 'Datenbank nicht erreichbar' }));
		await waitFor(() => expect(knopf.disabled).toBe(false));
	});

	it('schickt bei einem zweiten Auftrag zum Löschen vor der Antwort keine zweite Anfrage', async () => {
		/** @type {(antwort: any) => void} */
		let liefere = () => {};
		vi.mocked(apiFetch).mockImplementation(() => new Promise((fertig) => (liefere = fertig)));
		const screen = render(BuchakteVormerkungen, {
			vormerkungen: [VORMERKUNG],
			titel: { id: 't1' }
		});
		const knopf = /** @type {HTMLButtonElement} */ (
			screen.getByRole('button', { name: 'Vormerkung löschen' })
		);

		await fireEvent.click(knopf);
		await waitFor(() => expect(apiFetch).toHaveBeenCalledTimes(1));
		await fireEvent.click(knopf);
		await leereWarteschlange();

		expect(apiFetch, 'ein zweites DELETE vor der Antwort').toHaveBeenCalledTimes(1);
		expect(knopf.disabled, 'gesperrt, solange das Löschen läuft').toBe(true);

		liefere(serverAntwort(503, { error: 'Datenbank nicht erreichbar' }));
		await waitFor(() => expect(knopf.disabled).toBe(false));
	});
});

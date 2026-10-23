import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, fireEvent } from '@testing-library/svelte';

vi.mock('./apiFetch.js', async (importOriginal) => ({
	.../** @type {any} */ (await importOriginal()),
	apiClient: { patch: vi.fn() }
}));
vi.mock('./stores/snackbars.svelte.js', () => ({ snackbars: { addToast: vi.fn() } }));

import { apiClient } from './apiFetch.js';
import { snackbars } from './stores/snackbars.svelte.js';
import LeserSperreDialog from './LeserSperreDialog.svelte';
import { serverAntwort } from './serverAntwort.js';

// „Abbrechen" bleibt frei, während die Anfrage läuft (M3 Dialogs: „Dismissive actions are never
// disabled."). Die Antwort geht dann nicht verloren: Ist der Dialog zu, meldet die Snackbar
// Erfolg und Fehler, sonst hielte man die Sperre für nicht geschehen (components/ui/dialogAnfrage).

const LESER = { id: '7', vorname: 'Mia', nachname: 'Muster', art: 'schueler', ist_gesperrt: false };

/** Eine Antwort, die der Test von außen schickt. */
function spaeteAntwort() {
	/** @type {(v: any) => void} */
	let schicke = () => {};
	vi.mocked(apiClient.patch).mockReturnValue(new Promise((res) => (schicke = res)));
	return (/** @type {any} */ antwort) => schicke(antwort);
}

// Ein ganzer Takt: Danach ist jeder anstehende Mikroschritt gelaufen, wie viele der Dialog auch
// braucht, um die Antwort zu lesen.
const stillhalten = () => new Promise((fertig) => setTimeout(fertig, 0));

async function sperreMitGrund() {
	const onsuccess = vi.fn();
	const screen = render(LeserSperreDialog, { props: { open: true, leser: LESER, onsuccess } });
	await fireEvent.input(screen.getByRole('textbox'), { target: { value: 'Buch verloren' } });
	await fireEvent.click(screen.getByRole('button', { name: 'Bestätigen' }));
	return { screen, onsuccess };
}

describe('Leser sperren: Abbrechen während der Anfrage', () => {
	beforeEach(() => {
		vi.mocked(apiClient.patch).mockReset();
		vi.mocked(snackbars.addToast).mockReset();
	});

	it('schließt sofort und meldet einen späten Fehler in der Snackbar', async () => {
		const schicke = spaeteAntwort();
		const { screen } = await sperreMitGrund();

		const abbrechen = screen.getByRole('button', { name: 'Abbrechen' });
		expect(/** @type {HTMLButtonElement} */ (abbrechen).disabled).toBe(false);
		await fireEvent.click(abbrechen);
		expect(screen.queryByRole('dialog')).toBeNull();

		schicke(serverAntwort(409, { error: 'Der Leser liegt im Papierkorb.' }));
		await stillhalten();
		expect(snackbars.addToast).toHaveBeenCalledWith('Der Leser liegt im Papierkorb.', 'error');
	});

	it('meldet einen späten Erfolg in der Snackbar und gibt ihn an die Akte weiter', async () => {
		const schicke = spaeteAntwort();
		const { screen, onsuccess } = await sperreMitGrund();
		await fireEvent.click(screen.getByRole('button', { name: 'Abbrechen' }));

		const antwort = { ...LESER, is_manually_blocked: true, ist_gesperrt: true };
		schicke({ ok: true, json: async () => antwort });
		await stillhalten();
		expect(snackbars.addToast).toHaveBeenCalledWith('Ausleihe für Mia Muster gesperrt.', 'success');
		expect(onsuccess).toHaveBeenCalledWith(antwort);
	});

	it('zeigt den Fehler der alten Anfrage nicht im wieder geöffneten Dialog', async () => {
		const schicke = spaeteAntwort();
		const { screen } = await sperreMitGrund();
		await fireEvent.click(screen.getByRole('button', { name: 'Abbrechen' }));
		await screen.rerender({ open: true });

		schicke(serverAntwort(409, { error: 'Der Leser liegt im Papierkorb.' }));
		await stillhalten();
		expect(screen.queryByRole('alert')).toBeNull();
		expect(snackbars.addToast).toHaveBeenCalledWith('Der Leser liegt im Papierkorb.', 'error');
	});

	it('zeigt den Fehler im Dialog, solange er offen ist', async () => {
		const schicke = spaeteAntwort();
		const { screen } = await sperreMitGrund();

		schicke(serverAntwort(409, { error: 'Der Leser liegt im Papierkorb.' }));
		await stillhalten();
		expect(screen.getByRole('alert').textContent).toContain('Der Leser liegt im Papierkorb.');
		expect(snackbars.addToast).not.toHaveBeenCalled();
	});
});

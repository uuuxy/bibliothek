import { describe, it, expect, vi } from 'vitest';
import { render, screen, fireEvent, waitFor } from '@testing-library/svelte';

vi.mock('../../stores/snackbars.svelte.js', () => ({
	snackbars: { addToast: vi.fn() }
}));
vi.mock('../../stores/bestaetigung.svelte.js', () => ({
	loeschenBestaetigen: vi.fn(async () => true)
}));
// leseApiFehler bleibt echt (ARCHITEKTUR.md 8.14).
vi.mock('../../apiFetch.js', async (importOriginal) => ({
	.../** @type {any} */ (await importOriginal()),
	apiFetch: vi.fn()
}));

import { apiFetch } from '../../apiFetch.js';
import { serverAntwort } from '../../serverAntwort.js';
import LeserVormerkungenKarte from './LeserVormerkungenKarte.svelte';

// Ein zweiter Auftrag, während das Löschen einer Vormerkung lief, schickte eine zweite Anfrage;
// nach „Vormerkung gelöscht" stand dann „Vormerkung nicht gefunden", in Chromium, Firefox und
// WebKit (OFFEN.md 5.106).
describe('Vormerkung löschen: ein Auftrag je Vormerkung', () => {
	it('schickt bei einem zweiten Klick vor der Antwort keine zweite Anfrage', async () => {
		/** @type {(antwort: any) => void} */
		let liefere = () => {};
		vi.mocked(apiFetch).mockImplementation(() => new Promise((fertig) => (liefere = fertig)));
		render(LeserVormerkungenKarte, {
			vormerkungen: [{ id: 'v1', titel_name: 'Momo', status: 'wartend', erstellt_am: '2026-10-01' }]
		});
		const knopf = /** @type {HTMLButtonElement} */ (
			screen.getByRole('button', { name: 'Vormerkung löschen' })
		);

		await fireEvent.click(knopf);
		await waitFor(() => expect(apiFetch).toHaveBeenCalledTimes(1));
		await fireEvent.click(knopf);
		await Promise.resolve();

		expect(apiFetch).toHaveBeenCalledTimes(1);
		expect(knopf.disabled).toBe(true);

		liefere(serverAntwort(503, { error: 'Datenbank nicht erreichbar' }));
		await waitFor(() => expect(knopf.disabled).toBe(false));
	});
});

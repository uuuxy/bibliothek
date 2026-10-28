import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, fireEvent, waitFor } from '@testing-library/svelte';

vi.mock('../../apiFetch.js', async (original) => ({
	.../** @type {any} */ (await original()),
	apiFetch: vi.fn(),
	apiGet: vi.fn(),
	apiPost: vi.fn(),
	apiDelete: vi.fn()
}));
vi.mock('../../stores/snackbars.svelte.js', () => ({ snackbars: { addToast: vi.fn() } }));

import { apiFetch, apiGet, apiDelete } from '../../apiFetch.js';
import Klassenleitungen from './Klassenleitungen.svelte';

// Scheitert das Laden, steht weder „Noch keine Zuordnungen" noch das Formular da: Hinzufügen
// ersetzt die Zuordnung derselben Klasse, und die stünde gerade nicht in der Liste.
describe('Klassenleitungen', () => {
	beforeEach(() => {
		vi.clearAllMocks();
		vi.mocked(apiFetch).mockResolvedValue(
			/** @type {any} */ ({ ok: true, json: async () => ['05F'] })
		);
	});

	it('zeigt nach einem gescheiterten Laden den Fehler statt einer leeren Liste', async () => {
		vi.mocked(apiGet).mockRejectedValueOnce(new Error('Netz weg'));
		const screen = render(Klassenleitungen);

		await waitFor(() => expect(screen.getByRole('alert')).toBeTruthy());
		expect(screen.queryByText('Noch keine Zuordnungen vorhanden.')).toBeNull();
		expect(screen.queryByRole('button', { name: 'Hinzufügen' })).toBeNull();
	});

	it('lädt auf „Erneut versuchen" neu und zeigt dann Liste und Formular', async () => {
		vi.mocked(apiGet).mockRejectedValueOnce(new Error('Netz weg'));
		vi.mocked(apiGet).mockResolvedValueOnce([
			{ klasse: '05F', lehrer_email: 'lehrkraft@schule.example' }
		]);
		const screen = render(Klassenleitungen);

		await fireEvent.click(await screen.findByRole('button', { name: 'Erneut versuchen' }));

		expect(await screen.findByText('lehrkraft@schule.example')).toBeTruthy();
		expect(screen.getByRole('button', { name: 'Hinzufügen' })).toBeTruthy();
		expect(screen.queryByRole('alert')).toBeNull();
	});
});

const ZEILE = { klasse: '05F', lehrer_email: 'lehrkraft@schule.example' };

// Ein zweiter Klick vor der Antwort schickte ein zweites DELETE: Nach „Zuordnung für 05F
// gelöscht." stand „Kein Eintrag für diese Klasse". Ging die Antwort verloren, blieb die gelöschte
// Zeile stehen. Beides in Chromium, Firefox und WebKit (OFFEN.md 5.106).
describe('Klassenleitungen: Löschen', () => {
	beforeEach(() => {
		vi.clearAllMocks();
		// Ganz zurück: Ein unverbrauchter Wert aus einem Fall davor käme sonst als erste Antwort.
		vi.mocked(apiGet).mockReset();
		vi.mocked(apiDelete).mockReset();
		vi.mocked(apiFetch).mockResolvedValue(
			/** @type {any} */ ({ ok: true, json: async () => ['05F'] })
		);
	});

	it('schickt bei einem zweiten Klick vor der Antwort kein zweites DELETE', async () => {
		vi.mocked(apiGet).mockResolvedValue([ZEILE]);
		/** @type {(wert?: any) => void} */
		let antworte = () => {};
		vi.mocked(apiDelete).mockReturnValueOnce(new Promise((fertig) => (antworte = fertig)));
		const screen = render(Klassenleitungen);
		const knopf = await screen.findByRole('button', { name: 'Zuordnung für 05F löschen' });

		await fireEvent.click(knopf);
		await fireEvent.click(knopf);

		expect(apiDelete).toHaveBeenCalledTimes(1);
		expect(
			/** @type {HTMLButtonElement} */ (knopf).disabled,
			'gesperrt, solange das Löschen läuft'
		).toBe(true);
		antworte();
		await waitFor(() => expect(apiGet).toHaveBeenCalledTimes(2));
	});

	it('lädt die Liste auch nach einem Fehler neu', async () => {
		vi.mocked(apiGet).mockResolvedValueOnce([ZEILE]).mockResolvedValueOnce([]);
		vi.mocked(apiDelete).mockRejectedValueOnce(new Error('Keine Verbindung zum Server.'));
		const screen = render(Klassenleitungen);

		await fireEvent.click(await screen.findByRole('button', { name: 'Zuordnung für 05F löschen' }));

		await waitFor(() => expect(screen.queryByText(ZEILE.lehrer_email)).toBeNull());
		expect(apiGet).toHaveBeenCalledTimes(2);
	});
});

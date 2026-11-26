import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, fireEvent, waitFor } from '@testing-library/svelte';

vi.mock('../../apiFetch.js', async (original) => ({
	.../** @type {any} */ (await original()),
	apiGet: vi.fn(),
	apiPut: vi.fn(),
	apiPost: vi.fn()
}));
vi.mock('../../stores/snackbars.svelte.js', () => ({ snackbars: { addToast: vi.fn() } }));

import { apiGet, apiPost, apiPut } from '../../apiFetch.js';
import AdressabgleichEinstellungen from './AdressabgleichEinstellungen.svelte';

const zugang = {
	adresse: 'https://webmail.schule.example',
	benutzer: 'bibliothek@schule.example',
	has_password: true,
	letzter_lauf: null
};

describe('AdressabgleichEinstellungen', () => {
	beforeEach(() => vi.clearAllMocks());

	it('zeigt nach einem gescheiterten Laden kein Formular, das sich speichern ließe', async () => {
		vi.mocked(apiGet).mockRejectedValueOnce(new Error('Netz weg'));
		const screen = render(AdressabgleichEinstellungen);

		await waitFor(() => expect(screen.getByRole('alert')).toBeTruthy());
		expect(screen.queryByLabelText('Webmail-Adresse')).toBeNull();
		expect(apiPut).not.toHaveBeenCalled();
	});

	it('lässt ohne gespeicherten Zugang keinen Abgleich zu', async () => {
		vi.mocked(apiGet).mockResolvedValueOnce({ adresse: '', benutzer: '', has_password: false });
		const screen = render(AdressabgleichEinstellungen);

		const probe = /** @type {HTMLButtonElement} */ (
			await screen.findByRole('button', { name: 'Probelauf' })
		);
		expect(probe.disabled).toBe(true);
		expect(screen.getByText('Noch kein Abgleich gelaufen')).toBeTruthy();
	});

	it('zeigt im Probelauf, wer welche Adresse bekäme und wer keine bekommt', async () => {
		vi.mocked(apiGet).mockResolvedValueOnce(zugang);
		vi.mocked(apiPost).mockResolvedValueOnce({
			probelauf: true,
			stand: { zeitpunkt: '2026-11-16T07:00:00Z', eintraege: 3, uebernommen: 1, uebergangen: 1 },
			uebernahmen: [
				{ konto_id: 'k1', name: 'Anna Beispiel', adresse: 'abeispiel@schule.example', weg: 'Name' }
			],
			uebergangen: [{ konto_id: 'k2', name: 'Max Muster', grund: 'Name im Adressbuch mehrfach' }]
		});
		const screen = render(AdressabgleichEinstellungen);

		await fireEvent.click(await screen.findByRole('button', { name: 'Probelauf' }));

		// Die Frist ist länger als die des Servers, damit die Maske die Antwort noch sieht.
		expect(apiPost).toHaveBeenCalledWith(
			'/api/admin/settings/adressabgleich/lauf',
			{ probelauf: true },
			{ timeoutMs: 150000 }
		);
		expect(await screen.findByText(/Probelauf: 1 Konto bekäme eine Adresse/)).toBeTruthy();
		expect(screen.getByText('abeispiel@schule.example, gefunden über: Name')).toBeTruthy();
		expect(screen.getByText('Keine Adresse: Name im Adressbuch mehrfach')).toBeTruthy();
		// Der Probelauf liest den letzten Lauf nicht neu: Er hat nichts eingetragen.
		expect(apiGet).toHaveBeenCalledTimes(1);
	});

	it('zeigt den Satz des Servers, wenn das Webmail die Anmeldung ablehnt', async () => {
		vi.mocked(apiGet).mockResolvedValue(zugang);
		vi.mocked(apiPost).mockRejectedValueOnce(
			new Error('Das Webmail hat die Anmeldung abgelehnt. Bitte Benutzername und Passwort prüfen.')
		);
		const screen = render(AdressabgleichEinstellungen);

		await fireEvent.click(await screen.findByRole('button', { name: 'Jetzt abgleichen' }));

		const meldung = await screen.findByRole('alert');
		expect(meldung.textContent).toContain('Anmeldung abgelehnt');
	});

	it('speichert erst nach einer Änderung und schickt das Passwort nur, wenn eins getippt ist', async () => {
		vi.mocked(apiGet).mockResolvedValue(zugang);
		vi.mocked(apiPut).mockResolvedValueOnce({ status: 'ok' });
		const screen = render(AdressabgleichEinstellungen);

		const speichern = /** @type {HTMLButtonElement} */ (
			await screen.findByRole('button', { name: 'Zugang speichern' })
		);
		expect(speichern.disabled).toBe(true);
		// Der gespeicherte Benutzername steht im Feld; leer, schickte das Speichern einen leeren.
		expect(/** @type {HTMLInputElement} */ (screen.getByLabelText('Benutzername')).value).toBe(
			'bibliothek@schule.example'
		);
		await fireEvent.input(screen.getByLabelText('Benutzername'), {
			target: { value: 'neu@schule.example' }
		});
		await fireEvent.click(speichern);

		await waitFor(() =>
			expect(apiPut).toHaveBeenCalledWith('/api/admin/settings/adressabgleich', {
				adresse: 'https://webmail.schule.example',
				benutzer: 'neu@schule.example',
				passwort: ''
			})
		);
	});

	it('lässt das Formular stehen, während es nach dem Speichern nachlädt', async () => {
		vi.mocked(apiGet).mockResolvedValueOnce(zugang);
		vi.mocked(apiGet).mockReturnValueOnce(new Promise(() => {}));
		vi.mocked(apiPut).mockResolvedValueOnce({ status: 'ok' });
		const screen = render(AdressabgleichEinstellungen);

		await fireEvent.input(await screen.findByLabelText('Benutzername'), {
			target: { value: 'neu@schule.example' }
		});
		await fireEvent.click(screen.getByRole('button', { name: 'Zugang speichern' }));

		await waitFor(() => expect(apiGet).toHaveBeenCalledTimes(2));
		expect(screen.getByLabelText('Webmail-Adresse')).toBeTruthy();
	});
});

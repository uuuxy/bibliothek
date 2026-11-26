import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, fireEvent, waitFor } from '@testing-library/svelte';

vi.mock('../../apiFetch.js', async (original) => ({
	.../** @type {any} */ (await original()),
	apiGet: vi.fn(),
	apiPut: vi.fn(),
	apiPost: vi.fn()
}));
vi.mock('../../stores/snackbars.svelte.js', () => ({ snackbars: { addToast: vi.fn() } }));

import { apiGet, apiPut, apiPost } from '../../apiFetch.js';
import MailEinstellungen from './MailEinstellungen.svelte';

// Scheitert das Laden, steht kein Formular da: Mit leeren Feldern speicherte „Speichern" einen
// leeren Server, Port und Benutzer über die echten Angaben, und keine Mail ginge mehr hinaus.
describe('MailEinstellungen', () => {
	beforeEach(() => vi.clearAllMocks());

	it('zeigt nach einem gescheiterten Laden kein Formular, das sich speichern ließe', async () => {
		vi.mocked(apiGet).mockRejectedValueOnce(new Error('Netz weg'));
		const screen = render(MailEinstellungen);

		await waitFor(() => expect(screen.getByRole('alert')).toBeTruthy());
		expect(screen.queryByLabelText('SMTP Host*')).toBeNull();
		expect(screen.queryByRole('button', { name: /Speichern/ })).toBeNull();
		expect(apiPut).not.toHaveBeenCalled();
	});

	it('lädt auf „Erneut versuchen" neu und zeigt dann die gespeicherten Angaben', async () => {
		vi.mocked(apiGet).mockRejectedValueOnce(new Error('Netz weg'));
		vi.mocked(apiGet).mockResolvedValueOnce({
			smtp_host: 'mail.schule.example',
			smtp_port: '587',
			smtp_user: 'bibliothek',
			sender_email: 'bibliothek@schule.example',
			has_password: true
		});
		const screen = render(MailEinstellungen);

		await fireEvent.click(await screen.findByRole('button', { name: 'Erneut versuchen' }));

		const host = /** @type {HTMLInputElement} */ (await screen.findByLabelText('SMTP Host*'));
		expect(host.value).toBe('mail.schule.example');
		// Auch der Benutzername: Stünde er leer da, löschte ihn das nächste Speichern.
		expect(screen.getByDisplayValue('bibliothek')).toBeTruthy();
		expect(screen.queryByRole('alert')).toBeNull();
	});
});

// Die Testmail geht an die Adresse im Feld; vorbelegt ist der Absender.
describe('MailEinstellungen: Verbindung testen', () => {
	beforeEach(() => vi.clearAllMocks());

	it('schickt die Testmail an die Adresse im Feld und meldet den Versand', async () => {
		vi.mocked(apiGet).mockResolvedValueOnce({
			smtp_host: 'mail.schule.example',
			smtp_port: '587',
			smtp_user: 'bibliothek',
			sender_email: 'bibliothek@schule.example',
			has_password: true
		});
		vi.mocked(apiPost).mockResolvedValueOnce({ status: 'ok' });
		const screen = render(MailEinstellungen);

		await fireEvent.click(await screen.findByRole('button', { name: 'Test-E-Mail senden' }));

		await waitFor(() =>
			expect(apiPost).toHaveBeenCalledWith(
				'/api/admin/settings/mail/test',
				{ to: 'bibliothek@schule.example' },
				expect.anything()
			)
		);
		expect(await screen.findByText('Test-E-Mail versendet')).toBeTruthy();
	});
});

// Der zweite Absender: Die Maske lädt und speichert ihn samt den angehakten Arten von Mails mit dem
// Postausgang, und der Testversand prüft ihn mit, sobald er gespeichert ist.
describe('MailEinstellungen: zweiter Absender', () => {
	beforeEach(() => vi.clearAllMocks());

	const mitPortal = {
		smtp_host: 'mail.schule.example',
		smtp_port: '587',
		smtp_user: 'bibliothek',
		sender_email: 'bibliothek@schule.example',
		has_password: true,
		portal: {
			sender_email: 'anliegen@schule.example',
			smtp_user: '',
			has_password: false,
			fuer: ['portal'],
			arten: ['portal', 'mahnliste', 'abgaenger', 'bestellung']
		}
	};

	it('speichert den zweiten Absender samt den angehakten Arten mit dem Postausgang', async () => {
		vi.mocked(apiGet).mockResolvedValueOnce(mitPortal);
		vi.mocked(apiPut).mockResolvedValueOnce({ status: 'ok' });
		const screen = render(MailEinstellungen);

		const gruppe = await screen.findByRole('group', { name: 'Zweiter Absender' });
		const adresse = /** @type {HTMLInputElement} */ (gruppe.querySelector('input[type="email"]'));
		expect(adresse.value).toBe('anliegen@schule.example');
		const portal = /** @type {HTMLInputElement} */ (
			screen.getByRole('checkbox', { name: 'Mails aus Mein Portal' })
		);
		expect(portal.checked).toBe(true);

		// Die Bestellmails dazu, die Mails aus Mein Portal heraus.
		await fireEvent.click(screen.getByRole('checkbox', { name: 'Bestellmails an Händler' }));
		await fireEvent.click(portal);
		await fireEvent.click(screen.getByRole('button', { name: 'Postausgang speichern' }));

		await waitFor(() =>
			expect(apiPut).toHaveBeenCalledWith(
				'/api/admin/settings/mail',
				expect.objectContaining({
					portal: {
						sender_email: 'anliegen@schule.example',
						smtp_user: '',
						smtp_password: '',
						fuer: ['bestellung']
					}
				})
			)
		);
	});

	it('prüft beim Testversand beide Absender und nennt jedes Ergebnis', async () => {
		vi.mocked(apiGet).mockResolvedValueOnce(mitPortal);
		vi.mocked(apiPost)
			.mockResolvedValueOnce({ status: 'ok' })
			.mockRejectedValueOnce(new Error('535 Anmeldung abgelehnt'));
		const screen = render(MailEinstellungen);

		await fireEvent.click(await screen.findByRole('button', { name: 'Test-E-Mail senden' }));

		expect(await screen.findByText('Zweiter Absender: Testversand fehlgeschlagen')).toBeTruthy();
		expect(screen.getByText('Postausgang: Test-E-Mail versendet')).toBeTruthy();
		expect(screen.getByText('535 Anmeldung abgelehnt')).toBeTruthy();
		expect(apiPost).toHaveBeenNthCalledWith(
			2,
			'/api/admin/settings/mail/test',
			{ to: 'bibliothek@schule.example', postfach: 'portal' },
			expect.anything()
		);
	});
});

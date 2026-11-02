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
		expect(screen.queryByLabelText('SMTP Host *')).toBeNull();
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

		const host = /** @type {HTMLInputElement} */ (await screen.findByLabelText('SMTP Host *'));
		expect(host.value).toBe('mail.schule.example');
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

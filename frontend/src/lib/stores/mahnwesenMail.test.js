import { describe, it, expect, vi, beforeEach } from 'vitest';

vi.mock('../apiFetch.js', () => ({
	apiFetch: vi.fn()
}));
vi.mock('../../inventur/lib/store.svelte.js', () => ({
	showToast: vi.fn()
}));

import { apiFetch } from '../apiFetch.js';
import { showToast } from '../../inventur/lib/store.svelte.js';
import { useMahnwesenMail } from './mahnwesenMail.svelte.js';

const apiFetchMock = vi.mocked(apiFetch);
const showToastMock = vi.mocked(showToast);

describe('useMahnwesenMail.sendBulkOverdueMails', () => {
	beforeEach(() => {
		vi.clearAllMocks();
	});

	const auswahl = { klassen: ['5a', '6b'] };

	it('POSTet die gewählten Klassen an /api/mail/send-bulk-overdue', async () => {
		apiFetchMock.mockResolvedValueOnce(
			/** @type {any} */ ({ ok: true, json: async () => ({ sent_count: 3 }) })
		);
		const store = useMahnwesenMail();

		await store.sendBulkOverdueMails(auswahl);

		expect(apiFetch).toHaveBeenCalledWith('/api/mail/send-bulk-overdue', {
			method: 'POST',
			headers: { 'Content-Type': 'application/json' },
			body: JSON.stringify({ klassen: ['5a', '6b'], override_email: '' })
		});
		expect(showToast).toHaveBeenCalledWith('3 Klassen-Mahnliste(n) versendet.', 'success');
	});

	it('reicht die Override-Adresse mit — sie entscheidet, wer die Listen bekommt', async () => {
		apiFetchMock.mockResolvedValueOnce(
			/** @type {any} */ ({
				ok: true,
				json: async () => ({ sent_count: 2, message: 'an sek@schule.de' })
			})
		);
		const store = useMahnwesenMail();

		await store.sendBulkOverdueMails({ ...auswahl, overrideEmail: 'sek@schule.de' });

		const body = JSON.parse(/** @type {any} */ (apiFetchMock.mock.calls[0][1]).body);
		expect(body.override_email).toBe('sek@schule.de');
		// Die Server-Meldung nennt den Empfänger; ein selbstgebautes „n versendet" nicht.
		expect(showToast).toHaveBeenCalledWith('an sek@schule.de', 'success');
	});

	it('schickt ohne Auswahl gar nicht erst los (leeres Feld hiesse serverseitig ALLE Klassen)', async () => {
		const store = useMahnwesenMail();

		await store.sendBulkOverdueMails({ klassen: [] });

		expect(apiFetch).not.toHaveBeenCalled();
		expect(showToastMock.mock.calls[0][1]).toBe('error');
	});

	it('zeigt bei einer Fehlerantwort (res.ok=false) die Server-Meldung als Fehler-Toast', async () => {
		apiFetchMock.mockResolvedValueOnce(
			/** @type {any} */ ({
				ok: false,
				json: async () => ({ error: 'Mahnwesen ist derzeit pausiert (Ferien)' })
			})
		);
		const store = useMahnwesenMail();

		await store.sendBulkOverdueMails({ klassen: ['5a'] });

		expect(showToast).toHaveBeenCalledWith('Mahnwesen ist derzeit pausiert (Ferien)', 'error');
	});

	it('fängt Netzwerkfehler ab und meldet sie als Fehler-Toast', async () => {
		apiFetchMock.mockRejectedValueOnce(new Error('boom'));
		const store = useMahnwesenMail();

		await store.sendBulkOverdueMails({ klassen: ['5a'] });

		expect(showToastMock).toHaveBeenCalledTimes(1);
		expect(showToastMock.mock.calls[0][1]).toBe('error');
	});
});

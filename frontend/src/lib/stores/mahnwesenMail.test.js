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

	it('POSTet an /api/mail/send-bulk-overdue und meldet die Anzahl per Erfolgs-Toast', async () => {
		apiFetchMock.mockResolvedValueOnce(
			/** @type {any} */ ({ ok: true, json: async () => ({ sent_count: 3 }) })
		);
		const store = useMahnwesenMail();

		await store.sendBulkOverdueMails();

		expect(apiFetch).toHaveBeenCalledWith('/api/mail/send-bulk-overdue', { method: 'POST' });
		expect(showToast).toHaveBeenCalledWith('3 Klassen-Mahnliste(n) versendet.', 'success');
	});

	it('zeigt bei einer Fehlerantwort (res.ok=false) die Server-Meldung als Fehler-Toast', async () => {
		apiFetchMock.mockResolvedValueOnce(
			/** @type {any} */ ({
				ok: false,
				json: async () => ({ error: 'Mahnwesen ist derzeit pausiert (Ferien)' })
			})
		);
		const store = useMahnwesenMail();

		await store.sendBulkOverdueMails();

		expect(showToast).toHaveBeenCalledWith('Mahnwesen ist derzeit pausiert (Ferien)', 'error');
	});

	it('fängt Netzwerkfehler ab und meldet sie als Fehler-Toast', async () => {
		apiFetchMock.mockRejectedValueOnce(new Error('boom'));
		const store = useMahnwesenMail();

		await store.sendBulkOverdueMails();

		expect(showToastMock).toHaveBeenCalledTimes(1);
		expect(showToastMock.mock.calls[0][1]).toBe('error');
	});
});

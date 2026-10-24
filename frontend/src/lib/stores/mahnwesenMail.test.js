import { describe, it, expect, vi, beforeEach } from 'vitest';

vi.mock('../apiFetch.js', async (original) => ({
	.../** @type {any} */ (await original()),
	apiFetch: vi.fn()
}));
vi.mock('../../inventur/lib/store.svelte.js', () => ({
	melde: vi.fn()
}));

import { apiFetch, FRIST_SAMMELVERSAND_MS } from '../apiFetch.js';
import { melde } from '../../inventur/lib/store.svelte.js';
import { useMahnwesenMail } from './mahnwesenMail.svelte.js';
import { serverAntwort } from '../serverAntwort.js';

const apiFetchMock = vi.mocked(apiFetch);
const showToastMock = vi.mocked(melde);

describe('useMahnwesenMail.sende', () => {
	beforeEach(() => {
		vi.clearAllMocks();
	});

	const auswahl = { klassen: ['5a', '6b'] };

	it('POSTet die gewählten Klassen an /api/mail/send-bulk-overdue', async () => {
		apiFetchMock.mockResolvedValueOnce(
			/** @type {any} */ ({ ok: true, json: async () => ({ sent_count: 3 }) })
		);
		const store = useMahnwesenMail();

		await store.sende(auswahl);

		expect(apiFetch).toHaveBeenCalledWith('/api/mail/send-bulk-overdue', {
			method: 'POST',
			// Je Klasse eine Mail: Mit der Vorgabe von 10 s gäbe die Oberfläche auf, während
			// der Server weiter versendet, und der zweite Klick schickte jede Liste doppelt.
			timeoutMs: FRIST_SAMMELVERSAND_MS,
			headers: { 'Content-Type': 'application/json' },
			body: JSON.stringify({ klassen: ['5a', '6b'], override_email: '' })
		});
		expect(melde).toHaveBeenCalledWith('Mahnlisten von 3 Klassen versendet', 'success');
	});

	it('reicht die Override-Adresse mit — sie entscheidet, wer die Listen bekommt', async () => {
		apiFetchMock.mockResolvedValueOnce(
			/** @type {any} */ ({
				ok: true,
				json: async () => ({ sent_count: 2, message: 'an sek@schule.de' })
			})
		);
		const store = useMahnwesenMail();

		await store.sende({ ...auswahl, overrideEmail: 'sek@schule.de' });

		const body = JSON.parse(/** @type {any} */ (apiFetchMock.mock.calls[0][1]).body);
		expect(body.override_email).toBe('sek@schule.de');
		// Die Server-Meldung nennt den Empfänger; ein selbstgebautes „n versendet" nicht.
		expect(melde).toHaveBeenCalledWith('an sek@schule.de', 'success');
	});

	it('schickt ohne Auswahl gar nicht erst los', async () => {
		const store = useMahnwesenMail();

		await store.sende({ klassen: [] });

		expect(apiFetch).not.toHaveBeenCalled();
		expect(showToastMock.mock.calls[0][1]).toBe('error');
	});

	it('zeigt bei einer Fehlerantwort (res.ok=false) die Server-Meldung als Fehler-Toast', async () => {
		apiFetchMock.mockResolvedValueOnce(
			serverAntwort(409, { error: 'Mahnwesen ist derzeit pausiert (Ferien)' })
		);
		const store = useMahnwesenMail();

		await store.sende({ klassen: ['5a'] });

		expect(melde).toHaveBeenCalledWith('Mahnwesen ist derzeit pausiert (Ferien)', 'error');
	});

	it('fängt Netzwerkfehler ab und meldet sie als Fehler-Toast', async () => {
		apiFetchMock.mockRejectedValueOnce(new Error('boom'));
		const store = useMahnwesenMail();

		await store.sende({ klassen: ['5a'] });

		expect(showToastMock).toHaveBeenCalledTimes(1);
		expect(showToastMock.mock.calls[0][1]).toBe('error');
	});
});

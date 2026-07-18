import { describe, it, expect, vi, beforeEach } from 'vitest';
import { apiFetch, extractApiError } from './apiFetch.js';

/** Baut eine minimale Response-Attrappe mit gegebenem Body/Status. */
function fakeResponse(body, status = 400) {
	return /** @type {any} */ ({ status, text: async () => body });
}

// extractApiError kapselt das Auspacken der einheitlichen Backend-Fehlerform
// {"error": "..."} — ohne diese Stelle landete roher JSON-Text in Toasts/Bannern.
describe('extractApiError', () => {
	it('zieht die Meldung aus dem {"error":...}-JSON', async () => {
		const msg = await extractApiError(fakeResponse('{"error":"Bitte zuerst zurückbuchen"}', 409));
		expect(msg).toBe('Bitte zuerst zurückbuchen');
	});

	it('akzeptiert auch {"message":...} als Fallback-Feld', async () => {
		const msg = await extractApiError(fakeResponse('{"message":"kaputt"}'));
		expect(msg).toBe('kaputt');
	});

	it('gibt Rohtext zurück, wenn der Body kein JSON ist', async () => {
		const msg = await extractApiError(fakeResponse('plain text error'));
		expect(msg).toBe('plain text error');
	});

	it('fällt bei leerem Body auf den HTTP-Status zurück', async () => {
		const msg = await extractApiError(fakeResponse('', 503));
		expect(msg).toBe('Fehler 503');
	});

	it('gibt bei JSON ohne error/message den Rohtext zurück', async () => {
		const msg = await extractApiError(fakeResponse('{"foo":"bar"}'));
		expect(msg).toBe('{"foo":"bar"}');
	});

	it('fängt einen fehlschlagenden text()-Aufruf ab', async () => {
		const res = /** @type {any} */ ({
			status: 500,
			text: async () => {
				throw new Error('stream kaputt');
			}
		});
		expect(await extractApiError(res)).toBe('Fehler 500');
	});
});

// Regressionstests für den CSRF-Bootstrap: Die erste Mutation direkt nach dem
// Login lief ohne csrf_token-Cookie in einen 403, weil das Token nur aus dem
// Cookie gelesen, aber nie initial beschafft wurde.
describe('apiFetch CSRF-Bootstrap', () => {
	beforeEach(() => {
		// jsdom: Cookies leeren
		document.cookie.split(';').forEach((c) => {
			const name = c.split('=')[0].trim();
			if (name) document.cookie = `${name}=; expires=Thu, 01 Jan 1970 00:00:00 GMT; path=/`;
		});
		vi.restoreAllMocks();
	});

	it('holt das Token vom Bootstrap-Endpoint, wenn das Cookie fehlt', async () => {
		const fetchMock = vi.spyOn(globalThis, 'fetch').mockImplementation(async (url) => {
			if (String(url) === '/api/csrf-token') {
				return /** @type {any} */ ({ ok: true, json: async () => ({ csrf_token: 'boot-token' }) });
			}
			return /** @type {any} */ ({ ok: true, json: async () => ({}) });
		});

		await apiFetch('/api/lieferanten', { method: 'POST', body: '{}' });

		expect(fetchMock).toHaveBeenCalledWith('/api/csrf-token', { credentials: 'include' });
		const mutationCall = fetchMock.mock.calls.find(([u]) => String(u) === '/api/lieferanten');
		expect(mutationCall?.[1]?.headers?.['X-CSRF-Token']).toBe('boot-token');
	});

	it('macht keinen Bootstrap, wenn das Cookie schon existiert', async () => {
		document.cookie = 'csrf_token=cookie-token; path=/';
		const fetchMock = vi
			.spyOn(globalThis, 'fetch')
			.mockImplementation(async () => /** @type {any} */ ({ ok: true, json: async () => ({}) }));

		await apiFetch('/api/lieferanten', { method: 'POST', body: '{}' });

		expect(fetchMock.mock.calls.some(([u]) => String(u) === '/api/csrf-token')).toBe(false);
		const mutationCall = fetchMock.mock.calls.find(([u]) => String(u) === '/api/lieferanten');
		expect(mutationCall?.[1]?.headers?.['X-CSRF-Token']).toBe('cookie-token');
	});

	it('GETs lösen keinen Bootstrap aus', async () => {
		const fetchMock = vi
			.spyOn(globalThis, 'fetch')
			.mockImplementation(async () => /** @type {any} */ ({ ok: true, json: async () => ({}) }));

		await apiFetch('/api/lieferanten');

		expect(fetchMock.mock.calls.some(([u]) => String(u) === '/api/csrf-token')).toBe(false);
	});
});

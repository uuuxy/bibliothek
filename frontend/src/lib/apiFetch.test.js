import { describe, it, expect, vi, beforeEach } from 'vitest';
import { apiFetch } from './apiFetch.js';

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
        const fetchMock = vi.spyOn(global, 'fetch').mockImplementation(async (url) => {
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
        const fetchMock = vi.spyOn(global, 'fetch').mockImplementation(async () =>
            /** @type {any} */ ({ ok: true, json: async () => ({}) })
        );

        await apiFetch('/api/lieferanten', { method: 'POST', body: '{}' });

        expect(fetchMock.mock.calls.some(([u]) => String(u) === '/api/csrf-token')).toBe(false);
        const mutationCall = fetchMock.mock.calls.find(([u]) => String(u) === '/api/lieferanten');
        expect(mutationCall?.[1]?.headers?.['X-CSRF-Token']).toBe('cookie-token');
    });

    it('GETs lösen keinen Bootstrap aus', async () => {
        const fetchMock = vi.spyOn(global, 'fetch').mockImplementation(async () =>
            /** @type {any} */ ({ ok: true, json: async () => ({}) })
        );

        await apiFetch('/api/lieferanten');

        expect(fetchMock.mock.calls.some(([u]) => String(u) === '/api/csrf-token')).toBe(false);
    });
});

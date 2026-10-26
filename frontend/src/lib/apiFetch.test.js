import { describe, it, expect, vi, beforeEach } from 'vitest';
import { apiFetch, leseApiFehler } from './apiFetch.js';
import { programmstand, KOPF_OBERFLAECHENSTAND } from './stores/programmstand.svelte.js';

/** Baut eine minimale Response-Attrappe mit gegebenem Body/Status. */
function fakeResponse(body, status = 400) {
	return /** @type {any} */ ({ status, text: async () => body });
}

// leseApiFehler kapselt das Auspacken der einheitlichen Backend-Fehlerform
// {"error": "..."} — ohne diese Stelle landete roher JSON-Text in Toasts/Bannern.
describe('leseApiFehler', () => {
	it('zieht die Meldung aus dem {"error":...}-JSON', async () => {
		const msg = await leseApiFehler(fakeResponse('{"error":"Bitte zuerst zurückbuchen"}', 409));
		expect(msg).toBe('Bitte zuerst zurückbuchen');
	});

	it('akzeptiert auch {"message":...} als Fallback-Feld', async () => {
		const msg = await leseApiFehler(fakeResponse('{"message":"kaputt"}'));
		expect(msg).toBe('kaputt');
	});

	it('gibt Rohtext zurück, wenn der Body kein JSON ist', async () => {
		const msg = await leseApiFehler(fakeResponse('plain text error'));
		expect(msg).toBe('plain text error');
	});

	it('fällt bei leerem Body auf den HTTP-Status zurück', async () => {
		const msg = await leseApiFehler(fakeResponse('', 503));
		expect(msg).toBe('Fehler 503');
	});

	it('gibt bei JSON ohne error/message den Rohtext zurück', async () => {
		const msg = await leseApiFehler(fakeResponse('{"foo":"bar"}'));
		expect(msg).toBe('{"foo":"bar"}');
	});

	it('fängt einen fehlschlagenden text()-Aufruf ab', async () => {
		const res = /** @type {any} */ ({
			status: 500,
			text: async () => {
				throw new Error('stream kaputt');
			}
		});
		expect(await leseApiFehler(res)).toBe('Fehler 500');
	});

	it('nimmt die Vorgabe des Aufrufers, wo die Antwort keine Meldung trägt', async () => {
		const vorgabe = 'Der Papierkorb konnte nicht geladen werden.';
		expect(await leseApiFehler(fakeResponse('', 502), vorgabe)).toBe(vorgabe);
		expect(await leseApiFehler(fakeResponse('{"error":"Kein Recht"}', 403), vorgabe)).toBe(
			'Kein Recht'
		);
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

// Jede Antwort nennt den Stand der Oberfläche am Server. apiFetch reicht ihn weiter; ohne das
// merkte ein offenes Fenster ein Update erst über die Live-Leitung.
describe('apiFetch und der Stand der Oberfläche', () => {
	beforeEach(() => {
		vi.restoreAllMocks();
		programmstand.eigener = '/assets/index-alt111.js';
		programmstand.server = '';
	});

	it('merkt sich den Stand aus dem Kopf der Antwort', async () => {
		vi.spyOn(globalThis, 'fetch').mockImplementation(
			async () =>
				new Response('[]', { headers: { [KOPF_OBERFLAECHENSTAND]: '/assets/index-neu222.js' } })
		);

		await apiFetch('/api/lieferanten');

		expect(programmstand.server).toBe('/assets/index-neu222.js');
		expect(programmstand.veraltet).toBe(true);
	});
});

describe('apiFetch error handling', () => {
	beforeEach(() => {
		vi.restoreAllMocks();
	});

	it('re-throws generic errors without modification', async () => {
		vi.spyOn(globalThis, 'fetch').mockImplementation(async () => {
			throw new Error('Some generic network failure');
		});

		await expect(apiFetch('/api/test')).rejects.toThrow('Some generic network failure');
	});

	it('transforms AbortError into a user-friendly timeout message', async () => {
		vi.spyOn(globalThis, 'fetch').mockImplementation(async () => {
			const error = new Error('The operation was aborted');
			error.name = 'AbortError';
			throw error;
		});

		await expect(apiFetch('/api/test')).rejects.toThrow(
			'Netzwerk-Timeout: Die Anfrage hat zu lange gedauert.'
		);
	});

	// Ohne Antwort wirft fetch einen TypeError mit dem Text des Browsers (gemessen in Chromium,
	// Firefox und WebKit); die Meldung zeigt sonst diesen englischen Text.
	it.each(['Failed to fetch', 'NetworkError when attempting to fetch resource.', 'Load failed'])(
		'übersetzt den Netzfehler „%s"',
		async (text) => {
			vi.spyOn(globalThis, 'fetch').mockRejectedValue(new TypeError(text));
			await expect(apiFetch('/api/test')).rejects.toThrow(/^Keine Verbindung zum Server\.$/);
		}
	);
});

describe('Sitzungsverlust-Haken', () => {
	/**
	 * Nach Ablauf der 12h-Sitzung pollte das Badge im 30-Sekunden-Takt in ein 401
	 * und die Seite wirkte kaputt statt abgemeldet. Der Haken meldet den ERSTEN
	 * unerwarteten 401 — aber nie für die Auth-Endpunkte selbst (dort ist 401 die
	 * reguläre Antwort) und nie für Öffentliches.
	 */
	it('feuert bei 401 einer API-Antwort, aber nicht für Auth- und öffentliche Pfade', async () => {
		const { apiFetch, registriereSitzungAbgelaufenHandler } = await import('./apiFetch.js');
		const handler = vi.fn();
		registriereSitzungAbgelaufenHandler(handler);
		globalThis.fetch = vi.fn(async () => new Response('', { status: 401 }));

		await apiFetch('/api/reservierungen/klassensatz/anzahl');
		expect(handler).toHaveBeenCalledTimes(1);

		await apiFetch('/api/auth/refresh', { method: 'POST' });
		await apiFetch('/api/public/opac/suche?q=x');
		expect(handler).toHaveBeenCalledTimes(1);

		globalThis.fetch = vi.fn(async () => new Response('', { status: 200 }));
		await apiFetch('/api/books');
		expect(handler).toHaveBeenCalledTimes(1);
	});

	// 423 heißt „gesperrt nach Inaktivität": Die Anmeldung gilt weiter. Wer das als
	// Sitzungsverlust behandelte, meldete den Arbeitsplatz ab, statt den Sperrbildschirm zu zeigen.
	it('meldet 423 als Sperre, nicht als Sitzungsverlust — und nicht für Auth-Pfade', async () => {
		const { apiFetch, registriereSitzungAbgelaufenHandler, registriereSitzungGesperrtHandler } =
			await import('./apiFetch.js');
		const abgelaufen = vi.fn();
		const gesperrt = vi.fn();
		registriereSitzungAbgelaufenHandler(abgelaufen);
		registriereSitzungGesperrtHandler(gesperrt);
		globalThis.fetch = vi.fn(async () => new Response('', { status: 423 }));

		await apiFetch('/api/schueler');
		expect(gesperrt).toHaveBeenCalledTimes(1);
		expect(abgelaufen).not.toHaveBeenCalled();

		// Der eigene Zustand wird gesperrt mit 423 beantwortet — das ist die Auskunft, kein Anlass.
		await apiFetch('/api/auth/me');
		expect(gesperrt).toHaveBeenCalledTimes(1);
	});
});

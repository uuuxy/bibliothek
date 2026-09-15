import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { authStore } from './authStore.svelte.js';

describe('authStore', () => {
	beforeEach(() => {
		// Reset state before each test
		authStore.handleLogout();
		vi.clearAllMocks();
	});

	it('should be initially logged out', () => {
		expect(authStore.isLoggedIn).toBe(false);
		expect(authStore.currentUser).toBeNull();
	});

	it('should login successfully and set isLoggedIn to true', async () => {
		// Mock global fetch
		// @ts-expect-error  Test-Double: Teilobjekt statt vollständiger Response
		globalThis.fetch = vi.fn(async () => ({
			ok: true,
			json: async () => ({ id: 1, rolle: 'mitarbeiter', vorname: 'Test' }),
			text: async () => ''
		}));

		// Mock EventSource to prevent network errors in test
		// @ts-expect-error  Test-Double: Stub statt vollständiger EventSource
		globalThis.EventSource = vi.fn(function () {
			return {
				addEventListener: vi.fn(),
				close: vi.fn()
			};
		});

		// Set login credentials
		authStore.loginEmail = 'test@example.com';
		authStore.loginPassword = 'password123';

		// Trigger login
		await authStore.handleLogin(null);

		// Assertions
		expect(authStore.isLoggedIn).toBe(true);
		expect(authStore.currentUser).toEqual({ id: 1, rolle: 'mitarbeiter', vorname: 'Test' });
		expect(authStore.loginEmail).toBe('');
		expect(authStore.loginPassword).toBe('');
	});
});

describe('authStore Session-Restore (Boot)', () => {
	beforeEach(() => {
		// @ts-expect-error  Test-Double: Teilobjekt statt vollständiger Response
		globalThis.fetch = vi.fn(async () => ({ ok: true, status: 200, json: async () => ({}) }));
		// @ts-expect-error  Test-Double: Stub statt vollständiger EventSource
		globalThis.EventSource = vi.fn(function () {
			return { addEventListener: vi.fn(), close: vi.fn() };
		});
		authStore.handleLogout();
		authStore.sessionChecked = false;
		// Der Abmelde-Merker des Resets gehört nicht zu diesen Fällen (siehe unten).
		localStorage.clear();
		vi.clearAllMocks();
	});
	afterEach(() => {
		authStore.stopSessionRefresh();
	});

	it('stellt die Session aus einem gültigen Cookie wieder her', async () => {
		// @ts-expect-error  Test-Double: Teilobjekt statt vollständiger Response
		globalThis.fetch = vi.fn(async () => ({
			ok: true,
			status: 200,
			json: async () => ({
				user_id: 'u1',
				rolle: 'admin',
				vorname: 'Peter',
				nachname: 'F',
				permissions: ['*']
			})
		}));

		await authStore.restoreSession();

		expect(globalThis.fetch).toHaveBeenCalledWith('/api/auth/me');
		expect(authStore.isLoggedIn).toBe(true);
		expect(authStore.currentUser?.rolle).toBe('admin');
		expect(authStore.sessionChecked).toBe(true);
	});

	it('bleibt bei 401 ausgeloggt, markiert den Check aber als erledigt', async () => {
		// @ts-expect-error  Test-Double: Teilobjekt statt vollständiger Response
		globalThis.fetch = vi.fn(async () => ({ ok: false, status: 401 }));

		await authStore.restoreSession();

		expect(authStore.isLoggedIn).toBe(false);
		expect(authStore.sessionChecked).toBe(true);
	});

	it('wertet Netzwerkfehler als ausgeloggt statt zu hängen', async () => {
		globalThis.fetch = vi.fn(async () => {
			throw new TypeError('Failed to fetch');
		});

		await authStore.restoreSession();

		expect(authStore.isLoggedIn).toBe(false);
		expect(authStore.sessionChecked).toBe(true);
	});

	// Raster-Durchgang 12.09.2026 über die Änderungen vom 11.09.: Seit 5cc80b89 antwortet
	// der Server bei einem Datenbank-Aussetzer mit 503 statt 401 — damit ein kurzer
	// Aussetzer nicht alle Arbeitsplätze abmeldet. Der Boot-Restore las aber nur `res.ok`:
	// Wer währenddessen neu lud, landete trotz gültigem Cookie am Login.
	it('hält die Sitzung, wenn der Boot-Restore in einen 503 läuft', async () => {
		vi.useFakeTimers();
		try {
			const antworten = [
				{ ok: false, status: 503 },
				{ ok: false, status: 503 },
				{
					ok: true,
					status: 200,
					json: async () => ({ user_id: 'u1', rolle: 'admin', vorname: 'Peter' })
				}
			];
			// @ts-expect-error  Test-Double: Teilobjekt statt vollständiger Response
			globalThis.fetch = vi.fn(async () => antworten.shift());

			const lauf = authStore.restoreSession();
			await vi.advanceTimersByTimeAsync(5000);
			await lauf;

			expect(globalThis.fetch).toHaveBeenCalledTimes(3);
			expect(authStore.isLoggedIn).toBe(true);
			expect(authStore.sessionChecked).toBe(true);
		} finally {
			authStore.stopSessionRefresh();
			vi.useRealTimers();
		}
	});

	// Die Gegenprobe zum Warten: Ein 401 ist die reguläre Antwort für „kein Cookie". Würde
	// er auch wiederholt, stünde jeder frische Browser sechs Sekunden vor dem Ladekreis.
	it('wartet bei 401 nicht, sondern zeigt sofort den Login', async () => {
		// @ts-expect-error  Test-Double: Teilobjekt statt vollständiger Response
		globalThis.fetch = vi.fn(async () => ({ ok: false, status: 401 }));

		await authStore.restoreSession();

		expect(globalThis.fetch).toHaveBeenCalledTimes(1);
		expect(authStore.isLoggedIn).toBe(false);
		expect(authStore.sessionChecked).toBe(true);
	});

	it('handleLogout invalidiert die Session auch serverseitig', () => {
		authStore.handleLogout();
		// keepalive: sonst stirbt die Anfrage mit der Seite (Reload/Tab zu direkt nach dem
		// Klick). Das Verhalten selbst prüft e2e/abmelden-tab-zu.spec.js.
		expect(globalThis.fetch).toHaveBeenCalledWith('/api/auth/logout', {
			method: 'POST',
			keepalive: true
		});
		expect(authStore.sessionChecked).toBe(true);
	});
});

describe('authStore Session-Refresh', () => {
	beforeEach(() => {
		authStore.handleLogout();
		vi.clearAllMocks();
		vi.useFakeTimers();
		// @ts-expect-error  Test-Double: Stub statt vollständiger EventSource
		globalThis.EventSource = vi.fn(function () {
			return { addEventListener: vi.fn(), close: vi.fn() };
		});
	});
	afterEach(() => {
		authStore.stopSessionRefresh();
		vi.useRealTimers();
	});

	it('ruft nach dem Login alle 30 Minuten /api/auth/refresh auf', async () => {
		// @ts-expect-error  Test-Double: Teilobjekt statt vollständiger Response
		globalThis.fetch = vi.fn(async () => ({
			ok: true,
			status: 200,
			json: async () => ({}),
			text: async () => ''
		}));
		authStore.loginEmail = 'test@example.com';
		authStore.loginPassword = 'pw';
		await authStore.handleLogin(null);
		// @ts-expect-error  globalThis.fetch ist hier der vi.fn-Mock, nicht die DOM-Signatur
		globalThis.fetch.mockClear();

		await vi.advanceTimersByTimeAsync(30 * 60 * 1000);
		expect(globalThis.fetch).toHaveBeenCalledWith('/api/auth/refresh', { method: 'POST' });

		await vi.advanceTimersByTimeAsync(30 * 60 * 1000);
		expect(globalThis.fetch).toHaveBeenCalledTimes(2);
	});

	it('loggt aus, wenn der Refresh 401 liefert (Session serverseitig tot)', async () => {
		// @ts-expect-error  Test-Double: Teilobjekt statt vollständiger Response
		globalThis.fetch = vi.fn(async () => ({
			ok: true,
			status: 200,
			json: async () => ({}),
			text: async () => ''
		}));
		authStore.loginEmail = 'test@example.com';
		authStore.loginPassword = 'pw';
		await authStore.handleLogin(null);

		// @ts-expect-error  Test-Double: Teilobjekt statt vollständiger Response
		globalThis.fetch = vi.fn(async () => ({ ok: false, status: 401 }));
		await vi.advanceTimersByTimeAsync(30 * 60 * 1000);

		expect(authStore.isLoggedIn).toBe(false);
		// Nach dem Logout darf kein weiterer Refresh mehr feuern
		// @ts-expect-error  globalThis.fetch ist hier der vi.fn-Mock, nicht die DOM-Signatur
		globalThis.fetch.mockClear();
		await vi.advanceTimersByTimeAsync(60 * 60 * 1000);
		expect(globalThis.fetch).not.toHaveBeenCalled();
	});

	it('überlebt Netzwerkfehler ohne Logout (offline ≠ abgemeldet)', async () => {
		// @ts-expect-error  Test-Double: Teilobjekt statt vollständiger Response
		globalThis.fetch = vi.fn(async () => ({
			ok: true,
			status: 200,
			json: async () => ({}),
			text: async () => ''
		}));
		authStore.loginEmail = 'test@example.com';
		authStore.loginPassword = 'pw';
		await authStore.handleLogin(null);

		globalThis.fetch = vi.fn(async () => {
			throw new TypeError('Failed to fetch');
		});
		await vi.advanceTimersByTimeAsync(30 * 60 * 1000);

		expect(authStore.isLoggedIn).toBe(true);
	});
});

// Die Login-Meldung verschwand nach vier Sekunden — auch „Zugang beantragt — die
// Bibliothek muss ihn noch freischalten" (403). Wer sie nicht zu Ende gelesen hat,
// tippt das Passwort noch einmal. Ein 403 heißt: Zugangsdaten richtig, Wiederholen
// bringt nichts — die Meldung bleibt, bis der nächste Versuch sie ersetzt.
describe('authStore Login-Meldung', () => {
	beforeEach(() => {
		authStore.handleLogout();
		vi.clearAllMocks();
		vi.useFakeTimers();
	});
	afterEach(() => vi.useRealTimers());

	/** @param {number} status @param {string} error */
	async function loginMit(status, error) {
		// @ts-expect-error  Test-Double: Teilobjekt statt vollständiger Response
		globalThis.fetch = vi.fn(async () => ({
			ok: false,
			status,
			json: async () => ({ error }),
			text: async () => ''
		}));
		authStore.loginEmail = 'lehrkraft@schule.de';
		authStore.loginPassword = 'pw';
		await authStore.handleLogin(null);
	}

	it('401 (falsches Passwort): Meldung räumt sich nach vier Sekunden weg', async () => {
		await loginMit(401, 'invalid email or password');
		expect(authStore.loginError).toBe('invalid email or password');
		await vi.advanceTimersByTimeAsync(4000);
		expect(authStore.loginError).toBeNull();
	});

	it('403 (Zugang beantragt): Meldung bleibt stehen', async () => {
		await loginMit(403, 'Zugang beantragt — die Bibliothek muss ihn noch freischalten');
		await vi.advanceTimersByTimeAsync(60_000);
		expect(authStore.loginError).toBe(
			'Zugang beantragt — die Bibliothek muss ihn noch freischalten'
		);
	});
});

// Abmelden ohne Antwort des Servers (14.09.2026): Das Löschcookie setzt nur die Antwort des
// Servers. Ohne Netz oder bei 502 vom Proxy blieb das Cookie im Browser, und das nächste Neuladen
// meldete die vorige Person wieder an. Den Weg im Browser prüft e2e/abmelden-ohne-antwort.spec.js.
describe('authStore: Abmeldung, die den Server nicht erreicht', () => {
	const MERKER = 'bibliothek.abmeldungAusstehend';
	/** Wartet, bis die nicht abgewartete Abmelde-Anfrage durch ist. */
	const warteAufAbmeldung = () => new Promise((fertig) => setTimeout(fertig, 0));

	beforeEach(() => {
		// @ts-expect-error  Test-Double: Stub statt vollständiger EventSource
		globalThis.EventSource = vi.fn(function () {
			return { addEventListener: vi.fn(), close: vi.fn() };
		});
		localStorage.clear();
		// Jeder Fall beginnt abgemeldet — ohne dass ein Reset selbst einen Merker hinterlässt.
		authStore.isLoggedIn = false;
		authStore.currentUser = null;
		authStore.sessionChecked = false;
	});
	afterEach(() => {
		authStore.stopSessionRefresh();
		localStorage.clear();
	});

	it('ohne Netz bleibt die Abmeldung vermerkt', async () => {
		globalThis.fetch = vi.fn(async () => {
			throw new TypeError('Failed to fetch');
		});
		authStore.handleLogout();
		await warteAufAbmeldung();
		expect(localStorage.getItem(MERKER)).toBe('1');
	});

	it('502 vom Proxy: das Löschcookie kam nicht an, die Abmeldung bleibt vermerkt', async () => {
		// @ts-expect-error  Test-Double: Teilobjekt statt vollständiger Response
		globalThis.fetch = vi.fn(async () => ({ ok: false, status: 502 }));
		authStore.handleLogout();
		await warteAufAbmeldung();
		expect(localStorage.getItem(MERKER)).toBe('1');
	});

	it('eine Antwort des Servers (200 oder 503) löscht den Merker', async () => {
		for (const status of [200, 503]) {
			// @ts-expect-error  Test-Double: Teilobjekt statt vollständiger Response
			globalThis.fetch = vi.fn(async () => ({ ok: status === 200, status }));
			authStore.handleLogout();
			await warteAufAbmeldung();
			expect(localStorage.getItem(MERKER), `Status ${status}`).toBeNull();
		}
	});

	// 503 heißt: Das Löschcookie kam an, der Widerruf am Server nicht (api/logout_handler.go).
	// Dieser Browser ist abgemeldet, die Sitzung selbst gilt bis zu ihrem Ablauf weiter.
	// Entschieden am 13.09.2026 (OFFEN.md 3.4): abmelden wie bisher, dazu ein sichtbarer Hinweis.
	it('503: abgemeldet, aber mit Hinweis, dass die Sperre am Server nicht bestätigt ist', async () => {
		// @ts-expect-error  Test-Double: Teilobjekt statt vollständiger Response
		globalThis.fetch = vi.fn(async () => ({ ok: false, status: 503 }));
		authStore.handleLogout();
		await warteAufAbmeldung();
		expect(authStore.isLoggedIn).toBe(false);
		expect(authStore.abmeldeHinweis).toMatch(/nicht bestätigt/);
	});

	it('200: kein Hinweis', async () => {
		authStore.abmeldeHinweis = null;
		// @ts-expect-error  Test-Double: Teilobjekt statt vollständiger Response
		globalThis.fetch = vi.fn(async () => ({ ok: true, status: 200 }));
		authStore.handleLogout();
		await warteAufAbmeldung();
		expect(authStore.abmeldeHinweis).toBeNull();
	});

	it('die nächste Anmeldung räumt den Hinweis weg', async () => {
		authStore.abmeldeHinweis = 'alt';
		// @ts-expect-error  Test-Double: Teilobjekt statt vollständiger Response
		globalThis.fetch = vi.fn(async () => ({
			ok: true,
			status: 200,
			json: async () => ({ user_id: 'u1', rolle: 'admin', vorname: 'Peter' }),
			text: async () => ''
		}));
		authStore.loginEmail = 'p@schule.invalid';
		authStore.loginPassword = 'x';
		await authStore.handleLogin(null);
		expect(authStore.isLoggedIn).toBe(true);
		expect(authStore.abmeldeHinweis).toBeNull();
	});

	it('auch die nachgeholte Abmeldung beim nächsten Start zeigt den Hinweis bei 503', async () => {
		localStorage.setItem(MERKER, '1');
		// @ts-expect-error  Test-Double: Teilobjekt statt vollständiger Response
		globalThis.fetch = vi.fn(async () => ({ ok: false, status: 503 }));
		await authStore.restoreSession();
		expect(localStorage.getItem(MERKER)).toBeNull();
		expect(authStore.abmeldeHinweis).toMatch(/nicht bestätigt/);
	});

	it('der nächste Start holt die Abmeldung nach und stellt keine Sitzung wieder her', async () => {
		localStorage.setItem(MERKER, '1');
		// @ts-expect-error  Test-Double: Teilobjekt statt vollständiger Response
		globalThis.fetch = vi.fn(async () => ({
			ok: true,
			status: 200,
			json: async () => ({ user_id: 'u1', rolle: 'admin', vorname: 'Peter' })
		}));

		await authStore.restoreSession();

		expect(globalThis.fetch).toHaveBeenCalledWith('/api/auth/logout', { method: 'POST' });
		expect(
			globalThis.fetch,
			'die alte Sitzung darf nicht wiederhergestellt werden'
		).not.toHaveBeenCalledWith('/api/auth/me');
		expect(authStore.isLoggedIn).toBe(false);
		expect(authStore.sessionChecked).toBe(true);
		expect(localStorage.getItem(MERKER)).toBeNull();
	});

	it('ist beim Start noch kein Netz da, bleibt der Merker für den nächsten Start', async () => {
		localStorage.setItem(MERKER, '1');
		globalThis.fetch = vi.fn(async () => {
			throw new TypeError('Failed to fetch');
		});

		await authStore.restoreSession();

		expect(authStore.isLoggedIn).toBe(false);
		expect(authStore.sessionChecked).toBe(true);
		expect(localStorage.getItem(MERKER)).toBe('1');
	});

	it('eine neue Anmeldung löscht den Merker', async () => {
		localStorage.setItem(MERKER, '1');
		// @ts-expect-error  Test-Double: Teilobjekt statt vollständiger Response
		globalThis.fetch = vi.fn(async () => ({
			ok: true,
			status: 200,
			json: async () => ({ id: 1, rolle: 'mitarbeiter', vorname: 'Test' }),
			text: async () => ''
		}));
		authStore.loginEmail = 'test@example.com';
		authStore.loginPassword = 'pw';

		await authStore.handleLogin(null);

		expect(authStore.isLoggedIn).toBe(true);
		expect(localStorage.getItem(MERKER)).toBeNull();
	});
});

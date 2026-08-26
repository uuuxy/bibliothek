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

	it('handleLogout invalidiert die Session auch serverseitig', () => {
		authStore.handleLogout();
		expect(globalThis.fetch).toHaveBeenCalledWith('/api/auth/logout', { method: 'POST' });
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

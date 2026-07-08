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
        // @ts-ignore
        global.fetch = vi.fn(async () => ({
            ok: true,
            json: async () => ({ id: 1, rolle: 'mitarbeiter', vorname: 'Test' }),
            text: async () => ''
        }));
        
        // Mock EventSource to prevent network errors in test
        // @ts-ignore
        global.EventSource = vi.fn(function() {
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
        // @ts-ignore
        global.fetch = vi.fn(async () => ({ ok: true, status: 200, json: async () => ({}) }));
        // @ts-ignore
        global.EventSource = vi.fn(function() {
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
        // @ts-ignore
        global.fetch = vi.fn(async () => ({
            ok: true, status: 200,
            json: async () => ({ user_id: 'u1', rolle: 'admin', vorname: 'Peter', nachname: 'F', permissions: ['*'] }),
        }));

        await authStore.restoreSession();

        expect(global.fetch).toHaveBeenCalledWith('/api/auth/me');
        expect(authStore.isLoggedIn).toBe(true);
        expect(authStore.currentUser?.rolle).toBe('admin');
        expect(authStore.sessionChecked).toBe(true);
    });

    it('bleibt bei 401 ausgeloggt, markiert den Check aber als erledigt', async () => {
        // @ts-ignore
        global.fetch = vi.fn(async () => ({ ok: false, status: 401 }));

        await authStore.restoreSession();

        expect(authStore.isLoggedIn).toBe(false);
        expect(authStore.sessionChecked).toBe(true);
    });

    it('wertet Netzwerkfehler als ausgeloggt statt zu hängen', async () => {
        // @ts-ignore
        global.fetch = vi.fn(async () => { throw new TypeError('Failed to fetch'); });

        await authStore.restoreSession();

        expect(authStore.isLoggedIn).toBe(false);
        expect(authStore.sessionChecked).toBe(true);
    });

    it('handleLogout invalidiert die Session auch serverseitig', () => {
        authStore.handleLogout();
        expect(global.fetch).toHaveBeenCalledWith('/api/auth/logout', { method: 'POST' });
        expect(authStore.sessionChecked).toBe(true);
    });
});

describe('authStore Session-Refresh', () => {
    beforeEach(() => {
        authStore.handleLogout();
        vi.clearAllMocks();
        vi.useFakeTimers();
        // @ts-ignore
        global.EventSource = vi.fn(function() {
            return { addEventListener: vi.fn(), close: vi.fn() };
        });
    });
    afterEach(() => {
        authStore.stopSessionRefresh();
        vi.useRealTimers();
    });

    it('ruft nach dem Login alle 30 Minuten /api/auth/refresh auf', async () => {
        // @ts-ignore
        global.fetch = vi.fn(async () => ({ ok: true, status: 200, json: async () => ({}), text: async () => '' }));
        authStore.loginEmail = 'test@example.com';
        authStore.loginPassword = 'pw';
        await authStore.handleLogin(null);
        // @ts-ignore
        global.fetch.mockClear();

        await vi.advanceTimersByTimeAsync(30 * 60 * 1000);
        expect(global.fetch).toHaveBeenCalledWith('/api/auth/refresh', { method: 'POST' });

        await vi.advanceTimersByTimeAsync(30 * 60 * 1000);
        expect(global.fetch).toHaveBeenCalledTimes(2);
    });

    it('loggt aus, wenn der Refresh 401 liefert (Session serverseitig tot)', async () => {
        // @ts-ignore
        global.fetch = vi.fn(async () => ({ ok: true, status: 200, json: async () => ({}), text: async () => '' }));
        authStore.loginEmail = 'test@example.com';
        authStore.loginPassword = 'pw';
        await authStore.handleLogin(null);

        // @ts-ignore
        global.fetch = vi.fn(async () => ({ ok: false, status: 401 }));
        await vi.advanceTimersByTimeAsync(30 * 60 * 1000);

        expect(authStore.isLoggedIn).toBe(false);
        // Nach dem Logout darf kein weiterer Refresh mehr feuern
        // @ts-ignore
        global.fetch.mockClear();
        await vi.advanceTimersByTimeAsync(60 * 60 * 1000);
        expect(global.fetch).not.toHaveBeenCalled();
    });

    it('überlebt Netzwerkfehler ohne Logout (offline ≠ abgemeldet)', async () => {
        // @ts-ignore
        global.fetch = vi.fn(async () => ({ ok: true, status: 200, json: async () => ({}), text: async () => '' }));
        authStore.loginEmail = 'test@example.com';
        authStore.loginPassword = 'pw';
        await authStore.handleLogin(null);

        // @ts-ignore
        global.fetch = vi.fn(async () => { throw new TypeError('Failed to fetch'); });
        await vi.advanceTimersByTimeAsync(30 * 60 * 1000);

        expect(authStore.isLoggedIn).toBe(true);
    });
});

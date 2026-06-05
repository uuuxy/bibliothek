import { describe, it, expect, vi, beforeEach } from 'vitest';
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

        // Set login barcode
        authStore.loginBarcode = 'L-123';
        
        // Trigger login
        await authStore.handleLogin(null);
        
        // Assertions
        expect(authStore.isLoggedIn).toBe(true);
        expect(authStore.currentUser).toEqual({ id: 1, rolle: 'mitarbeiter', vorname: 'Test' });
        expect(authStore.loginBarcode).toBe('');
    });
});

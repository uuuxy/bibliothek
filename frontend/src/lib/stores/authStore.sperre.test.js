import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { authStore, registriereGesperrterStartHandler } from './authStore.svelte.js';

const KONTO = { user_id: 'u1', email: 'theke@schule.example', rolle: 'admin', permissions: ['*'] };

/**
 * Stellt den Server je Pfad und merkt die Reihenfolge der Anfragen.
 * @param {Record<string, { status: number, body?: any }>} antworten
 * @returns {string[]} die angefragten Pfade, in der Reihenfolge ihres Eintreffens
 */
function serverAntwortet(antworten) {
	/** @type {string[]} */
	const pfade = [];
	// @ts-expect-error  Test-Double: Teilobjekt statt vollständiger Response
	globalThis.fetch = vi.fn(async (url) => {
		pfade.push(String(url));
		const a = antworten[String(url)] ?? { status: 200, body: {} };
		return {
			ok: a.status >= 200 && a.status < 300,
			status: a.status,
			json: async () => a.body ?? {}
		};
	});
	return pfade;
}

/**
 * Neuladen und neuer Tab während der Sperre nach Inaktivität: Der Server beantwortet den
 * Start mit 423, und die Anwendung darf nicht aufgehen — auch nicht für einen Augenblick.
 */
describe('authStore: Start während der Sperre nach Inaktivität', () => {
	/** @type {ReturnType<typeof vi.fn>} */
	let eventSource;
	/** @type {boolean[]} */
	let angemeldetBeimVerdecken;

	beforeEach(() => {
		eventSource = vi.fn(function () {
			return { addEventListener: vi.fn(), close: vi.fn() };
		});
		// @ts-expect-error  Test-Double: Stub statt vollständiger EventSource
		globalThis.EventSource = eventSource;
		localStorage.clear();
		authStore.isLoggedIn = false;
		authStore.currentUser = null;
		authStore.sessionChecked = false;
		angemeldetBeimVerdecken = [];
		registriereGesperrterStartHandler(() => angemeldetBeimVerdecken.push(authStore.isLoggedIn));
	});
	afterEach(() => {
		authStore.stopSessionRefresh();
		authStore.haltLiveAn();
		registriereGesperrterStartHandler(() => {});
		localStorage.clear();
	});

	it('423: angemeldet, aber nur mit der E-Mail-Adresse — keine Rechte, keine Live-Leitung', async () => {
		serverAntwortet({
			'/api/auth/me': { status: 423, body: { error: 'gesperrt', email: KONTO.email } }
		});

		await authStore.restoreSession();

		expect(authStore.sessionChecked).toBe(true);
		expect(
			authStore.isLoggedIn,
			'sonst stünde die Anmeldemaske statt des Sperrbildschirms da'
		).toBe(true);
		expect(authStore.currentUser).toEqual({ email: KONTO.email, permissions: [] });
		expect(
			eventSource,
			'hinter der Sperre beantwortet der Server /events nicht'
		).not.toHaveBeenCalled();
	});

	it('423: verdeckt wird, bevor die Anwendung als angemeldet gilt', async () => {
		serverAntwortet({ '/api/auth/me': { status: 423, body: { email: KONTO.email } } });
		await authStore.restoreSession();
		expect(
			angemeldetBeimVerdecken,
			'erst angemeldet, dann verdeckt: Die Anwendung würde einmal mit leeren Rechten gerendert'
		).toEqual([false]);
	});

	it('200: der Start fragt nur den Zustand der Anmeldung und verdeckt nichts', async () => {
		const pfade = serverAntwortet({ '/api/auth/me': { status: 200, body: KONTO } });
		await authStore.restoreSession();
		expect(pfade).toEqual(['/api/auth/me']);
		expect(authStore.currentUser?.rolle).toBe('admin');
		expect(angemeldetBeimVerdecken).toEqual([]);
	});
});

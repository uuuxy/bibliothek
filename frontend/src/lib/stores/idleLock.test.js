import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';

vi.mock('../apiFetch.js', async (importOriginal) => ({
	.../** @type {any} */ (await importOriginal()),
	apiFetch: vi.fn()
}));
vi.mock('../liveEvents.js', () => ({
	abonniere: vi.fn(() => vi.fn()),
	verbinde: vi.fn(),
	trenne: vi.fn()
}));

import { apiFetch } from '../apiFetch.js';
import { abonniere } from '../liveEvents.js';
import { IdleLock } from './idleLock.svelte.js';
import { authStore } from './authStore.svelte.js';
import { omniboxStore } from './omnibox.svelte.js';

/**
 * Der Inaktivitäts-Wächter (A4): Die Theke muss nach der kurzen Frist leer sein, die
 * Sperre nach der langen kommen, echte Bedienung muss beide Uhren neu stellen, und die
 * Wiederanmeldung darf nur mit gültigem Passwort entsperren. Alles mit gestellter Uhr —
 * die Fristen sind Minuten, kein Test wartet fünf davon.
 */
describe('idleLock', () => {
	/** @type {IdleLock} */
	let lock;

	beforeEach(() => {
		vi.useFakeTimers();
		vi.mocked(apiFetch).mockReset();
		vi.mocked(abonniere).mockClear();
		lock = new IdleLock();
		lock.thekeLeerenMinuten = 1;
		lock.sperreMinuten = 3;
		omniboxStore.activeStudent = { id: 's1', vorname: 'Mia' };
		omniboxStore.queryVal = 'Mü';
		authStore.currentUser = { email: 'theke@schule.example', rolle: 'mitarbeiter' };
	});

	afterEach(() => {
		lock.stop();
		vi.useRealTimers();
	});

	it('leert die Theke nach der kurzen Frist, sperrt nach der langen', () => {
		lock.start();
		vi.advanceTimersByTime(59_000);
		expect(omniboxStore.activeStudent).not.toBeNull();
		vi.advanceTimersByTime(2_000);
		expect(omniboxStore.activeStudent).toBeNull();
		expect(omniboxStore.queryVal).toBe('');
		expect(lock.gesperrt).toBe(false);
		vi.advanceTimersByTime(2 * 60_000);
		expect(lock.gesperrt).toBe(true);
	});

	it('echte Bedienung stellt beide Uhren neu (Scanner = Tastatur)', () => {
		lock.start();
		for (let i = 0; i < 5; i++) {
			vi.advanceTimersByTime(50_000);
			window.dispatchEvent(new KeyboardEvent('keydown', { key: '1' }));
		}
		// 250 s vergangen, aber nie 60 s am Stück: Theke steht noch, keine Sperre.
		expect(omniboxStore.activeStudent).not.toBeNull();
		expect(lock.gesperrt).toBe(false);
	});

	it('0 = aus: ohne Fristen passiert nichts', () => {
		lock.thekeLeerenMinuten = 0;
		lock.sperreMinuten = 0;
		lock.start();
		vi.advanceTimersByTime(60 * 60_000);
		expect(omniboxStore.activeStudent).not.toBeNull();
		expect(lock.gesperrt).toBe(false);
	});

	it('sperren räumt die Theke mit ab; Bedienung im gesperrten Zustand entsperrt NICHT', () => {
		lock.start();
		vi.advanceTimersByTime(3 * 60_000 + 10);
		expect(lock.gesperrt).toBe(true);
		expect(omniboxStore.activeStudent).toBeNull();
		window.dispatchEvent(new MouseEvent('pointerdown'));
		vi.advanceTimersByTime(10 * 60_000);
		expect(lock.gesperrt).toBe(true);
	});

	it('entsperrt nur mit gültigem Passwort — gegen /login, nicht /api', async () => {
		lock.start();
		vi.advanceTimersByTime(3 * 60_000 + 10);
		expect(lock.gesperrt).toBe(true);

		vi.mocked(apiFetch).mockResolvedValueOnce(/** @type {any} */ ({ ok: false, status: 401 }));
		expect(await lock.entsperren('falsch')).toBe(false);
		expect(lock.gesperrt).toBe(true);
		expect(lock.entsperrFehler).toMatch(/Passwort falsch/);

		vi.mocked(apiFetch).mockResolvedValueOnce(/** @type {any} */ ({ ok: true, status: 200 }));
		expect(await lock.entsperren('richtig')).toBe(true);
		expect(lock.gesperrt).toBe(false);
		const [url, opts] = vi.mocked(apiFetch).mock.calls[1];
		expect(url).toBe('/login');
		expect(JSON.parse(String(opts?.body))).toEqual({
			email: 'theke@schule.example',
			password: 'richtig'
		});
		// Nach dem Entsperren läuft die Uhr wieder von vorn.
		vi.advanceTimersByTime(3 * 60_000 + 10);
		expect(lock.gesperrt).toBe(true);
	});

	it('Theke leeren stoppt den Kamera-Scanner — er würde sonst hinter der Sperre weiterbuchen', () => {
		const stop = vi.fn(() => Promise.resolve());
		omniboxStore.cameraScanner = /** @type {any} */ ({ stop });
		omniboxStore.showCamera = true;
		lock.start();
		vi.advanceTimersByTime(60_000 + 10);
		expect(stop).toHaveBeenCalledTimes(1);
		expect(omniboxStore.cameraScanner).toBeNull();
		expect(omniboxStore.showCamera).toBe(false);
	});

	it('lädt die Fristen neu, wenn der Server sitzungsfristen signalisiert', async () => {
		lock.start();
		const abo = vi.mocked(abonniere).mock.calls.find(([name]) => name === 'sitzungsfristen');
		expect(abo, 'start() muss das SSE-Ereignis sitzungsfristen abonnieren').toBeDefined();

		vi.mocked(apiFetch).mockResolvedValueOnce(
			/** @type {any} */ ({
				ok: true,
				json: async () => ({ theke_leeren_minuten: 7, sperre_minuten: 0 })
			})
		);
		await abo?.[1](/** @type {any} */ ({}));
		expect(vi.mocked(apiFetch).mock.calls[0]?.[0]).toBe('/api/einstellungen/sitzung');
		expect(lock.thekeLeerenMinuten).toBe(7);
		expect(lock.sperreMinuten).toBe(0);
	});

	it('stop meldet das Fristen-Abo ab — sonst lädt jeder Logout-Login-Zyklus doppelt', () => {
		lock.start();
		const stelle = vi.mocked(abonniere).mock.calls.findIndex(([n]) => n === 'sitzungsfristen');
		const abmelden = vi.mocked(abonniere).mock.results[stelle]?.value;
		lock.stop();
		expect(abmelden).toHaveBeenCalledTimes(1);
	});

	it('stop hebt Sperre und Uhren auf (Logout)', () => {
		lock.start();
		vi.advanceTimersByTime(3 * 60_000 + 10);
		expect(lock.gesperrt).toBe(true);
		lock.stop();
		expect(lock.gesperrt).toBe(false);
		omniboxStore.activeStudent = { id: 's2' };
		vi.advanceTimersByTime(60 * 60_000);
		expect(omniboxStore.activeStudent).not.toBeNull();
	});
});

import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';

vi.mock('../apiFetch.js', async (importOriginal) => ({
	.../** @type {any} */ (await importOriginal()),
	apiFetch: vi.fn()
}));
const netz = vi.hoisted(() => ({
	isOffline: false,
	pendingCount: 0,
	startSync: /** @type {() => Promise<void>} */ (async () => {})
}));
vi.mock('./offlineSync.svelte.js', () => ({ offlineSync: netz }));
const rueckkehr = vi.hoisted(() => /** @type {Set<() => void>} */ (new Set()));
vi.mock('./netzLage.svelte.js', () => ({
	netzLage: {
		beiRueckkehr: (/** @type {() => void} */ h) => {
			rueckkehr.add(h);
			return () => rueckkehr.delete(h);
		}
	}
}));
vi.mock('../liveEvents.js', () => ({
	abonniere: vi.fn(() => vi.fn()),
	verbinde: vi.fn(),
	trenne: vi.fn()
}));

import { apiFetch } from '../apiFetch.js';
import { trenne } from '../liveEvents.js';
import { IdleLock } from './idleLock.svelte.js';
import { authStore } from './authStore.svelte.js';
import { omniboxStore } from './omnibox.svelte.js';

const SPERRE = 'bibliothek.sperre';
const AKTIVITAET = 'bibliothek.aktivitaet';
const KONTO = { email: 'theke@schule.example', rolle: 'mitarbeiter', permissions: [] };
const SPERRBAR = { '/api/auth/sperren': { status: 200, body: { gesperrt: true } } };
const NICHT_SPERRBAR = { '/api/auth/sperren': { status: 200, body: { gesperrt: false } } };

/**
 * Antwort des Servers je Pfad; ohne Eintrag 200 ohne Inhalt.
 * @param {Record<string, { status: number, body?: any }>} antworten
 */
function serverAntwortet(antworten) {
	vi.mocked(apiFetch).mockImplementation(async (url) => {
		const a = antworten[String(url)] ?? { status: 200, body: {} };
		return /** @type {any} */ ({
			ok: a.status >= 200 && a.status < 300,
			status: a.status,
			json: async () => a.body ?? {}
		});
	});
}

const aufrufe = () => vi.mocked(apiFetch).mock.calls.map(([url]) => String(url));
const signal = () => (localStorage.getItem(SPERRE) ?? '').split(':')[0];

/**
 * Ein Signal aus einem anderen Fenster desselben Browsers.
 * @param {string} key @param {string} newValue
 */
function anderesFenster(key, newValue) {
	window.dispatchEvent(new StorageEvent('storage', { key, newValue }));
}

/**
 * Die Sperre nach Inaktivität gilt am Server und für alle Fenster eines Browsers. Der Server
 * selbst steht in auth/sperre_pg_test.go; hier steht, was das Fenster ihm sagt und glaubt.
 */
describe('idleLock: Sperre am Server', () => {
	/** @type {IdleLock} */
	let lock;
	/** @type {import('vitest').MockInstance} */
	let uebernimm;
	/** @type {import('vitest').MockInstance} */
	let abgelaufen;

	beforeEach(() => {
		vi.useFakeTimers();
		localStorage.clear();
		vi.mocked(apiFetch).mockReset();
		vi.mocked(trenne).mockClear();
		serverAntwortet(SPERRBAR);
		netz.isOffline = false;
		netz.pendingCount = 0;
		netz.startSync = async () => {};
		uebernimm = vi.spyOn(authStore, 'uebernimmKonto').mockImplementation(() => {});
		abgelaufen = vi.spyOn(authStore, 'sitzungAbgelaufen').mockImplementation(() => {});
		lock = new IdleLock();
		lock.thekeLeerenMinuten = 1;
		lock.sperreMinuten = 3;
		omniboxStore.activeStudent = { id: 's1', vorname: 'Mia' };
		authStore.currentUser = { ...KONTO };
	});

	afterEach(() => {
		lock.stop();
		uebernimm.mockRestore();
		abgelaufen.mockRestore();
		vi.useRealTimers();
	});

	/** Lässt die Frist ablaufen: verdeckt, und die Anfrage an den Server ist durch. */
	async function fristLaeuftAb() {
		lock.start();
		await vi.advanceTimersByTimeAsync(3 * 60_000 + 10);
		expect(lock.gesperrt).toBe(true);
	}

	it('sperrt nach der Frist am Server, sagt es den anderen Fenstern und beendet die Live-Leitung', async () => {
		await fristLaeuftAb();
		expect(aufrufe()).toContain('/api/auth/sperren');
		expect(signal()).toBe('gesperrt');
		expect(trenne).toHaveBeenCalled();
	});

	it('schickt erst die ohne Netz gescannten Vorgänge, dann die Sperre', async () => {
		/** @type {string[]} */
		const reihenfolge = [];
		netz.pendingCount = 2;
		netz.startSync = async () => {
			await new Promise((fertig) => setTimeout(fertig, 500));
			reihenfolge.push('warteschlange');
		};
		vi.mocked(apiFetch).mockImplementation(async (url) => {
			reihenfolge.push(String(url));
			return /** @type {any} */ ({ ok: true, status: 200, json: async () => ({ gesperrt: true }) });
		});
		lock.start();
		await vi.advanceTimersByTimeAsync(3 * 60_000 + 10);
		// Verdeckt ist sofort, am Server gesperrt erst nach der Warteschlange.
		expect(lock.gesperrt).toBe(true);
		expect(reihenfolge).toEqual([]);
		await vi.advanceTimersByTimeAsync(600);
		expect(reihenfolge).toEqual(['warteschlange', '/api/auth/sperren']);
	});

	it('sperrt nicht nachträglich, wenn während der Warteschlange aufgeschlossen wurde', async () => {
		netz.pendingCount = 1;
		netz.startSync = () => new Promise((fertig) => setTimeout(fertig, 5_000));
		serverAntwortet({ ...SPERRBAR, '/api/auth/entsperren': { status: 200, body: KONTO } });
		await fristLaeuftAb();
		expect(await lock.entsperren('richtig')).toBe(true);
		await vi.advanceTimersByTimeAsync(6_000);
		expect(aufrufe(), 'eine verspätete Sperre träfe jemanden bei der Arbeit').not.toContain(
			'/api/auth/sperren'
		);
	});

	it('aufschließen übernimmt das Konto vom Server und sagt es den anderen Fenstern', async () => {
		serverAntwortet({ ...SPERRBAR, '/api/auth/entsperren': { status: 200, body: KONTO } });
		await fristLaeuftAb();
		expect(await lock.entsperren('richtig')).toBe(true);
		expect(lock.gesperrt).toBe(false);
		expect(signal()).toBe('offen');
		expect(uebernimm).toHaveBeenCalledWith(KONTO);
	});

	it('401 beim Aufschließen, und die Anmeldung gibt es nicht mehr: zur Anmeldemaske statt „Passwort falsch"', async () => {
		serverAntwortet({
			...SPERRBAR,
			'/api/auth/entsperren': { status: 401 },
			'/api/auth/me': { status: 401 }
		});
		await fristLaeuftAb();
		expect(await lock.entsperren('richtig')).toBe(false);
		expect(abgelaufen).toHaveBeenCalledTimes(1);
		expect(lock.entsperrFehler).toBeNull();
	});

	it('423 auf eine Anfrage verdeckt das Fenster, ohne den Server erneut zu sperren', () => {
		lock.start();
		lock.vomServerGesperrt();
		expect(lock.gesperrt).toBe(true);
		expect(signal()).toBe('gesperrt');
		expect(aufrufe()).not.toContain('/api/auth/sperren');
	});

	it('der Start in eine gesperrte Anmeldung ist verdeckt, bevor der Wächter läuft', () => {
		lock.verdeckeVorDemStart();
		expect(lock.gesperrt, 'sonst würde die Anwendung einmal gerendert').toBe(true);
		lock.start();
		expect(lock.gesperrt).toBe(true);
		expect(trenne).toHaveBeenCalled();
		// Die Uhren ruhen: Auch nach der Frist fragt das Fenster den Server nicht nach einer Sperre.
		vi.advanceTimersByTime(10 * 60_000);
		expect(aufrufe()).not.toContain('/api/auth/sperren');
	});

	it('ein neu geladenes, gesperrtes Fenster geht mit auf, wenn nebenan aufgeschlossen wird', async () => {
		serverAntwortet({ '/api/auth/me': { status: 200, body: KONTO } });
		lock.verdeckeVorDemStart();
		lock.start();
		anderesFenster(SPERRE, `offen:${Date.now()}`);
		await vi.advanceTimersByTimeAsync(10);
		expect(lock.gesperrt).toBe(false);
		expect(uebernimm).toHaveBeenCalledWith(KONTO);
	});

	// Der Server sperrt nur, was sich auch ohne Mailserver wieder aufschließen lässt. Eine
	// Anmeldung ohne Prüfwert des Passworts verdeckt nur dieses Fenster.
	describe('eine Anmeldung, die der Server nicht sperren kann', () => {
		beforeEach(() =>
			serverAntwortet({ ...NICHT_SPERRBAR, '/api/auth/me': { status: 200, body: KONTO } })
		);

		it('verdeckt dieses Fenster und sagt den anderen nichts', async () => {
			await fristLaeuftAb();
			expect(signal(), 'am Server ist nichts gesperrt').toBe('');
		});

		it('ein falsches Passwort schließt nicht auf, obwohl der Server die Anmeldung als offen führt', async () => {
			serverAntwortet({
				...NICHT_SPERRBAR,
				'/api/auth/entsperren': { status: 401 },
				'/api/auth/me': { status: 200, body: KONTO }
			});
			await fristLaeuftAb();
			expect(await lock.entsperren('falsch')).toBe(false);
			expect(lock.gesperrt).toBe(true);
			expect(lock.entsperrFehler).toMatch(/Passwort falsch/);
			expect(uebernimm).not.toHaveBeenCalled();
		});

		it('weder der Blick ins Fenster noch ein Signal von nebenan schließen auf', async () => {
			await fristLaeuftAb();
			window.dispatchEvent(new Event('focus'));
			anderesFenster(SPERRE, `offen:${Date.now()}`);
			await vi.advanceTimersByTimeAsync(10);
			expect(lock.gesperrt, 'auf geht es nur mit dem Passwort').toBe(true);
			expect(uebernimm).not.toHaveBeenCalled();
		});
	});

	describe('nach der Rückkehr des Netzes', () => {
		const netzIstZurueck = () => [...rueckkehr].forEach((h) => h());

		it('geht eine Sperre, die den Server verfehlt hat, noch einmal an ihn', async () => {
			serverAntwortet({ '/api/auth/sperren': { status: 503 } });
			await fristLaeuftAb();
			expect(signal()).toBe('');

			serverAntwortet(SPERRBAR);
			vi.mocked(apiFetch).mockClear();
			netzIstZurueck();
			await vi.advanceTimersByTimeAsync(10);
			expect(aufrufe()).toEqual(['/api/auth/sperren']);
			expect(signal()).toBe('gesperrt');
		});

		it('fragt eine bestätigte Sperre nicht noch einmal an', async () => {
			await fristLaeuftAb();
			vi.mocked(apiFetch).mockClear();
			netzIstZurueck();
			await vi.advanceTimersByTimeAsync(10);
			expect(aufrufe()).toEqual([]);
		});
	});

	describe('mehrere Fenster eines Browsers', () => {
		it('Bedienung im anderen Fenster schiebt die Sperre hinaus, die eigene Theke leert sich trotzdem', async () => {
			lock.start();
			for (let i = 0; i < 6; i++) {
				await vi.advanceTimersByTimeAsync(40_000);
				anderesFenster(AKTIVITAET, String(Date.now()));
			}
			// Vier Minuten ohne Bedienung in diesem Fenster, im Browser nie mehr als 40 Sekunden.
			expect(lock.gesperrt, 'die Sperre träfe die Arbeit im anderen Fenster').toBe(false);
			expect(aufrufe()).not.toContain('/api/auth/sperren');
			expect(omniboxStore.activeStudent, 'an dieser Theke war über eine Minute niemand').toBeNull();
			await vi.advanceTimersByTimeAsync(3 * 60_000 + 10);
			expect(lock.gesperrt).toBe(true);
		});

		it('eigene Bedienung meldet sich den anderen Fenstern', () => {
			lock.start();
			vi.advanceTimersByTime(5_000);
			window.dispatchEvent(new KeyboardEvent('keydown', { key: '1' }));
			expect(localStorage.getItem(AKTIVITAET)).toBe(String(Date.now()));
		});

		it('sperrt das andere Fenster, verdeckt sich dieses mit — ohne zweite Anfrage', () => {
			lock.start();
			anderesFenster(SPERRE, `gesperrt:${Date.now()}`);
			expect(lock.gesperrt).toBe(true);
			expect(aufrufe()).not.toContain('/api/auth/sperren');
		});

		it('schließt das andere Fenster auf, fragt dieses den Server und geht mit auf', async () => {
			serverAntwortet({ '/api/auth/me': { status: 200, body: KONTO } });
			lock.start();
			anderesFenster(SPERRE, `gesperrt:${Date.now()}`);
			anderesFenster(SPERRE, `offen:${Date.now()}`);
			await vi.advanceTimersByTimeAsync(10);
			expect(lock.gesperrt).toBe(false);
			expect(uebernimm).toHaveBeenCalledWith(KONTO);
		});

		it('ein Signal „offen" ohne Aufschließen am Server ändert nichts', async () => {
			serverAntwortet({ '/api/auth/me': { status: 423, body: { email: KONTO.email } } });
			lock.start();
			anderesFenster(SPERRE, `gesperrt:${Date.now()}`);
			// Das Signal kann jeder auslösen, der vor dem Rechner sitzt (Entwicklerwerkzeuge).
			anderesFenster(SPERRE, `offen:${Date.now()}`);
			await vi.advanceTimersByTimeAsync(10);
			expect(lock.gesperrt, 'der Server führt die Anmeldung weiter als gesperrt').toBe(true);
			expect(uebernimm).not.toHaveBeenCalled();
		});

		it('meldet das andere Fenster ab, geht dieses zur Anmeldemaske', async () => {
			serverAntwortet({ '/api/auth/me': { status: 401 } });
			lock.start();
			anderesFenster(SPERRE, `gesperrt:${Date.now()}`);
			anderesFenster(SPERRE, `offen:${Date.now()}`);
			await vi.advanceTimersByTimeAsync(10);
			expect(abgelaufen).toHaveBeenCalledTimes(1);
		});
	});
});

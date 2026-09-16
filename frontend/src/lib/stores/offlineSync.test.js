import 'fake-indexeddb/auto';
import { describe, it, expect, vi, beforeEach } from 'vitest';

vi.mock('../apiFetch.js', () => ({
	apiClient: { post: vi.fn() }
}));
vi.mock('../../inventur/lib/store.svelte.js', () => ({ showToast: vi.fn() }));
vi.mock('../audio.js', () => ({
	playSoundSuccess: vi.fn(),
	playSoundError: vi.fn(),
	playSuccessBeep: vi.fn(),
	playErrorBeep: vi.fn()
}));

import { apiClient } from '../apiFetch.js';
import { enqueueOfflineAction, loadQueue, dequeueOfflineAction } from '../offlineQueue.js';
import { offlineSync } from './offlineSync.svelte.js';
import { showToast } from '../../inventur/lib/store.svelte.js';

/** Einträge in Format 2, wie schnappschuss() sie im Omnibox-Store baut. */
let zaehler = 0;
/** @param {string} barcode @param {string} schuelerId */
const ausleihe = (barcode, schuelerId) => ({
	id: crypto.randomUUID(),
	art: /** @type {const} */ ('ausleihe'),
	barcode,
	leser_id: schuelerId,
	gescannt_am: ++zaehler
});
/** @param {string} barcode */
const rueckgabe = (barcode) => ({
	id: crypto.randomUUID(),
	art: /** @type {const} */ ('rueckgabe'),
	barcode,
	leser_id: null,
	gescannt_am: ++zaehler
});

async function clearQueue() {
	for (const item of await loadQueue()) {
		await dequeueOfflineAction(item.id);
	}
}

/**
 * Antwortet wie die Nachbuch-Tuer: je Eintrag ein Ergebnis, gebaut AUS DEM GESENDETEN
 * RUMPF statt aus einer geratenen Reihenfolge. Die Warteschlange ordnet nach
 * Scan-Zeitpunkt, und zwei Eintraege derselben Millisekunde sind nicht geordnet — ein
 * Test, der die Reihenfolge raet, misst die falsche Zeile.
 *
 * @param {(eintrag: any) => any} jeEintrag Ergebnis-Objekt ohne `schluessel`
 */
function antworte(jeEintrag) {
	return /** @type {any} */ (
		async (/** @type {any} */ _pfad, /** @type {any} */ rumpf) => ({
			ok: true,
			json: async () => ({
				uhr_versatz_sekunden: 0,
				ergebnisse: rumpf.eintraege.map((/** @type {any} */ e) => ({
					schluessel: e.schluessel,
					...jeEintrag(e)
				}))
			})
		})
	);
}

/** Das haeufigste Ergebnis: gebucht wie gescannt. */
const wieGescannt = (/** @type {any} */ e) => ({
	ergebnis: e.absicht === 'ausleihe' ? 'ausgeliehen' : 'zurueckgegeben',
	daten: { type: e.absicht }
});

// Die Offline-Queue ist der riskante Teil des PWA-Verhaltens — bewusst als
// Unit-Test statt E2E (Service-Worker-Offline-Simulation ist CI-flaky).
describe('offlineQueue', () => {
	beforeEach(async () => {
		await clearQueue();
		vi.clearAllMocks();
	});

	it('reiht Aktionen mit eindeutigen Idempotenz-Keys ein und entfernt gezielt', async () => {
		await enqueueOfflineAction(rueckgabe('B-1'));
		await enqueueOfflineAction(ausleihe('B-2', 'schueler-1'));

		const q = await loadQueue();
		expect(q).toHaveLength(2);
		expect(new Set(q.map((i) => i.id)).size).toBe(2);

		const checkout = q.find((i) => i.art === 'ausleihe');
		expect(checkout?.leser_id).toBe('schueler-1');

		await dequeueOfflineAction(q[0].id);
		expect(await loadQueue()).toHaveLength(1);
	});
});

describe('offlineSync.startSync', () => {
	beforeEach(async () => {
		await clearQueue();
		vi.clearAllMocks();
	});

	// Seit dem 16.09.2026 geht der Sync an die NACHBUCH-Tuer. Die alte Stapel-Tuer war
	// nie die vorgesehene: Sie kennt weder den Scan-Zeitpunkt noch den Schluessel als
	// Buchungssperre noch einen offline gescannten Ausweis.
	it('sendet die Portion an die Nachbuch-Tuer und leert sie bei Erfolg', async () => {
		await enqueueOfflineAction(ausleihe('B-100', 'schueler-42'));
		vi.mocked(apiClient.post).mockImplementation(antworte(wieGescannt));

		await offlineSync.startSync();

		const [pfad, rumpf] = vi.mocked(apiClient.post).mock.calls[0];
		expect(pfad).toBe('/api/action/nachbuchen');
		// Die Sendezeit gehoert zur Portion, nicht zum Eintrag: Aus ihr rechnet der Server
		// den Uhrversatz dieses Rechners.
		expect(rumpf.gesendet_am).toMatch(/^\d{4}-\d{2}-\d{2}T/);
		expect(rumpf.eintraege).toHaveLength(1);
		expect(rumpf.eintraege[0]).toMatchObject({
			barcode: 'B-100',
			absicht: 'ausleihe',
			leser_id: 'schueler-42',
			schluessel: expect.any(String)
		});
		expect(rumpf.eintraege[0].gescannt_am).toMatch(/^\d{4}-\d{2}-\d{2}T/);
		expect(await loadQueue()).toHaveLength(0);
		expect(offlineSync.pendingCount).toBe(0);
	});

	it('wirft fachlich Abgelehntes (nicht_gebucht) aus der Queue statt endlos zu haengen', async () => {
		await enqueueOfflineAction(rueckgabe('B-KAPUTT'));
		vi.mocked(apiClient.post).mockImplementation(
			antworte(() => ({ ergebnis: 'nicht_gebucht', grund: 'Barcode nicht gefunden' }))
		);

		await offlineSync.startSync();
		expect(await loadQueue()).toHaveLength(0);
	});

	it('behält die Queue, wenn die Anfrage selbst scheitert (z. B. 502)', async () => {
		await enqueueOfflineAction(rueckgabe('B-200'));

		vi.mocked(apiClient.post).mockResolvedValue(/** @type {any} */ ({ ok: false, status: 502 }));

		await offlineSync.startSync();
		expect(await loadQueue()).toHaveLength(1);
		// 502 heißt „der Server kommt gleich wieder" — nichts anzusagen, die nächste Runde
		// läuft in einer Minute.
		expect(offlineSync.abgelehntMitStatus).toBeNull();
	});

	// Warten hilft nicht bei einer Antwort, die sich von selbst nicht ändert
	// (Rasterdurchgang 16.09.2026, OFFEN.md 5.19): 403, weil der gerade angemeldete Mensch
	// nicht buchen darf. Bis hierher endete die Runde genauso still wie bei 502 und lief
	// jede Minute erneut ins Leere; im Band stand weiter nur „noch nicht im System".
	it('merkt sich eine Ablehnung, bei der Warten nicht hilft (403)', async () => {
		await enqueueOfflineAction(rueckgabe('B-201'));

		vi.mocked(apiClient.post).mockResolvedValue(/** @type {any} */ ({ ok: false, status: 403 }));

		await offlineSync.startSync();
		expect(await loadQueue(), 'die Vorgänge bleiben gespeichert').toHaveLength(1);
		expect(offlineSync.abgelehntMitStatus).toBe(403);
	});

	it('vergisst die Ablehnung, sobald ein Stapel wieder durchgeht', async () => {
		await enqueueOfflineAction(rueckgabe('B-202'));
		vi.mocked(apiClient.post).mockResolvedValue(/** @type {any} */ ({ ok: false, status: 403 }));
		await offlineSync.startSync();
		expect(offlineSync.abgelehntMitStatus).toBe(403);

		vi.mocked(apiClient.post).mockResolvedValue(
			/** @type {any} */ ({ ok: true, json: async () => ({ ergebnisse: [] }) })
		);
		await offlineSync.startSync();
		expect(offlineSync.abgelehntMitStatus).toBeNull();
	});
});

// Was der Server ABLEHNT, darf nicht spurlos verschwinden (Rasterdurchgang 06.09.2026).
//
// Dass ein 4xx nicht ewig wiederholt wird, ist richtig — der Server hat fachlich
// entschieden. Falsch war, dass es niemand erfuhr: Der Eintrag flog aus der
// Warteschlange, der Zähler ging auf 0, und der Erfolgston lief. Eine Klasse gibt 18
// Bücher offline zurück, vier werden abgelehnt (abgeriebenes Etikett, gesperrter Schüler,
// falscher Zustand) — die Ausleihen bleiben offen und laufen ins Mahnwesen.
describe('offlineSync: abgelehnte Vorgänge', () => {
	beforeEach(async () => {
		await clearQueue();
		vi.clearAllMocks();
	});

	it('meldet abgelehnte Scans mit Barcode, statt sie still zu löschen', async () => {
		await enqueueOfflineAction(rueckgabe('B-10234'));
		await enqueueOfflineAction(rueckgabe('B-10243')); // Etikett abgerieben, gibt es nicht
		// Die Antwort wird aus dem Payload gebaut: Die Reihenfolge in der Warteschlange
		// hängt am Zeitstempel, und zwei Einträge derselben Millisekunde sind nicht
		// geordnet. Ein Test, der die Reihenfolge rät, misst die falsche Zeile.
		vi.mocked(apiClient.post).mockImplementation(
			antworte((e) =>
				e.barcode === 'B-10243'
					? { ergebnis: 'nicht_gebucht', grund: 'Barcode nicht gefunden' }
					: wieGescannt(e)
			)
		);

		await offlineSync.startSync();

		expect(await loadQueue(), 'die Warteschlange ist geleert').toHaveLength(0);
		const meldungen = vi.mocked(showToast).mock.calls.map((c) => String(c[0]));
		expect(meldungen.join(' | ')).toContain('B-10243');
		expect(meldungen.join(' | '), 'der angenommene Scan gehört nicht in die Warnung').not.toContain(
			'B-10234'
		);
	});

	it('schweigt, wenn alles angenommen wurde (Gegenprobe)', async () => {
		await enqueueOfflineAction(rueckgabe('B-10234'));
		vi.mocked(apiClient.post).mockImplementation(antworte(wieGescannt));
		await offlineSync.startSync();
		expect(vi.mocked(showToast)).not.toHaveBeenCalled();
	});
});

// Eine nachgesendete Ausleihe trägt ihre Person als active_leser_id mit (OFFEN.md 2.2,
// Commit 4). Seit Migration 125 ist das EIN Feld — vorher schickte der Payload-Bauer für eine
// Lehrkraft gar nichts, und der Server buchte aus der Ausleihe still eine Rückgabe.
describe('offlineSync: die Ausleihe trägt ihren Leser mit', () => {
	beforeEach(async () => {
		await clearQueue();
		vi.clearAllMocks();
	});

	/** @param {string | null} leserID */
	async function sendeAusleiheMit(leserID) {
		await enqueueOfflineAction({
			id: crypto.randomUUID(),
			art: 'ausleihe',
			barcode: 'B-10236',
			leser_id: leserID,
			gescannt_am: ++zaehler
		});
		vi.mocked(apiClient.post).mockImplementation(antworte(wieGescannt));
		await offlineSync.startSync();
		return vi.mocked(apiClient.post).mock.calls[0][1].eintraege[0];
	}

	it('schickt leser_id — auch für einen Kollegen', async () => {
		const e = await sendeAusleiheMit('leser-3');
		expect(e.leser_id, 'ohne Person bucht der Server eine Rückgabe').toBe('leser-3');
	});

	// Gegenprobe: Ohne Person darf das Feld NICHT mitgehen — sonst würde eine Rückgabe zur
	// Ausleihe an niemanden.
	it('schickt ohne Person kein Feld', async () => {
		const e = await sendeAusleiheMit(null);
		expect(e.leser_id).toBeUndefined();
	});
});

// Erledigt ist nur, was der Server WIE GESCANNT gebucht hat (OFFEN.md 2.2, Commit 6). Bis
// zum 15.09.2026 galt als erledigt: success, jeder 4xx außer 429 und „Server nannte den Index
// nicht"; der Antworttyp wurde nie mit der Absicht verglichen, und ein liegengebliebener
// Eintrag (5xx, 429) wurde ohne Pause sofort erneut gesendet.
describe('offlineSync: erledigt nur, was wie gescannt gebucht wurde', () => {
	beforeEach(async () => {
		await clearQueue();
		vi.clearAllMocks();
	});

	it('Schweigen des Servers (kein Ergebnis zum Index) ist kein Erfolg', async () => {
		await enqueueOfflineAction(rueckgabe('B-10234'));
		vi.mocked(apiClient.post).mockResolvedValue(
			/** @type {any} */ ({ ok: true, json: async () => ({ ergebnisse: [] }) })
		);
		await offlineSync.startSync();
		expect(await loadQueue(), 'der Eintrag bleibt, bis der Server ihn beantwortet').toHaveLength(1);
		expect(apiClient.post, 'keine Endlosschleife: die Runde endet').toHaveBeenCalledTimes(1);
	});

	it('Ausleihe gescannt, Rückgabe gebucht: ausgebucht UND gemeldet, mit Barcode und beiden Typen', async () => {
		await enqueueOfflineAction(ausleihe('B-10234', 'schueler-7'));
		vi.mocked(apiClient.post).mockImplementation(
			antworte(() => ({ ergebnis: 'zurueckgegeben', daten: { type: 'rueckgabe' } }))
		);
		await offlineSync.startSync();
		expect(await loadQueue(), 'endgueltig entschieden, also ausgebucht').toHaveLength(0);
		const meldungen = vi
			.mocked(showToast)
			.mock.calls.map((c) => String(c[0]))
			.join(' | ');
		expect(meldungen).toContain('B-10234');
		expect(meldungen).toMatch(/Ausleihe/);
		expect(meldungen).toMatch(/Rückgabe/);
	});

	it('wie gescannt gebucht: still ausgebucht', async () => {
		await enqueueOfflineAction(ausleihe('B-10234', 'schueler-7'));
		vi.mocked(apiClient.post).mockImplementation(antworte(wieGescannt));
		await offlineSync.startSync();
		expect(await loadQueue()).toHaveLength(0);
		expect(showToast).not.toHaveBeenCalled();
	});

	// „wiederholen" ist das einzige der neun Woerter, das NICHT endgueltig ist: Der Server
	// war nicht erreichbar, oder derselbe Schluessel wird gerade gebucht.
	it('„wiederholen" bleibt liegen, und die Runde endet, statt sofort erneut zu senden', async () => {
		await enqueueOfflineAction(rueckgabe('B-10234'));
		vi.mocked(apiClient.post)
			.mockImplementationOnce(
				antworte(() => ({ ergebnis: 'wiederholen', grund: 'Server nicht erreichbar' }))
			)
			.mockImplementation(antworte(wieGescannt));
		await offlineSync.startSync();
		expect(apiClient.post, 'ein Versuch je Runde').toHaveBeenCalledTimes(1);
		expect(await loadQueue(), 'der Eintrag wartet auf die nächste Runde').toHaveLength(1);
	});
});

// Der Offline-Scan mit geladenem Schüler ist eine AUSLEIHE, keine Rückgabe
// (Rasterdurchgang 06.09.2026). Bis dahin legte die Omnibox jeden Offline-Scan als
// „checkin" ab, und der Payload-Bauer schickte `active_leser_id` nur bei „checkout" —
// einem Typ, den niemand je einreihte. Der Server las das Schweigen als Rückgabe: Das
// Buch war schon draußen, die Rückgabe scheiterte, der Eintrag flog aus der Warteschlange.
describe('offlineSync: Ausleihe trägt den Schüler mit', () => {
	beforeEach(async () => {
		await clearQueue();
		vi.clearAllMocks();
	});

	it('schickt leser_id für eine offline gespeicherte Ausleihe', async () => {
		await enqueueOfflineAction(ausleihe('B-10234', 'schueler-7'));
		await enqueueOfflineAction(rueckgabe('B-10235'));
		vi.mocked(apiClient.post).mockImplementation(antworte(wieGescannt));

		await offlineSync.startSync();

		const eintraege = vi.mocked(apiClient.post).mock.calls[0][1].eintraege;
		const mitSchueler = eintraege.find((/** @type {any} */ e) => e.barcode === 'B-10234');
		const ohneSchueler = eintraege.find((/** @type {any} */ e) => e.barcode === 'B-10235');
		expect(mitSchueler.leser_id, 'ohne Schüler bucht der Server eine Rückgabe').toBe('schueler-7');
		expect(ohneSchueler.leser_id).toBeUndefined();
	});
});

// Was nur die Nachbuch-Tuer kann — und was der alte Stapel-Sync nie konnte.
describe('offlineSync: die Nachbuch-Tuer', () => {
	beforeEach(async () => {
		await clearQueue();
		vi.clearAllMocks();
	});

	// „umgebucht" heisst: Das Buch lag bei jemand anderem, wurde dort zurueckgenommen und
	// neu ausgeliehen. Ein Buch hat die Person gewechselt — das gehoert gesagt.
	it('meldet ein umgebuchtes Exemplar mit Barcode', async () => {
		await enqueueOfflineAction(ausleihe('B-10234', 'schueler-7'));
		vi.mocked(apiClient.post).mockImplementation(
			antworte(() => ({ ergebnis: 'umgebucht', daten: { type: 'ausleihe' } }))
		);

		await offlineSync.startSync();

		expect(await loadQueue(), 'endgueltig entschieden').toHaveLength(0);
		const meldungen = vi
			.mocked(showToast)
			.mock.calls.map((c) => String(c[0]))
			.join(' | ');
		expect(meldungen).toContain('B-10234');
		expect(meldungen).toMatch(/jemand anderem/);
	});

	// Die Uhr des Theken-Rechners wird oft in dem Moment korrigiert, in dem das Netz
	// zurueckkommt — also zwischen Scan und Versand. Der Server rechnet den Scan-Zeitpunkt
	// ueber den Versatz zur Sendezeit um; passt der Scan-Zeitpunkt nicht zur Sendezeit,
	// ist die Umrechnung falsch, und Frist und Mahnwesen rechnen ab dem falschen Tag.
	it('rechnet den Scan-Zeitpunkt aus dem Uhr-Anker, wenn die Wanduhr gesprungen ist', async () => {
		const jetztMono = Math.round(performance.now());
		await enqueueOfflineAction({
			id: crypto.randomUUID(),
			art: 'rueckgabe',
			barcode: 'B-10234',
			leser_id: null,
			// Die gespeicherte Wanduhr-Zeit ist um zwei Stunden falsch — so, wie sie eine
			// ungestellte Theken-Uhr liefert.
			gescannt_am: Date.now() - 2 * 60 * 60 * 1000,
			// Der Anker sagt die Wahrheit: Der Scan war vor 5 Sekunden, in DIESEM Seitenaufruf.
			mono: jetztMono - 5000,
			ursprung: Math.round(performance.timeOrigin ?? 0)
		});
		vi.mocked(apiClient.post).mockImplementation(antworte(wieGescannt));

		await offlineSync.startSync();

		const rumpf = vi.mocked(apiClient.post).mock.calls[0][1];
		const gescannt = new Date(rumpf.eintraege[0].gescannt_am).getTime();
		const gesendet = new Date(rumpf.gesendet_am).getTime();
		// Der Abstand zwischen Scan und Versand ist rund 5 Sekunden, nicht zwei Stunden.
		expect(gesendet - gescannt).toBeGreaterThanOrEqual(4000);
		expect(gesendet - gescannt).toBeLessThan(15000);
	});

	// Ohne Anker (Eintrag aus einem frueheren Seitenaufruf oder aus einer eingespielten
	// Sicherung) gilt der gespeicherte Wert — dort gibt es nichts Besseres, und Raten
	// waere schlechter als der ehrliche Zeitstempel.
	it('laesst den gespeicherten Zeitpunkt stehen, wenn kein Anker dabei ist', async () => {
		const vorhin = Date.now() - 90 * 60 * 1000;
		await enqueueOfflineAction({
			id: crypto.randomUUID(),
			art: 'rueckgabe',
			barcode: 'B-10235',
			leser_id: null,
			gescannt_am: vorhin
		});
		vi.mocked(apiClient.post).mockImplementation(antworte(wieGescannt));

		await offlineSync.startSync();

		const rumpf = vi.mocked(apiClient.post).mock.calls[0][1];
		expect(new Date(rumpf.eintraege[0].gescannt_am).getTime()).toBe(vorhin);
	});

	// Portion 25: Die Tuer nimmt bis zu 50, aber eine kleinere Portion laesst nach einem
	// Abbruch weniger erneut laufen.
	it('sendet hoechstens 25 Eintraege je Portion', async () => {
		for (let i = 0; i < 30; i++) await enqueueOfflineAction(rueckgabe(`B-${1000 + i}`));
		vi.mocked(apiClient.post).mockImplementation(antworte(wieGescannt));

		await offlineSync.startSync();

		expect(vi.mocked(apiClient.post).mock.calls[0][1].eintraege).toHaveLength(25);
		expect(await loadQueue(), 'die zweite Portion folgt in derselben Runde').toHaveLength(0);
	});
});

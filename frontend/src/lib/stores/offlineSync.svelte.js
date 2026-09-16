import {
	loadQueue,
	dequeueOfflineAction,
	enqueueOfflineAction,
	normalisiereEintrag
} from '../offlineQueue.js';
import { apiClient } from '../apiFetch.js';
import { playSoundSuccess } from '../audio.js';
import { showToast } from '../../inventur/lib/store.svelte.js';

// Baut die Portion fuer die Nachbuch-Tuer (POST /api/action/nachbuchen).
//
// Bis zum 16.09.2026 ging der Sync an /api/action/batch — die Tuer, die laut Entscheidung
// vom 13.09. nur noch "eine Version laenger" fuer Theken-Tabs mit altem Stand bestehen
// bleibt. Die Nachbuch-Tuer war gebaut, geroutet und getestet, und niemand rief sie auf.
// Sie ist die richtige, weil sie Dinge kann, die der Stapel nicht kann: den
// Scan-Zeitpunkt buchen, einen Schluessel genau einmal buchen, einen offline gescannten
// Ausweis aufloesen und jede Abweichung als Meldung festhalten.
//
// DIE UHR (Vorgabe aus dem Plan, Stufe 3 Commit 14): `gescannt_am` und `gesendet_am`
// muessen von DERSELBEN Uhr kommen, sonst rechnet der Server den Versatz falsch. Genau das
// ist hier kein Randfall: Die Uhr eines Theken-Rechners wird oft in dem Moment korrigiert,
// in dem das Netz zurueckkommt — also zwischen Scan und Versand.
//
// Deshalb wird der Scan-Zeitpunkt neu bestimmt, wenn der Eintrag aus DIESEM Seitenaufruf
// stammt: `performance.now()` laeuft gleichmaessig weiter und springt nicht. Aus dem
// Abstand seit dem Scan und der Wanduhr von JETZT ergibt sich ein Scan-Zeitpunkt, der zu
// `gesendet_am` passt, auch wenn die Wanduhr dazwischen gesprungen ist. Stammt der Eintrag
// aus einem frueheren Seitenaufruf (Neuladen, eingespielte Sicherung), gilt der
// gespeicherte Wert — dort gibt es nichts Besseres.
/**
 * @param {import('../offlineQueue.js').OfflineEintrag[]} batchItems
 * @returns {{ gesendet_am: string, eintraege: any[] }}
 */
function baueNachbuchPayload(batchItems) {
	const jetzt = Date.now();
	const mono = typeof performance !== 'undefined' ? Math.round(performance.now()) : null;
	const ursprung =
		typeof performance !== 'undefined' ? Math.round(performance.timeOrigin ?? 0) : null;

	return {
		gesendet_am: new Date(jetzt).toISOString(),
		eintraege: batchItems.map((item) => {
			let gescannt = item.gescannt_am;
			if (mono !== null && item.ursprung === ursprung && typeof item.mono === 'number') {
				gescannt = jetzt - (mono - item.mono);
			}
			/** @type {any} */
			const e = {
				schluessel: item.id,
				absicht: item.art,
				barcode: item.barcode,
				gescannt_am: new Date(gescannt).toISOString()
			};
			// Eine Ausleihe traegt ihre Person, eine Rueckgabe nicht — der Server findet
			// den Vorbesitzer selbst. `ausweis_barcode` kennt die Tuer bereits; gefuellt
			// wird es, sobald die Theke Ausweise offline annimmt.
			if (item.art === 'ausleihe' && item.leser_id) e.leser_id = item.leser_id;
			if (item.ausweis_barcode) e.ausweis_barcode = item.ausweis_barcode;
			return e;
		})
	};
}

// Endgueltige Ergebnisse: Der Eintrag ist erledigt und fliegt aus der Warteschlange.
// „wiederholen" ist das einzige, das NICHT endgueltig ist (Server nicht erreichbar, oder
// der Schluessel wird gerade gebucht) — dort endet die Runde.
const NACHBUCH_ENDGUELTIG = new Set([
	'ausgeliehen',
	'umgebucht',
	'bereits_ausgeliehen',
	'zurueckgegeben',
	'nur_reaktiviert',
	'nicht_gebucht',
	'veraltet',
	'bereits_gebucht'
]);

/** @type {Record<string, string>} */
const TYPNAME = { ausleihe: 'Ausleihe', rueckgabe: 'Rückgabe' };
/** @param {string | undefined} typ */
const nenne = (typ) => TYPNAME[typ ?? ''] ?? (typ ? `„${typ}“` : 'nichts');

// Erledigt ist nur, was der Server wirklich entschieden hat.
//
// Die Nachbuch-Tuer antwortet je Schluessel mit einem von neun Woertern. Acht davon sind
// endgueltig — gebucht, umgebucht, schon dagewesen, abgelehnt: In allen Faellen hat der
// Server den Fall abschliessend behandelt, und der Eintrag gehoert aus der Warteschlange.
// Das neunte, „wiederholen", heisst ausdruecklich das Gegenteil: Der Server war nicht
// erreichbar, oder derselbe Schluessel wird gerade gebucht. Dann bleibt der Eintrag liegen
// und die Runde endet, statt gegen dieselbe Wand zu laufen.
//
// Ein Schluessel, den der Server GAR NICHT beantwortet hat, bleibt ebenfalls liegen.
// Schweigen ist kein Erfolg — bis zum 15.09.2026 galt es als erledigt.
//
// Gemeldet wird, was ein Mensch wissen muss:
//   - `nicht_gebucht` und `veraltet` tragen ihren Grund; ohne die Meldung erfaehrt niemand,
//     dass vier von achtzehn Rueckgaben abgelehnt wurden, und die Ausleihen laufen ins
//     Mahnwesen bis zur Rechnung an die Eltern (Rasterdurchgang 06.09.2026).
//   - `umgebucht` heisst: Das Buch lag bei jemand anderem, wurde dort zurueckgenommen und
//     neu ausgeliehen. Ein Buch hat die Person gewechselt — das gehoert gesagt, auch wenn
//     der Server es zusaetzlich in seiner Meldungsliste festhaelt.
//   - Weicht die gebuchte Wirkung von der Absicht des Scans ab (als Ausleihe gescannt,
//     als Rueckgabe gebucht), wird das gemeldet. Derselbe Typvergleich wie bisher, und er
//     gilt auch fuer `bereits_gebucht`: Das ist die Wirkung des Online-Versands, der
//     damals durchkam, ohne dass die Theke die Antwort noch sah.
//   - `aufsicht_informieren` ist keine Meldung, sondern eine Aufgabe: Das Buch stand auf
//     einem Bescheid, der schon bei der Schulaufsicht liegt.
/**
 * @param {any} data
 * @param {import('../offlineQueue.js').OfflineEintrag[]} batchItems
 * @returns {Promise<{ pruefen: { barcode: string, meldung: string }[], weiter: boolean }>}
 */
async function verarbeiteNachbuchErgebnisse(data, batchItems) {
	/** @type {{ barcode: string, meldung: string }[]} */
	const pruefen = [];
	let weiter = true;

	const versatz = Number(data?.uhr_versatz_sekunden ?? 0);
	if (Math.abs(versatz) > 60) {
		// Kein Abbruch: Der Server rechnet den Versatz selbst heraus. Die Zeile sagt nur,
		// dass die Uhr dieses Rechners deutlich falsch geht — das gehoert in die Wartung.
		console.warn(`Offline-Sync: Uhr dieses Rechners weicht um ${versatz} s ab.`);
	}

	// Nachschlagewerk als schlichtes Objekt, nicht als Map: In einer `.svelte.js` ist eine
	// gewoehnliche Map verboten (svelte/prefer-svelte-reactivity), und reaktiv muss hier
	// nichts sein — das Ding lebt nur fuer die Dauer dieser Auswertung.
	/** @type {Record<string, any>} */
	const nachSchluessel = Object.create(null);
	for (const e of data?.ergebnisse ?? []) if (e?.schluessel) nachSchluessel[e.schluessel] = e;

	for (const item of batchItems) {
		const erg = nachSchluessel[item.id];
		if (!erg || !NACHBUCH_ENDGUELTIG.has(erg.ergebnis)) {
			weiter = false;
			continue;
		}

		if (erg.ergebnis === 'nicht_gebucht' || erg.ergebnis === 'veraltet') {
			pruefen.push({
				barcode: item.barcode,
				meldung: erg.grund ? `nicht gebucht (${erg.grund})` : 'nicht gebucht'
			});
		} else if (erg.ergebnis === 'umgebucht') {
			pruefen.push({
				barcode: item.barcode,
				meldung: 'lag bei jemand anderem — dort zurückgenommen und neu ausgeliehen'
			});
		} else {
			const gebucht = erg.daten?.type;
			if (gebucht && gebucht !== item.art) {
				pruefen.push({
					barcode: item.barcode,
					meldung: `als ${nenne(item.art)} gescannt, der Server buchte ${nenne(gebucht)}`
				});
			}
		}

		if (erg.aufsicht_informieren) {
			pruefen.push({ barcode: item.barcode, meldung: erg.aufsicht_informieren });
		}

		await dequeueOfflineAction(item.id);
	}
	return { pruefen, weiter };
}

async function exportQueueAsJSON() {
	const q = await loadQueue();
	if (q.length === 0) return;
	const blob = new Blob([JSON.stringify(q, null, 2)], { type: 'application/json' });
	const url = URL.createObjectURL(blob);
	const a = document.createElement('a');
	a.href = url;
	a.download = `offline_scans_backup_${new Date().toISOString().slice(0, 10)}.json`;
	document.body.appendChild(a);
	a.click();
	a.remove();
	URL.revokeObjectURL(url);
}

// Sagt dem Bediener, was er von Hand klären muss — mit Barcode und Grund je Eintrag. Das
// Muster steht im Haus schon fertig (useFehlbestand: „n gelöscht, m übersprungen").
/** @param {{ barcode: string, meldung: string }[]} pruefen */
function meldeZuPruefende(pruefen) {
	if (pruefen.length === 0) return;
	const liste = pruefen.map((p) => `„${p.barcode}“: ${p.meldung}`).join('; ');
	showToast(
		pruefen.length === 1
			? `Offline-Scan ${liste} — bitte von Hand prüfen.`
			: `${pruefen.length} Offline-Scans brauchen Prüfung: ${liste} — bitte von Hand prüfen.`,
		'error'
	);
}

function createOfflineSyncStore() {
	let pendingCount = $state(0);
	let isSyncing = $state(false);
	let isOffline = $state(typeof navigator !== 'undefined' ? !navigator.onLine : false);
	// Die Warteschlange ließ sich nicht lesen (Commit 5): Das Band sagt es, statt 0 zu zeigen.
	let warteschlangeFehler = $state(false);

	async function updateCount() {
		try {
			const q = await loadQueue();
			pendingCount = q.length;
			warteschlangeFehler = false;
		} catch {
			warteschlangeFehler = true;
		}
	}

	// Verschickt einen Batch und verarbeitet dessen Ergebnisse. Liefert false, wenn der
	// Sync abbrechen soll (kompletter Batch-Fehler wie 502, oder Netzwerkfehler).
	async function sendeBatch(payload, batchItems, queueLength) {
		try {
			const res = await apiClient.post('/api/action/nachbuchen', payload);

			if (!res.ok) {
				// Batch request failed completely (e.g. 502 Bad Gateway), stop syncing
				return false;
			}

			const data = await res.json();
			const { pruefen, weiter } = await verarbeiteNachbuchErgebnisse(data, batchItems);
			meldeZuPruefende(pruefen);
			await updateCount();
			if (!weiter) return false;

			// Network Jitter: 200-500 ms Pause vor dem nächsten Batch, damit mehrere
			// Geräte nach einer Offline-Phase nicht im Gleichtakt auf den Server laufen.
			//
			// Math.random() ist hier bewusst richtig und kein Sicherheitsmangel
			// (SonarQube javascript:S2245 meldet jede Verwendung): Das Ergebnis ist eine
			// Wartedauer, kein Geheimnis. Es schützt nichts, identifiziert nichts und ist
			// für niemanden von Vorteil, wenn er es vorhersagt. Eine kryptografische
			// Quelle brächte hier keinerlei Schutz — nur Aufwand.
			if (queueLength > 50) {
				const jitter = 200 + Math.random() * 300; // NOSONAR — Jitter, kein Sicherheitskontext (siehe Kommentar oben; S2245 ist hier ein False Positive)
				await new Promise((resolve) => setTimeout(resolve, jitter));
			}
			return true;
		} catch (err) {
			console.warn('Offline-Sync: Netzwerkfehler beim Batch-Versand:', err);
			return false;
		}
	}

	async function startSync() {
		if (isSyncing || !navigator.onLine) return;
		isSyncing = true;

		let syncedAny = false;

		while (navigator.onLine) {
			/** @type {import('../offlineQueue.js').OfflineEintrag[]} */
			let q;
			try {
				q = await loadQueue();
				warteschlangeFehler = false;
			} catch {
				warteschlangeFehler = true;
				break;
			}
			if (q.length === 0) break;

			// loadQueue liefert nach Scan-Zeitpunkt geordnet.
			//
			// Portion 25 (Vorgabe des Plans): Die Tuer nimmt bis zu 50, aber jede Portion
			// laeuft in EINER Transaktion je Eintrag, und eine kleinere Portion laesst nach
			// einem Abbruch weniger erneut laufen.
			const batchItems = q.slice(0, 25);
			const payload = baueNachbuchPayload(batchItems);

			const ok = await sendeBatch(payload, batchItems, q.length);
			if (!ok) break;
			syncedAny = true;
		}

		if (syncedAny && pendingCount === 0) {
			playSoundSuccess();
		}
		isSyncing = false;
	}

	/**
	 * Spielt eine Notfall-Sicherung zurück in die lokale Warteschlange; der Auto-Sync
	 * schiebt sie danach zum Server.
	 *
	 * item.id MUSS mitwandern: Diese ID ist der Idempotenz-Schlüssel, den der Server
	 * kennt (siehe baueNachbuchPayload und api/nachbuchen_schluessel.go). Ohne sie
	 * vergibt normalisiereEintrag eine frische UUID — und dieselbe Datei zweimal
	 * eingespielt würde jede Aktion ZWEIMAL ausführen. Bei zehn Kiosk-Rechnern mit
	 * einem gemeinsamen Sicherungsordner ist doppeltes Einspielen der Normalfall,
	 * nicht der Ausnahmefall: Zwei Admins, oder einer, der unsicher ist, ob er es
	 * schon getan hat. Mit dem Schlüssel ist der zweite Durchlauf wirkungslos.
	 *
	 * Sicherungen in Format 1 (vor dem 15.09.2026, `action_type`/`barcode_id`) und
	 * Format 2 (`art`/`barcode`, mit Scan-Zeitpunkt) werden gleich behandelt.
	 * @param {File} file
	 */
	async function importQueueFromJSON(file) {
		try {
			const text = await file.text();
			const items = JSON.parse(text);
			if (!Array.isArray(items)) throw new Error('Invalid format');

			let importedCount = 0;
			const promises = [];
			for (const roh of items) {
				const eintrag = normalisiereEintrag(roh);
				if (!eintrag) continue;
				promises.push(enqueueOfflineAction(eintrag));
				importedCount++;
			}
			await Promise.all(promises);

			await updateCount();
			startSync();
			return importedCount;
		} catch (e) {
			console.error(e);
			throw new Error('Fehler beim Einlesen der Backup-Datei.');
		}
	}

	// Der Dialog nennt den Ausweg, nicht nur die Gefahr: "Nicht ausschalten" ist an
	// einem Schulrechner keine Handlungsanweisung, die jemand bis Feierabend
	// durchhalten kann. Verlassen darf man sich darauf ohnehin nicht — beim
	// Herunterfahren von Windows bekommt der Browser oft keine Gelegenheit mehr,
	// diesen Dialog zu zeigen. Deshalb steht dieselbe Anweisung dauerhaft im Banner.
	function handleBeforeUnload(e) {
		if (pendingCount > 0) {
			e.preventDefault();
			const msg = `${pendingCount} Vorgang/Vorgänge sind nur auf diesem Rechner gespeichert. Bitte zuerst "Sicherung speichern" — danach darf der Rechner aus.`;
			e.returnValue = msg;
			return msg;
		}
	}

	// Einmalig je Seite: Jede Anmeldung ruft init() (hintergrundAbrufe.svelte.js); ohne
	// Sperre stapelten Logout→Login Listener und 30-s-Intervalle, und startSync() lief
	// mehrfach parallel gegen dieselbe Warteschlange.
	let initialisiert = false;
	function init() {
		if (initialisiert) return;
		initialisiert = true;
		if (typeof window !== 'undefined') {
			isOffline = !navigator.onLine;
			updateCount();

			window.addEventListener('online', () => {
				isOffline = false;
				startSync();
			});

			window.addEventListener('offline', () => {
				isOffline = true;
			});

			window.addEventListener('beforeunload', handleBeforeUnload);
			// Jede Minute ein Anlauf, falls das online-Ereignis fehlte oder ein Eintrag nach
			// 5xx/429 liegen blieb (die Runde endet dort, statt sofort erneut zu senden).
			setInterval(() => {
				if (pendingCount > 0 && !isOffline) startSync();
			}, 60000);
		}
	}

	return {
		get isOffline() {
			return isOffline;
		},
		get pendingCount() {
			return pendingCount;
		},
		get isSyncing() {
			return isSyncing;
		},
		get warteschlangeFehler() {
			return warteschlangeFehler;
		},
		updateCount,
		startSync,
		exportQueueAsJSON,
		importQueueFromJSON,
		init
	};
}

export const offlineSync = createOfflineSyncStore();

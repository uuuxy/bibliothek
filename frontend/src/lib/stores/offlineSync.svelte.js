import {
	ladeWarteschlange,
	dequeueOfflineAction,
	enqueueOfflineAction,
	normalisiereEintrag
} from '../offlineWarteschlange.js';
import { apiClient } from '../apiFetch.js';
import { spieleErfolgston } from '../audio.js';
import { melde } from '../../inventur/lib/store.svelte.js';
import { netzLage } from './netzLage.svelte.js';

// Baut die Portion für die Nachbuch-Tür (POST /api/action/nachbuchen). Sie kann den
// Scan-Zeitpunkt buchen, einen Schlüssel genau einmal buchen, einen offline gescannten
// Ausweis auflösen und jede Abweichung als Meldung festhalten.
//
// Die Uhr: `gescannt_am` und `gesendet_am` müssen von derselben Uhr kommen, sonst rechnet
// der Server den Versatz falsch. Das ist hier kein Randfall: Die Uhr eines Theken-Rechners
// wird oft in dem Moment korrigiert, in dem das Netz zurückkommt, also zwischen Scan und
// Versand.
//
// Deshalb wird der Scan-Zeitpunkt neu bestimmt, wenn der Eintrag aus diesem Seitenaufruf
// stammt: `performance.now()` läuft gleichmäßig weiter und springt nicht. Aus dem Abstand
// seit dem Scan und der Wanduhr von jetzt ergibt sich ein Scan-Zeitpunkt, der zu
// `gesendet_am` passt, auch wenn die Wanduhr dazwischen gesprungen ist. Stammt der Eintrag
// aus einem früheren Seitenaufruf (Neuladen, eingespielte Sicherung), gilt der gespeicherte
// Wert; dort gibt es nichts Besseres.
/**
 * @param {import('../offlineWarteschlange.js').OfflineEintrag[]} batchItems
 * @returns {{ gesendet_am: string, eintraege: any[] }}
 */
function baueNachbuchPayload(batchItems) {
	const jetzt = Date.now();
	const mono = typeof performance !== 'undefined' ? Math.round(performance.now()) : null;
	const ursprung =
		typeof performance !== 'undefined' ? Math.round(performance.timeOrigin ?? 0) : null;

	return {
		gesendet_am: new Date(jetzt).toISOString(),
		eintraege: batchItems.map((posten) => {
			let gescannt = posten.gescannt_am;
			if (mono !== null && posten.ursprung === ursprung && typeof posten.mono === 'number') {
				gescannt = jetzt - (mono - posten.mono);
			}
			/** @type {any} */
			const e = {
				schluessel: posten.id,
				absicht: posten.art,
				barcode: posten.barcode,
				gescannt_am: new Date(gescannt).toISOString()
			};
			// Eine Ausleihe traegt ihre Person, eine Rueckgabe nicht — der Server findet
			// den Vorbesitzer selbst. `ausweis_barcode` kennt die Tuer bereits; gefuellt
			// wird es, sobald die Theke Ausweise offline annimmt.
			if (posten.art === 'ausleihe' && posten.leser_id) e.leser_id = posten.leser_id;
			if (posten.ausweis_barcode) e.ausweis_barcode = posten.ausweis_barcode;
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

// Gemeldet wird, was ein Mensch wissen muss:
//   - `nicht_gebucht` und `veraltet` tragen ihren Grund; ohne die Meldung erfaehrt niemand,
//     dass vier von achtzehn Rueckgaben abgelehnt wurden, und die Ausleihen laufen ins
//     Mahnwesen bis zur Rechnung an die Eltern.
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
 * @param {import('../offlineWarteschlange.js').OfflineEintrag} posten
 * @param {any} erg das endgültige Ergebnis des Servers zu diesem Eintrag
 * @returns {{ barcode: string, meldung: string }[]}
 */
function zuPruefendeFuer(posten, erg) {
	/** @type {{ barcode: string, meldung: string }[]} */
	const pruefen = [];
	if (erg.ergebnis === 'nicht_gebucht' || erg.ergebnis === 'veraltet') {
		pruefen.push({
			barcode: posten.barcode,
			meldung: erg.grund ? `nicht gebucht (${erg.grund})` : 'nicht gebucht'
		});
	} else if (erg.ergebnis === 'umgebucht') {
		pruefen.push({
			barcode: posten.barcode,
			meldung: 'lag bei jemand anderem — dort zurückgenommen und neu ausgeliehen'
		});
	} else {
		const gebucht = erg.daten?.type;
		if (gebucht && gebucht !== posten.art) {
			pruefen.push({
				barcode: posten.barcode,
				meldung: `als ${nenne(posten.art)} gescannt, der Server buchte ${nenne(gebucht)}`
			});
		}
	}

	if (erg.aufsicht_informieren) {
		pruefen.push({ barcode: posten.barcode, meldung: erg.aufsicht_informieren });
	}

	return pruefen;
}

// Erledigt ist nur, was der Server wirklich entschieden hat.
//
// Die Nachbuch-Tuer antwortet je Schluessel mit einem von neun Woertern. Acht davon sind
// endgueltig — gebucht, umgebucht, schon dagewesen, abgelehnt: In allen Faellen hat der
// Server den Fall abschliessend behandelt, und der Eintrag gehoert aus der Warteschlange.
// Das neunte, „wiederholen", heisst ausdruecklich das Gegenteil: Der Server war nicht
// erreichbar, oder derselbe Schluessel wird gerade gebucht. Dann bleibt der Eintrag liegen
// und die Runde endet, statt gegen dieselbe Wand zu laufen.
//
// Ein Schlüssel, den der Server gar nicht beantwortet hat, bleibt ebenfalls liegen:
// Schweigen ist kein Erfolg.
/**
 * @param {any} antwort
 * @param {import('../offlineWarteschlange.js').OfflineEintrag[]} batchItems
 * @returns {Promise<{ pruefen: { barcode: string, meldung: string }[], weiter: boolean }>}
 */
async function verarbeiteNachbuchErgebnisse(antwort, batchItems) {
	/** @type {{ barcode: string, meldung: string }[]} */
	const pruefen = [];
	let weiter = true;

	const versatz = Number(antwort?.uhr_versatz_sekunden ?? 0);
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
	for (const e of antwort?.ergebnisse ?? []) if (e?.schluessel) nachSchluessel[e.schluessel] = e;

	for (const posten of batchItems) {
		const erg = nachSchluessel[posten.id];
		if (!erg || !NACHBUCH_ENDGUELTIG.has(erg.ergebnis)) {
			weiter = false;
			continue;
		}

		pruefen.push(...zuPruefendeFuer(posten, erg));
		await dequeueOfflineAction(posten.id);
	}
	return { pruefen, weiter };
}

async function exportiereWarteschlange() {
	const q = await ladeWarteschlange();
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
	melde(
		pruefen.length === 1
			? `Offline-Scan ${liste} — bitte von Hand prüfen.`
			: `${pruefen.length} Offline-Scans brauchen Prüfung: ${liste} — bitte von Hand prüfen.`,
		'error'
	);
}

function erzeugeOfflineSync() {
	let wartendeAnzahl = $state(0);
	let gleichtAb = $state(false);
	// Die Warteschlange ließ sich nicht lesen (Commit 5): Das Band sagt es, statt 0 zu zeigen.
	let warteschlangeFehler = $state(false);
	// Der Server hat den Stapel abgelehnt, und Warten hilft nicht (4xx). Siehe sendeBatch.
	/** @type {number | null} */
	let abgelehntMitStatus = $state(null);

	async function zaehleNeu() {
		try {
			const q = await ladeWarteschlange();
			wartendeAnzahl = q.length;
			warteschlangeFehler = false;
		} catch {
			warteschlangeFehler = true;
		}
	}

	// Verschickt einen Batch und verarbeitet dessen Ergebnisse. Liefert false, wenn der
	// Sync abbrechen soll (kompletter Batch-Fehler wie 502, oder Netzwerkfehler).
	//
	// Dabei wird unterschieden, ob Warten hilft: Ein 502/503 heißt „der Server kommt gleich
	// wieder", die Runde endet still, in einer Minute läuft die nächste. Eine Antwort, die
	// sich von selbst nicht ändert (403, weil der gerade angemeldete Mensch nicht buchen darf;
	// 400 bei einem Rumpf, den der Server nicht annimmt), liefe sonst genauso still jede
	// Minute ins Leere, und der Zähler im Band sagt nur „noch nicht im System", nicht „geht
	// so nicht mehr". Deshalb merkt sich der Store solche Antworten, und das Band sagt sie an.
	async function sendeBatch(payload, batchItems, queueLength) {
		try {
			const res = await apiClient.post('/api/action/nachbuchen', payload);

			if (!res.ok) {
				// 5xx, 429 und 423 (gesperrt nach Inaktivität, bis jemand aufschließt): Warten
				// hilft. Sonst 4xx: Warten hilft nicht — das gehört gesagt.
				const wartenHilft = res.status >= 500 || res.status === 429 || res.status === 423;
				abgelehntMitStatus = wartenHilft ? null : res.status;
				return false;
			}
			abgelehntMitStatus = null;

			const antwort = await res.json();
			const { pruefen, weiter } = await verarbeiteNachbuchErgebnisse(antwort, batchItems);
			meldeZuPruefende(pruefen);
			await zaehleNeu();
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

	// Ein Lauf zur Zeit. Wer während eines Laufs ruft, wartet auf dessen Ende — die Sperre
	// nach Inaktivität lässt so erst die Warteschlange durch (leerlaufSperre).
	/** @type {Promise<void> | null} */
	let laufenderSync = null;
	function starteAbgleich() {
		if (netzLage.offline) return Promise.resolve();
		laufenderSync ??= sendeWarteschlange().finally(() => {
			laufenderSync = null;
		});
		return laufenderSync;
	}

	async function sendeWarteschlange() {
		gleichtAb = true;

		let syncedAny = false;

		while (!netzLage.offline) {
			/** @type {import('../offlineWarteschlange.js').OfflineEintrag[]} */
			let q;
			try {
				q = await ladeWarteschlange();
				warteschlangeFehler = false;
			} catch {
				warteschlangeFehler = true;
				break;
			}
			if (q.length === 0) break;

			// ladeWarteschlange liefert nach Scan-Zeitpunkt geordnet.
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

		if (syncedAny && wartendeAnzahl === 0) {
			spieleErfolgston();
		}
		gleichtAb = false;
	}

	/**
	 * Spielt eine Notfall-Sicherung zurück in die lokale Warteschlange; der Auto-Sync
	 * schiebt sie danach zum Server.
	 *
	 * item.id muss mitwandern: Diese ID ist der Idempotenz-Schlüssel, den der Server kennt
	 * (siehe baueNachbuchPayload und api/theke/nachbuchen_schluessel.go). Ohne sie vergibt
	 * normalisiereEintrag eine frische UUID, und dieselbe Datei zweimal eingespielt würde
	 * jede Aktion zweimal ausführen. Bei zehn Kiosk-Rechnern mit
	 * einem gemeinsamen Sicherungsordner ist doppeltes Einspielen der Normalfall,
	 * nicht der Ausnahmefall: Zwei Admins, oder einer, der unsicher ist, ob er es
	 * schon getan hat. Mit dem Schlüssel ist der zweite Durchlauf wirkungslos.
	 *
	 * Sicherungen in Format 1 (`action_type`/`barcode_id`) und
	 * Format 2 (`art`/`barcode`, mit Scan-Zeitpunkt) werden gleich behandelt.
	 * @param {File} file
	 */
	async function importiereWarteschlange(file) {
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

			await zaehleNeu();
			starteAbgleich();
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
	function warneVorDemSchliessen(e) {
		if (wartendeAnzahl > 0) {
			e.preventDefault();
			const msg = `${wartendeAnzahl} Vorgang/Vorgänge sind nur auf diesem Rechner gespeichert. Bitte zuerst "Sicherung speichern" — danach darf der Rechner aus.`;
			e.returnValue = msg;
			return msg;
		}
	}

	// Einmalig je Seite: Jede Anmeldung ruft init() (hintergrundAbrufe.svelte.js); ohne
	// Sperre stapelten Logout→Login Listener und 30-s-Intervalle, und starteAbgleich() lief
	// mehrfach parallel gegen dieselbe Warteschlange.
	let initialisiert = false;
	function init() {
		if (initialisiert) return;
		initialisiert = true;
		if (typeof window !== 'undefined') {
			zaehleNeu();

			// Ob das Netz da ist, misst netzLage — navigator.onLine allein ist eine Behauptung.
			netzLage.beiRueckkehr(() => starteAbgleich());

			window.addEventListener('beforeunload', warneVorDemSchliessen);
			// Jede Minute ein Anlauf, falls das online-Ereignis fehlte oder ein Eintrag nach
			// 5xx/429 liegen blieb (die Runde endet dort, statt sofort erneut zu senden).
			setInterval(() => {
				if (wartendeAnzahl > 0 && !netzLage.offline) starteAbgleich();
			}, 60000);
		}
	}

	return /** @satisfies {object} */ ({
		get ohneNetz() {
			return netzLage.offline;
		},
		get wartendeAnzahl() {
			return wartendeAnzahl;
		},
		get gleichtAb() {
			return gleichtAb;
		},
		get warteschlangeFehler() {
			return warteschlangeFehler;
		},
		/** Status der letzten Ablehnung, bei der Warten nicht hilft — sonst null. */
		get abgelehntMitStatus() {
			return abgelehntMitStatus;
		},
		zaehleNeu,
		starteAbgleich,
		exportiereWarteschlange,
		importiereWarteschlange,
		init
	});
}

export const offlineSync = erzeugeOfflineSync();

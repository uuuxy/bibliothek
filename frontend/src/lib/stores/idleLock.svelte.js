// stores/idleLock.svelte.js
// Inaktivitäts-Wächter der Sitzung (A4 in docs/datenschutz_offene_punkte.md).
//
// Zwei Stufen, beide in den Einstellungen justierbar (0 = aus):
//   1. Theke leeren — der geladene Schüler/Lehrer verschwindet aus der Omnibox, damit
//      der nächste an der Theke nicht das Profil des vorigen sieht.
//   2. Sperrbildschirm — die ganze Anwendung wird verdeckt, und der Server sperrt die
//      Anmeldung: Bis das Passwort der angemeldeten Person eingegeben ist, beantwortet er
//      keine Anfrage mehr, auch nicht nach dem Neuladen oder in einem neuen Tab. Die
//      Anmeldung selbst läuft weiter (der 30-Minuten-Refresh hält Kiosk-Tabs über Nacht).
//
// Als Aktivität zählt nur echte Bedienung (Zeiger, Tastatur, Scanner = Tastatur,
// Berührung, Rad). SSE-Pings und Poller zählen nicht — sonst wäre ein offener Tab nie
// inaktiv.
//
// Die Anmeldung gehört dem ganzen Browser. Für die Sperre zählt deshalb die Bedienung in
// jedem seiner Fenster, und gesperrt wie aufgeschlossen wird in allen zugleich. Ob gesperrt
// ist, sagt dabei allein der Server; die Fenster geben sich nur Bescheid (fensterSignale.js).

import { untrack } from 'svelte';
import { apiFetch, registriereSitzungGesperrtHandler } from '../apiFetch.js';
import { abonniere } from '../liveEvents.js';
import { authStore, registriereGesperrterStartHandler } from './authStore.svelte.js';
import { beobachteAndereFenster, meldeAktivitaet, meldeSperre } from './fensterSignale.js';
import { netzLage } from './netzLage.svelte.js';
import { offlineSync } from './offlineSync.svelte.js';
import { entsperreAmServer, leseSperrzustand, sperreAmServer } from './sperreAmServer.js';
import { thekeLeeren as thekeLeerenAusfuehren } from './thekeLeeren.js';

const AKTIVITAETS_EREIGNISSE = ['pointerdown', 'pointermove', 'keydown', 'wheel', 'touchstart'];
/** Mehr als einmal pro Sekunde muss die Uhr nicht neu gestellt werden. */
const DROSSEL_MS = 1000;

export class IdleLock {
	gesperrt = $state(false);
	/** Läuft gerade eine Wiederanmeldung (Passwort geht zum IMAP-Server)? */
	entsperreLaeuft = $state(false);
	entsperrFehler = $state(/** @type {string | null} */ (null));
	/** Minuten bis Theke leeren / Sperre; 0 = aus. Vorgaben wie im Backend. */
	thekeLeerenMinuten = $state(5);
	sperreMinuten = $state(15);

	/** @type {ReturnType<typeof setTimeout> | null} */
	#timerTheke = null;
	/** @type {ReturnType<typeof setTimeout> | null} */
	#timerSperre = null;
	#letzteAktivitaet = 0;
	#laeuft = false;
	// Führt der Server die Anmeldung als gesperrt? Nur dann folgt dieses Fenster, wenn
	// nebenan aufgeschlossen wird. Kann er sie nicht sperren, ist nur dieses Fenster
	// verdeckt und geht allein mit dem Passwort auf.
	#amServerGesperrt = false;
	#aktivitaetHandler = () => this.aktivitaet();
	/** @type {(() => void) | null} */
	#abmeldenFristen = null;
	/** @type {(() => void) | null} */
	#abmeldenNetz = null;
	/** @type {(() => void) | null} */
	#abmeldenFenster = null;
	// Die Sperre war fällig, konnte aber nicht greifen, weil das Netz weg war. Sie wird
	// nachgeholt, sobald die Verbindung zurück ist.
	#sperreFaellig = false;
	#onlineHandler = () => this.#netzZurueck();
	// Ohne `storage`-Ereignis (privates Fenster) erfährt ein gesperrtes Fenster erst beim
	// Hinsehen, dass nebenan aufgeschlossen wurde.
	#fokusHandler = () => void this.#folgeDemServer();

	/** Holt die Fristen vom Server; bei Fehler bleiben die Vorgaben. */
	async ladeFristen() {
		try {
			const res = await apiFetch('/api/einstellungen/sitzung');
			if (!res.ok) return;
			const d = await res.json();
			if (Number.isFinite(d.theke_leeren_minuten)) this.thekeLeerenMinuten = d.theke_leeren_minuten;
			if (Number.isFinite(d.sperre_minuten)) this.sperreMinuten = d.sperre_minuten;
		} catch {
			/* offline — Vorgaben gelten */
		}
		if (this.#laeuft && !this.gesperrt) this.#planeTimer();
	}

	/**
	 * Wächter scharf stellen (nach Login / Session-Restore). Idempotent.
	 *
	 * Der Aufrufer ist ein $effect (App.svelte). Was hier gelesen und geschrieben wird, darf
	 * ihn nicht neu anstoßen: Er liefe sonst in sich selbst, sobald die Sperre schon steht.
	 */
	start() {
		untrack(() => this.#stelleScharf());
	}

	#stelleScharf() {
		if (this.#laeuft) return;
		this.#laeuft = true;
		for (const ev of AKTIVITAETS_EREIGNISSE) {
			window.addEventListener(ev, this.#aktivitaetHandler, { passive: true });
		}
		window.addEventListener('focus', this.#fokusHandler);
		this.#letzteAktivitaet = 0;
		// Geänderte Fristen erreichen offene Tabs sofort: Das Speichern der Kategorie
		// „Datenschutz & Sitzung" sendet `sitzungsfristen` über die SSE-Leitung — sonst
		// liefe der zweite Arbeitsplatz bis zum nächsten F5 mit den alten Werten.
		this.#abmeldenFristen = abonniere('sitzungsfristen', () => this.ladeFristen());
		this.#abmeldenNetz = netzLage.beiRueckkehr(this.#onlineHandler);
		this.#abmeldenFenster = beobachteAndereFenster({
			beiSperre: () => this.#verdecke(true),
			beiEntsperrt: () => void this.#folgeDemServer(),
			beiAktivitaet: () => this.#fremdeAktivitaet()
		});
		this.#planeTimer();
		// Der Start fand die Anmeldung gesperrt vor: Uhren und Live-Leitung ruhen.
		if (this.gesperrt) this.#verdecke(true);
	}

	/**
	 * Der Server hat den Start mit „gesperrt" beantwortet (authStore.restoreSession): neu
	 * geladen oder neuer Tab. Verdeckt wird, bevor die Anwendung zum ersten Mal gerendert wird.
	 */
	verdeckeVorDemStart() {
		this.gesperrt = true;
	}

	/** Wächter abschalten (Logout). Hebt auch eine Sperre auf — ohne Sitzung gibt es nichts zu verdecken. */
	stop() {
		if (!this.#laeuft) return;
		this.#laeuft = false;
		for (const ev of AKTIVITAETS_EREIGNISSE) {
			window.removeEventListener(ev, this.#aktivitaetHandler);
		}
		window.removeEventListener('focus', this.#fokusHandler);
		this.#abmeldenFristen?.();
		this.#abmeldenFristen = null;
		this.#abmeldenNetz?.();
		this.#abmeldenNetz = null;
		this.#abmeldenFenster?.();
		this.#abmeldenFenster = null;
		this.#loescheTimer();
		this.#sperreFaellig = false;
		this.#amServerGesperrt = false;
		this.gesperrt = false;
		this.entsperrFehler = null;
	}

	/** Echte Bedienung: Uhr neu stellen. Im gesperrten Zustand zählt nichts. */
	aktivitaet() {
		if (!this.#laeuft || this.gesperrt) return;
		const jetzt = Date.now();
		if (jetzt - this.#letzteAktivitaet < DROSSEL_MS) return;
		this.#letzteAktivitaet = jetzt;
		// Eine ohne Netz fällig gewordene Sperre ist damit erledigt: Jetzt ist jemand da.
		// Sonst sperrte der Bildschirm mitten in der Arbeit, sobald das Netz zurückkommt.
		this.#sperreFaellig = false;
		meldeAktivitaet(jetzt);
		this.#planeTimer();
	}

	/**
	 * In einem anderen Fenster dieses Browsers wird gearbeitet: Die Sperre wartet. Die Theke
	 * dieses Fensters leert sich trotzdem nach ihrer eigenen Frist — an ihr war niemand.
	 */
	#fremdeAktivitaet() {
		if (!this.#laeuft || this.gesperrt) return;
		this.#sperreFaellig = false;
		this.#planeSperre();
	}

	/** Theken-Ansicht leeren — die Liste lebt in stores/thekeLeeren.js, weil das Abmelden dasselbe braucht. */
	thekeLeeren() {
		thekeLeerenAusfuehren();
	}

	/**
	 * Die Frist ist abgelaufen: Theke leeren, verdecken, am Server sperren.
	 *
	 * Ohne Netz wird nicht gesperrt: Aufgeschlossen wird am Server, und ohne ihn lägen die
	 * offline gescannten Vorgänge hinter einer Tür, die niemand öffnen kann. Geleert wird die
	 * Theke trotzdem; die Sperre holt #netzZurueck nach.
	 */
	sperren() {
		if (!this.#laeuft) return;
		if (offlineSync.isOffline) {
			this.thekeLeeren();
			this.#loescheTimer();
			this.#sperreFaellig = true;
			return;
		}
		this.#verdecke(false);
		void this.#sperreAmServer();
	}

	/** Der Server hat eine Anfrage mit „gesperrt" beantwortet (ein anderes Fenster hat gesperrt). */
	vomServerGesperrt() {
		if (!this.#laeuft) return;
		const neu = !this.#amServerGesperrt;
		this.#verdecke(true);
		if (neu) meldeSperre(true);
	}

	/**
	 * Verdeckt dieses Fenster. Den Server fragt es nicht; mehrfach gerufen ändert sich nichts.
	 * @param {boolean} amServer  Der Server führt die Anmeldung als gesperrt.
	 */
	#verdecke(amServer) {
		if (!this.#laeuft) return;
		this.thekeLeeren();
		this.#loescheTimer();
		authStore.haltLiveAn();
		if (amServer) this.#amServerGesperrt = true;
		if (this.gesperrt) return;
		this.entsperrFehler = null;
		this.gesperrt = true;
	}

	/**
	 * Erst die ohne Netz gescannten Vorgänge, dann die Sperre: Hinter ihr nimmt der Server
	 * sie nicht an, und sie lägen bis zum Aufschließen nur auf diesem Rechner.
	 */
	async #sperreAmServer() {
		if (offlineSync.pendingCount > 0) await offlineSync.startSync();
		// Inzwischen aufgeschlossen: Eine verspätete Sperre träfe jemanden bei der Arbeit.
		if (!this.#laeuft || !this.gesperrt) return;
		const ergebnis = await sperreAmServer();
		if (ergebnis !== 'gesperrt' || !this.#laeuft || !this.gesperrt) return;
		this.#amServerGesperrt = true;
		meldeSperre(true);
	}

	/**
	 * Das Netz ist zurück. War die Sperre fällig, greift sie jetzt. Steht sie nur in diesem
	 * Fenster, geht sie noch einmal an den Server — die erste Anfrage kann ihn verfehlt haben.
	 */
	#netzZurueck() {
		if (!this.#laeuft) return;
		if (this.gesperrt) {
			if (!this.#amServerGesperrt) void this.#sperreAmServer();
			return;
		}
		if (!this.#sperreFaellig) return;
		this.#sperreFaellig = false;
		this.sperren();
	}

	/**
	 * Schließt mit dem Passwort der angemeldeten Person auf. Der Server prüft es beim
	 * Mailserver der Schule und, wenn der nicht erreichbar ist, gegen den Prüfwert der Anmeldung.
	 * @param {string} passwort
	 * @returns {Promise<boolean>}
	 */
	async entsperren(passwort) {
		if (!passwort) {
			this.entsperrFehler = 'Bitte Passwort eingeben.';
			return false;
		}
		// Gesperrt wurde noch mit Netz, verloren ging es danach. „Netzwerkfehler" sagte
		// weder, dass Warten hilft, noch dass die gescannten Vorgänge gespeichert bleiben.
		if (offlineSync.isOffline) {
			this.entsperrFehler =
				'Ohne Netz lässt sich das Passwort nicht prüfen — die Anmeldung läuft über den Schulserver. ' +
				'Sobald die Verbindung zurück ist, geht es hier weiter; gescannte Vorgänge bleiben gespeichert.';
			return false;
		}
		this.entsperreLaeuft = true;
		this.entsperrFehler = null;
		try {
			const res = await entsperreAmServer(passwort);
			if (res.ok) {
				this.#schliesseAuf(await res.json());
				return true;
			}
			// 401 heißt „Passwort falsch" oder „diese Anmeldung gibt es nicht mehr".
			if (res.status === 401 && (await leseSperrzustand()).zustand === 'beendet') {
				authStore.sitzungAbgelaufen();
				return false;
			}
			this.entsperrFehler = await fehlertext(res);
			return false;
		} catch (err) {
			this.entsperrFehler = err instanceof Error ? err.message : 'Netzwerkfehler';
			return false;
		} finally {
			this.entsperreLaeuft = false;
		}
	}

	/** @param {any} konto  Antwort des Servers wie bei der Anmeldung */
	#schliesseAuf(konto) {
		this.#amServerGesperrt = false;
		this.gesperrt = false;
		this.entsperrFehler = null;
		authStore.uebernimmKonto(konto);
		this.#letzteAktivitaet = Date.now();
		meldeAktivitaet(this.#letzteAktivitaet);
		meldeSperre(false);
		this.#planeTimer();
		void this.ladeFristen();
	}

	/**
	 * Ein anderes Fenster hat aufgeschlossen oder abgemeldet. Ob das stimmt, sagt der Server —
	 * das Signal im Browser kann jeder auslösen, der davor sitzt.
	 */
	async #folgeDemServer() {
		if (!this.#laeuft || !this.gesperrt || !this.#amServerGesperrt) return;
		const stand = await leseSperrzustand();
		if (!this.#laeuft || !this.gesperrt) return;
		if (stand.zustand === 'beendet') authStore.sitzungAbgelaufen();
		else if (stand.zustand === 'offen') this.#schliesseAuf(stand.konto);
	}

	#planeTimer() {
		this.#planeTheke();
		this.#planeSperre();
	}

	#planeTheke() {
		if (this.#timerTheke) clearTimeout(this.#timerTheke);
		this.#timerTheke = null;
		if (this.thekeLeerenMinuten > 0) {
			this.#timerTheke = setTimeout(() => this.thekeLeeren(), this.thekeLeerenMinuten * 60_000);
		}
	}

	#planeSperre() {
		if (this.#timerSperre) clearTimeout(this.#timerSperre);
		this.#timerSperre = null;
		if (this.sperreMinuten > 0) {
			this.#timerSperre = setTimeout(() => this.sperren(), this.sperreMinuten * 60_000);
		}
	}

	#loescheTimer() {
		if (this.#timerTheke) clearTimeout(this.#timerTheke);
		if (this.#timerSperre) clearTimeout(this.#timerSperre);
		this.#timerTheke = null;
		this.#timerSperre = null;
	}
}

/** @param {Response} res */
async function fehlertext(res) {
	if (res.status === 401) return 'Passwort falsch.';
	if (res.status === 429) return 'Zu viele Versuche — bitte kurz warten.';
	if (res.status === 503) return 'Anmeldedienst (Mailserver) nicht erreichbar.';
	try {
		const d = await res.json();
		if (d?.error) return String(d.error);
	} catch {
		/* kein JSON */
	}
	return `Wiederanmeldung fehlgeschlagen (${res.status}).`;
}

export const idleLock = new IdleLock();

// Einmal beim Modul-Laden, wie der 401-Haken des authStore.
registriereSitzungGesperrtHandler(() => idleLock.vomServerGesperrt());
registriereGesperrterStartHandler(() => idleLock.verdeckeVorDemStart());

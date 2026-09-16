// stores/idleLock.svelte.js
// Inaktivitäts-Wächter der Sitzung (A4 in docs/datenschutz_offene_punkte.md).
//
// Zwei Stufen, beide in den Einstellungen justierbar (0 = aus):
//   1. Theke leeren — der geladene Schüler/Lehrer verschwindet aus der Omnibox, damit
//      der NÄCHSTE an der Theke nicht das Profil des vorigen sieht.
//   2. Sperrbildschirm — die ganze Anwendung wird verdeckt; weiter geht es nur mit dem
//      Passwort der angemeldeten Person (echte Wiederanmeldung gegen /login) oder per
//      Abmelden. Die Sitzung selbst läuft weiter (der 30-Minuten-Refresh hält
//      Kiosk-Tabs über Nacht am Leben) — sie ist nur nicht mehr einsehbar.
//
// Als Aktivität zählt nur echte Bedienung (Zeiger, Tastatur, Scanner = Tastatur,
// Berührung, Rad). SSE-Pings und Poller zählen NICHT — sonst wäre ein offener Tab nie
// inaktiv.

import { apiFetch } from '../apiFetch.js';
import { abonniere } from '../liveEvents.js';
import { authStore } from './authStore.svelte.js';
import { offlineSync } from './offlineSync.svelte.js';
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
	#aktivitaetHandler = () => this.aktivitaet();
	/** @type {(() => void) | null} */
	#abmeldenFristen = null;
	// Die Sperre WAR faellig, konnte aber nicht greifen, weil das Netz weg war. Sie wird
	// nachgeholt, sobald die Verbindung zurueck ist (Stufe 3, 16.09.2026).
	#sperreFaellig = false;
	#onlineHandler = () => this.#netzZurueck();

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
		if (this.#laeuft) this.#planeTimer();
	}

	/** Wächter scharf stellen (nach Login / Session-Restore). Idempotent. */
	start() {
		if (this.#laeuft) return;
		this.#laeuft = true;
		for (const ev of AKTIVITAETS_EREIGNISSE) {
			window.addEventListener(ev, this.#aktivitaetHandler, { passive: true });
		}
		this.#letzteAktivitaet = 0;
		// Geänderte Fristen erreichen offene Tabs sofort: Das Speichern der Kategorie
		// „Datenschutz & Sitzung" sendet `sitzungsfristen` über die SSE-Leitung — sonst
		// liefe der zweite Arbeitsplatz bis zum nächsten F5 mit den alten Werten.
		this.#abmeldenFristen = abonniere('sitzungsfristen', () => this.ladeFristen());
		window.addEventListener('online', this.#onlineHandler);
		this.#planeTimer();
	}

	/** Wächter abschalten (Logout). Hebt auch eine Sperre auf — ohne Sitzung gibt es nichts zu verdecken. */
	stop() {
		if (!this.#laeuft) return;
		this.#laeuft = false;
		for (const ev of AKTIVITAETS_EREIGNISSE) {
			window.removeEventListener(ev, this.#aktivitaetHandler);
		}
		this.#abmeldenFristen?.();
		this.#abmeldenFristen = null;
		window.removeEventListener('online', this.#onlineHandler);
		this.#loescheTimer();
		this.#sperreFaellig = false;
		this.gesperrt = false;
		this.entsperrFehler = null;
	}

	/** Echte Bedienung: Uhr neu stellen. Im gesperrten Zustand zählt nichts. */
	aktivitaet() {
		if (!this.#laeuft || this.gesperrt) return;
		const jetzt = Date.now();
		if (jetzt - this.#letzteAktivitaet < DROSSEL_MS) return;
		this.#letzteAktivitaet = jetzt;
		// Eine offline fällig gewordene Sperre ist damit erledigt: Sie steht für „es war
		// länger als die Frist niemand da" — und jetzt ist jemand da. Ohne diese Zeile
		// sperrte der Bildschirm mitten in der Arbeit, sobald das Netz zurückkam, und
		// nahm den ohne Netz gemerkten Ausweis mit (thekeLeeren).
		this.#sperreFaellig = false;
		this.#planeTimer();
	}

	/**
	 * Theken-Ansicht leeren — die Liste lebt in stores/thekeLeeren.js, weil das
	 * Abmelden dasselbe braucht (bis 31.08.2026 konnte es nur dieser Wächter, und
	 * der nächste Bediener sah das Profil des vorigen).
	 */
	thekeLeeren() {
		thekeLeerenAusfuehren();
	}

	/**
	 * Sperrbildschirm: Theke leeren und alles verdecken.
	 *
	 * OHNE NETZ wird NICHT gesperrt (Stufe 3, entschieden am 13.09.2026). Der Grund ist
	 * kein Komfort: Aufgemacht wird der Sperrbildschirm mit dem Passwort gegen den
	 * Schul-Mailserver (`entsperren` ruft `/login`, das gegen IMAP prueft). Ohne Netz
	 * passt der Schluessel nicht ins Schloss — die Sitzung dahinter laeuft weiter, ist
	 * aber nicht mehr erreichbar, und die offline gescannten Vorgaenge liegen hinter
	 * einer Tuer, die niemand oeffnen kann. Der Stufe-1-Nachweis am 16.09.2026 hat genau
	 * das gezeigt.
	 *
	 * Was trotzdem passiert: Die Theke wird GELEERT. Das ist der datenschutzrechtliche
	 * Teil (A4) — der naechste Bediener sieht das Profil des vorigen nicht. Verdeckt
	 * wird nur nicht, denn Verdecken ohne Aufschliessen ist Aussperren.
	 *
	 * Nachgeholt wird die Sperre, sobald die Verbindung zurueck ist (#netzZurueck).
	 */
	sperren() {
		if (!this.#laeuft) return;
		this.thekeLeeren();
		this.#loescheTimer();
		if (offlineSync.isOffline) {
			this.#sperreFaellig = true;
			return;
		}
		this.entsperrFehler = null;
		this.gesperrt = true;
	}

	/**
	 * Das Netz ist zurueck. War die Sperre faellig, greift sie JETZT — es war laenger
	 * als die eingestellte Frist niemand da, und der Bildschirm steht seither offen.
	 */
	#netzZurueck() {
		if (!this.#laeuft || !this.#sperreFaellig) return;
		this.#sperreFaellig = false;
		this.sperren();
	}

	/**
	 * Wiederanmeldung der angemeldeten Person. Geht gegen /login (nicht /api/…), also
	 * löst ein falsches Passwort (401) NICHT den Sitzungs-abgelaufen-Haken aus.
	 * @param {string} passwort
	 * @returns {Promise<boolean>}
	 */
	async entsperren(passwort) {
		const email = authStore.currentUser?.email;
		if (!email || !passwort) {
			this.entsperrFehler = 'Bitte Passwort eingeben.';
			return false;
		}
		// Seit dem 16.09.2026 wird ohne Netz gar nicht erst gesperrt. Wer hier trotzdem
		// landet, wurde NOCH MIT Netz gesperrt und hat es seither verloren. Dann ist
		// „Netzwerkfehler" die technisch richtige, aber unbrauchbare Auskunft: Sie sagt
		// nicht, dass Warten hilft und dass die Vorgaenge nicht verloren sind.
		if (offlineSync.isOffline) {
			this.entsperrFehler =
				'Ohne Netz lässt sich das Passwort nicht prüfen — die Anmeldung läuft über den Schulserver. ' +
				'Sobald die Verbindung zurück ist, geht es hier weiter; gescannte Vorgänge bleiben gespeichert.';
			return false;
		}
		this.entsperreLaeuft = true;
		this.entsperrFehler = null;
		try {
			const res = await apiFetch('/login', {
				method: 'POST',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify({ email, password: passwort }),
				timeoutMs: 20000
			});
			if (res.ok) {
				this.gesperrt = false;
				this.#letzteAktivitaet = Date.now();
				this.#planeTimer();
				return true;
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

	#planeTimer() {
		this.#loescheTimer();
		if (this.thekeLeerenMinuten > 0) {
			this.#timerTheke = setTimeout(() => this.thekeLeeren(), this.thekeLeerenMinuten * 60_000);
		}
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

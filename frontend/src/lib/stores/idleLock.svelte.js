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
import { authStore } from './authStore.svelte.js';
import { omniboxStore } from './omnibox.svelte.js';
import { uiStore } from './uiStore.svelte.js';

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
		this.#planeTimer();
	}

	/** Wächter abschalten (Logout). Hebt auch eine Sperre auf — ohne Sitzung gibt es nichts zu verdecken. */
	stop() {
		if (!this.#laeuft) return;
		this.#laeuft = false;
		for (const ev of AKTIVITAETS_EREIGNISSE) {
			window.removeEventListener(ev, this.#aktivitaetHandler);
		}
		this.#loescheTimer();
		this.gesperrt = false;
		this.entsperrFehler = null;
	}

	/** Echte Bedienung: Uhr neu stellen. Im gesperrten Zustand zählt nichts. */
	aktivitaet() {
		if (!this.#laeuft || this.gesperrt) return;
		const jetzt = Date.now();
		if (jetzt - this.#letzteAktivitaet < DROSSEL_MS) return;
		this.#letzteAktivitaet = jetzt;
		this.#planeTimer();
	}

	/** Theken-Ansicht leeren: kein geladener Schüler/Lehrer, keine Suchreste, keine Kamera. */
	thekeLeeren() {
		// Kamera-Scanner dekodiert den Stream unabhängig von der Sichtbarkeit — lief er
		// weiter, buchte ein vor die Kamera gehaltener Barcode hinter der Sperre
		// (Prüfung 22.08.2026, A6). Stop ist fire-and-forget; der Store-Zeiger wird sofort
		// genullt, damit kein Decode-Callback mehr in submitAction läuft.
		const scanner = omniboxStore.cameraScanner;
		omniboxStore.cameraScanner = null;
		omniboxStore.showCamera = false;
		if (scanner) {
			try {
				Promise.resolve(scanner.stop()).catch(() => {});
			} catch {
				/* Scanner war schon aus */
			}
		}
		omniboxStore.activeStudent = null;
		omniboxStore.activeTeacher = null;
		omniboxStore.queryVal = '';
		omniboxStore.isDropdownOpen = false;
		omniboxStore.unifiedSearchResults = {
			students: [],
			books: [],
			studentsTotal: 0,
			booksTotal: 0
		};
		omniboxStore.vormerkungAlert = null;
		omniboxStore.blockAlert = null;
		omniboxStore.checklistAnfrage = null;
		uiStore.requestedStudentId = null;
	}

	/** Sperrbildschirm: Theke leeren und alles verdecken. */
	sperren() {
		if (!this.#laeuft) return;
		this.thekeLeeren();
		this.#loescheTimer();
		this.entsperrFehler = null;
		this.gesperrt = true;
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

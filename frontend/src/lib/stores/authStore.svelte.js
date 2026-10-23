import { appState } from '../../inventur/lib/store.svelte.js';
import { apiFetch, leseApiFehler, registriereSitzungAbgelaufenHandler } from '../apiFetch.js';
import { abonniere, trenne, verbinde } from '../liveEvents.js';
import { hatRecht } from '../menue.js';
import { thekeLeeren } from './thekeLeeren.js';
import { snackbars } from './snackbars.svelte.js';

// Boot-Restore gegen einen Datenbank-Aussetzer: dreimal fragen, 2 s auseinander. Die
// Frist des Servers bei der Token-Prüfung ist 2 s (auth/token.go), ein kurzer Aussetzer ist
// damit überbrückt; länger darf der Ladekreis vor dem Login nicht stehen.
const RESTORE_VERSUCHE = 3;
const RESTORE_WARTEZEIT_MS = 2000;

// Eine Abmeldung, die den Server nicht erreicht hat. Das Löschcookie setzt nur die
// Antwort des Servers (api/abmeldung.go, bei 200 und bei 503). Ohne Netz, oder bei 502/504
// vom Proxy, kommt keine an: Das HttpOnly-Cookie bleibt im Browser, und JavaScript kann es nicht
// löschen. Ohne diesen Merker meldete das nächste Neuladen mit Netz die vorige Person wieder an —
// am geteilten Theken-Rechner arbeitete der Nächste unter fremdem Konto. Der Merker trägt keine
// Personendaten; er lebt im Browser wie das Cookie selbst. Nachgestellt in
// e2e/abmelden-ohne-antwort.spec.js.
const ABMELDUNG_AUSSTEHEND = 'bibliothek.abmeldungAusstehend';

/** @param {boolean} ausstehend */
function merkeAbmeldung(ausstehend) {
	try {
		if (ausstehend) localStorage.setItem(ABMELDUNG_AUSSTEHEND, '1');
		else localStorage.removeItem(ABMELDUNG_AUSSTEHEND);
	} catch {
		/* Ohne Speicher (privates Fenster, gesperrte Website-Daten) bleibt es beim bisherigen Weg. */
	}
}

function abmeldungAusstehend() {
	try {
		return localStorage.getItem(ABMELDUNG_AUSSTEHEND) === '1';
	} catch {
		return false;
	}
}

/**
 * Hat die Antwort das Löschcookie mitgebracht? Das tun genau die zwei Antworten des
 * Abmelde-Handlers; 429 vom Rate-Limiter und 502/504 vom Proxy tun es nicht.
 * @param {number} status
 */
function abmeldungZugestellt(status) {
	return status === 200 || status === 503;
}

// Findet der Start die Anmeldung gesperrt, verdeckt leerlaufSperre das Fenster, bevor `isLoggedIn`
// die Anwendung rendert. Eine Registrierung statt eines Imports: leerlaufSperre importiert diesen Store.
/** @type {(() => void) | null} */
let gesperrterStartHandler = null;

/** @param {() => void} handler */
export function registriereGesperrterStartHandler(handler) {
	gesperrterStartHandler = handler;
}

class AuthStore {
	isLoggedIn = $state(false);
	currentUser = $state(/** @type {any} */ (null));
	/** Erst nach dem Boot-Restore-Versuch true — davor zeigt die App weder Login noch Inhalt. */
	sessionChecked = $state(false);
	heartbeatOk = $state(true);
	lastHeartbeatTime = $state(Date.now());
	loginEmail = $state('');
	loginPassword = $state('');
	loginError = $state(/** @type {string | null} */ (null));
	isLoggingIn = $state(false);
	/**
	 * Steht nach einer Abmeldung, deren Widerruf der Server nicht bestätigt hat (503,
	 * api/abmeldung.go): Das Löschcookie kam an, dieser Browser ist abgemeldet, die Sitzung
	 * selbst gilt bis zu ihrem Ablauf weiter. Abgemeldet wird trotzdem, dazu steht dieser
	 * Hinweis auf der Anmeldemaske.
	 */
	abmeldeHinweis = $state(/** @type {string | null} */ (null));

	/** @type {ReturnType<typeof setInterval> | null} */
	#refreshTimer = null;
	/** @type {Array<() => void>} Abmeldungen der Herzschlag-Zuhörer. */
	#abmeldungen = [];

	// Sliding Session: Der Server erneuert das Cookie erst bei <50% Restlaufzeit
	// (12h-Token → Erneuerung frühestens nach 6h). Der 30-Minuten-Tick ist also
	// fast immer ein billiges "skipped" — hält aber Kiosk-Tabs über Nacht am Leben.
	startSessionRefresh() {
		this.stopSessionRefresh();
		this.#refreshTimer = setInterval(
			async () => {
				try {
					const res = await fetch('/api/auth/refresh', { method: 'POST' });
					if (res.status === 401) this.handleLogout();
				} catch {
					/* offline — der SSE-Heartbeat meldet das bereits */
				}
			},
			30 * 60 * 1000
		);
	}

	stopSessionRefresh() {
		if (this.#refreshTimer) {
			clearInterval(this.#refreshTimer);
			this.#refreshTimer = null;
		}
	}

	// Die Verbindung selbst gehört liveEvents.js — hier hängt nur der Herzschlag daran.
	//
	// Frische kommt ausschließlich von echten Server-Signalen (connected/ping). Ob wir
	// offline sind, entscheidet allein der 25s-Checker in App.svelte — ein transienter
	// Fehler (Druckdialog, Standby, Netzwechsel) darf das Vollbild-Overlay nicht sofort
	// auslösen.
	connectSSE() {
		verbinde();
		if (this.#abmeldungen.length > 0) return; // Zuhörer hängen schon.

		const markAlive = () => {
			this.lastHeartbeatTime = Date.now();
			this.heartbeatOk = true;
		};
		this.#abmeldungen.push(abonniere('connected', markAlive), abonniere('ping', markAlive));
	}

	/**
	 * Gemeinsamer Einstieg für Login und Session-Restore: setzt den Store-Zustand,
	 * startet SSE + Refresh-Loop und pflegt die appState-Rollen-Flags.
	 * @param {any} konto
	 * @param {Function} [onRoleCallback]
	 */
	/**
	 * Die Antwort des Abmelde-Handlers auswerten — beim Abmelden selbst und beim Nachholen
	 * im Boot-Restore.
	 * @param {{ status: number }} res
	 */
	#verarbeiteAbmeldeAntwort(res) {
		if (abmeldungZugestellt(res.status)) merkeAbmeldung(false);
		if (res.status === 503) {
			this.abmeldeHinweis =
				'Abgemeldet, aber der Server hat die Sperre der Sitzung nicht bestätigt. ' +
				'Dieser Browser ist abgemeldet; kommt das öfter vor, bitte Bescheid geben.';
		}
	}

	/**
	 * Boot-Restore einer gesperrten Anmeldung (Inaktivität oder Höchstdauer): Der Server nennt
	 * nur E-Mail-Adresse und Grund. Es gibt weder Rechte noch Live-Leitung, bis das Passwort
	 * eingegeben ist.
	 * @param {{ email?: string }} zustand
	 */
	#applyGesperrt(zustand) {
		gesperrterStartHandler?.();
		this.currentUser = { email: zustand?.email ?? '', permissions: [] };
		this.isLoggedIn = true;
		this.lastHeartbeatTime = Date.now();
		this.heartbeatOk = true;
		this.startSessionRefresh();
		appState.guestAuthenticated = false;
		appState.adminAuthenticated = false;
	}

	/**
	 * Die Sperre ist aufgeschlossen: Der Server hat das Konto wie bei der Anmeldung geliefert.
	 * @param {any} konto
	 */
	uebernimmKonto(konto) {
		this.#applyLogin(konto);
	}

	/** Die Anmeldung ist gesperrt: Die Live-Leitung endet, bis aufgeschlossen ist. */
	haltLiveAn() {
		trenne();
	}

	#applyLogin(konto, onRoleCallback) {
		// Eine neue Anmeldung ist gewollt: Ein alter Merker darf sie beim nächsten Laden nicht beenden.
		merkeAbmeldung(false);
		this.abmeldeHinweis = null;
		this.currentUser = konto;
		this.isLoggedIn = true;
		// Grace-Period: direkt nach dem Login gilt die Verbindung als frisch,
		// bis der erste SSE-Ping eintrifft (Server pingt alle 15s).
		this.lastHeartbeatTime = Date.now();
		this.heartbeatOk = true;
		this.connectSSE();
		this.startSessionRefresh();

		// Portal oder Verwaltung, entschieden am Recht, nicht am Rollennamen: Eine Aufzählung
		// von Rollen wäre bei jeder weiteren Rolle nachzupflegen, und wer sie vergisst, schickt
		// die neue Rolle still ins Portal; der Fehler zeigt sich als „die Leitung sieht
		// nichts", nicht als Meldung.
		//
		// perform_actions ist das Recht, an der Theke und im Bestand zu arbeiten (/api/action,
		// /scan, /search). Wer es hat, gehört in die Verwaltung; das Kollegium hat es nicht
		// (db/rechte_vorgabe.go) und sieht sein Portal. Welchen Bildschirm jemand öffnen darf,
		// liest der Router ohnehin aus denselben Rechten (menue.js, erlaubteBildschirme).
		const arbeitetInDerBibliothek = hatRecht(konto, 'perform_actions');
		appState.guestAuthenticated = true;
		appState.adminAuthenticated = arbeitetInDerBibliothek;
		if (onRoleCallback) onRoleCallback(konto?.rolle || '');
	}

	/**
	 * Boot-Restore: stellt eine bestehende Session aus dem HttpOnly-Cookie wieder her.
	 * Ohne diesen Check zeigte jeder Reload den Login-Screen (F5 = UI-Logout),
	 * obwohl die Session serverseitig noch 12h gültig war.
	 *
	 * Ein 5xx beantwortet die Frage „gilt diese Sitzung?" nicht — er sagt, dass der Server
	 * sie gerade nicht prüfen kann. Bei einem Datenbank-Aussetzer antwortet er mit 503
	 * statt 401, damit ein Arbeitsplatz nicht abgemeldet wird (auth/anmelden.go). Hieße
	 * „nicht ok" hier Login-Bildschirm, müsste sich, wer während des Aussetzers neu lädt (F5,
	 * Kiosk-Neustart, Bildschirm morgens an), neu anmelden, obwohl sein Cookie noch gilt.
	 * Deshalb kurz warten und erneut fragen; der Boot zeigt so lange den Ladekreis
	 * (App.svelte). Ein 401 wartet nicht: Er ist die
	 * reguläre Antwort für „kein Cookie", und jeder frische Browser käme sonst erst nach
	 * Sekunden an den Login.
	 */
	async restoreSession() {
		try {
			// Zuerst eine ausstehende Abmeldung nachholen und dann nichts wiederherstellen: Wer
			// abgemeldet hat, meldet sich neu an. Ohne Netz bleibt der Merker für den nächsten Start.
			if (abmeldungAusstehend()) {
				const res = await fetch('/api/auth/logout', { method: 'POST' });
				this.#verarbeiteAbmeldeAntwort(res);
				return;
			}
			for (let versuch = 1; ; versuch++) {
				const res = await fetch('/api/auth/me');
				if (res.ok) {
					this.#applyLogin(await res.json());
					return;
				}
				if (res.status === 423) {
					this.#applyGesperrt(await res.json());
					return;
				}
				if (res.status < 500 || versuch >= RESTORE_VERSUCHE) return;
				await new Promise((fertig) => setTimeout(fertig, RESTORE_WARTEZEIT_MS));
			}
		} catch {
			/* offline/Server weg → Login-Screen ist der richtige Fallback */
		} finally {
			this.sessionChecked = true;
		}
	}

	/**
	 * @param {Event|null} e
	 * @param {Function} [onRoleCallback]
	 */
	async handleLogin(e, onRoleCallback) {
		if (e) e.preventDefault();
		if (!this.loginEmail.trim() || !this.loginPassword) return;
		this.loginError = null;
		this.isLoggingIn = true;

		try {
			// apiFetch statt fetch: POST /login unterliegt der CSRF-Prüfung (api/csrf.go) und
			// braucht den X-CSRF-Token-Header. apiFetch holt das Token notfalls selbst per
			// Bootstrap (/api/csrf-token), es geht also auch beim allerersten Seitenaufruf ohne
			// vorheriges Cookie.
			//
			// timeoutMs über dem Standard von 10 s für Mutationen: Die Anmeldung wartet auf den
			// IMAP-Server der Schule, und pruefeIMAPAnmeldung räumt dem selbst 15 s ein. Mit dem
			// Standardwert bräche apiFetch eine langsame, aber erfolgreiche Anmeldung als
			// "Netzwerk-Timeout" ab.
			const res = await apiFetch('/login', {
				method: 'POST',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify({ email: this.loginEmail, password: this.loginPassword }),
				timeoutMs: 20000
			});
			if (!res.ok) {
				const fehler = new Error(await leseApiFehler(res, 'Login fehlgeschlagen'));
				// 403 = Zugangsdaten richtig, Zugang trotzdem nicht („Zugang beantragt",
				// „Konto deaktiviert"). Diese Meldung muss STEHEN bleiben: Sie sagt der
				// Lehrkraft, dass Wiederholen nichts bringt — nach vier Sekunden weg, liest
				// sie sie nicht zu Ende und tippt das Passwort noch einmal.
				/** @type {any} */ (fehler).bleibtStehen = res.status === 403;
				throw fehler;
			}
			const konto = await res.json();
			this.loginEmail = '';
			this.loginPassword = '';
			this.#applyLogin(konto, onRoleCallback);
		} catch (err) {
			const meldung = /** @type {any} */ (err).message || String(err);
			this.loginError = meldung;
			this.loginPassword = '';
			if (!(/** @type {any} */ (err)?.bleibtStehen)) {
				setTimeout(() => {
					this.loginError = null;
				}, 4000);
			}
		} finally {
			this.isLoggingIn = false;
		}
	}

	// sitzungAbgelaufen: der zentrale 401-Haken aus apiFetch. Nur beim ERSTEN
	// Treffer abmelden — die parallel laufenden Poller liefern denselben 401
	// mehrfach, und mehrfache Toasts/Abmeldungen wären nur Lärm.
	sitzungAbgelaufen() {
		if (!this.isLoggedIn) return;
		this.handleLogout();
		snackbars.addToast('Sitzung abgelaufen — bitte neu anmelden.', 'warning');
	}

	handleLogout(onLogoutCallback) {
		// Serverseitig invalidieren (Token-Blacklist + Cookie löschen) — sonst würde
		// der Boot-Restore die Session beim nächsten Reload wiederbeleben.
		// Fire-and-forget: der lokale Zustand wird unabhängig vom Netz geleert.
		//
		// keepalive: Ohne ihn bricht der Browser die Anfrage ab, sobald die Seite verschwindet;
		// nach einem Reload oder Tab zu direkt nach dem Klick hätte der Server die Sitzung nie
		// gesperrt, und nach dem nächsten Öffnen wäre man wieder angemeldet. Nachgestellt in
		// e2e/abmelden-tab-zu.spec.js.
		//
		// Bis der Server antwortet, steht die Abmeldung als ausstehend im Browser
		// (ABMELDUNG_AUSSTEHEND); restoreSession holt sie beim nächsten Start nach.
		merkeAbmeldung(true);
		fetch('/api/auth/logout', { method: 'POST', keepalive: true })
			.then((res) => this.#verarbeiteAbmeldeAntwort(res))
			.catch(() => {});
		this.sessionChecked = true;
		this.isLoggedIn = false;
		this.currentUser = null;
		this.loginEmail = '';
		this.loginPassword = '';
		this.loginError = null;
		appState.adminAuthenticated = false;
		appState.guestAuthenticated = false;
		this.stopSessionRefresh();
		// Theke leeren: Die Omnibox-Stores sind Modul-Singletons und überleben das Abmelden.
		// Ohne diese Zeile sähe der nächste Bediener nach seiner Anmeldung weiter das
		// Schülerprofil des vorigen, mit Ausleihen, Sperren und Mahnstufen. Dieselbe Liste wie
		// beim Inaktivitäts-Wächter.
		thekeLeeren();
		// Verbindung UND jeden geplanten Wiederaufbau beenden: Nach der Abmeldung
		// antwortet /events nur noch mit 401, und ein weiterlaufender Wiederaufbau
		// klopfte im Hintergrund weiter an.
		trenne();
		for (const abmelden of this.#abmeldungen) abmelden();
		this.#abmeldungen = [];
		if (onLogoutCallback) onLogoutCallback();
	}
}

export const authStore = new AuthStore();

// Der Haken wird EINMAL beim Modul-Laden registriert — ab dann führt jeder
// unerwartete 401 einer API-Antwort zur sauberen Abmeldung statt zu Dauer-Lärm.
registriereSitzungAbgelaufenHandler(() => authStore.sitzungAbgelaufen());

import { appState } from '../../inventur/lib/store.svelte.js';
import { apiFetch, registriereSitzungAbgelaufenHandler } from '../apiFetch.js';
import { abonniere, trenne, verbinde } from '../liveEvents.js';
import { toastStore } from './toastStore.svelte.js';

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
	 * @param {any} user
	 * @param {Function} [onRoleCallback]
	 */
	#applyLogin(user, onRoleCallback) {
		this.currentUser = user;
		this.isLoggedIn = true;
		// Grace-Period: direkt nach dem Login gilt die Verbindung als frisch,
		// bis der erste SSE-Ping eintrifft (Server pingt alle 15s).
		this.lastHeartbeatTime = Date.now();
		this.heartbeatOk = true;
		this.connectSSE();
		this.startSessionRefresh();

		if (user && (user.rolle === 'admin' || user.rolle === 'mitarbeiter')) {
			appState.adminAuthenticated = true;
			appState.guestAuthenticated = true;
			if (onRoleCallback) onRoleCallback(user.rolle);
		} else if (user?.rolle === 'kollegium') {
			appState.guestAuthenticated = true;
			if (onRoleCallback) onRoleCallback('kollegium');
		} else if (onRoleCallback) {
			onRoleCallback(user?.rolle || '');
		}
	}

	/**
	 * Boot-Restore: stellt eine bestehende Session aus dem HttpOnly-Cookie wieder her.
	 * Ohne diesen Check zeigte jeder Reload den Login-Screen (F5 = UI-Logout),
	 * obwohl die Session serverseitig noch 12h gültig war.
	 */
	async restoreSession() {
		try {
			const res = await fetch('/api/auth/me');
			if (res.ok) {
				this.#applyLogin(await res.json());
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
			// apiFetch statt fetch: POST /login unterliegt seit dem 04.08.2026 der
			// CSRF-Prüfung (api/csrf.go) und braucht den X-CSRF-Token-Header. apiFetch
			// holt das Token notfalls selbst per Bootstrap (/api/csrf-token), es geht also
			// auch beim allerersten Seitenaufruf ohne vorheriges Cookie.
			//
			// timeoutMs bewusst über dem Standard von 10 s für Mutationen: Die Anmeldung
			// wartet auf den IMAP-Server der Schule, und AuthenticateIMAP räumt dem
			// selbst 15 s ein. Mit dem Standardwert hätte apiFetch eine langsame, aber
			// erfolgreiche Anmeldung als "Netzwerk-Timeout" abgebrochen.
			const res = await apiFetch('/login', {
				method: 'POST',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify({ email: this.loginEmail, password: this.loginPassword }),
				timeoutMs: 20000
			});
			if (!res.ok) {
				let msg = 'Login fehlgeschlagen';
				try {
					const d = await res.json();
					msg = d.error || msg;
				} catch {
					try {
						msg = (await res.text()) || msg;
					} catch {}
				}
				throw new Error(msg);
			}
			const user = await res.json();
			this.loginEmail = '';
			this.loginPassword = '';
			this.#applyLogin(user, onRoleCallback);
		} catch (err) {
			const errorMessage = /** @type {any} */ (err).message || String(err);
			this.loginError = errorMessage;
			this.loginPassword = '';
			setTimeout(() => {
				this.loginError = null;
			}, 4000);
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
		toastStore.addToast('Sitzung abgelaufen — bitte neu anmelden.', 'warning');
	}

	handleLogout(onLogoutCallback) {
		// Serverseitig invalidieren (Token-Blacklist + Cookie löschen) — sonst würde
		// der Boot-Restore die Session beim nächsten Reload wiederbeleben.
		// Fire-and-forget: der lokale Zustand wird unabhängig vom Netz geleert.
		fetch('/api/auth/logout', { method: 'POST' }).catch(() => {});
		this.sessionChecked = true;
		this.isLoggedIn = false;
		this.currentUser = null;
		this.loginEmail = '';
		this.loginPassword = '';
		this.loginError = null;
		appState.adminAuthenticated = false;
		appState.guestAuthenticated = false;
		this.stopSessionRefresh();
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

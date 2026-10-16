import { apiFetch } from './apiFetch.js';
import { ladeOffeneSessions, starteSession, brichAb } from './inventurApi.js';

/**
 * Session-Start und -Verwaltung: Scope wählen (global/Signatur/Filter), Inventur
 * starten, mit einer bereits laufenden Session (409) fortsetzen oder sie verwerfen.
 * Aus useInventur herausgelöst — das Scannen selbst braucht diese Auswahl-
 * Mechanik nicht mehr, sobald eine Session aktiv ist.
 * @param {{ onActivate?: () => void }} [opts] onActivate wird aufgerufen, sobald eine
 *   Session aktiv wird (Start oder Fortsetzen) — der Aufrufer nutzt das, um scan-
 *   lokalen Zustand (letzterScan) zu leeren, den dieses Modul nicht selbst besitzt.
 */
export function useInventurlauf(opts = {}) {
	const { onActivate } = opts;

	let status = $state('idle'); // 'idle' | 'active'
	let laufKennung = $state('');
	let bereichArt = $state('global');
	// Die Signatur ist der Text vom Buchruecken, kein Fremdschluessel mehr. Sie wirkt
	// als Praefix: "LMF Deu 7" erfasst auch "LMF Deu 7 / Bie" (Migration 060).
	let gewaehlteSignatur = $state('');
	let signaturen = $state(/** @type {any[]} */ ([]));
	// Filter-Scope: gezielte Teil-Inventur nach Fach und/oder Klasse ("nur Mathe, Kl. 5").
	let gewaehltesFach = $state('');
	let gewaehlterJahrgang = $state('');
	let faecher = $state(/** @type {string[]} */ ([]));
	let offeneLaeufe = $state(/** @type {any[]} */ ([]));
	let stand = $state({ erwartet: 0, erfasst: 0, label: '' });
	let startOffen = $state(false);
	let fehler = $state('');

	async function ladeSignaturen() {
		try {
			const res = await apiFetch('/api/signaturen');
			if (res.ok) signaturen = (await res.json()) || [];
		} catch (e) {
			console.error('Failed to load signaturen', e);
		}
	}

	async function ladeFaecher() {
		try {
			const res = await apiFetch('/api/faecher');
			if (res.ok) faecher = (await res.json()) || [];
		} catch (e) {
			console.error('Failed to load faecher', e);
		}
	}

	async function ladeOffeneLaeufe() {
		offeneLaeufe = await ladeOffeneSessions();
	}

	async function starteInventur() {
		fehler = '';
		const payload = { type: bereichArt };
		if (bereichArt === 'signature') {
			if (!gewaehlteSignatur.trim()) {
				fehler = 'Bitte wähle eine Signatur aus.';
				return;
			}
			payload.signatur = gewaehlteSignatur.trim();
		} else if (bereichArt === 'filter') {
			if (!gewaehltesFach && !gewaehlterJahrgang) {
				fehler = 'Bitte wähle mindestens ein Fach oder eine Klasse.';
				return;
			}
			if (gewaehltesFach) payload.subject = gewaehltesFach;
			if (gewaehlterJahrgang) payload.grade = Number(gewaehlterJahrgang);
		}

		const r = await starteSession(payload);
		if (r.ok) {
			laufKennung = r.data.session_id;
			stand = { erwartet: r.data.erwartet, erfasst: 0, label: r.data.label };
			status = 'active';
			startOffen = false;
			onActivate?.();
			return;
		}
		if (r.status === 409) {
			// Für diesen Bereich läuft bereits eine Inventur — statt sie zu überschreiben
			// (der alte Datenverlust-Bug), die laufende anzeigen und zum Fortsetzen anbieten.
			fehler = 'Für diesen Bereich läuft bereits eine Inventur. Unten fortsetzen oder verwerfen.';
			await ladeOffeneLaeufe();
			startOffen = false;
			return;
		}
		fehler = r.error || 'Fehler beim Starten der Inventur.';
	}

	/** @param {any} session laufende Session aus offeneLaeufe */
	function setzeLaufFort(session) {
		laufKennung = session.session_id;
		stand = { erwartet: session.erwartet, erfasst: session.erfasst, label: session.label };
		fehler = '';
		status = 'active';
		onActivate?.();
	}

	/** @param {any} session */
	async function verwerfeLauf(session) {
		fehler = '';
		await brichAb(session.session_id);
		await ladeOffeneLaeufe();
	}

	// fehler wird an zwei Stellen angezeigt (Start-Modal + Hauptschirm). Ohne diesen
	// Reset blieb eine modal-lokale Meldung (z. B. „Bitte wähle eine Signatur aus.“) nach
	// dem Abbrechen kontextlos als Banner auf dem Hauptschirm stehen.
	function leereFehler() {
		fehler = '';
	}

	/** Aufrufer ist der Abschluss der Session (finishInventory in useInventur). */
	function setzeZurueck() {
		status = 'idle';
		laufKennung = '';
		stand = { erwartet: 0, erfasst: 0, label: '' };
	}

	return {
		get status() {
			return status;
		},
		get laufKennung() {
			return laufKennung;
		},
		get bereichArt() {
			return bereichArt;
		},
		set bereichArt(v) {
			bereichArt = v;
		},
		get gewaehlteSignatur() {
			return gewaehlteSignatur;
		},
		set gewaehlteSignatur(v) {
			gewaehlteSignatur = v;
		},
		get signaturen() {
			return signaturen;
		},
		get gewaehltesFach() {
			return gewaehltesFach;
		},
		set gewaehltesFach(v) {
			gewaehltesFach = v;
		},
		get gewaehlterJahrgang() {
			return gewaehlterJahrgang;
		},
		set gewaehlterJahrgang(v) {
			gewaehlterJahrgang = v;
		},
		get faecher() {
			return faecher;
		},
		get offeneLaeufe() {
			return offeneLaeufe;
		},
		get stand() {
			return stand;
		},
		get startOffen() {
			return startOffen;
		},
		set startOffen(v) {
			startOffen = v;
		},
		get fehler() {
			return fehler;
		},
		leereFehler,
		ladeSignaturen,
		ladeFaecher,
		ladeOffeneLaeufe,
		starteInventur,
		setzeLaufFort,
		verwerfeLauf,
		setzeZurueck
	};
}

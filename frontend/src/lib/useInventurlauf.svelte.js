import { apiFetch } from './apiFetch.js';
import { ladeOffeneSessions, starteSession, brichAb } from './inventurApi.js';

/**
 * Session-Start und -Verwaltung: Scope wählen (global/Signatur/Filter), Inventur
 * starten, mit einer bereits laufenden Session (409) fortsetzen oder sie verwerfen.
 * Aus useInventur herausgelöst — das Scannen selbst braucht diese Auswahl-
 * Mechanik nicht mehr, sobald eine Session aktiv ist.
 * @param {{ onActivate?: () => void }} [opts] onActivate wird aufgerufen, sobald eine
 *   Session aktiv wird (Start oder Fortsetzen) — der Aufrufer nutzt das, um scan-
 *   lokalen Zustand (lastScan) zu leeren, den dieses Modul nicht selbst besitzt.
 */
export function useInventurlauf(opts = {}) {
	const { onActivate } = opts;

	let status = $state('idle'); // 'idle' | 'active'
	let sessionId = $state('');
	let scopeType = $state('global');
	// Die Signatur ist der Text vom Buchruecken, kein Fremdschluessel mehr. Sie wirkt
	// als Praefix: "LMF Deu 7" erfasst auch "LMF Deu 7 / Bie" (Migration 060).
	let selectedSignatur = $state('');
	let signaturen = $state(/** @type {any[]} */ ([]));
	// Filter-Scope: gezielte Teil-Inventur nach Fach und/oder Klasse ("nur Mathe, Kl. 5").
	let selectedFach = $state('');
	let selectedGrade = $state('');
	let faecher = $state(/** @type {string[]} */ ([]));
	let offeneSessions = $state(/** @type {any[]} */ ([]));
	let stats = $state({ erwartet: 0, erfasst: 0, label: '' });
	let showStartModal = $state(false);
	let fehler = $state('');

	async function loadSignaturen() {
		try {
			const res = await apiFetch('/api/signaturen');
			if (res.ok) signaturen = (await res.json()) || [];
		} catch (e) {
			console.error('Failed to load signaturen', e);
		}
	}

	async function loadFaecher() {
		try {
			const res = await apiFetch('/api/faecher');
			if (res.ok) faecher = (await res.json()) || [];
		} catch (e) {
			console.error('Failed to load faecher', e);
		}
	}

	async function loadOffeneSessions() {
		offeneSessions = await ladeOffeneSessions();
	}

	async function startInventory() {
		fehler = '';
		const payload = { type: scopeType };
		if (scopeType === 'signature') {
			if (!selectedSignatur.trim()) {
				fehler = 'Bitte wähle eine Signatur aus.';
				return;
			}
			payload.signatur = selectedSignatur.trim();
		} else if (scopeType === 'filter') {
			if (!selectedFach && !selectedGrade) {
				fehler = 'Bitte wähle mindestens ein Fach oder eine Klasse.';
				return;
			}
			if (selectedFach) payload.subject = selectedFach;
			if (selectedGrade) payload.grade = Number(selectedGrade);
		}

		const r = await starteSession(payload);
		if (r.ok) {
			sessionId = r.data.session_id;
			stats = { erwartet: r.data.erwartet, erfasst: 0, label: r.data.label };
			status = 'active';
			showStartModal = false;
			onActivate?.();
			return;
		}
		if (r.status === 409) {
			// Für diesen Bereich läuft bereits eine Inventur — statt sie zu überschreiben
			// (der alte Datenverlust-Bug), die laufende anzeigen und zum Fortsetzen anbieten.
			fehler = 'Für diesen Bereich läuft bereits eine Inventur. Unten fortsetzen oder verwerfen.';
			await loadOffeneSessions();
			showStartModal = false;
			return;
		}
		fehler = r.error || 'Fehler beim Starten der Inventur.';
	}

	/** @param {any} session laufende Session aus offeneSessions */
	function resumeSession(session) {
		sessionId = session.session_id;
		stats = { erwartet: session.erwartet, erfasst: session.erfasst, label: session.label };
		fehler = '';
		status = 'active';
		onActivate?.();
	}

	/** @param {any} session */
	async function verwerfeSession(session) {
		fehler = '';
		await brichAb(session.session_id);
		await loadOffeneSessions();
	}

	// errorMessage wird an zwei Stellen angezeigt (Start-Modal + Hauptschirm). Ohne diesen
	// Reset blieb eine modal-lokale Meldung (z. B. „Bitte wähle eine Signatur aus.“) nach
	// dem Abbrechen kontextlos als Banner auf dem Hauptschirm stehen.
	function clearError() {
		fehler = '';
	}

	/** Aufrufer ist der Abschluss der Session (finishInventory in useInventur). */
	function resetToIdle() {
		status = 'idle';
		sessionId = '';
		stats = { erwartet: 0, erfasst: 0, label: '' };
	}

	return {
		get status() {
			return status;
		},
		get sessionId() {
			return sessionId;
		},
		get scopeType() {
			return scopeType;
		},
		set scopeType(v) {
			scopeType = v;
		},
		get selectedSignatur() {
			return selectedSignatur;
		},
		set selectedSignatur(v) {
			selectedSignatur = v;
		},
		get signaturen() {
			return signaturen;
		},
		get selectedFach() {
			return selectedFach;
		},
		set selectedFach(v) {
			selectedFach = v;
		},
		get selectedGrade() {
			return selectedGrade;
		},
		set selectedGrade(v) {
			selectedGrade = v;
		},
		get faecher() {
			return faecher;
		},
		get offeneSessions() {
			return offeneSessions;
		},
		get stats() {
			return stats;
		},
		get showStartModal() {
			return showStartModal;
		},
		set showStartModal(v) {
			showStartModal = v;
		},
		get errorMessage() {
			return fehler;
		},
		clearError,
		loadSignaturen,
		loadFaecher,
		loadOffeneSessions,
		startInventory,
		resumeSession,
		verwerfeSession,
		resetToIdle
	};
}

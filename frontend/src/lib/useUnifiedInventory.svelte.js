import { apiFetch } from './apiFetch.js';
import {
	ladeOffeneSessions,
	starteSession,
	scanne,
	schliesseAb,
	brichAb,
	deuteScanErgebnis
} from './inventurApi.js';

/**
 * Hook für die Inventur. Der Fortschritt ist seit dem Session-Umbau an eine
 * session_id gebunden (Backend: inventur_sessions); mehrere Inventuren können parallel
 * laufen, ohne sich zu überschreiben.
 */
export function useUnifiedInventory() {
	let status = $state('idle'); // 'idle' | 'active'
	let sessionId = $state('');
	let scopeType = $state('global');
	let selectedSignatureId = $state('');
	let signatures = $state(/** @type {any[]} */ ([]));
	// Filter-Scope: gezielte Teil-Inventur nach Fach und/oder Klasse ("nur Mathe, Kl. 5").
	let selectedFach = $state('');
	let selectedGrade = $state('');
	let faecher = $state(/** @type {string[]} */ ([]));
	let offeneSessions = $state(/** @type {any[]} */ ([]));
	let stats = $state({ erwartet: 0, erfasst: 0, label: '' });
	let lastScan = $state(/** @type {any} */ (null));
	let barcodeInput = $state('');
	let isScanning = $state(false);
	let showStartModal = $state(false);
	let showFinishModal = $state(false);
	let errorMessage = $state('');

	async function loadSignatures() {
		try {
			const res = await apiFetch('/api/signatures');
			if (res.ok) signatures = await res.json();
		} catch (e) {
			console.error('Failed to load signatures', e);
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
		errorMessage = '';
		const payload = { type: scopeType };
		if (scopeType === 'signature') {
			if (!selectedSignatureId) {
				errorMessage = 'Bitte wähle eine Signatur aus.';
				return;
			}
			payload.signature_id = Number(selectedSignatureId);
		} else if (scopeType === 'filter') {
			if (!selectedFach && !selectedGrade) {
				errorMessage = 'Bitte wähle mindestens ein Fach oder eine Klasse.';
				return;
			}
			if (selectedFach) payload.subject = selectedFach;
			if (selectedGrade) payload.grade = Number(selectedGrade);
		}

		const r = await starteSession(payload);
		if (r.ok) {
			sessionId = r.data.session_id;
			stats = { erwartet: r.data.erwartet, erfasst: 0, label: r.data.label };
			lastScan = null;
			status = 'active';
			showStartModal = false;
			return;
		}
		if (r.status === 409) {
			// Für diesen Bereich läuft bereits eine Inventur — statt sie zu überschreiben
			// (der alte Datenverlust-Bug), die laufende anzeigen und zum Fortsetzen anbieten.
			errorMessage = 'Für diesen Bereich läuft bereits eine Inventur. Unten fortsetzen oder verwerfen.';
			await loadOffeneSessions();
			showStartModal = false;
			return;
		}
		errorMessage = r.error || 'Fehler beim Starten der Inventur.';
	}

	/** @param {any} session laufende Session aus offeneSessions */
	function resumeSession(session) {
		sessionId = session.session_id;
		stats = { erwartet: session.erwartet, erfasst: session.erfasst, label: session.label };
		lastScan = null;
		errorMessage = '';
		status = 'active';
	}

	/** @param {any} session */
	async function verwerfeSession(session) {
		errorMessage = '';
		await brichAb(session.session_id);
		await loadOffeneSessions();
	}

	// errorMessage wird an zwei Stellen angezeigt (Start-Modal + Hauptschirm). Ohne diesen
	// Reset blieb eine modal-lokale Meldung (z. B. „Bitte wähle eine Signatur aus.“) nach
	// dem Abbrechen kontextlos als Banner auf dem Hauptschirm stehen.
	function clearError() {
		errorMessage = '';
	}

	/** @param {string} barcodeVal @param {Function} [focusInput] */
	async function handleScan(barcodeVal, focusInput) {
		if (!barcodeVal.trim() || isScanning) return;
		isScanning = true;
		const barcode = barcodeVal.trim();
		barcodeInput = '';

		try {
			const r = await scanne(sessionId, barcode);
			const ergebnis = deuteScanErgebnis(r, barcode);
			if (ergebnis.zaehlen) stats.erfasst++;
			lastScan = ergebnis.lastScan;
		} catch (e) {
			console.error('Scan fehlgeschlagen:', e);
			lastScan = { success: false, barcode, title: 'Fehler', warnings: ['Netzwerkfehler beim Scannen'] };
		} finally {
			isScanning = false;
			if (focusInput) focusInput();
		}
	}

	async function finishInventory() {
		const r = await schliesseAb(sessionId);
		if (r.ok) {
			alert(`Inventur abgeschlossen! ${r.data.verloren_gemeldet} Bücher wurden als verloren markiert.`);
			resetToIdle();
			await loadOffeneSessions();
		} else {
			alert(r.error || 'Fehler beim Abschließen der Inventur.');
		}
	}

	function resetToIdle() {
		status = 'idle';
		sessionId = '';
		showFinishModal = false;
		stats = { erwartet: 0, erfasst: 0, label: '' };
		lastScan = null;
	}

	function getProgressPercent() {
		if (stats.erwartet === 0) return 0;
		return Math.min(100, Math.round((stats.erfasst / stats.erwartet) * 100));
	}

	return {
		get status() { return status; },
		get scopeType() { return scopeType; },
		set scopeType(v) { scopeType = v; },
		get selectedSignatureId() { return selectedSignatureId; },
		set selectedSignatureId(v) { selectedSignatureId = v; },
		get signatures() { return signatures; },
		get selectedFach() { return selectedFach; },
		set selectedFach(v) { selectedFach = v; },
		get selectedGrade() { return selectedGrade; },
		set selectedGrade(v) { selectedGrade = v; },
		get faecher() { return faecher; },
		get offeneSessions() { return offeneSessions; },
		get stats() { return stats; },
		get lastScan() { return lastScan; },
		get barcodeInput() { return barcodeInput; },
		set barcodeInput(v) { barcodeInput = v; },
		get isScanning() { return isScanning; },
		get showStartModal() { return showStartModal; },
		set showStartModal(v) { showStartModal = v; },
		get showFinishModal() { return showFinishModal; },
		set showFinishModal(v) { showFinishModal = v; },
		get errorMessage() { return errorMessage; },
		clearError,
		loadSignatures,
		loadFaecher,
		loadOffeneSessions,
		startInventory,
		resumeSession,
		verwerfeSession,
		handleScan,
		finishInventory,
		getProgressPercent
	};
}

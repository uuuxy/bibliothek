import { toastStore } from './stores/toastStore.svelte.js';
import { scanne, schliesseAb, deuteScanErgebnis } from './inventurApi.js';
import { useFehlbestand } from './useFehlbestand.svelte.js';
import { useInventurSession } from './useInventurSession.svelte.js';

/**
 * Hook für die Inventur. Der Fortschritt ist seit dem Session-Umbau an eine
 * session_id gebunden (Backend: inventur_sessions); mehrere Inventuren können parallel
 * laufen, ohne sich zu überschreiben.
 */
export function useUnifiedInventory() {
	let lastScan = $state(/** @type {any} */ (null));
	let barcodeInput = $state('');
	let isScanning = $state(false);
	let showFinishModal = $state(false);

	const fb = useFehlbestand();
	// onActivate: Start/Fortsetzen einer Session gehört zu useInventurSession, aber
	// lastScan bleibt hier — es ist Zustand des Scan-Bildschirms, nicht der Session.
	const session = useInventurSession({ onActivate: () => (lastScan = null) });

	/** @param {string} barcodeVal @param {Function} [focusInput] */
	async function handleScan(barcodeVal, focusInput) {
		if (!barcodeVal.trim() || isScanning) return;
		isScanning = true;
		const barcode = barcodeVal.trim();
		barcodeInput = '';

		try {
			const r = await scanne(session.sessionId, barcode);
			const ergebnis = deuteScanErgebnis(r, barcode);
			if (ergebnis.zaehlen) session.stats.erfasst++;
			lastScan = ergebnis.lastScan;
		} catch (e) {
			console.error('Scan fehlgeschlagen:', e);
			lastScan = {
				success: false,
				barcode,
				title: 'Fehler',
				warnings: ['Netzwerkfehler beim Scannen']
			};
		} finally {
			isScanning = false;
			if (focusInput) focusInput();
		}
	}

	async function finishInventory() {
		const r = await schliesseAb(session.sessionId);
		if (r.ok) {
			toastStore.addToast(
				`Inventur abgeschlossen! ${r.data.verloren_gemeldet} Bücher wurden als verloren markiert.`,
				'success'
			);
			// Den Fehlbestand VOR dem Zuruecksetzen sichern.
			//
			// Vorher endete der Abschluss mit einer Zahl im Toast, und die Liste war weg —
			// mit „47 Bücher verloren" kann niemand ins Regal gehen und nachsehen, ob eines
			// davon nur falsch einsortiert war. Rekonstruieren liess sie sich danach auch
			// nicht: Durch die Aussonderung fallen die Exemplare aus dem Scope.
			fb.setzeFehlbestand(r.data.fehlbestand ?? [], session.stats.label);
			// Achtung: raeumt bewusst NICHT den Fehlbestand weg — der Bericht soll den
			// Abschluss ueberleben, sonst waere er im selben Moment wieder verschwunden.
			session.resetToIdle();
			showFinishModal = false;
			lastScan = null;
			await session.loadOffeneSessions();
			// Die gerade beendete Inventur gehört sofort in die Auswahl früherer Läufe —
			// sonst müsste man die Seite neu laden, um sie dort zu finden.
			await fb.loadAbgeschlosseneInventuren();
		} else {
			toastStore.addToast(r.error || 'Fehler beim Abschließen der Inventur.', 'error');
		}
	}

	function getProgressPercent() {
		if (session.stats.erwartet === 0) return 0;
		return Math.min(100, Math.round((session.stats.erfasst / session.stats.erwartet) * 100));
	}

	return {
		get fehlbestand() {
			return fb.fehlbestand;
		},
		get fehlbestandLabel() {
			return fb.fehlbestandLabel;
		},
		fehlbestandSchliessen: fb.fehlbestandSchliessen,
		fehlbestandGefunden: fb.fehlbestandGefunden,
		fehlbestandEndgueltigLoeschen: fb.fehlbestandEndgueltigLoeschen,
		get abgeschlosseneInventuren() {
			return fb.abgeschlosseneInventuren;
		},
		get ladeFruehereLaeuft() {
			return fb.ladeFruehereLaeuft;
		},
		loadAbgeschlosseneInventuren: fb.loadAbgeschlosseneInventuren,
		zeigeFrueherenFehlbestand: fb.zeigeFrueherenFehlbestand,
		get status() {
			return session.status;
		},
		get scopeType() {
			return session.scopeType;
		},
		set scopeType(v) {
			session.scopeType = v;
		},
		get selectedSignatur() {
			return session.selectedSignatur;
		},
		set selectedSignatur(v) {
			session.selectedSignatur = v;
		},
		get signaturen() {
			return session.signaturen;
		},
		get selectedFach() {
			return session.selectedFach;
		},
		set selectedFach(v) {
			session.selectedFach = v;
		},
		get selectedGrade() {
			return session.selectedGrade;
		},
		set selectedGrade(v) {
			session.selectedGrade = v;
		},
		get faecher() {
			return session.faecher;
		},
		get offeneSessions() {
			return session.offeneSessions;
		},
		get stats() {
			return session.stats;
		},
		get lastScan() {
			return lastScan;
		},
		get barcodeInput() {
			return barcodeInput;
		},
		set barcodeInput(v) {
			barcodeInput = v;
		},
		get isScanning() {
			return isScanning;
		},
		get showStartModal() {
			return session.showStartModal;
		},
		set showStartModal(v) {
			session.showStartModal = v;
		},
		get showFinishModal() {
			return showFinishModal;
		},
		set showFinishModal(v) {
			showFinishModal = v;
		},
		get errorMessage() {
			return session.errorMessage;
		},
		clearError: session.clearError,
		loadSignaturen: session.loadSignaturen,
		loadFaecher: session.loadFaecher,
		loadOffeneSessions: session.loadOffeneSessions,
		startInventory: session.startInventory,
		resumeSession: session.resumeSession,
		verwerfeSession: session.verwerfeSession,
		handleScan,
		finishInventory,
		getProgressPercent
	};
}

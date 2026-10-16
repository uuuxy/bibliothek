import { snackbars } from './stores/snackbars.svelte.js';
import { bestaetigen } from './stores/bestaetigung.svelte.js';
import { scanne, schliesseAb, deuteScanErgebnis } from './inventurApi.js';
import { useFehlbestand } from './useFehlbestand.svelte.js';
import { useInventurlauf } from './useInventurlauf.svelte.js';

/**
 * Was die Rückfrage vor dem Abschluss sagt: wie viele Bücher als verloren gebucht werden.
 * @param {number} offen erwartete, nicht gescannte Exemplare
 */
export function abschlussText(offen) {
	if (offen <= 0) {
		return 'Alle Bücher aus dem Bereich dieser Inventur sind gescannt. Es wird keines als verloren markiert.';
	}
	return offen === 1
		? '1 Buch aus dem Bereich dieser Inventur ist nicht gescannt. Es wird unwiderruflich als verloren markiert und ausgesondert.'
		: `${offen} Bücher aus dem Bereich dieser Inventur sind nicht gescannt. Sie werden unwiderruflich als verloren markiert und ausgesondert.`;
}

/**
 * Hook für die Inventur. Der Fortschritt ist seit dem Session-Umbau an eine
 * session_id gebunden (Backend: inventur_sessions); mehrere Inventuren können parallel
 * laufen, ohne sich zu überschreiben.
 */
export function useInventur() {
	let lastScan = $state(/** @type {any} */ (null));
	let barcodeInput = $state('');
	let isScanning = $state(false);

	const fb = useFehlbestand();
	// onActivate: Start/Fortsetzen einer Session gehört zu useInventurlauf, aber
	// lastScan bleibt hier — es ist Zustand des Scan-Bildschirms, nicht der Session.
	const lauf = useInventurlauf({ onActivate: () => (lastScan = null) });

	// Scans, die eintreffen, solange einer am Server ist. Ein Handscanner wartet nicht auf die
	// Antwort; ein verworfener Scan fiele beim Abschluss als Verlust auf.
	/** @type {string[]} */
	const wartend = [];
	// Löst sich auf, sobald kein Scan mehr unterwegs oder eingereiht ist.
	let scansGebucht = Promise.resolve();

	/** @param {string} barcode */
	async function bucheScan(barcode) {
		try {
			const r = await scanne(lauf.sessionId, barcode);
			const ergebnis = deuteScanErgebnis(r, barcode);
			if (ergebnis.zaehlen) lauf.stats.erfasst++;
			lastScan = ergebnis.lastScan;
		} catch (e) {
			console.error('Scan fehlgeschlagen:', e);
			lastScan = {
				success: false,
				barcode,
				title: 'Fehler',
				warnings: ['Netzwerkfehler beim Scannen']
			};
		}
	}

	/**
	 * Reiht den Scan ein und bucht die Reihe ab, einen nach dem anderen.
	 * @param {string} barcodeVal @param {Function} [focusInput]
	 */
	async function handleScan(barcodeVal, focusInput) {
		const barcode = barcodeVal.trim();
		if (!barcode) return;
		barcodeInput = '';
		wartend.push(barcode);
		if (isScanning) return scansGebucht;
		isScanning = true;
		scansGebucht = (async () => {
			try {
				while (wartend.length > 0) {
					await bucheScan(/** @type {string} */ (wartend.shift()));
				}
			} finally {
				isScanning = false;
				if (focusInput) focusInput();
			}
		})();
		return scansGebucht;
	}

	async function schliesseInventurAb() {
		const r = await schliesseAb(lauf.sessionId);
		if (r.ok) {
			snackbars.addToast(
				`Inventur abgeschlossen! ${r.data.verloren_gemeldet} Bücher wurden als verloren markiert.`,
				'success'
			);
			// Den Fehlbestand VOR dem Zuruecksetzen sichern.
			//
			// Vorher endete der Abschluss mit einer Zahl im Toast, und die Liste war weg —
			// mit „47 Bücher verloren" kann niemand ins Regal gehen und nachsehen, ob eines
			// davon nur falsch einsortiert war. Rekonstruieren liess sie sich danach auch
			// nicht: Durch die Aussonderung fallen die Exemplare aus dem Scope.
			fb.setzeFehlbestand(r.data.fehlbestand ?? [], lauf.stats.label);
			// Achtung: raeumt bewusst NICHT den Fehlbestand weg — der Bericht soll den
			// Abschluss ueberleben, sonst waere er im selben Moment wieder verschwunden.
			lauf.resetToIdle();
			lastScan = null;
			await lauf.loadOffeneSessions();
			// Die gerade beendete Inventur gehört sofort in die Auswahl früherer Läufe —
			// sonst müsste man die Seite neu laden, um sie dort zu finden.
			await fb.loadAbgeschlosseneInventuren();
		} else {
			snackbars.addToast(r.error || 'Fehler beim Abschließen der Inventur.', 'error');
		}
	}

	/**
	 * Fragt vor dem Abschluss nach. Bei „nein" läuft die Inventur weiter, und der Fokus geht
	 * zurück ins Scanfeld, damit der nächste Scan gezählt wird.
	 * @param {Function} [focusInput]
	 */
	async function abschliessenNachRueckfrage(focusInput) {
		// Erst wenn jeder Scan gebucht ist, stimmt die Zahl in der Rückfrage.
		await scansGebucht;
		const ja = await bestaetigen({
			titel: 'Inventur abschließen?',
			text: abschlussText(lauf.stats.erwartet - lauf.stats.erfasst),
			aktion: 'Ja, unwiderruflich abschließen',
			gefaehrlich: true
		});
		if (!ja) {
			focusInput?.();
			return;
		}
		await schliesseInventurAb();
	}

	/** @param {any} offene laufende Session aus offeneSessions */
	async function verwerfeSession(offene) {
		await lauf.verwerfeSession(offene);
		// Wie nach dem Abschluss: Die verworfene Inventur steht sofort unter den früheren —
		// als verworfen, nicht als vollständig (Migration 149).
		await fb.loadAbgeschlosseneInventuren();
	}

	function getProgressPercent() {
		if (lauf.stats.erwartet === 0) return 0;
		return Math.min(100, Math.round((lauf.stats.erfasst / lauf.stats.erwartet) * 100));
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
			return lauf.status;
		},
		get scopeType() {
			return lauf.scopeType;
		},
		set scopeType(v) {
			lauf.scopeType = v;
		},
		get selectedSignatur() {
			return lauf.selectedSignatur;
		},
		set selectedSignatur(v) {
			lauf.selectedSignatur = v;
		},
		get signaturen() {
			return lauf.signaturen;
		},
		get selectedFach() {
			return lauf.selectedFach;
		},
		set selectedFach(v) {
			lauf.selectedFach = v;
		},
		get selectedGrade() {
			return lauf.selectedGrade;
		},
		set selectedGrade(v) {
			lauf.selectedGrade = v;
		},
		get faecher() {
			return lauf.faecher;
		},
		get offeneSessions() {
			return lauf.offeneSessions;
		},
		get stats() {
			return lauf.stats;
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
			return lauf.showStartModal;
		},
		set showStartModal(v) {
			lauf.showStartModal = v;
		},
		get errorMessage() {
			return lauf.errorMessage;
		},
		clearError: lauf.clearError,
		loadSignaturen: lauf.loadSignaturen,
		loadFaecher: lauf.loadFaecher,
		loadOffeneSessions: lauf.loadOffeneSessions,
		startInventory: lauf.startInventory,
		resumeSession: lauf.resumeSession,
		verwerfeSession,
		handleScan,
		abschliessenNachRueckfrage,
		getProgressPercent
	};
}

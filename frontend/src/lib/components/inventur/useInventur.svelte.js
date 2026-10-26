import { snackbars } from '../../stores/snackbars.svelte.js';
import { bestaetigen } from '../../stores/bestaetigung.svelte.js';
import { scanne, schliesseAb, deuteScanErgebnis } from './inventurApi.js';
import { useFehlbestand } from './useFehlbestand.svelte.js';
import { useInventurlauf } from './useInventurlauf.svelte.js';
import { spieleErfolgston, spieleFehlerton } from '../../utils/audio.js';

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
 * Hook für die Inventur. Der Fortschritt hängt an einer session_id (inventur_sessions); mehrere
 * Inventuren laufen parallel, ohne sich zu überschreiben.
 */
export function useInventur() {
	let letzterScan = $state(/** @type {any} */ (null));
	let barcodeEingabe = $state('');
	let scanLaeuft = $state(false);

	// onActivate: Start/Fortsetzen einer Session gehört zu useInventurlauf, aber
	// letzterScan bleibt hier — es ist Zustand des Scan-Bildschirms, nicht der Session.
	const lauf = useInventurlauf({ onActivate: () => (letzterScan = null) });
	const fb = useFehlbestand(lauf.nichtGeladen);

	// Scans, die eintreffen, solange einer am Server ist. Ein Handscanner wartet nicht auf die
	// Antwort; ein verworfener Scan fiele beim Abschluss als Verlust auf.
	/** @type {string[]} */
	const wartend = [];
	// Löst sich auf, sobald kein Scan mehr unterwegs oder eingereiht ist.
	let scansGebucht = Promise.resolve();

	/** @param {string} barcode */
	async function bucheScan(barcode) {
		try {
			const r = await scanne(lauf.laufKennung, barcode);
			const ergebnis = deuteScanErgebnis(r, barcode);
			if (ergebnis.zaehlen) lauf.stand.erfasst++;
			letzterScan = ergebnis.lastScan;
			// Der Ton folgt wie an der Theke der Buchung: Gezählt klingt nach Erfolg, auch mit
			// Hinweis; was nicht zählt, gibt den Fehlerton, damit man auf den Bildschirm sieht.
			(ergebnis.zaehlen ? spieleErfolgston : spieleFehlerton)();
		} catch (e) {
			console.error('Scan fehlgeschlagen:', e);
			spieleFehlerton();
			letzterScan = {
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
	async function verarbeiteScan(barcodeVal, focusInput) {
		const barcode = barcodeVal.trim();
		if (!barcode) return;
		barcodeEingabe = '';
		wartend.push(barcode);
		if (scanLaeuft) return scansGebucht;
		scanLaeuft = true;
		scansGebucht = (async () => {
			try {
				while (wartend.length > 0) {
					await bucheScan(/** @type {string} */ (wartend.shift()));
				}
			} finally {
				scanLaeuft = false;
				if (focusInput) focusInput();
			}
		})();
		return scansGebucht;
	}

	async function schliesseInventurAb() {
		const r = await schliesseAb(lauf.laufKennung);
		if (r.ok) {
			snackbars.addToast(
				`Inventur abgeschlossen! ${r.data.verloren_gemeldet} Bücher wurden als verloren markiert.`,
				'success'
			);
			// Die Liste vor dem Zurücksetzen sichern: Mit der Zahl allein geht niemand ins Regal, und
			// nach der Aussonderung fallen die Exemplare aus dem Bereich der Inventur.
			fb.setzeFehlbestand(r.data.fehlbestand ?? [], lauf.stand.label);
			// Lässt den Fehlbestand stehen: Der Bericht soll den Abschluss überleben.
			lauf.setzeZurueck();
			letzterScan = null;
			await lauf.ladeOffeneLaeufe();
			// Die gerade beendete Inventur gehört sofort in die Auswahl früherer Läufe —
			// sonst müsste man die Seite neu laden, um sie dort zu finden.
			await fb.ladeFruehereInventuren();
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
			text: abschlussText(lauf.stand.erwartet - lauf.stand.erfasst),
			aktion: 'Ja, unwiderruflich abschließen',
			gefaehrlich: true
		});
		if (!ja) {
			focusInput?.();
			return;
		}
		await schliesseInventurAb();
	}

	/** @param {any} offene laufende Session aus offeneLaeufe */
	async function verwerfeLauf(offene) {
		await lauf.verwerfeLauf(offene);
		// Wie nach dem Abschluss: Die verworfene Inventur steht sofort unter den früheren —
		// als verworfen, nicht als vollständig (Migration 149).
		await fb.ladeFruehereInventuren();
	}

	function fortschrittProzent() {
		if (lauf.stand.erwartet === 0) return 0;
		return Math.min(100, Math.round((lauf.stand.erfasst / lauf.stand.erwartet) * 100));
	}

	return /** @satisfies {object} */ ({
		get fehlbestand() {
			return fb.fehlbestand;
		},
		get fehlbestandBezeichnung() {
			return fb.fehlbestandBezeichnung;
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
		ladeFruehereInventuren: fb.ladeFruehereInventuren,
		zeigeFrueherenFehlbestand: fb.zeigeFrueherenFehlbestand,
		get status() {
			return lauf.status;
		},
		get bereichArt() {
			return lauf.bereichArt;
		},
		set bereichArt(v) {
			lauf.bereichArt = v;
		},
		get gewaehlteSignatur() {
			return lauf.gewaehlteSignatur;
		},
		set gewaehlteSignatur(v) {
			lauf.gewaehlteSignatur = v;
		},
		get signaturen() {
			return lauf.signaturen;
		},
		get gewaehltesFach() {
			return lauf.gewaehltesFach;
		},
		set gewaehltesFach(v) {
			lauf.gewaehltesFach = v;
		},
		get gewaehlterJahrgang() {
			return lauf.gewaehlterJahrgang;
		},
		set gewaehlterJahrgang(v) {
			lauf.gewaehlterJahrgang = v;
		},
		get faecher() {
			return lauf.faecher;
		},
		get offeneLaeufe() {
			return lauf.offeneLaeufe;
		},
		nichtGeladen: lauf.nichtGeladen,
		get stand() {
			return lauf.stand;
		},
		get letzterScan() {
			return letzterScan;
		},
		get barcodeEingabe() {
			return barcodeEingabe;
		},
		set barcodeEingabe(v) {
			barcodeEingabe = v;
		},
		get scanLaeuft() {
			return scanLaeuft;
		},
		get startOffen() {
			return lauf.startOffen;
		},
		set startOffen(v) {
			lauf.startOffen = v;
		},
		get fehler() {
			return lauf.fehler;
		},
		leereFehler: lauf.leereFehler,
		ladeSignaturen: lauf.ladeSignaturen,
		ladeFaecher: lauf.ladeFaecher,
		ladeOffeneLaeufe: lauf.ladeOffeneLaeufe,
		starteInventur: lauf.starteInventur,
		setzeLaufFort: lauf.setzeLaufFort,
		verwerfeLauf,
		verarbeiteScan,
		abschliessenNachRueckfrage,
		fortschrittProzent
	});
}

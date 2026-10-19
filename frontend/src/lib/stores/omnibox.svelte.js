// Zustand und Abläufe der Omnibox an der Theke: Eingabe, Suche, Scan und seine Antwort.

import { apiFetch, apiClient } from '../apiFetch.js';
import { spieleFehlerton } from '../audio.js';
import { buchBarcodes } from './buchBarcodes.svelte.js';
import { normalisiereScan, ordneScanEin } from '../scanEinordnen.js';
import { snackbars } from './snackbars.svelte.js';
import { uiStore } from './uiStore.svelte.js';
import { fehlertext } from '../utils/fehlertext.js';
import { escapeIstBelegt } from '../components/ui/escapeSchliesst.js';
import { erzeugeOmniboxRueckmeldung } from './omniboxRueckmeldung.svelte.js';
import { erzeugeOmniboxSuche } from './omniboxSuche.svelte.js';
import { erzeugeOmniboxAntwort } from './omniboxAntwort.svelte.js';
import { erzeugeOmniboxOhneNetz } from './omniboxOhneNetz.svelte.js';

// Meldungen gehen an die gemeinsamen snackbars (SnackbarAnzeige.svelte), damit die Theke keinen
// zweiten Meldungsweg mit eigener Optik hat. Auf Modulebene, weil sie nichts aus dem Store
// einfängt (sonarjs S7721).
/**
 * @param {string} message
 * @param {import('./snackbars.svelte.js').ToastTyp} [type='success']
 */
function melde(message, type = 'success') {
	snackbars.addToast(message, type);
}

export function erzeugeOmniboxStore() {
	// Der Leser, der an der Theke steht, ob Schüler oder Kollege (unterscheidbar an `art`). Es
	// gibt nur diesen einen Platz, damit keine Stelle zwischen zweien wählen muss.
	let aktiverLeser = $state(/** @type {any} */ (null));
	// Schnellrückgabe: Jeder Scan nimmt nur zurück. Sie und ein geladener Leser schließen
	// einander aus; ohne Leser bekommt der Server keinen und leiht nie aus.
	let schnellrueckgabe = $state(false);
	let eingabe = $state('');

	const rueckmeldung = erzeugeOmniboxRueckmeldung();
	const {
		leuchteBildschirm,
		wackle,
		leuchteRand,
		zeigeFehlerBanner,
		clearFehlerBanner,
		nenneNichtGebucht
	} = rueckmeldung;

	let letzteFremdrueckgabe = $state(/** @type {any} */ (null));
	// Gehört das ausgeliehene Buch zu einer anderen Auflage als die der Klasse, steht der Hinweis
	// bis zum nächsten Scan über dem Konto, wie die Fremdrückgabe (OmniboxThekeHinweise).
	let letzterAuflagenHinweis = $state(/** @type {any} */ (null));
	let vormerkungHinweis = $state(/** @type {{titel?: string, user?: string} | null} */ (null));
	// Die abholbereiten Vormerkungen des geladenen Lesers, damit an der Theke ins Abholfach
	// gegriffen wird, solange er davorsteht. Jeder Scan, der einen Leser lädt, überschreibt sie
	// (?? []), sonst bliebe der Hinweis des vorigen stehen.
	let abholbereit = $state(/** @type {{titel: string, bereitgestellt_bis?: string}[]} */ ([]));
	/**
	 * Geräte-Scan mit Zubehör: Der Server unterbricht mit type=geraet_check und
	 * wartet auf die Bestätigung der Checkliste. Hier liegt die Anfrage, bis der
	 * Dialog bestätigt (erneuter Versand mit confirmed_checklist) oder abbricht.
	 * overrideBlock: Der Scan war schon ein Übergehen (Sperr-Dialog) — die Bestätigung
	 * schickt es mit, sonst hielte die Sperre den zweiten Versand wieder auf.
	 * @type {{query: string, geraet: any, overrideBlock: boolean} | null}
	 */
	let prueflisteAnfrage = $state(null);
	/**
	 * Der Sperr-Dialog. art ist das Merkmal X-Sperre: „uebergehbar" (Hinweis — einmalig
	 * übergehen) oder „leser" (Sperre am Leser — aufheben).
	 * @type {{message: string, query: string, art: 'uebergehbar' | 'leser'} | null}
	 */
	let sperrHinweis = $state(null);

	// Kamerascanner-Status
	let kameraOffen = $state(false);
	let kamera = $state(/** @type {any} */ (null));

	const suche = erzeugeOmniboxSuche({
		leseEingabe: () => eingabe,
		setzeEingabe: (text) => (eingabe = text),
		fuehreAktionAus,
		melde
	});
	const { verarbeiteEingabe, waehleTreffer } = suche;

	const { verarbeiteAktionsErgebnis } = erzeugeOmniboxAntwort({
		leseLeser: () => aktiverLeser,
		// Die Schnellrückgabe endet, weil sie und ein geladener Leser einander ausschließen.
		ladeLeser: (leser, fach) => {
			schnellrueckgabe = false;
			aktiverLeser = leser;
			abholbereit = fach ?? [];
		},
		leseSchnellrueckgabe: () => schnellrueckgabe,
		setzeFremdrueckgabe: (hinweis) => (letzteFremdrueckgabe = hinweis),
		setzeAuflagenHinweis: (hinweis) => (letzterAuflagenHinweis = hinweis),
		setzeVormerkungHinweis: (hinweis) => (vormerkungHinweis = hinweis),
		setzePrueflisteAnfrage: (anfrage) => (prueflisteAnfrage = anfrage),
		leuchteBildschirm,
		leuchteRand,
		wackle,
		melde
	});

	const ohneNetz = erzeugeOmniboxOhneNetz({
		leseLeser: () => aktiverLeser,
		beendeSchnellrueckgabe: () => (schnellrueckgabe = false),
		entferneLeser: () => (aktiverLeser = null),
		leuchteBildschirm,
		zeigeFehlerBanner,
		melde
	});
	const { schnappschuss, verarbeiteVersandfehler } = ohneNetz;

	let belegt = $derived(!!(aktiverLeser || suche.trefferlisteOffen));

	// Jeder Zeitgeber hat einen Griff und wird vor dem Neuplanen verworfen: Einer, der die Seite
	// überlebt, feuert nach dem Abbau von jsdom ins Leere und reißt den ganzen Testlauf mit.
	/** @type {ReturnType<typeof setTimeout> | null} */
	let fokusTimer = null;

	// Die Reihe der Scans. Ein Handscanner wartet nicht auf die Antwort: Ein Scan, der
	// eintrifft, solange eine Buchung läuft, wartet hier und wird danach gebucht. Verworfen
	// bliebe er unbemerkt, in der Schnellrückgabe bliebe das Buch verliehen.
	/**
	 * @typedef {{ q: string, reloadProfileCb?: (() => void) | null, overrideBlock: boolean,
	 *   confirmedChecklist: boolean, absicht: 'ausleihe' | 'rueckgabe' | null }} Auftrag
	 */
	let buchungLaeuft = false;
	/** @type {Auftrag[]} */
	const wartend = [];
	// Die Nummer, die gerade gebucht wird.
	let laufendeNummer = '';
	// Löst sich auf, sobald kein Scan mehr unterwegs ist oder wartet.
	let reiheGebucht = Promise.resolve();

	// Stoppt jeden laufenden Zeitgeber dieses Stores, beim Verlassen der Seite (Omnibox.svelte)
	// und beim Abräumen der Tests: Danach greift kein Rückruf auf eine Seite, die es nicht mehr gibt.
	function stoppeZeitgeber() {
		rueckmeldung.stoppeZeitgeber();
		suche.stoppeZeitgeber();
		if (fokusTimer) clearTimeout(fokusTimer);
		fokusTimer = null;
	}

	// Liest die Fehlermeldung aus einer nicht-ok HTTP-Antwort. Bei einem
	// Sperr-/Überfälligkeits-403 wird sperrHinweis gesetzt und BLOCK_ALERT geworfen,
	// sonst ein generischer Fehler. Wirft in jedem Fall.
	async function verarbeiteAktionsFehler(res, q) {
		let errStr = await res.text();
		try {
			const fehlerAntwort = JSON.parse(errStr);
			if (fehlerAntwort.error) errStr = fehlerAntwort.error;
		} catch (e) {
			console.debug('Fehlerantwort war kein JSON, nutze Rohtext:', e);
		}

		// Der Dialog hängt am Merkmal des Servers, nicht am Wortlaut der Meldung, der je Sperre
		// anders lautet (omniboxSperrDialog.test.js).
		const art = res.headers.get('X-Sperre');
		if (res.status === 403 && (art === 'uebergehbar' || art === 'leser')) {
			sperrHinweis = { message: errStr, query: q, art };
			throw new Error('BLOCK_ALERT');
		}

		throw new Error(errStr || 'Aktion fehlgeschlagen');
	}

	// Die Schnellrückgabe ein- oder ausschalten. Beim Einschalten weicht, wer geladen oder
	// ohne Netz gemerkt ist: Sonst ginge das nächste freie Buch an ihn.
	/** @param {boolean} an */
	function schalteSchnellrueckgabe(an) {
		verwirfWartende();
		schnellrueckgabe = an;
		if (!an) return;
		aktiverLeser = null;
		abholbereit = [];
		ohneNetz.offlineAusweis = '';
		letzteFremdrueckgabe = null;
		letzterAuflagenHinweis = null;
	}

	// Escape beendet die Schnellrückgabe. Schließt es gerade einen Hinweis (Vormerkung),
	// bleibt sie an: Der Stapel ist dann noch nicht durch.
	function escapeGedrueckt() {
		if (!escapeIstBelegt()) schalteSchnellrueckgabe(false);
	}

	/**
	 * Löst einen ohne Netz gemerkten Ausweis auf, bevor das nächste Buch hinausgeht: Ohne
	 * Leser-Kennung läse der Server den Scan als Rückgabe.
	 *
	 * Entschieden wird an der Wirkung eines Ausweisscans, der nichts bucht, nicht an
	 * `navigator.onLine`, das bei einem Ausfall des Servers weiter „online" meldet. Lädt er die
	 * Person, geht das Buch normal hinaus. Scheitert der Versand, bleibt der Merker, und das Buch
	 * geht mit `ausweis_barcode` in die Warteschlange. Antwortet der Server ohne Person, ist
	 * unklar, wem das Buch gehört: Es wird nicht gebucht, und der Merker fällt.
	 *
	 * @param {string} q der rohe Scan
	 * @param {(() => void) | null} [reloadProfileCb]
	 * @returns {Promise<boolean>} false = dieser Scan wurde bewusst nicht ausgeführt
	 */
	async function merkerAufgeloest(q, reloadProfileCb) {
		if (!ohneNetz.offlineAusweis) return true;
		// Ein Ausweis selbst darf durch — er ist der Ausweg und ersetzt den Merker.
		if (ordneScanEin(q, buchBarcodes.istBuch).art === 'ausweis') {
			ohneNetz.offlineAusweis = '';
			return true;
		}

		const gemerkt = ohneNetz.offlineAusweis;
		let res;
		try {
			res = await apiClient.post('/api/action', {
				query: gemerkt,
				idempotency_key: crypto.randomUUID()
			});
		} catch (e) {
			// Keine Verbindung — der erwartete Fall, kein Fehler. Der Merker bleibt.
			console.warn('Ausweis-Merker nicht auflösbar, bleibt stehen:', e);
			return true;
		}

		try {
			if (!res.ok) await verarbeiteAktionsFehler(res, gemerkt); // wirft immer
			verarbeiteAktionsErgebnis(await res.json(), reloadProfileCb, gemerkt);
		} catch (e) {
			verarbeiteAntwortfehler(e);
		}

		ohneNetz.offlineAusweis = '';
		if (aktiverLeser?.id) return true;

		leuchteBildschirm('error');
		spieleFehlerton();
		zeigeFehlerBanner(
			`Der ohne Netz gemerkte Ausweis \u201e${gemerkt}\u201c ließ sich nicht laden. Dieser Scan wurde NICHT gebucht — ` +
				`bitte den Ausweis erneut scannen.`
		);
		return false;
	}

	// Eine Antwort kam an — nichts ist offline. Was ihre Auswertung wirft, wird gezeigt.
	/** @param {unknown} e */
	function verarbeiteAntwortfehler(e) {
		// Wer scannt, sieht auf das Buch: Ein Scan, der nichts gebucht hat, ist zu hören. Nach
		// einem gescheiterten Ausweis steht der Leser davor noch an der Theke.
		leuchteBildschirm('error');
		spieleFehlerton();
		if (e instanceof Error && e.message === 'BLOCK_ALERT') return;
		// Nur das Banner an der Omnibox, kein Toast dazu: Derselbe Fehler stünde sonst zweimal da.
		zeigeFehlerBanner(`Fehler: ${fehlertext(e)}`);
	}

	// Haupt-Scan-Aktion. `absicht` nur, wenn der Aufrufer sie kennt (gibZurueck); ein Scan
	// im Feld lässt sie offen.
	/**
	 * @param {Event | null} e
	 * @param {(() => void) | null} [reloadProfileCb]
	 * @param {boolean} [overrideBlock]
	 * @param {boolean} [confirmedChecklist]
	 * @param {'ausleihe' | 'rueckgabe' | null} [absicht]
	 */
	async function fuehreAktionAus(
		e,
		reloadProfileCb,
		overrideBlock = false,
		confirmedChecklist = false,
		absicht = null
	) {
		if (e) e.preventDefault();
		if (suche.trefferlisteOffen && suche.markierterTreffer >= 0) {
			waehleTreffer(suche.markierterTreffer, null);
			return;
		}

		// Enter ohne Auswahl nimmt den einen Schüler, wenn die Liste genau ihn zeigt: Der rohe Text
		// liefe am Server in die Titelsuche, die nur Bücher kennt. Bei Namensgleichen rät Enter
		// nicht, und ein Buch braucht den Rückruf onSelectBook, den es hier nicht gibt.
		if (
			suche.trefferlisteOffen &&
			suche.suchTreffer.students.length === 1 &&
			suche.suchTreffer.books.length === 0
		) {
			waehleTreffer(0, null);
			return;
		}

		// Einmal vereinheitlichen, gleich am Anfang: Dann gilt derselbe Wert für den Versand, den
		// Schnappschuss und die Einordnung ohne Netz.
		const q = normalisiereScan(eingabe);
		if (!q) return;

		eingabe = '';
		suche.trefferlisteOffen = false;

		/** @type {Auftrag} */
		const auftrag = { q, reloadProfileCb, overrideBlock, confirmedChecklist, absicht };
		if (buchungLaeuft) {
			// Dieselbe Nummer, bevor sie gebucht ist, ist der Doppelscan: Er fällt weg. Gebucht
			// gäbe er das eben geliehene Buch gleich wieder zurück.
			if (!istUnterwegs(q)) wartend.push(auftrag);
			return reiheGebucht;
		}
		buchungLaeuft = true;
		reiheGebucht = bucheReihe(auftrag);
		return reiheGebucht;
	}

	/** Wird die Nummer gerade gebucht oder wartet sie? @param {string} q */
	const istUnterwegs = (q) => q === laufendeNummer || wartend.some((w) => w.q === q);

	// Bucht den Scan und danach, der Reihe nach, was inzwischen gescannt wurde. Jeder Scan
	// findet vor, was der vorige hinterlassen hat: Nach einem Ausweis gehen die Bücher an
	// diesen Leser.
	//
	// Weiter geht es nur nach einer glatten Buchung. Scheitert ein Scan, kann er ein Ausweis
	// gewesen sein, und die folgenden Bücher gingen an den Leser davor; nach einer Rückfrage
	// entscheidet ein Mensch, was gilt. Was dann noch wartet, wird nicht gebucht und genannt.
	/** @param {Auftrag} erster */
	async function bucheReihe(erster) {
		try {
			/** @type {Auftrag | undefined} */
			let auftrag = erster;
			while (auftrag) {
				laufendeNummer = auftrag.q;
				const glatt = await bucheScan(auftrag);
				laufendeNummer = '';
				if (rueckfrageOffen()) {
					// Die Rückfrage hat jetzt die Tastatur. Was vom nächsten Scan schon im Feld
					// steht, stünde sonst vor dem übernächsten.
					eingabe = '';
				}
				if (!glatt || rueckfrageOffen()) break;
				auftrag = wartend.shift();
			}
		} catch (e) {
			// Unerwartet abgebrochen: Ob der laufende Scan gebucht ist, weiß hier niemand.
			console.error('Buchung abgebrochen:', e);
			leuchteRand('red');
			verarbeiteAntwortfehler(e);
		} finally {
			laufendeNummer = '';
			verwirfWartende();
			buchungLaeuft = false;
			scanfeldWiederScharfstellen();
		}
	}

	// Die wartenden Scans werden nicht gebucht. Das Banner nennt sie, bis der nächste Scan
	// kommt; der Ton sagt es dem, der nicht hinsieht.
	function verwirfWartende() {
		if (wartend.length === 0) return;
		const nummern = wartend.splice(0).map((w) => w.q);
		leuchteBildschirm('error');
		spieleFehlerton();
		leuchteRand('red');
		nenneNichtGebucht(nummern);
	}

	// Bucht einen Scan. Das Ergebnis sagt, ob er glatt durchging: Leser geladen, gebucht oder
	// für später abgelegt.
	/**
	 * @param {Auftrag} auftrag
	 * @returns {Promise<boolean>}
	 */
	async function bucheScan({ q, reloadProfileCb, overrideBlock, confirmedChecklist, absicht }) {
		letzteFremdrueckgabe = null;
		letzterAuflagenHinweis = null;
		clearFehlerBanner();

		// Der Schnappschuss entsteht vor allem Weiteren, damit Escape oder „Theke leeren" die
		// Absicht des Scans nicht mehr ändern.
		let eintrag = schnappschuss(q, crypto.randomUUID(), absicht);

		// Steht ein ohne Netz gemerkter Ausweis, gehört ihm das nächste Buch: erst auflösen, dann
		// buchen (merkerAufgeloest).
		if (ohneNetz.offlineAusweis) {
			if (!(await merkerAufgeloest(q, reloadProfileCb))) return false;
			// Die Auflösung hat die Person geladen: Sie gehört in den Schnappschuss, der Merker
			// nicht mehr. Trüge der Eintrag beides, entschiede beim Nachbuchen die Reihenfolge der
			// Felder, wem das Buch gehört.
			if (eintrag.ausweis_barcode && aktiverLeser?.id) {
				const ohneMerker = { ...eintrag };
				delete ohneMerker.ausweis_barcode;
				eintrag = { ...ohneMerker, leser_id: aktiverLeser.id, art: absicht ?? 'ausleihe' };
			}
		}

		// Scheitert der Versand, hat der Server nichts gesehen: Der Schnappschuss geht in die
		// Warteschlange. Kam eine Antwort an, wird ihr Fehler gezeigt und nicht eingereiht; nach
		// Ablauf seines Idempotenz-Caches (24 h) buchte der Server sonst ein zweites Mal.
		//
		// Rot nur im Fehlerfall: Sonst überschriebe es das Grün des Erfolgs und das Orange der
		// Fremdrückgabe, und niemand sähe mehr auf die Farbe.
		let res;
		try {
			res = await apiClient.post('/api/action', {
				query: q,
				active_leser_id: eintrag.leser_id ?? undefined,
				confirmed_checklist: confirmedChecklist,
				override_block: overrideBlock,
				idempotency_key: eintrag.id
			});
		} catch (e) {
			leuchteRand('red');
			return verarbeiteVersandfehler(e, eintrag);
		}

		try {
			if (!res.ok) {
				await verarbeiteAktionsFehler(res, q); // wirft immer
			}
			const daten = await res.json();
			return verarbeiteAktionsErgebnis(daten, reloadProfileCb, q, overrideBlock);
		} catch (e) {
			leuchteRand('red');
			verarbeiteAntwortfehler(e);
			return false;
		}
	}

	/**
	 * Gibt dem Scanfeld nach einer Buchung den Fokus: Ein Handscanner tippt blind, ohne Fokus
	 * landen seine Zeichen im Nichts. Der $effect in Omnibox.svelte hilft hier nicht, er läuft
	 * nur ohne geladenen Leser (`!belegt`).
	 *
	 * Nicht, solange eine Rückfrage offen ist oder die Kamera scannt: Der Fokussprung machte den
	 * Dialog wegtippbar.
	 */
	function scanfeldWiederScharfstellen() {
		if (entscheidungOffen()) return;
		fokussiereScanfeld();
	}

	// Eine Rückfrage, die ein Mensch beantwortet: Sperre, Vormerkung, Zubehör.
	const rueckfrageOffen = () => !!(sperrHinweis || vormerkungHinweis || prueflisteAnfrage);

	// Ein Scan bei offener Rückfrage ist nicht gebucht, und sie bleibt stehen (scanOhneFokus.js).
	// Wer auf das Buch sieht, hört es.
	function scanBeiRueckfrage() {
		leuchteBildschirm('error');
		spieleFehlerton();
	}

	// Eine offene Rückfrage oder die laufende Kamera: Beide behalten ihre Eingabe.
	const entscheidungOffen = () => kameraOffen || rueckfrageOffen();

	// Darf ein Zeichen, das ohne Fokus getippt wird, ins Scanfeld (scanOhneFokus.js)? Auch
	// während einer Buchung: Der Scan wird eingereiht (bucheReihe). Nicht, solange eine
	// Rückfrage offen ist oder die Kamera scannt.
	const scanfeldBereit = () => !entscheidungOffen();

	// Die einzige Stelle, an der ein Zeitgeber dieses Stores die Seite anfasst (stoppeZeitgeber).
	// Sie wartet, bis Svelte das Profil neu gezeichnet hat, und fragt beim Feuern noch einmal:
	// Ein zweiter Scan kann inzwischen den Sperr-Dialog geöffnet haben, der den Fokus behält.
	function fokussiereScanfeld() {
		if (fokusTimer) clearTimeout(fokusTimer);
		fokusTimer = setTimeout(() => {
			fokusTimer = null;
			if (entscheidungOffen()) return;
			document.getElementById('omnibox-input')?.focus();
		}, 50);
	}

	// Auch der Wechsel zur Ausleihe gibt dem Scanfeld den Fokus: Sonst bliebe er auf dem Knopf
	// der Seitenleiste, und der nächste Scan liefe ins Leere (e2e/scanner-fokus-menue.spec.js).
	uiStore.beimWechselZurTheke = scanfeldWiederScharfstellen;

	// Nach dem Zusammenführen zweier Datensätze (Leserakte → onMerged) hängt die Theke auf das
	// Ziel um. Die Kennung steht sofort, damit kein Scan dazwischen die gelöschte erwischt; Name
	// und Sperren kommen nach.
	/** @param {string} zielId */
	async function uebernimmZusammengefuehrt(zielId) {
		aktiverLeser = { id: zielId };
		const res = await apiFetch(`/api/schueler/${zielId}`);
		// Inzwischen ein anderer Ausweis? Dann gehört das Nachgeladene niemandem mehr.
		if (!res.ok || aktiverLeser?.id !== zielId) return;
		aktiverLeser = await res.json();
	}

	// „Buch zurückgeben" aus der Akte: derselbe Weg wie ein Scan, aber mit bekannter Absicht.
	// Ohne Netz reihte ein geladener Leser den Scan sonst als Ausleihe ein.
	/** @param {string} barcode @param {(() => void) | null} [reloadProfileCb] */
	function gibZurueck(barcode, reloadProfileCb = null) {
		eingabe = barcode;
		return fuehreAktionAus(null, reloadProfileCb, false, false, 'rueckgabe');
	}

	return /** @satisfies {object} */ ({
		get aktiverLeser() {
			return aktiverLeser;
		},
		/** Die ohne Netz gemerkte Ausweisnummer ("" = keine). Nur die Nummer, kein Name. */
		get offlineAusweis() {
			return ohneNetz.offlineAusweis;
		},
		// Von außen gesetzt (Escape, Akte schließen, Theke leeren): Was noch wartet, wurde für
		// den Zustand davor gescannt und wird nicht gebucht.
		set offlineAusweis(v) {
			verwirfWartende();
			ohneNetz.offlineAusweis = v;
		},
		set aktiverLeser(v) {
			verwirfWartende();
			if (v) schnellrueckgabe = false;
			aktiverLeser = v;
		},
		get schnellrueckgabe() {
			return schnellrueckgabe;
		},
		get abholbereit() {
			return abholbereit;
		},
		get eingabe() {
			return eingabe;
		},
		set eingabe(v) {
			eingabe = v;
		},
		get randLeuchten() {
			return rueckmeldung.randLeuchten;
		},
		get bildschirmLeuchten() {
			return rueckmeldung.bildschirmLeuchten;
		},
		get letzteFremdrueckgabe() {
			return letzteFremdrueckgabe;
		},
		set letzteFremdrueckgabe(v) {
			letzteFremdrueckgabe = v;
		},
		get letzterAuflagenHinweis() {
			return letzterAuflagenHinweis;
		},
		get wackelt() {
			return rueckmeldung.wackelt;
		},
		get fehler() {
			return rueckmeldung.fehler;
		},
		get vormerkungHinweis() {
			return vormerkungHinweis;
		},
		set vormerkungHinweis(v) {
			vormerkungHinweis = v;
		},
		get prueflisteAnfrage() {
			return prueflisteAnfrage;
		},
		set prueflisteAnfrage(v) {
			prueflisteAnfrage = v;
		},
		get sperrHinweis() {
			return sperrHinweis;
		},
		set sperrHinweis(v) {
			sperrHinweis = v;
		},
		get kameraOffen() {
			return kameraOffen;
		},
		set kameraOffen(v) {
			kameraOffen = v;
		},
		get kamera() {
			return kamera;
		},
		set kamera(v) {
			kamera = v;
		},
		get trefferlisteOffen() {
			return suche.trefferlisteOffen;
		},
		set trefferlisteOffen(v) {
			suche.trefferlisteOffen = v;
		},
		get suchTreffer() {
			return suche.suchTreffer;
		},
		set suchTreffer(v) {
			suche.suchTreffer = v;
		},
		get markierterTreffer() {
			return suche.markierterTreffer;
		},
		set markierterTreffer(v) {
			suche.markierterTreffer = v;
		},
		get trefferAnzahl() {
			return suche.trefferAnzahl;
		},
		get belegt() {
			return belegt;
		},

		// Exportierte Methoden
		stoppeZeitgeber,
		fokussiereScanfeld,
		scanfeldBereit,
		rueckfrageOffen,
		scanBeiRueckfrage,
		leuchteBildschirm,
		wackle,
		leuchteRand,
		melde,
		verarbeiteEingabe,
		uebernimmZusammengefuehrt,
		schalteSchnellrueckgabe,
		escapeGedrueckt,
		gibZurueck,
		waehleTreffer,
		fuehreAktionAus
	});
}

export const omniboxStore = erzeugeOmniboxStore();

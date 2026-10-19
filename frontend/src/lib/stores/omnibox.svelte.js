// Die Omnibox an der Theke: der Zustand der Theke und die Fassade, über die Bauteile und Tests
// gehen. Hier stehen außerdem die Eingabe (fuehreAktionAus), Rückfrage und Fokus, die Auswertung
// einer Fehlerantwort, Zusammenführen und gibZurueck. Die übrigen Sachen stehen in den Teilen
// omnibox<Teil>.svelte.js, die nur diese Datei lädt (frontend-hygiene-omnibox-teile.test.js).

import { apiFetch } from '../apiFetch.js';
import { spieleFehlerton } from '../audio.js';
import { normalisiereScan } from '../scanEinordnen.js';
import { snackbars } from './snackbars.svelte.js';
import { uiStore } from './uiStore.svelte.js';
import { fehlertext } from '../utils/fehlertext.js';
import { escapeIstBelegt } from '../components/ui/escapeSchliesst.js';
import { erzeugeOmniboxRueckmeldung } from './omniboxRueckmeldung.svelte.js';
import { erzeugeOmniboxSuche } from './omniboxSuche.svelte.js';
import { erzeugeOmniboxAntwort } from './omniboxAntwort.svelte.js';
import { erzeugeOmniboxOhneNetz } from './omniboxOhneNetz.svelte.js';
import { erzeugeOmniboxBuchung } from './omniboxBuchung.svelte.js';
import { erzeugeOmniboxReihe } from './omniboxReihe.svelte.js';

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
	// Eine Rückfrage, die ein Mensch beantwortet: Sperre, Vormerkung, Zubehör.
	const rueckfrageOffen = () => !!(sperrHinweis || vormerkungHinweis || prueflisteAnfrage);

	let kameraOffen = $state(false);
	let kamera = $state(/** @type {any} */ (null));

	const suche = erzeugeOmniboxSuche({
		leseEingabe: () => eingabe,
		setzeEingabe: (text) => (eingabe = text),
		fuehreAktionAus,
		melde
	});

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

	const { bucheScan, verwirfLaufendeAntworten } = erzeugeOmniboxBuchung({
		leseLeser: () => aktiverLeser,
		leseMerker: () => ohneNetz.offlineAusweis,
		vergissMerker: () => (ohneNetz.offlineAusweis = ''),
		// Die Hinweiszeilen über dem Konto stehen bis zum nächsten Scan.
		leereHinweiszeilen: () => {
			letzteFremdrueckgabe = null;
			letzterAuflagenHinweis = null;
		},
		schnappschuss,
		verarbeiteVersandfehler,
		verarbeiteAktionsFehler,
		verarbeiteAktionsErgebnis,
		verarbeiteAntwortfehler,
		leuchteBildschirm,
		leuchteRand,
		zeigeFehlerBanner,
		clearFehlerBanner
	});

	const { reiheEin, verwirfWartende } = erzeugeOmniboxReihe({
		bucheScan,
		rueckfrageOffen,
		ungespeichert: () => !!uiStore.verlassenSperre?.(),
		leereEingabe: () => (eingabe = ''),
		verarbeiteAntwortfehler,
		scanfeldWiederScharfstellen,
		leuchteBildschirm,
		leuchteRand,
		zeigeFehlerBanner,
		nenneNichtGebucht
	});

	let belegt = $derived(!!(aktiverLeser || suche.trefferlisteOffen));

	// Jeder Zeitgeber hat einen Griff und wird vor dem Neuplanen verworfen: Einer, der die Seite
	// überlebt, feuert nach dem Abbau von jsdom ins Leere und reißt den ganzen Testlauf mit.
	/** @type {ReturnType<typeof setTimeout> | null} */
	let fokusTimer = null;

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
			suche.waehleTreffer(suche.markierterTreffer, null);
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
			suche.waehleTreffer(0, null);
			return;
		}

		// Einmal vereinheitlichen, gleich am Anfang: Dann gilt derselbe Wert für den Versand, den
		// Schnappschuss und die Einordnung ohne Netz.
		const q = normalisiereScan(eingabe);
		if (!q) return;

		eingabe = '';
		suche.trefferlisteOffen = false;

		return reiheEin({ q, reloadProfileCb, overrideBlock, confirmedChecklist, absicht });
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
		get letzteFremdrueckgabe() {
			return letzteFremdrueckgabe;
		},
		set letzteFremdrueckgabe(v) {
			letzteFremdrueckgabe = v;
		},
		get letzterAuflagenHinweis() {
			return letzterAuflagenHinweis;
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
		get belegt() {
			return belegt;
		},
		// Suche und Rückmeldung sind eigene Teile; der Aufruf nennt den Teil.
		suche,
		rueckmeldung,

		stoppeZeitgeber,
		fokussiereScanfeld,
		scanfeldBereit,
		rueckfrageOffen,
		scanBeiRueckfrage,
		uebernimmZusammengefuehrt,
		schalteSchnellrueckgabe,
		verwirfLaufendeAntworten,
		escapeGedrueckt,
		gibZurueck,
		fuehreAktionAus
	});
}

export const omniboxStore = erzeugeOmniboxStore();

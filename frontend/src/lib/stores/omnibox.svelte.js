// stores/omnibox.svelte.js
// Status- und Logikverwaltung für die Omnibox (Svelte 5 Runes)

import { apiFetch, apiClient } from '../apiFetch.js';
import { spieleErfolgston, spieleFehlerton } from '../audio.js';
import { enqueueOfflineAction } from '../offlineWarteschlange.js';
import { offlineSync } from './offlineSync.svelte.js';
import { buchBarcodes } from './buchBarcodes.svelte.js';
import { normalisiereScan, ordneScanEin } from '../scanEinordnen.js';
import { snackbars } from './snackbars.svelte.js';
import { uiStore } from './uiStore.svelte.js';
import { fehlertext } from '../utils/fehlertext.js';
import { escapeIstBelegt } from '../components/ui/escapeSchliesst.js';

// Name des Vorbesitzers bei einer Fremdrückgabe (Schüler bevorzugt, dann Lehrer).
function formatVorbesitzerName(daten) {
	if (daten.vorbesitzer) {
		return `${daten.vorbesitzer.vorname} ${daten.vorbesitzer.nachname}`;
	}
	if (daten.vorbesitzer_user) {
		return `${daten.vorbesitzer_user.vorname} ${daten.vorbesitzer_user.nachname}`;
	}
	return 'unbekannt';
}

// Name und Klasse eines Lesers; das Kollegium hat keine Klasse.
/** @param {{vorname: string, nachname: string, klasse?: string}} leser */
function leserMitKlasse(leser) {
	const name = `${leser.vorname} ${leser.nachname}`;
	return leser.klasse ? `${name} (${leser.klasse})` : name;
}

// Geht an den globalen snackbars (SnackbarAnzeige.svelte in App.svelte). Vorher
// hielt die Omnibox einen eigenen Toast-Zustand samt eigenem Markup und eigenem
// Auto-Dismiss — zwei Meldungswege mit zwei Optiken für dieselbe Sache.
//
// Steht auf Modulebene, weil die Funktion nichts aus dem Store einfängt: Sonst würde
// bei jedem erzeugeOmniboxStore() eine neue, identische Funktion angelegt (sonarjs S7721).
/**
 * @param {string} message
 * @param {import('./snackbars.svelte.js').ToastTyp} [type='success']
 */
function melde(message, type = 'success') {
	snackbars.addToast(message, type);
}

export function erzeugeOmniboxStore() {
	// aktiverLeser ist der LESER, der gerade an der Theke steht — Schüler oder Kollege,
	// unterscheidbar an `art`. Ein zweiter Platz für Lehrkräfte ist mit Migration 125
	// weggefallen; solange es zwei gab, musste jede Stelle beide abfragen und sich für
	// eine entscheiden.
	let aktiverLeser = $state(/** @type {any} */ (null));
	// Der ohne Netz gescannte Ausweis — nur die NUMMER, kein Name, keine Ausleihen.
	// Aufloesen kann ihn nur der Server; bis dahin tragen die folgenden Buecher sie mit
	// (offlineWarteschlange: `ausweis_barcode`, die Nachbuch-Tuer kennt das Feld).
	let offlineAusweis = $state('');
	// Schnellrückgabe: Jeder Scan nimmt nur zurück. Sie und ein geladener Leser schließen
	// einander aus; ohne Leser bekommt der Server keinen und leiht nie aus.
	let schnellrueckgabe = $state(false);
	let eingabe = $state('');

	let randLeuchten = $state('');
	let bildschirmLeuchten = $state(''); // "success" | "error" | ""
	let letzteFremdrueckgabe = $state(/** @type {any} */ (null));
	// Gemischte Auflagen (docs/OFFEN.md 4.18, Stufe 5): der Hinweis der letzten Ausleihe, bis
	// zum nächsten Scan — wie die Fremdrückgabe, als Zeile über dem Konto (OmniboxThekeHinweise).
	let letzterAuflagenHinweis = $state(/** @type {any} */ (null));
	let wackelt = $state(false);
	let fehler = $state('');
	/** @type {any} Auto-Dismiss-Timer des Inline-Fehlerbanners */
	let meldungsZeitgeber = null;
	let vormerkungHinweis = $state(/** @type {{titel?: string, user?: string} | null} */ (null));
	// Abholfach-Hinweis zum AKTIVEN Schüler (Betreiber-Entscheidung 01.09.2026):
	// beim Ausweis-Scan liefert der Server die abholbereiten Vormerkungen mit —
	// die Mitarbeiterin greift ins Abholfach, solange der Schüler vor ihr steht.
	// Wird bei JEDEM Setzen von aktiverLeser mit überschrieben (?? []), damit
	// kein Hinweis eines vorherigen Schülers stehen bleibt (bekannte Bugklasse
	// „nie zurückgesetzter UI-State").
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

	// Such-Status
	let suchVerzoegerung = $state(/** @type {any} */ (null));
	let trefferlisteOffen = $state(false);
	let suchTreffer = $state({ students: [], books: [], studentsTotal: 0, booksTotal: 0 });
	let markierterTreffer = $state(-1);
	let trefferAnzahl = $derived(suchTreffer.students.length + suchTreffer.books.length);

	let belegt = $derived(!!(aktiverLeser || trefferlisteOffen));

	// UI Feedback-Methoden
	//
	// Jeder dieser Zeitgeber bekommt ein HANDLE und wird vor dem Neuplanen verworfen —
	// dasselbe Muster wie in actions/tastaturNavigation.js und actions/tooltip.js. Der Grund ist
	// nicht Ordnungsliebe: Ein Timer, der die Seite überlebt, feuert ins Leere. In der
	// Testumgebung ist „ins Leere" ein Absturz — nach dem Abbau von jsdom gibt es kein
	// `document` mehr, und der Rückruf reisst den GANZEN Lauf mit („Unhandled Errors:
	// document is not defined"), obwohl jeder einzelne Test grün ist. Genau so stand die
	// CI am 16.09.2026 rot bei 697 grünen Tests; am 11.09.2026 war es tastaturNavigation.
	/** @type {ReturnType<typeof setTimeout> | null} */
	let bildschirmZeitgeber = null;
	/** @type {ReturnType<typeof setTimeout> | null} */
	let shakeTimer = null;
	/** @type {ReturnType<typeof setTimeout> | null} */
	let randZeitgeber = null;
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

	function leuchteBildschirm(type) {
		bildschirmLeuchten = type;
		if (bildschirmZeitgeber) clearTimeout(bildschirmZeitgeber);
		bildschirmZeitgeber = setTimeout(() => {
			bildschirmZeitgeber = null;
			bildschirmLeuchten = '';
		}, 300);
	}

	function wackle() {
		wackelt = true;
		if (shakeTimer) clearTimeout(shakeTimer);
		shakeTimer = setTimeout(() => {
			shakeTimer = null;
			wackelt = false;
		}, 500);
	}

	function leuchteRand(color) {
		randLeuchten = color;
		if (randZeitgeber) clearTimeout(randZeitgeber);
		randZeitgeber = setTimeout(() => {
			randZeitgeber = null;
			randLeuchten = '';
		}, 1000);
	}

	// Stoppt JEDEN laufenden Zeitgeber dieses Stores. Die Theke ruft es beim Verlassen der
	// Seite (Omnibox.svelte, onDestroy), die Tests beim Abräumen — danach kann kein
	// Rückruf mehr auf eine Seite greifen, die es nicht mehr gibt.
	function stoppeZeitgeber() {
		for (const t of [
			bildschirmZeitgeber,
			shakeTimer,
			randZeitgeber,
			fokusTimer,
			meldungsZeitgeber,
			suchVerzoegerung
		]) {
			if (t) clearTimeout(t);
		}
		bildschirmZeitgeber = null;
		shakeTimer = null;
		randZeitgeber = null;
		fokusTimer = null;
		meldungsZeitgeber = null;
		suchVerzoegerung = null;
	}

	// Das Fehlerbanner hat zwei Teile. Was ein Scan meldet, verschwindet nach 6 s von selbst.
	// Die Nummern der Scans, die eine Reihe nicht gebucht hat, bleiben stehen, bis der nächste
	// Scan kommt: Die Rückfrage, vor der sie gescannt wurden, kann länger offen sein.
	let bannerMeldung = '';
	/** @type {string[]} */
	let nichtGebucht = [];

	function zeichneBanner() {
		const nummern = nichtGebucht.map((nummer) => `„${nummer}“`).join(', ');
		const rest = nummern ? `Gescannt und NICHT gebucht: ${nummern} — bitte erneut scannen.` : '';
		fehler = [bannerMeldung, rest].filter(Boolean).join(' · ');
	}

	function stoppeBannerUhr() {
		if (meldungsZeitgeber) {
			clearTimeout(meldungsZeitgeber);
			meldungsZeitgeber = null;
		}
	}

	// Zeigt die Meldung im Fehlerbanner an der Omnibox. Eine laufende Uhr wird verworfen,
	// damit ein neuer Fehler die volle Anzeigedauer bekommt.
	function zeigeFehlerBanner(message) {
		stoppeBannerUhr();
		bannerMeldung = message;
		meldungsZeitgeber = setTimeout(() => {
			meldungsZeitgeber = null;
			bannerMeldung = '';
			zeichneBanner();
		}, 6000);
		zeichneBanner();
	}

	// Leert das Fehlerbanner ganz, auch die Nummern der nicht gebuchten Scans.
	function clearFehlerBanner() {
		stoppeBannerUhr();
		bannerMeldung = '';
		nichtGebucht = [];
		zeichneBanner();
	}

	// Such-Logik. suchLauf zählt jeden gestarteten Abruf — nur die Antwort des zuletzt
	// gestarteten darf die Trefferliste schreiben.
	let suchLauf = 0;

	/** Trefferliste und Dropdown leeren — eine Liste darf ihren Suchtext nie überleben. */
	function verwirfTreffer() {
		suchTreffer = { students: [], books: [], studentsTotal: 0, booksTotal: 0 };
		trefferlisteOffen = false;
		markierterTreffer = -1;
	}

	function verarbeiteEingabe() {
		clearTimeout(suchVerzoegerung);
		if (!eingabe.trim()) {
			// Auch hier hochzählen: Ein noch laufender Abruf darf die eben geleerte
			// Liste nicht wieder füllen.
			suchLauf++;
			trefferlisteOffen = false;
			suchTreffer = { students: [], books: [], studentsTotal: 0, booksTotal: 0 };
			return;
		}
		suchVerzoegerung = setTimeout(async () => {
			if (!eingabe.trim()) return;
			// Sequenznummer wie im bestellStore: Der Entprell-Timer verwirft nur NOCH NICHT
			// gestartete Abrufe — zwei gleichzeitig laufende beantwortet der Server in
			// beliebiger Reihenfolge. Ohne diese Prüfung tauschte eine verspätete Antwort
			// die Trefferliste unter dem sichtbaren Suchtext aus, und der nächste Klick
			// (bzw. Enter) buchte auf den FALSCHEN Schüler — ohne Rückfrage
			// (Fund 31.08.2026).
			const seq = ++suchLauf;
			try {
				const res = await apiFetch(`/api/search?q=${encodeURIComponent(eingabe.trim())}`);
				if (seq !== suchLauf) return;
				if (!res.ok) {
					// Dieselbe Gefahr wie oben, nur über den anderen Ausgang (Sweep
					// „verschluckte Fehlantwort", 06.09.2026): Bis hierher blieb bei einem
					// Fehlschlag die Liste des VORIGEN Suchtextes stehen — samt offenem
					// Dropdown. „Müller" getippt, Treffer da; „Schmidt" getippt, Abruf
					// scheitert (500, oder 429 vom Rate-Limiter) — und der nächste Klick
					// oder Enter buchte auf Müller, während im Feld Schmidt stand.
					verwirfTreffer();
					melde('Suche fehlgeschlagen — bitte erneut versuchen.', 'error');
					return;
				}
				const ergebnisse = await res.json();
				suchTreffer = {
					students: ergebnisse.students || [],
					books: ergebnisse.books || [],
					studentsTotal: ergebnisse.students_total ?? (ergebnisse.students || []).length,
					booksTotal: ergebnisse.books_total ?? (ergebnisse.books || []).length
				};
				trefferlisteOffen = suchTreffer.students.length > 0 || suchTreffer.books.length > 0;
				markierterTreffer = -1;
			} catch (err) {
				if (seq !== suchLauf) return;
				// Netzfehler: ebenfalls verwerfen, aber ohne Toast — im WLAN-Loch käme
				// bei jedem Tastendruck einer, und das Offline-Overlay sagt es bereits.
				verwirfTreffer();
				console.error('Suche fehlgeschlagen:', err);
			}
		}, 300);
	}

	// Dropdown-Auswahl
	//
	// Ein Leser wird über seine ID geladen, nicht über die Ausweisnummer: Ein Kollege aus
	// der Selbstanmeldung hat keine, und bis zum 16.09.2026 schickte der Klick auf ihn
	// eine leere Eingabe los — sichtbar passierte gar nichts. Aufgelöst wird die ID auf
	// demselben Weg wie ein Scan (POST /api/action), damit der Abholfach-Hinweis und die
	// Sperrprüfung an beiden Wegen gleich sind.
	function waehleTreffer(index, onSelectBook) {
		const { students, books } = suchTreffer;
		if (index < students.length) {
			const gewaehlterLeser = students[index];
			eingabe = `leser:${gewaehlterLeser.id}`;
			trefferlisteOffen = false;
			fuehreAktionAus(null, null); // Ohne Event
		} else {
			const buch = books[index - students.length];
			eingabe = '';
			trefferlisteOffen = false;
			if (onSelectBook) onSelectBook(buch);
		}
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

		// Der Dialog hängt am Merkmal des Servers, nicht am Wortlaut (omniboxSperrDialog.test.js).
		// Bis zum 13.09.2026 entschieden hier die Wörter „Sperre", „Sperr-Automatik" und
		// „überfällig": Die Schadens-Sperre traf keins, und bei der System-Sperre hing es am
		// Sperrgrund, den eine Helferin gar nicht zu sehen bekommt.
		const art = res.headers.get('X-Sperre');
		if (res.status === 403 && (art === 'uebergehbar' || art === 'leser')) {
			sperrHinweis = { message: errStr, query: q, art };
			throw new Error('BLOCK_ALERT');
		}

		throw new Error(errStr || 'Aktion fehlgeschlagen');
	}

	// Fremdes Buch in aktiver Sitzung: NUR beim Vorbesitzer ausgebucht — bewusst kein
	// automatisches Umbuchen. Info ohne Unterbrechung; erneuter Scan leiht es an die
	// aktive Sitzung aus.
	function verarbeiteFremdrueckgabe(daten) {
		leuchteBildschirm('warning');
		spieleFehlerton();
		leuchteRand('orange');
		const prevName = formatVorbesitzerName(daten);
		letzteFremdrueckgabe = { vorbesitzerName: prevName };
		const aktiv = aktiverLeser?.vorname;
		const nachsatz = aktiv ? ` Erneut scannen, um es an ${aktiv} auszuleihen.` : '';
		melde(
			`„${daten.book.titel}" war auf ${prevName} verbucht — dort zurückgegeben.${nachsatz}`,
			'warning'
		);
	}

	// Verarbeitet eine Rückgabe-Antwort (fremd/normal, Vormerkung, Session-Fortführung).
	function verarbeiteRueckgabe(daten, reloadProfileCb) {
		if (daten.fremdrueckgabe) {
			verarbeiteFremdrueckgabe(daten);
		} else {
			leuchteBildschirm('success');
			spieleErfolgston();
			leuchteRand('green');
			const titel = daten.book?.titel ?? daten.geraet?.modellname;
			// In der Schnellrückgabe erscheint kein Konto: Die Meldung nennt, bei wem das Buch war.
			melde(
				schnellrueckgabe && daten.student
					? `„${titel}" zurückgegeben, war bei ${leserMitKlasse(daten.student)}.`
					: `„${titel}" erfolgreich zurückgegeben.`
			);
		}
		if (daten.has_vormerkung) {
			vormerkungHinweis = {
				titel: daten.vormerkung_titel || daten.book?.titel,
				user: daten.vormerkung_user
			};
		}
		if (reloadProfileCb) reloadProfileCb();

		if (daten.student && !aktiverLeser && !schnellrueckgabe) {
			aktiverLeser = daten.student;
			abholbereit = daten.abholbereit ?? [];
		}
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
		offlineAusweis = '';
		letzteFremdrueckgabe = null;
		letzterAuflagenHinweis = null;
	}

	// Escape beendet die Schnellrückgabe. Schließt es gerade einen Hinweis (Vormerkung),
	// bleibt sie an: Der Stapel ist dann noch nicht durch.
	function escapeGedrueckt() {
		if (!escapeIstBelegt()) schalteSchnellrueckgabe(false);
	}

	// Die Rückkehr eines abgeschriebenen Buches (#597): Der Server sagt in `message`, was
	// er erledigt hat (Forderung storniert — grün), und in `aufsicht_informieren`, was ein
	// Mensch noch tun muss (der Bescheid liegt bei der Schulaufsicht — Warnung). Zwei
	// Kanäle, weil es zwei Dinge sind: Eine offene Aufgabe in einer grünen Erfolgsmeldung
	// wird überlesen.
	function zeigeRueckkehrHinweise(daten, { ohneMeldung = false } = {}) {
		if (!ohneMeldung && daten.message) melde(daten.message, 'success');
		if (daten.aufsicht_informieren) melde(daten.aufsicht_informieren, 'warning');
	}

	// Ein Leser ist geladen: Er steht jetzt an der Theke, sein Abholfach kommt mit.
	function verarbeiteLeser(daten) {
		schnellrueckgabe = false;
		aktiverLeser = daten.student;
		abholbereit = daten.abholbereit ?? [];
		leuchteBildschirm('success');
		spieleErfolgston();
		leuchteRand('green');
		// Bei einem Kollegen sagen, was der Scan bedeutet: Die Karte daneben zeigt
		// keine Klasse, und ohne Ansage sieht ein geladener Kollege aus wie ein
		// Schüler mit fehlender Angabe.
		if (daten.student?.art && daten.student.art !== 'schueler') {
			melde(`Geladen: ${daten.student.vorname} ${daten.student.nachname}`);
		}
	}

	function verarbeiteAusleihe(daten, reloadProfileCb) {
		// Gebucht ist die Ausleihe in jedem Fall, deshalb ihr Ton. Gehört das Buch zu einer
		// anderen Auflage als die, die die Klasse schon hat, blitzt es wie bei der
		// Fremdrückgabe, und die Zeile über dem Konto sagt, welche (4.18, Stufe 5).
		letzterAuflagenHinweis = daten.auflagen_hinweis ?? null;
		leuchteBildschirm(letzterAuflagenHinweis ? 'warning' : 'success');
		spieleErfolgston('ausleihe');
		leuchteRand(letzterAuflagenHinweis ? 'orange' : 'green');
		melde(
			`„${daten.book?.titel ?? daten.geraet?.modellname}" ausgeliehen an ${aktiverLeser?.vorname}.`
		);
		// Der Schüler hatte ein ANDERES Exemplar reserviert und ein Freihand-Exemplar
		// genommen — das reservierte muss zurück ins Regal, sonst bleibt es im Fach liegen.
		if (daten.regalfreigabe_barcode) {
			melde(
				`Hinweis: Reserviertes Exemplar ${daten.regalfreigabe_barcode} zurück ins Regal räumen.`,
				'warning'
			);
		}
		// Das Exemplar war abgeschrieben und ist beim Scan zurückgeholt worden (die
		// Ausleihe folgte im selben Zug): Was dabei mit der Forderung geschah, gehört
		// gesagt — sonst sieht die Theke nur „ausgeliehen".
		zeigeRueckkehrHinweise(daten);
		if (reloadProfileCb) reloadProfileCb();
	}

	// Verarbeitet die erfolgreiche Server-Antwort je nach data.type. overrideBlock ist das
	// Übergehen, mit dem der Scan geschickt wurde — die Zubehör-Liste reicht es weiter.
	//
	// Das Ergebnis sagt, ob danach ein wartender Scan gebucht werden darf: nur nach einer
	// glatten Buchung. Eine Hinweiszeile über dem Konto (Fremdrückgabe, andere Auflage) steht
	// bis zum nächsten Scan und verlangt einen Blick; der wartende Scan nähme sie gleich weg.
	/** @returns {boolean} */
	function verarbeiteAktionsErgebnis(daten, reloadProfileCb, q = '', overrideBlock = false) {
		if (daten.type === 'student') {
			verarbeiteLeser(daten);
			return true;
		}
		if (daten.type === 'ausleihe') {
			verarbeiteAusleihe(daten, reloadProfileCb);
			return !daten.auflagen_hinweis;
		}
		if (daten.type === 'rueckgabe') {
			verarbeiteRueckgabe(daten, reloadProfileCb);
			return !daten.fremdrueckgabe;
		}
		if (daten.type === 'info') {
			leuchteBildschirm('success');
			spieleErfolgston();
			leuchteRand('green');
			melde(daten.message, 'success');
			zeigeRueckkehrHinweise(daten, { ohneMeldung: true });
			if (reloadProfileCb) reloadProfileCb();
			return true;
		}
		if (daten.type === 'geraet_check') {
			// Kein Fehler, kein Erfolg: Der Scan wartet auf die Zubehör-Bestätigung.
			prueflisteAnfrage = { query: q, geraet: daten.geraet, overrideBlock };
		} else if (daten.type === 'search_results') {
			wackle();
			melde('Bitte wähle ein Ergebnis aus der Liste.', 'warning');
		}
		return false;
	}

	// Der Schnappschuss VOM SCAN: Absicht, Person, Zeitpunkt und Idempotenz-Schlüssel, bevor
	// die Anfrage hinausgeht (OFFEN.md 2.2, Commit 1). Bis zum 15.09.2026 las der Offline-Pfad
	// Person und Absicht erst NACH der hängenden Anfrage (Timeout 10 s): Escape oder „Theke
	// leeren" in dieser Zeit, und das Buch ging als Rückgabe ohne Person in die Warteschlange.
	//
	// Ist ein Schüler geladen, ist der Scan eine AUSLEIHE — bis zum Rasterdurchgang am
	// 06.09.2026 wurde jeder Offline-Scan als Rückgabe abgelegt, und der Server las das
	// Schweigen als Rückgabe: Das Buch war schon draußen, die Rückgabe scheiterte, der
	// Eintrag flog aus der Warteschlange. Das Kind hatte das Buch, das System sagte
	// „verfügbar".
	//
	// Die Absicht kann der Aufrufer vorgeben: „Buch zurückgeben" in der Akte ist eine
	// Rückgabe, auch mit geladenem Schüler (Commit 3). Ohne Vorgabe gilt: Schüler ODER
	// Ein geladener Leser → Ausleihe, sonst Rückgabe. Bis zum 15.09.2026 entschied nur
	// der Schüler, und ein Buch mit geladener Lehrkraft wurde als Rückgabe eingereiht.
	/**
	 * @param {string} q
	 * @param {string} idempotencyKey
	 * @param {'ausleihe' | 'rueckgabe' | null} absicht
	 * @returns {import('../offlineWarteschlange.js').OfflineEintrag}
	 */
	function schnappschuss(q, idempotencyKey, absicht) {
		// Uhr-Anker neben dem Zeitstempel: `performance.now()` laeuft gleichmaessig und
		// springt nicht, die Wanduhr schon — und zwar gern genau dann, wenn das Netz
		// zurueckkommt, also zwischen Scan und Versand. Der Sync rechnet daraus den
		// Scan-Zeitpunkt neu, solange der Eintrag aus demselben Seitenaufruf stammt
		// (offlineWarteschlange.js, OfflineEintrag).
		// Ohne geladene Person, aber mit Ausweis-Merker: Das ist eine AUSLEIHE an die
		// Person hinter dem Merker. Ohne beides bleibt es eine Rueckgabe.
		const merker = !aktiverLeser?.id && offlineAusweis ? offlineAusweis : '';
		return {
			id: idempotencyKey,
			art: absicht ?? (aktiverLeser?.id || merker ? 'ausleihe' : 'rueckgabe'),
			barcode: q,
			leser_id: aktiverLeser?.id ?? null,
			...(merker ? { ausweis_barcode: merker } : {}),
			gescannt_am: Date.now(),
			mono: Math.round(performance.now()),
			ursprung: Math.round(performance.timeOrigin ?? 0)
		};
	}

	// Speichert einen Scan offline.
	//
	// Bis zum 16.09.2026 stand hier `if (!barcode.startsWith('B-')) { Toast; return; }` —
	// jede andere Form, also Littera-Ziffern, `LMF-` und JEDER Ausweis, bekam einen nackten
	// „Netzwerkfehler" und war weg. Das ist Wort fuer Wort der Punkt, den Stufe 1 haette
	// beheben sollen; gefunden hat es erst der Nachweis von Hand am Stack, weil jeder
	// Offline-Testfall `B-10234` scannte.
	//
	// Eingeordnet wird jetzt wie am Server (scanEinordnen.js, Zwilling von
	// omnibox.go), mit der Barcode-Liste dieses Rechners als Nachschlagewerk.
	// Gebucht wird unter der NUMMER aus der Einordnung, nicht unter dem Aufdruck: Ein
	// Littera-Etikett traegt im Strichcode eine EAN-13, der Server kennt nur die Nummer
	// darin.
	/**
	 * @param {import('../offlineWarteschlange.js').OfflineEintrag} eintrag
	 * @returns {Promise<boolean>} true: gespeichert oder als Ausweis gemerkt
	 */
	async function speichereOfflineAktion(eintrag) {
		const einordnung = ordneScanEin(eintrag.barcode, buchBarcodes.istBuch);
		if (einordnung.art === 'ausweis') {
			merkeOfflineAusweis(einordnung.nummer);
			return true;
		}
		if (einordnung.art !== 'buch') {
			// „unklar" sperrt die Zuordnung bis zum naechsten eindeutigen Ausweis
			// (Entscheidung vom 13.09.2026): Der Merker faellt, damit das naechste Buch
			// nicht einer Person zugeschrieben wird, bei der niemand mehr sicher ist.
			if (einordnung.art === 'unklar') offlineAusweis = '';
			verwirfOfflineScan(einordnung);
			return false;
		}
		eintrag = { ...eintrag, barcode: einordnung.nummer };
		try {
			await enqueueOfflineAction(eintrag);
		} catch (err) {
			// Laut, nicht „gespeichert" (Commit 5): Ohne Warteschlange ist der Scan verloren —
			// das Buch muss zurück ins Regal oder auf den Zettel.
			console.error('Offline-Eintrag nicht gespeichert:', err);
			leuchteBildschirm('error');
			spieleFehlerton();
			zeigeFehlerBanner(
				`NICHT gespeichert: „${eintrag.barcode}“ konnte nicht auf diesem Rechner abgelegt werden — Buch zurücklegen und den Vorgang notieren.`
			);
			offlineSync.zaehleNeu();
			return false;
		}
		offlineSync.zaehleNeu();
		leuchteBildschirm('warning');
		spieleErfolgston();
		melde(`Offline: Aktion für „${eintrag.barcode}“ gespeichert.`, 'warning');
		return true;
	}

	// Was die Theke ohne Netz NICHT annimmt — und warum der Bediener das erfahren muss.
	//
	// Ein nackter „Netzwerkfehler" war die schlechteste aller Auskuenfte: Er sagte weder,
	// dass der Scan verworfen wurde, noch warum, noch was jetzt zu tun ist. Wer ihn sah,
	// durfte annehmen, es habe trotzdem geklappt.
	// Der Ausweis ohne Netz: Die Theke merkt sich die NUMMER und sonst nichts. Ein Name
	// stuende hier nur, wenn Personendaten auf dem Rechner laegen — und genau das soll
	// nicht sein (Entscheidung vom 13.09.2026). Der Server loest die Nummer beim
	// Nachbuchen auf.
	/** @param {string} nummer */
	function merkeOfflineAusweis(nummer) {
		schnellrueckgabe = false;
		offlineAusweis = nummer;
		// Die zuvor geladene Person weicht: Sonst zeigte die Theke einen Namen, waehrend
		// die folgenden Buecher an eine ANDERE Person gingen.
		aktiverLeser = null;
		leuchteBildschirm('warning');
		spieleErfolgston();
		melde(
			`Ohne Netz gemerkt: Ausweis „${nummer}". Die folgenden Bücher werden ihm zugeordnet.`,
			'warning'
		);
	}

	/**
	 * Der gefaehrlichste Fall des Offline-Betriebs: Ein ohne Netz gemerkter Ausweis steht
	 * noch, und das naechste Buch wird gescannt.
	 *
	 * Der Merker traegt nur die NUMMER; der Online-Weg braucht die Leser-Kennung. Ginge das
	 * Buch einfach hinaus, schickte es `active_leser_id` gar nicht mit — und der Server
	 * liest das Schweigen als RUECKGABE. Aus einer Ausleihe an A-10001 wuerde still eine
	 * Rueckgabe.
	 *
	 * Entschieden wird an der WIRKUNG, nicht an einer Vorhersage. Bis zum 16.09.2026 fragte
	 * diese Stelle `navigator.onLine` — und lag damit ausgerechnet im haeufigsten Ausfall
	 * falsch: Faellt der SERVER aus (Neustart, Reverse Proxy, Docker), bleibt das WLAN da.
	 * `navigator.onLine` sagte dann „online", der Ausweis war aber gerade offline gemerkt
	 * worden; die Theke bat um einen erneuten Ausweisscan, der wieder nur offline ankam,
	 * und bat erneut. Eine Schleife, in der sich KEIN Buch mehr auf eine Karte buchen liess
	 * (Rasterdurchgang 16.09.2026).
	 *
	 * Deshalb wird der Merker zuerst aufgeloest — mit genau dem Ausweisscan, den die Theke
	 * sonst vom Bediener verlangt hat. Ein Ausweisscan ist harmlos: Er bucht nichts, er
	 * laedt eine Person. Aus seinem Ergebnis ergibt sich alles Weitere:
	 *
	 *  - Person geladen: Der Merker ist verbraucht, das Buch geht normal online hinaus.
	 *  - Versand gescheitert: Es gibt wirklich keine Verbindung. Der Merker bleibt stehen,
	 *    das Buch nimmt seinen gewoehnlichen Weg und landet mit `ausweis_barcode` in der
	 *    Warteschlange — der Server loest die Karte beim Nachbuchen auf.
	 *  - Der Server hat geantwortet, aber keine Person geladen (Nummer unbekannt, gesperrt,
	 *    Suchtreffer): Dann ist nicht sicher, WEM das Buch gehoert. Es wird NICHT gebucht,
	 *    der Merker faellt, und das steht als Meldung da.
	 *
	 * @param {string} q der rohe Scan
	 * @param {(() => void) | null} [reloadProfileCb]
	 * @returns {Promise<boolean>} false = dieser Scan wurde bewusst nicht ausgefuehrt
	 */
	async function merkerAufgeloest(q, reloadProfileCb) {
		if (!offlineAusweis) return true;
		// Ein Ausweis selbst darf durch — er ist der Ausweg und ersetzt den Merker.
		if (ordneScanEin(q, buchBarcodes.istBuch).art === 'ausweis') {
			offlineAusweis = '';
			return true;
		}

		const gemerkt = offlineAusweis;
		let res;
		try {
			res = await apiClient.post('/api/action', {
				query: gemerkt,
				idempotency_key: crypto.randomUUID()
			});
		} catch (e) {
			// Keine Verbindung — der erwartete Fall, kein Fehler. Der Merker bleibt.
			console.warn('Ausweis-Merker nicht aufloesbar, bleibt stehen:', e);
			return true;
		}

		try {
			if (!res.ok) await verarbeiteAktionsFehler(res, gemerkt); // wirft immer
			verarbeiteAktionsErgebnis(await res.json(), reloadProfileCb, gemerkt);
		} catch (e) {
			verarbeiteAntwortfehler(e);
		}

		offlineAusweis = '';
		if (aktiverLeser?.id) return true;

		leuchteBildschirm('error');
		spieleFehlerton();
		zeigeFehlerBanner(
			`Der ohne Netz gemerkte Ausweis \u201e${gemerkt}\u201c liess sich nicht laden. Dieser Scan wurde NICHT gebucht — ` +
				`bitte den Ausweis erneut scannen.`
		);
		return false;
	}

	/** @param {import('../scanEinordnen.js').ScanEinordnung} einordnung */
	function verwirfOfflineScan(einordnung) {
		const meldungen = {
			// Ausweise ohne Netz bekommen ihren eigenen Weg (Stufe 1, naechster Schritt).
			// Bis dahin: sagen, dass es nicht geht, statt es stillschweigend zu schlucken.
			ausweis:
				`Ohne Netz laesst sich der Ausweis \u201e${einordnung.nummer}\u201c noch nicht laden. ` +
				`Buecher dieser Person bitte notieren und nach der Rueckkehr der Verbindung buchen.`,
			// Geraete offline stehen nicht im Umfang (OFFEN.md 2.4): Sie haengen an einer
			// Checkliste, die es ohne Netz nicht gibt.
			geraet:
				`Geraete lassen sich ohne Netz nicht ausgeben oder zuruecknehmen — die Checkliste dazu gibt es nur online. ` +
				`\u201e${einordnung.nummer}\u201c wurde NICHT gebucht.`,
			// Eine Namenssuche ohne Netz kann es nicht geben: Auf dem Theken-Rechner liegen
			// keine Personendaten (Entscheidung vom 13.09.2026). Das ist keine Luecke, die
			// noch zugeht — es ist die Zusage.
			suche:
				`Ohne Netz laesst sich nicht nach Namen suchen — auf diesem Rechner stehen keine Personendaten. ` +
				`Bitte den Ausweis scannen oder den Vorgang notieren.`,
			// Die sichere Seite: Wer hier raet, schreibt das naechste Buch einer fremden Person zu.
			unklar:
				`\u201e${einordnung.nummer}\u201c ist ohne Netz nicht eindeutig — die Nummer steht nicht in der Buchliste dieses Rechners. ` +
				`NICHT gebucht; bitte notieren.`
		};
		leuchteBildschirm('error');
		spieleFehlerton();
		zeigeFehlerBanner(meldungen[einordnung.art] ?? 'Dieser Scan wurde ohne Netz NICHT gebucht.');
	}

	// Der Versand ist gescheitert (Netzfehler, Timeout, CSRF-Bootstrap ohne Netz): Der
	// Server hat den Scan nicht gesehen, der Schnappschuss geht in die Warteschlange.
	/**
	 * @param {unknown} e @param {import('../offlineWarteschlange.js').OfflineEintrag} eintrag
	 * @returns {Promise<boolean>} true: gespeichert oder als Ausweis gemerkt
	 */
	function verarbeiteVersandfehler(e, eintrag) {
		console.warn('Scan nicht zugestellt, wird eingereiht:', e);
		return speichereOfflineAktion(eintrag);
	}

	// Eine Antwort kam an — nichts ist offline. Was ihre Auswertung wirft, wird gezeigt.
	/** @param {unknown} e */
	function verarbeiteAntwortfehler(e) {
		// Wer scannt, sieht auf das Buch: Ein Scan, der nichts gebucht hat, ist zu hören. Nach
		// einem gescheiterten Ausweis steht der Leser davor noch an der Theke.
		leuchteBildschirm('error');
		spieleFehlerton();
		if (e instanceof Error && e.message === 'BLOCK_ALERT') return;
		// Nur das Inline-Banner an der Omnibox (verschwindet nach 6s von selbst).
		// Kein zusätzlicher Toast — das war die doppelte Anzeige desselben Fehlers.
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
		if (trefferlisteOffen && markierterTreffer >= 0) {
			waehleTreffer(markierterTreffer, null);
			return;
		}

		// Enter direkt nach dem Tippen, ohne vorher mit den Pfeiltasten auszuwählen: Wenn
		// die Vorschlagsliste genau EINEN Treffer hat, ist der gemeint.
		//
		// Ohne das ging der rohe Text an /api/action — und der Weg dort endet für eine
		// Eingabe ohne Präfix in der Titelsuche (verarbeiteOhneVorsilbe → verarbeiteSuche),
		// die nur Bücher kennt. Ein getippter Nachname lieferte damit zuverlässig "nichts
		// gefunden", obwohl der Schüler im Vorschlag direkt darüber stand.
		//
		// Bewusst eng gefasst — genau ein Schüler und kein Buch daneben:
		//   • Bei "Müller" mit zwölf Namensgleichen darf Enter nicht raten und ein fremdes
		//     Konto öffnen. Dann bleibt die Liste zur Auswahl stehen.
		//   • Bücher bleiben außen vor. Ihr Zweig in waehleTreffer braucht einen
		//     onSelectBook-Rückruf, den es hier nicht gibt; automatisch ausgelöst würde ein
		//     Buchscan still ins Leere laufen statt auszuleihen.
		if (trefferlisteOffen && suchTreffer.students.length === 1 && suchTreffer.books.length === 0) {
			waehleTreffer(0, null);
			return;
		}

		// EINMAL vereinheitlichen, gleich am Anfang: Dann gilt derselbe Wert fuer den
		// Online-Versand, fuer den Schnappschuss und fuer die Einordnung ohne Netz. Ein
		// von Hand getipptes `s-10001` traf sonst weder hier noch dort etwas.
		const q = normalisiereScan(eingabe);
		if (!q) return;

		eingabe = '';
		trefferlisteOffen = false;

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
		nichtGebucht = [...nichtGebucht, ...wartend.splice(0).map((w) => w.q)];
		leuchteBildschirm('error');
		spieleFehlerton();
		leuchteRand('red');
		zeichneBanner();
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

		// Der Schnappschuss entsteht VOR allem Weiteren (OFFEN.md 2.2, Commit 1): Escape
		// oder „Theke leeren" waehrend einer laufenden Anfrage darf die Absicht des Scans
		// nicht mehr aendern.
		let eintrag = schnappschuss(q, crypto.randomUUID(), absicht);

		// Steht ein ohne Netz gemerkter Ausweis, gehoert das naechste Buch IHM — erst
		// aufloesen, dann buchen (merkerAufgeloest). Ohne Merker bleibt dieser Weg
		// unberuehrt: kein zusaetzlicher Wartepunkt zwischen Scan und Versand.
		if (offlineAusweis) {
			if (!(await merkerAufgeloest(q, reloadProfileCb))) return false;
			// Die Aufloesung hat die Person geladen: Sie gehoert in den Schnappschuss, und
			// der Merker faellt aus ihm heraus — sonst truege der Eintrag beides, und beim
			// Nachbuchen entschiede die Reihenfolge der Felder, wem das Buch gehoert.
			if (eintrag.ausweis_barcode && aktiverLeser?.id) {
				const ohneMerker = { ...eintrag };
				delete ohneMerker.ausweis_barcode;
				eintrag = { ...ohneMerker, leser_id: aktiverLeser.id, art: absicht ?? 'ausleihe' };
			}
		}

		// Zwei Fehlerklassen, zwei Zweige: Scheitert der Versand, hat der Server nichts gesehen —
		// der Schnappschuss geht in die Warteschlange. Kam eine Antwort an, ist nichts offline:
		// Was ihre Auswertung wirft, wird gezeigt und nicht eingereiht. Der Server kennt den
		// Idempotenz-Schlüssel schon; nach Ablauf seines Caches (24 h) würde neu gebucht.
		//
		// Rot gehört in den Fehlerfall: Feuerte es bei jedem Scan, überschriebe es das Grün des
		// Erfolgspfads und das Orange der Fremdrückgabe. Sieht Erfolg wie Fehler aus, schaut
		// niemand mehr auf die Farbe.
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
	 * Gibt dem Scanfeld nach einer Buchung den Fokus.
	 *
	 * Ein Handscanner ist eine Tastatur und tippt blind. Ohne Fokus landen seine Zeichen im
	 * Nichts — keine Ausleihe, keine Fehlermeldung. Das Feld behält den Fokus während der
	 * Buchung; ein Klick davor oder das neu gerenderte Profil kann ihn woanders gelassen haben.
	 *
	 * Der $effect in Omnibox.svelte fängt das nicht ab: Er läuft nur, solange KEIN Leser
	 * geladen ist (`!belegt`) — also genau nicht während des Ausleihens.
	 *
	 * Nicht zurückholen, solange ein Dialog eine menschliche Entscheidung braucht
	 * (Sperre, Vormerkung) oder die Kamera scannt — dort würde der Fokussprung die
	 * Bedienung stören und den Dialog wegtippbar machen.
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

	// Der Fokussprung selbst — DIE Stelle, an der ein Zeitgeber dieses Stores die Seite
	// anfasst, und damit die, die nach dem Abbau der Testumgebung den ganzen Lauf riss
	// (siehe stoppeZeitgeber). Bis zum 16.09.2026 stand derselbe Dreizeiler ein zweites
	// Mal in Omnibox.svelte; zwei Timer für eine Sache sind auch zwei Stellen, an denen
	// das Aufräumen fehlen kann.
	//
	// Nach der Aktion rendert Svelte das Profil neu; erst danach steht das Feld wieder.
	//
	// Beim Feuern wird noch einmal gefragt, nicht nur beim Planen: Zwei Scans kurz
	// hintereinander öffneten den Sperr-Dialog, BEVOR der Zeitgeber des ersten Scans feuerte
	// (e2e am 24.09.2026, drei von vier Läufen) — er zog den Fokus dann aus dem Dialog ins
	// Scanfeld dahinter, und der nächste Scan wäre dort gelandet statt im Dialog.
	function fokussiereScanfeld() {
		if (fokusTimer) clearTimeout(fokusTimer);
		fokusTimer = setTimeout(() => {
			fokusTimer = null;
			if (entscheidungOffen()) return;
			document.getElementById('omnibox-input')?.focus();
		}, 50);
	}

	// Auch jeder Wechsel zur Ausleihe gibt dem Scanfeld den Fokus zurück (14.09.2026): Ein
	// Klick auf „Ausleihe" ließ ihn auf dem Knopf der Seitenleiste, und der nächste Scan lief
	// ohne Meldung ins Leere (e2e/scanner-fokus-menue.spec.js).
	uiStore.beimWechselZurTheke = scanfeldWiederScharfstellen;

	// Nach dem Zusammenführen zweier Datensätze (Leserakte → onMerged): Der aktive
	// Schüler wird auf das Ziel umgehängt. Bis zum 15.09.2026 blieb an der Theke die
	// gelöschte Kennung stehen — die Akte zeigte schon das Ziel, aber jede weitere Buchung
	// und jeder Eintrag in der Offline-Warteschlange (enqueueOfflineAction) lief auf einen
	// Datensatz, den es nicht mehr gab (OFFEN.md 3.2). Die Kennung wird SOFORT gesetzt, damit
	// kein Scan dazwischen die alte erwischt; Name und Sperrflags kommen nach.
	/** @param {string} zielId */
	async function uebernimmZusammengefuehrt(zielId) {
		aktiverLeser = { id: zielId };
		const res = await apiFetch(`/api/schueler/${zielId}`);
		// Inzwischen ein anderer Ausweis? Dann gehört das Nachgeladene niemandem mehr.
		if (!res.ok || aktiverLeser?.id !== zielId) return;
		aktiverLeser = await res.json();
	}

	// „Buch zurückgeben" aus der Akte: derselbe Weg wie ein Scan des Barcodes, aber mit
	// bekannter Absicht. Online entscheidet ohnehin der Server (das Buch liegt beim
	// geladenen Schüler → Rückgabe); offline hätte „Schüler geladen" bis zum 15.09.2026 eine
	// Ausleihe eingereiht — beim Doppelklick eine echte zweite (OFFEN.md 2.2, Commit 3).
	/** @param {string} barcode @param {(() => void) | null} [reloadProfileCb] */
	function gibZurueck(barcode, reloadProfileCb = null) {
		eingabe = barcode;
		return fuehreAktionAus(null, reloadProfileCb, false, false, 'rueckgabe');
	}

	// Ohne Kontexttyp gibt TypeScript einem Objektliteral in JS einen offenen Typ, und ein Zugriff
	// auf einen Namen, den der Store nicht hat, bliebe ohne Meldung; @satisfies schließt ihn.
	return /** @satisfies {object} */ ({
		get aktiverLeser() {
			return aktiverLeser;
		},
		/** Die ohne Netz gemerkte Ausweisnummer ("" = keine). Nur die Nummer, kein Name. */
		get offlineAusweis() {
			return offlineAusweis;
		},
		// Von außen gesetzt (Escape, Akte schließen, Theke leeren): Was noch wartet, wurde für
		// den Zustand davor gescannt und wird nicht gebucht.
		set offlineAusweis(v) {
			verwirfWartende();
			offlineAusweis = v;
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
			return randLeuchten;
		},
		get bildschirmLeuchten() {
			return bildschirmLeuchten;
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
			return wackelt;
		},
		get fehler() {
			return fehler;
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
			return trefferlisteOffen;
		},
		set trefferlisteOffen(v) {
			trefferlisteOffen = v;
		},
		get suchTreffer() {
			return suchTreffer;
		},
		set suchTreffer(v) {
			suchTreffer = v;
		},
		get markierterTreffer() {
			return markierterTreffer;
		},
		set markierterTreffer(v) {
			markierterTreffer = v;
		},
		get trefferAnzahl() {
			return trefferAnzahl;
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

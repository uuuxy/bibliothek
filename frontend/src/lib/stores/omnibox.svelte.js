// Zustand und Abläufe der Omnibox an der Theke: Eingabe, Suche, Scan und seine Antwort.

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
import { erzeugeOmniboxRueckmeldung } from './omniboxRueckmeldung.svelte.js';

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
	// Der ohne Netz gescannte Ausweis, nur seine Nummer: Auflösen kann ihn nur der Server, bis
	// dahin tragen die folgenden Bücher sie mit (`ausweis_barcode` in der Warteschlange).
	let offlineAusweis = $state('');
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

	// Such-Status
	let suchVerzoegerung = $state(/** @type {any} */ (null));
	let trefferlisteOffen = $state(false);
	let suchTreffer = $state({ students: [], books: [], studentsTotal: 0, booksTotal: 0 });
	let markierterTreffer = $state(-1);
	let trefferAnzahl = $derived(suchTreffer.students.length + suchTreffer.books.length);

	let belegt = $derived(!!(aktiverLeser || trefferlisteOffen));

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
		for (const t of [fokusTimer, suchVerzoegerung]) {
			if (t) clearTimeout(t);
		}
		fokusTimer = null;
		suchVerzoegerung = null;
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
			// Zwei laufende Abrufe beantwortet der Server in beliebiger Reihenfolge. Ohne die
			// Nummer tauschte eine verspätete Antwort die Trefferliste unter dem Suchtext aus,
			// und der nächste Klick buchte auf den falschen Schüler.
			const seq = ++suchLauf;
			try {
				const res = await apiFetch(`/api/search?q=${encodeURIComponent(eingabe.trim())}`);
				if (seq !== suchLauf) return;
				if (!res.ok) {
					// Dieselbe Gefahr beim Fehlschlag: Bliebe die Liste des vorigen Suchtextes stehen,
					// buchte der nächste Klick auf einen Treffer, der zum Feld nicht mehr passt.
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
				// Netzfehler: ebenfalls verwerfen, aber ohne Toast. Im WLAN-Loch käme bei jedem
				// Tastendruck einer, und die Offline-Anzeige sagt es bereits.
				verwirfTreffer();
				console.error('Suche fehlgeschlagen:', err);
			}
		}, 300);
	}

	// Ein Leser aus der Trefferliste wird über seine ID geladen, weil ein Kollege aus der
	// Selbstanmeldung keine Ausweisnummer hat; auf demselben Weg wie ein Scan (POST /api/action),
	// damit Abholfach-Hinweis und Sperrprüfung an beiden Wegen gleich sind.
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

		// Der Dialog hängt am Merkmal des Servers, nicht am Wortlaut der Meldung, der je Sperre
		// anders lautet (omniboxSperrDialog.test.js).
		const art = res.headers.get('X-Sperre');
		if (res.status === 403 && (art === 'uebergehbar' || art === 'leser')) {
			sperrHinweis = { message: errStr, query: q, art };
			throw new Error('BLOCK_ALERT');
		}

		throw new Error(errStr || 'Aktion fehlgeschlagen');
	}

	// Ein fremdes Buch bei geladenem Leser wird nur beim Vorbesitzer ausgebucht, bewusst ohne
	// Umbuchen. Ein erneuter Scan leiht es dem geladenen Leser aus.
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

	// Die Rückkehr eines abgeschriebenen Buches: `message` sagt, was der Server erledigt hat
	// (grün), `aufsicht_informieren`, was ein Mensch noch tun muss (Warnung). Eine offene Aufgabe
	// in einer grünen Erfolgsmeldung würde überlesen.
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
		// Gebucht ist die Ausleihe in jedem Fall, deshalb ihr Ton. Gehört das Buch zu einer anderen
		// Auflage als die der Klasse, blitzt es wie bei der Fremdrückgabe, und die Zeile über dem
		// Konto sagt, welche.
		letzterAuflagenHinweis = daten.auflagen_hinweis ?? null;
		leuchteBildschirm(letzterAuflagenHinweis ? 'warning' : 'success');
		spieleErfolgston('ausleihe');
		leuchteRand(letzterAuflagenHinweis ? 'orange' : 'green');
		melde(
			`„${daten.book?.titel ?? daten.geraet?.modellname}" ausgeliehen an ${aktiverLeser?.vorname}.`
		);
		// Der Schüler hatte ein anderes Exemplar reserviert und eines aus dem Regal genommen: Das
		// reservierte muss zurück ins Regal, sonst bleibt es im Fach liegen.
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

	// Der Schnappschuss des Scans, bevor die Anfrage hinausgeht: Escape oder „Theke leeren"
	// während einer hängenden Anfrage ändern weder Person noch Absicht.
	//
	// Ohne Vorgabe ist der Scan eine Ausleihe, wenn ein Leser geladen oder ein Ausweis gemerkt
	// ist, sonst eine Rückgabe; „Buch zurückgeben" in der Akte gibt die Rückgabe vor.
	/**
	 * @param {string} q
	 * @param {string} idempotencyKey
	 * @param {'ausleihe' | 'rueckgabe' | null} absicht
	 * @returns {import('../offlineWarteschlange.js').OfflineEintrag}
	 */
	function schnappschuss(q, idempotencyKey, absicht) {
		// Ohne geladenen Leser, aber mit gemerktem Ausweis geht das Buch an die Person dahinter.
		const merker = !aktiverLeser?.id && offlineAusweis ? offlineAusweis : '';
		return {
			id: idempotencyKey,
			art: absicht ?? (aktiverLeser?.id || merker ? 'ausleihe' : 'rueckgabe'),
			barcode: q,
			leser_id: aktiverLeser?.id ?? null,
			...(merker ? { ausweis_barcode: merker } : {}),
			gescannt_am: Date.now(),
			// Die Wanduhr springt gern, wenn das Netz zurückkommt, performance.now() nicht. Daraus
			// rechnet das Nachbuchen den Zeitpunkt des Scans neu (offlineWarteschlange.js).
			mono: Math.round(performance.now()),
			ursprung: Math.round(performance.timeOrigin ?? 0)
		};
	}

	// Legt einen Scan ohne Netz ab. Eingeordnet wird wie am Server (scanEinordnen.js), mit der
	// Buchliste dieses Rechners; abgelegt wird die Nummer aus der Einordnung, weil ein
	// Littera-Etikett eine EAN-13 trägt und der Server nur die Nummer darin kennt.
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
			// Nach einem unklaren Scan fällt der Merker: Das nächste Buch soll keiner Person
			// zugeschrieben werden, bei der niemand mehr sicher ist.
			if (einordnung.art === 'unklar') offlineAusweis = '';
			verwirfOfflineScan(einordnung);
			return false;
		}
		eintrag = { ...eintrag, barcode: einordnung.nummer };
		try {
			await enqueueOfflineAction(eintrag);
		} catch (err) {
			// Laut, nicht „gespeichert": Ohne Warteschlange ist der Scan verloren, das Buch muss
			// zurück ins Regal oder auf den Zettel.
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

	// Ohne Netz merkt sich die Theke vom Ausweis nur die Nummer, weil auf dem Rechner keine
	// Personendaten liegen sollen. Der Server löst sie beim Nachbuchen auf.
	/** @param {string} nummer */
	function merkeOfflineAusweis(nummer) {
		schnellrueckgabe = false;
		offlineAusweis = nummer;
		// Die zuvor geladene Person weicht: Sonst zeigte die Theke einen Namen, während die
		// folgenden Bücher an eine andere Person gingen.
		aktiverLeser = null;
		leuchteBildschirm('warning');
		spieleErfolgston();
		melde(
			`Ohne Netz gemerkt: Ausweis „${nummer}". Die folgenden Bücher werden ihm zugeordnet.`,
			'warning'
		);
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
			console.warn('Ausweis-Merker nicht auflösbar, bleibt stehen:', e);
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
			`Der ohne Netz gemerkte Ausweis \u201e${gemerkt}\u201c ließ sich nicht laden. Dieser Scan wurde NICHT gebucht — ` +
				`bitte den Ausweis erneut scannen.`
		);
		return false;
	}

	/** @param {import('../scanEinordnen.js').ScanEinordnung} einordnung */
	function verwirfOfflineScan(einordnung) {
		// Den Ausweis merkt speichereOfflineAktion vorher; hier kommt er nicht an.
		const meldungen = {
			// Geräte gehen ohne Netz nicht: Sie hängen an einer Checkliste, die es nur online gibt.
			geraet:
				`Geräte lassen sich ohne Netz nicht ausgeben oder zurücknehmen — die Checkliste dazu gibt es nur online. ` +
				`\u201e${einordnung.nummer}\u201c wurde NICHT gebucht.`,
			// Ohne Netz gibt es keine Namenssuche, weil auf dem Rechner der Theke keine
			// Personendaten liegen sollen.
			suche:
				`Ohne Netz lässt sich nicht nach Namen suchen — auf diesem Rechner stehen keine Personendaten. ` +
				`Bitte den Ausweis scannen oder den Vorgang notieren.`,
			// Die sichere Seite: Wer hier rät, schreibt das nächste Buch einer fremden Person zu.
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
		if (trefferlisteOffen && markierterTreffer >= 0) {
			waehleTreffer(markierterTreffer, null);
			return;
		}

		// Enter ohne Auswahl nimmt den einen Schüler, wenn die Liste genau ihn zeigt: Der rohe Text
		// liefe am Server in die Titelsuche, die nur Bücher kennt. Bei Namensgleichen rät Enter
		// nicht, und ein Buch braucht den Rückruf onSelectBook, den es hier nicht gibt.
		if (trefferlisteOffen && suchTreffer.students.length === 1 && suchTreffer.books.length === 0) {
			waehleTreffer(0, null);
			return;
		}

		// Einmal vereinheitlichen, gleich am Anfang: Dann gilt derselbe Wert für den Versand, den
		// Schnappschuss und die Einordnung ohne Netz.
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
		if (offlineAusweis) {
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

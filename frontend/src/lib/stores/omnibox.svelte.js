// stores/omnibox.svelte.js
// Status- und Logikverwaltung für die Omnibox (Svelte 5 Runes)

import { apiFetch, apiClient } from '../apiFetch.js';
import { playSoundSuccess, playSoundError } from '../audio.js';
import { enqueueOfflineAction } from '../offlineQueue.js';
import { offlineSync } from './offlineSync.svelte.js';
import { buchBarcodes } from './buchBarcodes.svelte.js';
import { normalisiereScan, ordneScanEin } from '../scanEinordnen.js';
import { toastStore } from './toastStore.svelte.js';
import { uiStore } from './uiStore.svelte.js';

// Name des Vorbesitzers bei einer Fremdrückgabe (Schüler bevorzugt, dann Lehrer).
function formatVorbesitzerName(data) {
	if (data.vorbesitzer) {
		return `${data.vorbesitzer.vorname} ${data.vorbesitzer.nachname}`;
	}
	if (data.vorbesitzer_user) {
		return `${data.vorbesitzer_user.vorname} ${data.vorbesitzer_user.nachname}`;
	}
	return 'unbekannt';
}

// Geht an den globalen toastStore (ToastContainer.svelte in App.svelte). Vorher
// hielt die Omnibox einen eigenen Toast-Zustand samt eigenem Markup und eigenem
// Auto-Dismiss — zwei Meldungswege mit zwei Optiken für dieselbe Sache.
//
// Steht auf Modulebene, weil die Funktion nichts aus dem Store einfängt: Sonst würde
// bei jedem createOmniboxStore() eine neue, identische Funktion angelegt (sonarjs S7721).
/**
 * @param {string} message
 * @param {import('./toastStore.svelte.js').ToastTyp} [type='success']
 */
function showToast(message, type = 'success') {
	toastStore.addToast(message, type);
}

export function createOmniboxStore() {
	// activeStudent ist der LESER, der gerade an der Theke steht — Schüler oder Kollege,
	// unterscheidbar an `art`. Ein zweiter Platz für Lehrkräfte ist mit Migration 125
	// weggefallen; solange es zwei gab, musste jede Stelle beide abfragen und sich für
	// eine entscheiden.
	let activeStudent = $state(/** @type {any} */ (null));
	// Der ohne Netz gescannte Ausweis — nur die NUMMER, kein Name, keine Ausleihen.
	// Aufloesen kann ihn nur der Server; bis dahin tragen die folgenden Buecher sie mit
	// (offlineQueue: `ausweis_barcode`, die Nachbuch-Tuer kennt das Feld).
	let offlineAusweis = $state('');
	let queryVal = $state('');

	let flashBorder = $state('');
	let screenFlash = $state(''); // "success" | "error" | ""
	let lastFremdrueckgabe = $state(/** @type {any} */ (null));
	let isShaking = $state(false);
	let scanError = $state(false);
	let errorMessage = $state('');
	/** @type {any} Auto-Dismiss-Timer des Inline-Fehlerbanners */
	let errorMessageTimer = null;
	let vormerkungAlert = $state(/** @type {{titel?: string, user?: string} | null} */ (null));
	// Abholfach-Hinweis zum AKTIVEN Schüler (Betreiber-Entscheidung 01.09.2026):
	// beim Ausweis-Scan liefert der Server die abholbereiten Vormerkungen mit —
	// die Mitarbeiterin greift ins Abholfach, solange der Schüler vor ihr steht.
	// Wird bei JEDEM Setzen von activeStudent mit überschrieben (?? []), damit
	// kein Hinweis eines vorherigen Schülers stehen bleibt (bekannte Bugklasse
	// „nie zurückgesetzter UI-State").
	let abholbereit = $state(/** @type {{titel: string, bereitgestellt_bis?: string}[]} */ ([]));
	/**
	 * Geräte-Scan mit Zubehör: Der Server unterbricht mit type=geraet_check und
	 * wartet auf die Bestätigung der Checkliste. Hier liegt die Anfrage, bis der
	 * Dialog bestätigt (erneuter Versand mit confirmed_checklist) oder abbricht.
	 * @type {{query: string, geraet: any} | null}
	 */
	let checklistAnfrage = $state(null);
	let blockAlert = $state(/** @type {{message: string, query: string} | null} */ (null));
	// isOffline is now handled globally via offlineSync
	let offlineQueueCount = $state(0);

	// Kamerascanner-Status
	let showCamera = $state(false);
	let cameraScanner = $state(/** @type {any} */ (null));

	// Such-Status
	let debounceTimer = $state(/** @type {any} */ (null));
	let isDropdownOpen = $state(false);
	let unifiedSearchResults = $state({ students: [], books: [], studentsTotal: 0, booksTotal: 0 });
	let selectedDropdownIndex = $state(-1);
	let totalDropdownItems = $derived(
		unifiedSearchResults.students.length + unifiedSearchResults.books.length
	);

	let isActive = $derived(!!(activeStudent || isDropdownOpen));

	// UI Feedback-Methoden
	//
	// Jeder dieser Zeitgeber bekommt ein HANDLE und wird vor dem Neuplanen verworfen —
	// dasselbe Muster wie in actions/keyboardNav.js und actions/tooltip.js. Der Grund ist
	// nicht Ordnungsliebe: Ein Timer, der die Seite überlebt, feuert ins Leere. In der
	// Testumgebung ist „ins Leere" ein Absturz — nach dem Abbau von jsdom gibt es kein
	// `document` mehr, und der Rückruf reisst den GANZEN Lauf mit („Unhandled Errors:
	// document is not defined"), obwohl jeder einzelne Test grün ist. Genau so stand die
	// CI am 16.09.2026 rot bei 697 grünen Tests; am 11.09.2026 war es keyboardNav.
	/** @type {ReturnType<typeof setTimeout> | null} */
	let screenFlashTimer = null;
	/** @type {ReturnType<typeof setTimeout> | null} */
	let shakeTimer = null;
	/** @type {ReturnType<typeof setTimeout> | null} */
	let flashTimer = null;
	/** @type {ReturnType<typeof setTimeout> | null} */
	let fokusTimer = null;

	function triggerScreenFlash(type) {
		screenFlash = type;
		if (screenFlashTimer) clearTimeout(screenFlashTimer);
		screenFlashTimer = setTimeout(() => {
			screenFlashTimer = null;
			screenFlash = '';
		}, 300);
	}

	function triggerShake() {
		isShaking = true;
		if (shakeTimer) clearTimeout(shakeTimer);
		shakeTimer = setTimeout(() => {
			shakeTimer = null;
			isShaking = false;
		}, 500);
	}

	function triggerFlash(color) {
		flashBorder = color;
		if (flashTimer) clearTimeout(flashTimer);
		flashTimer = setTimeout(() => {
			flashTimer = null;
			flashBorder = '';
		}, 1000);
	}

	// Stoppt JEDEN laufenden Zeitgeber dieses Stores. Die Theke ruft es beim Verlassen der
	// Seite (Omnibox.svelte, onDestroy), die Tests beim Abräumen — danach kann kein
	// Rückruf mehr auf eine Seite greifen, die es nicht mehr gibt.
	function stoppeZeitgeber() {
		for (const t of [
			screenFlashTimer,
			shakeTimer,
			flashTimer,
			fokusTimer,
			errorMessageTimer,
			debounceTimer
		]) {
			if (t) clearTimeout(t);
		}
		screenFlashTimer = null;
		shakeTimer = null;
		flashTimer = null;
		fokusTimer = null;
		errorMessageTimer = null;
		debounceTimer = null;
	}

	// Zeigt das Inline-Fehlerbanner an der Omnibox und blendet es nach 6s automatisch
	// aus. Ein noch laufender Timer wird verworfen, damit ein neuer Fehler die volle
	// Anzeigedauer bekommt (und nicht der alte Timer das frische Banner sofort löscht).
	function zeigeFehlerBanner(message) {
		clearFehlerBanner();
		errorMessage = message;
		errorMessageTimer = setTimeout(() => {
			errorMessage = '';
			errorMessageTimer = null;
		}, 6000);
	}

	// Blendet das Inline-Fehlerbanner sofort aus und stoppt den Auto-Dismiss-Timer.
	function clearFehlerBanner() {
		if (errorMessageTimer) {
			clearTimeout(errorMessageTimer);
			errorMessageTimer = null;
		}
		errorMessage = '';
	}

	// Such-Logik. suchLauf zählt jeden gestarteten Abruf — nur die Antwort des zuletzt
	// gestarteten darf die Trefferliste schreiben.
	let suchLauf = 0;

	/** Trefferliste und Dropdown leeren — eine Liste darf ihren Suchtext nie überleben. */
	function verwirfTreffer() {
		unifiedSearchResults = { students: [], books: [], studentsTotal: 0, booksTotal: 0 };
		isDropdownOpen = false;
		selectedDropdownIndex = -1;
	}

	function handleInput() {
		clearTimeout(debounceTimer);
		if (!queryVal.trim()) {
			// Auch hier hochzählen: Ein noch laufender Abruf darf die eben geleerte
			// Liste nicht wieder füllen.
			suchLauf++;
			isDropdownOpen = false;
			unifiedSearchResults = { students: [], books: [], studentsTotal: 0, booksTotal: 0 };
			return;
		}
		debounceTimer = setTimeout(async () => {
			if (!queryVal.trim()) return;
			// Sequenznummer wie im orderStore: Der Entprell-Timer verwirft nur NOCH NICHT
			// gestartete Abrufe — zwei gleichzeitig laufende beantwortet der Server in
			// beliebiger Reihenfolge. Ohne diese Prüfung tauschte eine verspätete Antwort
			// die Trefferliste unter dem sichtbaren Suchtext aus, und der nächste Klick
			// (bzw. Enter) buchte auf den FALSCHEN Schüler — ohne Rückfrage
			// (Fund 31.08.2026).
			const seq = ++suchLauf;
			try {
				const res = await apiFetch(`/api/search?q=${encodeURIComponent(queryVal.trim())}`);
				if (seq !== suchLauf) return;
				if (!res.ok) {
					// Dieselbe Gefahr wie oben, nur über den anderen Ausgang (Sweep
					// „verschluckte Fehlantwort", 06.09.2026): Bis hierher blieb bei einem
					// Fehlschlag die Liste des VORIGEN Suchtextes stehen — samt offenem
					// Dropdown. „Müller" getippt, Treffer da; „Schmidt" getippt, Abruf
					// scheitert (500, oder 429 vom Rate-Limiter) — und der nächste Klick
					// oder Enter buchte auf Müller, während im Feld Schmidt stand.
					verwirfTreffer();
					showToast('Suche fehlgeschlagen — bitte erneut versuchen.', 'error');
					return;
				}
				const results = await res.json();
				unifiedSearchResults = {
					students: results.students || [],
					books: results.books || [],
					studentsTotal: results.students_total ?? (results.students || []).length,
					booksTotal: results.books_total ?? (results.books || []).length
				};
				isDropdownOpen =
					unifiedSearchResults.students.length > 0 || unifiedSearchResults.books.length > 0;
				selectedDropdownIndex = -1;
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
	function selectDropdownItem(index, onSelectBook) {
		const { students, books } = unifiedSearchResults;
		if (index < students.length) {
			const student = students[index];
			queryVal = `leser:${student.id}`;
			isDropdownOpen = false;
			submitAction(null, null); // Ohne Event
		} else {
			const book = books[index - students.length];
			queryVal = '';
			isDropdownOpen = false;
			if (onSelectBook) onSelectBook(book);
		}
	}

	// Liest die Fehlermeldung aus einer nicht-ok HTTP-Antwort. Bei einem
	// Sperr-/Überfälligkeits-403 wird blockAlert gesetzt und BLOCK_ALERT geworfen,
	// sonst ein generischer Fehler. Wirft in jedem Fall.
	async function handleActionHttpError(res, q) {
		let errStr = await res.text();
		try {
			const errData = JSON.parse(errStr);
			if (errData.error) errStr = errData.error;
		} catch (e) {
			console.debug('Fehlerantwort war kein JSON, nutze Rohtext:', e);
		}

		// Der Dialog hängt am Merkmal des Servers, nicht am Wortlaut (omniboxSperrDialog.test.js).
		// Bis zum 13.09.2026 entschieden hier die Wörter „Sperre", „Sperr-Automatik" und
		// „überfällig": Die Schadens-Sperre traf keins, und bei der System-Sperre hing es am
		// Sperrgrund, den eine Helferin gar nicht zu sehen bekommt.
		if (res.status === 403 && res.headers.get('X-Sperre') === 'uebergehbar') {
			blockAlert = { message: errStr, query: q };
			throw new Error('BLOCK_ALERT');
		}

		throw new Error(errStr || 'Aktion fehlgeschlagen');
	}

	// Fremdes Buch in aktiver Sitzung: NUR beim Vorbesitzer ausgebucht — bewusst kein
	// automatisches Umbuchen. Info ohne Unterbrechung; erneuter Scan leiht es an die
	// aktive Sitzung aus.
	function verarbeiteFremdrueckgabe(data) {
		triggerScreenFlash('warning');
		playSoundError();
		triggerFlash('orange');
		const prevName = formatVorbesitzerName(data);
		lastFremdrueckgabe = { vorbesitzerName: prevName };
		const aktiv = activeStudent?.vorname;
		const nachsatz = aktiv ? ` Erneut scannen, um es an ${aktiv} auszuleihen.` : '';
		showToast(
			`„${data.book.titel}" war auf ${prevName} verbucht — dort zurückgegeben.${nachsatz}`,
			'warning'
		);
	}

	// Verarbeitet eine Rückgabe-Antwort (fremd/normal, Vormerkung, Session-Fortführung).
	function verarbeiteRueckgabe(data, reloadProfileCb) {
		if (data.fremdrueckgabe) {
			verarbeiteFremdrueckgabe(data);
		} else {
			triggerScreenFlash('success');
			playSoundSuccess();
			triggerFlash('green');
			showToast(`„${data.book?.titel ?? data.geraet?.modellname}" erfolgreich zurückgegeben.`);
		}
		if (data.has_vormerkung) {
			vormerkungAlert = {
				titel: data.vormerkung_titel || data.book?.titel,
				user: data.vormerkung_user
			};
		}
		if (reloadProfileCb) reloadProfileCb();

		if (data.student && !activeStudent) {
			activeStudent = data.student;
			abholbereit = data.abholbereit ?? [];
		}
	}

	// Die Rückkehr eines abgeschriebenen Buches (#597): Der Server sagt in `message`, was
	// er erledigt hat (Forderung storniert — grün), und in `aufsicht_informieren`, was ein
	// Mensch noch tun muss (der Bescheid liegt bei der Schulaufsicht — Warnung). Zwei
	// Kanäle, weil es zwei Dinge sind: Eine offene Aufgabe in einer grünen Erfolgsmeldung
	// wird überlesen.
	function zeigeRueckkehrHinweise(data, { ohneMeldung = false } = {}) {
		if (!ohneMeldung && data.message) showToast(data.message, 'success');
		if (data.aufsicht_informieren) showToast(data.aufsicht_informieren, 'warning');
	}

	// Verarbeitet die erfolgreiche Server-Antwort je nach data.type.
	function verarbeiteAktionsErgebnis(data, reloadProfileCb, q = '') {
		if (data.type === 'student') {
			activeStudent = data.student;
			abholbereit = data.abholbereit ?? [];
			triggerScreenFlash('success');
			playSoundSuccess();
			triggerFlash('green');
			// Bei einem Kollegen sagen, was der Scan bedeutet: Die Karte daneben zeigt
			// keine Klasse, und ohne Ansage sieht ein geladener Kollege aus wie ein
			// Schüler mit fehlender Angabe.
			if (data.student?.art && data.student.art !== 'schueler') {
				showToast(`Geladen: ${data.student.vorname} ${data.student.nachname}`);
			}
		} else if (data.type === 'geraet_check') {
			// Kein Fehler, kein Erfolg: Der Scan wartet auf die Zubehör-Bestätigung.
			checklistAnfrage = { query: q, geraet: data.geraet };
		} else if (data.type === 'ausleihe') {
			triggerScreenFlash('success');
			playSoundSuccess();
			triggerFlash('green');
			showToast(
				`„${data.book?.titel ?? data.geraet?.modellname}" ausgeliehen an ${activeStudent?.vorname}.`
			);
			// Der Schüler hatte ein ANDERES Exemplar reserviert und ein Freihand-Exemplar
			// genommen — das reservierte muss zurück ins Regal, sonst bleibt es im Fach liegen.
			if (data.regalfreigabe_barcode) {
				showToast(
					`Hinweis: Reserviertes Exemplar ${data.regalfreigabe_barcode} zurück ins Regal räumen.`,
					'warning'
				);
			}
			// Das Exemplar war abgeschrieben und ist beim Scan zurückgeholt worden (die
			// Ausleihe folgte im selben Zug): Was dabei mit der Forderung geschah, gehört
			// gesagt — sonst sieht die Theke nur „ausgeliehen".
			zeigeRueckkehrHinweise(data);
			if (reloadProfileCb) reloadProfileCb();
		} else if (data.type === 'rueckgabe') {
			verarbeiteRueckgabe(data, reloadProfileCb);
		} else if (data.type === 'info') {
			triggerScreenFlash('success');
			playSoundSuccess();
			triggerFlash('green');
			showToast(data.message, 'success');
			zeigeRueckkehrHinweise(data, { ohneMeldung: true });
			if (reloadProfileCb) reloadProfileCb();
		} else if (data.type === 'search_results') {
			triggerShake();
			showToast('Bitte wähle ein Ergebnis aus der Liste.', 'warning');
		}
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
	 * @returns {import('../offlineQueue.js').OfflineEintrag}
	 */
	function schnappschuss(q, idempotencyKey, absicht) {
		// Uhr-Anker neben dem Zeitstempel: `performance.now()` laeuft gleichmaessig und
		// springt nicht, die Wanduhr schon — und zwar gern genau dann, wenn das Netz
		// zurueckkommt, also zwischen Scan und Versand. Der Sync rechnet daraus den
		// Scan-Zeitpunkt neu, solange der Eintrag aus demselben Seitenaufruf stammt
		// (offlineQueue.js, OfflineEintrag).
		// Ohne geladene Person, aber mit Ausweis-Merker: Das ist eine AUSLEIHE an die
		// Person hinter dem Merker. Ohne beides bleibt es eine Rueckgabe.
		const merker = !activeStudent?.id && offlineAusweis ? offlineAusweis : '';
		return {
			id: idempotencyKey,
			art: absicht ?? (activeStudent?.id || merker ? 'ausleihe' : 'rueckgabe'),
			barcode: q,
			leser_id: activeStudent?.id ?? null,
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
	// omnibox_service.go), mit der Barcode-Liste dieses Rechners als Nachschlagewerk.
	// Gebucht wird unter der NUMMER aus der Einordnung, nicht unter dem Aufdruck: Ein
	// Littera-Etikett traegt im Strichcode eine EAN-13, der Server kennt nur die Nummer
	// darin.
	/** @param {import('../offlineQueue.js').OfflineEintrag} eintrag */
	async function speichereOfflineAktion(eintrag) {
		const einordnung = ordneScanEin(eintrag.barcode, buchBarcodes.istBuch);
		if (einordnung.art === 'ausweis') {
			merkeOfflineAusweis(einordnung.nummer);
			return;
		}
		if (einordnung.art !== 'buch') {
			// „unklar" sperrt die Zuordnung bis zum naechsten eindeutigen Ausweis
			// (Entscheidung vom 13.09.2026): Der Merker faellt, damit das naechste Buch
			// nicht einer Person zugeschrieben wird, bei der niemand mehr sicher ist.
			if (einordnung.art === 'unklar') offlineAusweis = '';
			verwirfOfflineScan(einordnung);
			return;
		}
		eintrag = { ...eintrag, barcode: einordnung.nummer };
		try {
			await enqueueOfflineAction(eintrag);
		} catch (err) {
			// Laut, nicht „gespeichert" (Commit 5): Ohne Warteschlange ist der Scan verloren —
			// das Buch muss zurück ins Regal oder auf den Zettel.
			console.error('Offline-Eintrag nicht gespeichert:', err);
			triggerScreenFlash('error');
			playSoundError();
			zeigeFehlerBanner(
				`NICHT gespeichert: „${eintrag.barcode}“ konnte nicht auf diesem Rechner abgelegt werden — Buch zurücklegen und den Vorgang notieren.`
			);
			offlineSync.updateCount();
			return;
		}
		offlineSync.updateCount();
		triggerScreenFlash('warning');
		playSoundSuccess();
		showToast(`Offline: Aktion für „${eintrag.barcode}“ gespeichert.`, 'warning');
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
		offlineAusweis = nummer;
		// Die zuvor geladene Person weicht: Sonst zeigte die Theke einen Namen, waehrend
		// die folgenden Buecher an eine ANDERE Person gingen.
		activeStudent = null;
		triggerScreenFlash('warning');
		playSoundSuccess();
		showToast(
			`Ohne Netz gemerkt: Ausweis „${nummer}". Die folgenden Bücher werden ihm zugeordnet.`,
			'warning'
		);
	}

	/**
	 * Der gefaehrlichste Fall des Offline-Betriebs: Ein ohne Netz gemerkter Ausweis steht
	 * noch, und die Verbindung ist zurueck.
	 *
	 * Der Merker traegt nur die NUMMER; der Online-Weg braucht die Leser-Kennung. Ginge
	 * das Buch jetzt hinaus, schickte es `active_leser_id` gar nicht mit — und der Server
	 * liest das Schweigen als RUECKGABE. Aus einer Ausleihe an S-10001 wuerde still eine
	 * Rueckgabe. Genau davor warnt der Plan (Stufe 3, Commit 15).
	 *
	 * Deshalb wird ein Buch bei stehendem Merker NICHT gesendet, sondern um einen erneuten
	 * Ausweis-Scan gebeten. Ein Ausweis selbst darf durch — er ist ja der Ausweg.
	 *
	 * Warum an `navigator.onLine` und nicht am Versuch: Der Versuch entscheidet sich erst
	 * NACH dem Senden, und dann ist es zu spaet. `navigator.onLine` kann luegen (WLAN da,
	 * Server weg) — dann scheitert auch der erbetene Ausweis-Scan, der Merker entsteht neu,
	 * und der Preis ist ein zusaetzlicher Scan. Eine falsche Buchung entsteht in KEINEM
	 * Zweig; das ist der Punkt.
	 *
	 * @param {string} q der rohe Scan
	 * @returns {boolean} false = dieser Scan wurde bewusst nicht ausgefuehrt
	 */
	function merkerVertraegtDiesenScan(q) {
		if (!offlineAusweis) return true;
		if (typeof navigator !== 'undefined' && !navigator.onLine) return true;
		if (ordneScanEin(q, buchBarcodes.istBuch).art === 'ausweis') {
			offlineAusweis = '';
			return true;
		}
		const gemerkt = offlineAusweis;
		offlineAusweis = '';
		triggerScreenFlash('error');
		playSoundError();
		zeigeFehlerBanner(
			`Die Verbindung ist zurück. Bitte den Ausweis „${gemerkt}" noch einmal scannen — dann wird die Person richtig geladen. ` +
				`Dieser Scan wurde NICHT gebucht.`
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
		triggerScreenFlash('error');
		playSoundError();
		zeigeFehlerBanner(meldungen[einordnung.art] ?? 'Dieser Scan wurde ohne Netz NICHT gebucht.');
	}

	// Der Versand ist gescheitert (Netzfehler, Timeout, CSRF-Bootstrap ohne Netz): Der
	// Server hat den Scan nicht gesehen, der Schnappschuss geht in die Warteschlange.
	/** @param {unknown} e @param {import('../offlineQueue.js').OfflineEintrag} eintrag */
	async function verarbeiteVersandfehler(e, eintrag) {
		console.warn('Scan nicht zugestellt, wird eingereiht:', e);
		await speichereOfflineAktion(eintrag);
	}

	// Eine Antwort kam an — nichts ist offline. Was ihre Auswertung wirft, wird gezeigt.
	/** @param {unknown} e */
	function verarbeiteAntwortfehler(e) {
		if (e instanceof Error && e.message === 'BLOCK_ALERT') {
			triggerScreenFlash('error');
			playSoundError();
			return;
		}
		// Nur das Inline-Banner an der Omnibox (verschwindet nach 6s von selbst).
		// Kein zusätzlicher Toast — das war die doppelte Anzeige desselben Fehlers.
		zeigeFehlerBanner(`Fehler: ${e instanceof Error ? e.message : String(e)}`);
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
	async function submitAction(
		e,
		reloadProfileCb,
		overrideBlock = false,
		confirmedChecklist = false,
		absicht = null
	) {
		if (e) e.preventDefault();
		if (isDropdownOpen && selectedDropdownIndex >= 0) {
			selectDropdownItem(selectedDropdownIndex, null);
			return;
		}

		// Enter direkt nach dem Tippen, ohne vorher mit den Pfeiltasten auszuwählen: Wenn
		// die Vorschlagsliste genau EINEN Treffer hat, ist der gemeint.
		//
		// Ohne das ging der rohe Text an /api/action — und der Weg dort endet für eine
		// Eingabe ohne Präfix in der Titelsuche (resolveOhnePraefix → handleSearchAction),
		// die nur Bücher kennt. Ein getippter Nachname lieferte damit zuverlässig "nichts
		// gefunden", obwohl der Schüler im Vorschlag direkt darüber stand.
		//
		// Bewusst eng gefasst — genau ein Schüler und kein Buch daneben:
		//   • Bei "Müller" mit zwölf Namensgleichen darf Enter nicht raten und ein fremdes
		//     Konto öffnen. Dann bleibt die Liste zur Auswahl stehen.
		//   • Bücher bleiben außen vor. Ihr Zweig in selectDropdownItem braucht einen
		//     onSelectBook-Rückruf, den es hier nicht gibt; automatisch ausgelöst würde ein
		//     Buchscan still ins Leere laufen statt auszuleihen.
		if (
			isDropdownOpen &&
			unifiedSearchResults.students.length === 1 &&
			unifiedSearchResults.books.length === 0
		) {
			selectDropdownItem(0, null);
			return;
		}

		// EINMAL vereinheitlichen, gleich am Anfang: Dann gilt derselbe Wert fuer den
		// Online-Versand, fuer den Schnappschuss und fuer die Einordnung ohne Netz. Ein
		// von Hand getipptes `s-10001` traf sonst weder hier noch dort etwas.
		const q = normalisiereScan(queryVal);
		if (!q) return;

		queryVal = '';
		isDropdownOpen = false;
		lastFremdrueckgabe = null;
		clearFehlerBanner();

		// Disable input while processing
		document.getElementById('omnibox-input')?.blur();

		if (!merkerVertraegtDiesenScan(q)) return;

		const eintrag = schnappschuss(q, crypto.randomUUID(), absicht);

		// Zwei Fehlerklassen, zwei Zweige (OFFEN.md 2.2, Commit 2): Scheitert der VERSAND, hat
		// der Server nichts gesehen — der Schnappschuss geht in die Warteschlange. Kam eine
		// Antwort an, ist nichts offline: Was ihre Auswertung wirft, wird gezeigt. Bis zum
		// 15.09.2026 lag beides in einem catch, und ein TypeError aus der Auswertung einer
		// 200-Antwort wurde eingereiht — mit demselben Idempotenz-Schlüssel, den der Server
		// schon kannte; nach Ablauf des Caches (24 h) wäre neu gebucht worden.
		//
		// Rot gehört in den Fehlerfall, nicht ins finally: Dort feuerte es bei JEDEM
		// Scan und überschrieb das Grün, das der Erfolgspfad Millisekunden vorher
		// gesetzt hatte — die Leiste stand also nach einer geglückten Ausleihe über
		// eine Sekunde auf Fehlerfarbe (gemessen: bg-red-50/border-red-500 bei t=200
		// bis t=1200 ms). Dieselbe Zeile schluckte das Orange der Fremdrückgabe.
		// Wenn Erfolg wie Fehler aussieht, hört man auf, auf die Farbe zu schauen —
		// und übersieht dann den echten Fehler.
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
			triggerFlash('red');
			await verarbeiteVersandfehler(e, eintrag);
			scanfeldWiederScharfstellen();
			return;
		}

		try {
			if (!res.ok) {
				await handleActionHttpError(res, q); // wirft immer
			}
			const data = await res.json();
			verarbeiteAktionsErgebnis(data, reloadProfileCb, q);
		} catch (e) {
			triggerFlash('red');
			verarbeiteAntwortfehler(e);
		} finally {
			scanfeldWiederScharfstellen();
		}
	}

	/**
	 * Gibt dem Scanfeld den Fokus zurück, den submitAction oben bewusst weggenommen hat.
	 *
	 * Der blur() ist richtig — er verhindert Doppel-Scans, während die Aktion läuft. Ihm
	 * fehlte nur das Gegenstück: Ein Handscanner ist eine Tastatur und tippt blind. Ohne
	 * Fokus landen seine Zeichen im Nichts — keine Ausleihe, keine Fehlermeldung, gar
	 * nichts. Am Tresen musste man deshalb vor JEDEM Buch erst ins Feld klicken.
	 *
	 * Der bestehende $effect in Omnibox.svelte fängt das nicht ab: Er läuft nur, solange
	 * KEIN Schüler geladen ist (`!isActive`) — also genau nicht während des Ausleihens.
	 *
	 * Nicht zurückholen, solange ein Dialog eine menschliche Entscheidung braucht
	 * (Sperre, Vormerkung) oder die Kamera scannt — dort würde der Fokussprung die
	 * Bedienung stören und den Dialog wegtippbar machen.
	 */
	function scanfeldWiederScharfstellen() {
		if (showCamera || blockAlert || vormerkungAlert || checklistAnfrage) return;
		fokussiereScanfeld();
	}

	// Der Fokussprung selbst — DIE Stelle, an der ein Zeitgeber dieses Stores die Seite
	// anfasst, und damit die, die nach dem Abbau der Testumgebung den ganzen Lauf riss
	// (siehe stoppeZeitgeber). Bis zum 16.09.2026 stand derselbe Dreizeiler ein zweites
	// Mal in Omnibox.svelte; zwei Timer für eine Sache sind auch zwei Stellen, an denen
	// das Aufräumen fehlen kann.
	//
	// Nach der Aktion rendert Svelte das Profil neu; erst danach steht das Feld wieder.
	function fokussiereScanfeld() {
		if (fokusTimer) clearTimeout(fokusTimer);
		fokusTimer = setTimeout(() => {
			fokusTimer = null;
			document.getElementById('omnibox-input')?.focus();
		}, 50);
	}

	// Auch jeder Wechsel zur Ausleihe gibt dem Scanfeld den Fokus zurück (14.09.2026): Ein
	// Klick auf „Ausleihe" ließ ihn auf dem Knopf der Seitenleiste, und der nächste Scan lief
	// ohne Meldung ins Leere (e2e/scanner-fokus-menue.spec.js).
	uiStore.beimWechselZurTheke = scanfeldWiederScharfstellen;

	// Nach dem Zusammenführen zweier Datensätze (StudentProfile → onMerged): Der aktive
	// Schüler wird auf das Ziel umgehängt. Bis zum 15.09.2026 blieb an der Theke die
	// gelöschte Kennung stehen — die Akte zeigte schon das Ziel, aber jede weitere Buchung
	// und jeder Eintrag in der Offline-Warteschlange (enqueueOfflineAction) lief auf einen
	// Datensatz, den es nicht mehr gab (OFFEN.md 3.2). Die Kennung wird SOFORT gesetzt, damit
	// kein Scan dazwischen die alte erwischt; Name und Sperrflags kommen nach.
	/** @param {string} zielId */
	async function uebernimmZusammengefuehrt(zielId) {
		activeStudent = { id: zielId };
		const res = await apiFetch(`/api/schueler/${zielId}`);
		// Inzwischen ein anderer Ausweis? Dann gehört das Nachgeladene niemandem mehr.
		if (!res.ok || activeStudent?.id !== zielId) return;
		activeStudent = await res.json();
	}

	// „Buch zurückgeben" aus der Akte: derselbe Weg wie ein Scan des Barcodes, aber mit
	// bekannter Absicht. Online entscheidet ohnehin der Server (das Buch liegt beim
	// geladenen Schüler → Rückgabe); offline hätte „Schüler geladen" bis zum 15.09.2026 eine
	// Ausleihe eingereiht — beim Doppelklick eine echte zweite (OFFEN.md 2.2, Commit 3).
	/** @param {string} barcode @param {(() => void) | null} [reloadProfileCb] */
	function gibZurueck(barcode, reloadProfileCb = null) {
		queryVal = barcode;
		return submitAction(null, reloadProfileCb, false, false, 'rueckgabe');
	}

	return {
		get activeStudent() {
			return activeStudent;
		},
		/** Die ohne Netz gemerkte Ausweisnummer ("" = keine). Nur die Nummer, kein Name. */
		get offlineAusweis() {
			return offlineAusweis;
		},
		set offlineAusweis(v) {
			offlineAusweis = v;
		},
		set activeStudent(v) {
			activeStudent = v;
		},
		get abholbereit() {
			return abholbereit;
		},
		get queryVal() {
			return queryVal;
		},
		set queryVal(v) {
			queryVal = v;
		},
		get flashBorder() {
			return flashBorder;
		},
		set flashBorder(v) {
			flashBorder = v;
		},
		get screenFlash() {
			return screenFlash;
		},
		set screenFlash(v) {
			screenFlash = v;
		},
		get lastFremdrueckgabe() {
			return lastFremdrueckgabe;
		},
		set lastFremdrueckgabe(v) {
			lastFremdrueckgabe = v;
		},
		get isShaking() {
			return isShaking;
		},
		set isShaking(v) {
			isShaking = v;
		},
		get scanError() {
			return scanError;
		},
		set scanError(v) {
			scanError = v;
		},
		get errorMessage() {
			return errorMessage;
		},
		set errorMessage(v) {
			errorMessage = v;
		},
		get vormerkungAlert() {
			return vormerkungAlert;
		},
		set vormerkungAlert(v) {
			vormerkungAlert = v;
		},
		get checklistAnfrage() {
			return checklistAnfrage;
		},
		set checklistAnfrage(v) {
			checklistAnfrage = v;
		},
		get blockAlert() {
			return blockAlert;
		},
		set blockAlert(v) {
			blockAlert = v;
		},
		// isOffline handled globally
		get offlineQueueCount() {
			return offlineQueueCount;
		},
		set offlineQueueCount(v) {
			offlineQueueCount = v;
		},
		get showCamera() {
			return showCamera;
		},
		set showCamera(v) {
			showCamera = v;
		},
		get cameraScanner() {
			return cameraScanner;
		},
		set cameraScanner(v) {
			cameraScanner = v;
		},
		get debounceTimer() {
			return debounceTimer;
		},
		set debounceTimer(v) {
			debounceTimer = v;
		},
		get isDropdownOpen() {
			return isDropdownOpen;
		},
		set isDropdownOpen(v) {
			isDropdownOpen = v;
		},
		get unifiedSearchResults() {
			return unifiedSearchResults;
		},
		set unifiedSearchResults(v) {
			unifiedSearchResults = v;
		},
		get selectedDropdownIndex() {
			return selectedDropdownIndex;
		},
		set selectedDropdownIndex(v) {
			selectedDropdownIndex = v;
		},
		get totalDropdownItems() {
			return totalDropdownItems;
		},
		get isActive() {
			return isActive;
		},

		// Exportierte Methoden
		stoppeZeitgeber,
		fokussiereScanfeld,
		triggerScreenFlash,
		triggerShake,
		triggerFlash,
		showToast,
		handleInput,
		uebernimmZusammengefuehrt,
		gibZurueck,
		selectDropdownItem,
		submitAction
	};
}

export const omniboxStore = createOmniboxStore();

import { apiFetch, extractApiError } from '../../apiFetch.js';
import { uiStore } from '../../stores/uiStore.svelte.js';

/**
 * Die Serversuche der Schülerdatei. Eigene Datei wie ausweisdruck.svelte.js:
 * StudentDirectory steht an der Größen-Ratsche, und die Suche ist das Stück, das
 * nichts mit dem Führen der Liste zu tun hat.
 *
 * Gesucht wird auf dem SERVER. Vorher filterte die Ansicht im Browser über die
 * gelieferte Liste — und die ist bei 500 Zeilen gekappt. Bei 875 Schülern waren 375
 * über die Suche schlicht nicht erreichbar, welche genau hing an der alphabetischen
 * Reihenfolge der Klassennamen. Für den Benutzer sah das nach Zufall aus.
 *
 * Nebeneffekt, der den Ausschlag gab: Die Serversuche ist dieselbe wie an der Theke
 * (suchnorm) — "Muller" findet Müller, "Hoffmann Lena" dasselbe wie "Lena Hoffmann".
 * Der Browser-Filter konnte beides nicht.
 *
 * @param {() => void} nachKlassenDruck — läuft, wenn ein aus dem Druck-Center
 *   angeforderter Klassen-Stapeldruck fertig geladen ist (Treffer markieren).
 */
export function erzeugeSchuelerSuche(nachKlassenDruck) {
	/** @type {any[]} */
	let students = $state.raw([]);
	let laedt = $state(false);
	/** Leer heißt leer — ein Ladefehler heißt Ladefehler (wie im Papierkorb). */
	let ladefehler = $state('');
	let sucheLaeuft = $state(false);
	let query = $state('');
	/** Jahrgangsfilter, '' = alle (17.09.2026, OFFEN.md 9.5). Serverseitig wie die Suche:
	 *  Ein Filter im Browser säße hinter der Kappung bei 500 Zeilen und zeigte dann einen
	 *  Teil des Jahrgangs, ohne das zu sagen. */
	let jahrgang = $state('');
	/** Sortierung, '' = Reihenfolge der Kartei (17.09.2026, OFFEN.md 9.5, zweite Hälfte).
	 *  Serverseitig wie Suche und Filter: Im Browser sortiert säße die Sortierung HINTER
	 *  der Kappung bei 500 Zeilen — sie ordnete dann die ersten 500 der Kartei-Reihenfolge
	 *  um, statt die ersten 500 der gewählten. Das sieht richtig aus und ist es nicht. */
	let sortSpalte = $state('');
	let sortAbsteigend = $state(false);
	/** Die besetzten Jahrgänge fürs Auswahlfeld — vom Server, damit die Ableitung
	 *  „Klassenname → Jahrgang" nicht ein zweites Mal in JavaScript entsteht. */
	let jahrgaenge = $state.raw(/** @type {number[]} */ ([]));
	/** Wahr, wenn die Jahrgänge nicht geladen werden konnten. Leer heißt leer, ein
	 *  Ladefehler heißt Ladefehler — dieselbe Regel wie bei der Liste darüber. Ohne
	 *  diesen Zustand stünde im Auswahlfeld nur „Alle Jahrgänge", und das sähe aus wie
	 *  „diese Schule hat keine Jahrgänge". */
	let jahrgaengeFehler = $state(false);
	/** @type {ReturnType<typeof setTimeout> | undefined} */
	let timer;

	/** Muss zu ListStudentsWithStatsLimit im Backend passen: Erreicht die ungefilterte
	 *  Liste diese Länge, ist sie gekappt und die Ansicht sagt das auch. */
	const LISTEN_GRENZE = 500;

	// Nur die JÜNGSTE Anfrage darf die Liste schreiben: Beim Sprung aus dem
	// Druck-Center laufen die ungefilterte Mount-Ladung und die Klassensuche
	// gleichzeitig — welche Antwort zuletzt eintrifft, entschiede sonst der Server.
	let ladeNr = 0;

	async function lade() {
		const nr = ++ladeNr;
		laedt = true;
		try {
			const q = query.trim();
			const jg = jahrgang ? `&jahrgang=${encodeURIComponent(jahrgang)}` : '';
			const so = sortSpalte
				? `&sortierung=${encodeURIComponent(sortSpalte)}&richtung=${sortAbsteigend ? 'ab' : 'auf'}`
				: '';
			// art=alle: die LESERDATEI. Ohne diesen Zusatz liefert die Tür nur Schüler —
			// die Vorgabe gilt den anderen Aufrufern (Reiter „Ehemalige", Schülersuche des
			// Vormerkungs-Reiters), für die ein Kollege in der Liste falsch wäre.
			const res = await apiFetch(
				`/api/schueler?art=alle${q ? `&q=${encodeURIComponent(q)}` : ''}${jg}${so}`
			);
			// Nur die jüngste Anfrage schreibt — aber sie schreibt IN JEDEM FALL. Bis zum
			// 12.09.2026 hing am `nr === ladeNr` auch das `res.ok`: Scheiterte der Lauf,
			// blieben die Treffer der vorigen Suche unter dem neuen Suchtext stehen, und
			// an der Theke hat genau diese Form schon einmal auf den falschen Schüler
			// gebucht (Sweep „verschluckte Fehlantwort", Register 10.09.2026).
			if (nr !== ladeNr) return;
			if (res.ok) {
				students = (await res.json()) || [];
				ladefehler = '';
			} else {
				students = [];
				ladefehler = await extractApiError(res);
			}
		} catch (err) {
			if (nr === ladeNr) {
				students = [];
				ladefehler = 'Die Leserdatei konnte nicht geladen werden (Netzwerkfehler).';
			}
			console.error('Fehler beim Laden der Leserdatei:', err);
		} finally {
			if (nr === ladeNr) {
				laedt = false;
				sucheLaeuft = false;
			}
		}
	}

	// Erste Ladung sofort (wie erzeugeAusweisdruck sein Design lädt) — AUSSER ein
	// Klassen-Stapeldruck ist angefordert: Dann lädt gleich der Effekt unten mit der
	// Klasse im Suchfeld. Eine ungefilterte Parallel-Ladung daneben hieße, dass die
	// Antwort-Reihenfolge entscheidet, ob der Rückruf eine leere Liste markiert —
	// genau so ist es beim ersten Bau passiert (der Effekt läuft vor onMount).
	if (!uiStore.requestedKlassenDruck) lade();

	// Aus dem Druck-Center angeforderter Klassen-Stapeldruck (gleiche Mechanik wie
	// requestedStudentId): Klasse suchen, dann markiert der Aufrufer die Treffer.
	// Gedruckt wird in der Schülerdatei, hinter der Aktionsleiste — ihre Warnungen
	// (fehlendes Ablaufdatum, Etiketten-Startposition) stehen damit auch auf diesem
	// Weg vor dem Stapel.
	$effect(() => {
		const klasse = uiStore.requestedKlassenDruck;
		if (!klasse) return;
		uiStore.requestedKlassenDruck = null;
		query = klasse;
		lade().then(nachKlassenDruck);
	});

	/** Die Jahrgangsliste einmal holen.
	 *
	 *  Scheitert der Abruf, sagt das Auswahlfeld es und sperrt sich, statt eine leere
	 *  Liste anzubieten: Ein Filter, der keine Jahrgänge kennt, sieht sonst aus wie
	 *  einer, der nichts zu filtern findet. Dieselbe Entscheidung wie im
	 *  Kollegiums-Portal am 17.09.2026 — ein gescheiterter erster Abruf sagt das auch,
	 *  statt „nichts da" zu zeigen. Die Leserdatei selbst bleibt benutzbar; gesucht
	 *  werden kann weiter. */
	async function ladeJahrgaenge() {
		try {
			const res = await apiFetch('/api/jahrgaenge');
			if (res.ok) {
				jahrgaenge = (await res.json()) || [];
				jahrgaengeFehler = false;
			} else {
				jahrgaenge = [];
				jahrgaengeFehler = true;
			}
		} catch (err) {
			jahrgaenge = [];
			jahrgaengeFehler = true;
			console.error('Jahrgänge konnten nicht geladen werden:', err);
		}
	}
	ladeJahrgaenge();

	return {
		get students() {
			return students;
		},
		get jahrgang() {
			return jahrgang;
		},
		set jahrgang(wert) {
			jahrgang = wert;
		},
		get jahrgaenge() {
			return jahrgaenge;
		},
		get jahrgaengeFehler() {
			return jahrgaengeFehler;
		},
		get beschaeftigt() {
			return laedt || sucheLaeuft;
		},
		get ladefehler() {
			return ladefehler;
		},
		get query() {
			return query;
		},
		set query(wert) {
			query = wert;
		},
		get sortierung() {
			return { spalte: sortSpalte, absteigend: sortAbsteigend };
		},
		/** Ein Klick auf einen Spaltenkopf: fremde Spalte → aufsteigend, dieselbe →
		 *  Richtung umdrehen. Kein dritter Zustand „unsortiert": Eine Liste ist immer
		 *  irgendwie sortiert, und ein Klick, der die Ordnung wegnimmt, verwirrt mehr
		 *  als er hilft. */
		sortiere(spalte) {
			if (sortSpalte === spalte) {
				sortAbsteigend = !sortAbsteigend;
			} else {
				sortSpalte = spalte;
				sortAbsteigend = false;
			}
			lade();
		},
		get suchend() {
			return query.trim().length > 0 || jahrgang !== '';
		},
		get gekuerzt() {
			// NUR der Suchtext hebt die Kappung auf, ein Filter NICHT.
			//
			// Hier stand bis zum 17.09.2026 zusätzlich `jahrgang === ''` — mit dem Kommentar,
			// ein Jahrgangsfilter laufe „wie die Suche auf dem Server OHNE Kappung". Der Satz
			// war falsch: Das LIMIT 500 hängt allein daran, ob ein SUCHTEXT da ist
			// (repository/student_profile_queries.go), der Klassenfilter wird erst danach
			// angehängt. Am echten Postgres nachgemessen: 520 Leser einer Klasse ergeben 500
			// Zeilen. Ein Jahrgang dieser Schule bleibt darunter, aber das ist Glück und keine
			// Zusicherung — und ein Schutz, den nur ein Kommentar behauptet, ist keiner.
			return !query.trim() && students.length >= LISTEN_GRENZE;
		},
		lade,

		// Tippen wird entprellt, damit nicht jeder Tastendruck eine Abfrage auslöst.
		// 300 ms wie in der Omnibox — dieselbe Eingabegeschwindigkeit, dieselbe Wartezeit.
		angestossen() {
			sucheLaeuft = true;
			clearTimeout(timer);
			timer = setTimeout(lade, 300);
		}
	};
}

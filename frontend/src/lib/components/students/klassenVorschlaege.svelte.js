import { apiFetch, extractApiError } from '../../apiFetch.js';

/**
 * Die Klassen der Schule für jede Klassenauswahl — Anlegen-Dialog, Leserakte,
 * Mahnwesen-Routing, Klassensätze, LMF-Verlängerung. Eigene Datei aus demselben Grund wie
 * schuelerSuche.svelte.js und ausweisdruck.svelte.js: StudentDirectory steht an der
 * Größen-Ratsche (200 Zeilen), und das Nachschlagen der Klassen hat mit dem Führen der
 * Liste nichts zu tun.
 *
 * Quelle ist GET /api/klassen (die Klassen der aktiven Schüler, also die aus der LUSD). Bis
 * zum 15.09.2026 las die Auswahl die Tabelle lesergruppen, die kein Schreibweg füllt. Seit
 * dem 30.09.2026 wird die Klasse an allen diesen Stellen nur noch gewählt, nicht getippt
 * (docs/OFFEN.md 5.18).
 *
 * Damit ist die Liste kein Vorschlag mehr, sondern die einzige Wahl. Bis zum 30.09.2026
 * schwieg ein gescheiterter Abruf, denn man tippte die Klasse dann von Hand; danach sagte
 * die leere Auswahl „Keine Klassen" und der Anlegen-Dialog „Die Klassen kommen mit dem
 * LUSD-Abgleich" — ein falscher Grund für eine gesperrte Eingabe. Leer heißt leer, ein
 * Ladefehler heißt Ladefehler (dieselbe Regel wie geraeteListe.svelte.js).
 */
export function erzeugeKlassenVorschlaege() {
	/** @type {string[]} */
	let liste = $state.raw([]);
	let ladefehler = $state(false);

	async function lade() {
		try {
			const res = await apiFetch('/api/klassen');
			if (!res.ok) throw new Error(await extractApiError(res));
			const daten = await res.json();
			liste = Array.isArray(daten) ? daten : [];
			ladefehler = false;
		} catch (err) {
			// Eine Liste von vorher bleibt stehen: Sie ist älter, aber nicht falsch.
			ladefehler = true;
			console.error('Fehler beim Laden der Klassen:', err);
		}
	}

	return {
		get liste() {
			return liste;
		},
		get ladefehler() {
			return ladefehler;
		},
		lade
	};
}

/**
 * Der Fehlertext unter einer Klassenauswahl. Er ersetzt den Hinweis darunter und sagt, wie es
 * weitergeht — M3 Text fields, „Error text": „replace supporting text with error text" und „If
 * only one error is possible, error text should describe how to avoid the error".
 */
export const KLASSEN_LADEFEHLER = 'Die Klassen ließen sich nicht laden. Bitte neu öffnen.';

/**
 * Der Platzhalter einer Klassenauswahl — an allen Stellen dieselben Wörter.
 * @param {number} anzahl die Zahl der wählbaren Klassen
 * @param {boolean} ladefehler ob der Abruf der Klassen gescheitert ist
 * @param {string} [leer] was bei einer leeren Liste dasteht
 * @returns {string}
 */
export function klassenPlatzhalter(anzahl, ladefehler, leer = 'Keine Klassen') {
	if (anzahl > 0) return 'Klasse wählen';
	return ladefehler ? 'Klassen nicht geladen' : leer;
}

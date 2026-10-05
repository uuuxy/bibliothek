import { apiGet } from '../../apiFetch.js';

/**
 * Die Liste der Schlagwort-Pflege: GET /api/schlagworte/pflege?suche=… Gesucht wird am
 * Server über alle Wörter und Verweise; die Antwort bringt höchstens 200 Zeilen und die
 * Zahlen dazu (gesamt, verweise, filter, treffer). Genutzt von der Pflegeseite
 * (SchlagworteKategorie) und von der Wortsuche im Zusammenführen-Dialog
 * (SchlagwortPflegeDialog) — eine Suche, eine Regel.
 *
 * Bis zum 30.09.2026 lud die Seite die ersten 5.000 Wörter in alphabetischer Folge und suchte
 * im Browser nur darunter. Littera bringt 13.207 Wörter mit (docs/OFFEN.md 4.20); zwei
 * Drittel davon wären weder zu finden noch als Filter zu markieren gewesen. Dasselbe Muster
 * wie die Leserdatei (students/schuelerSuche.svelte.js): Tippen wird entprellt, und nur die
 * jüngste Anfrage schreibt die Liste.
 */
export function erzeugeSchlagwortPflegeListe() {
	/** @type {{ zeilen: import('./schlagwortPflege.js').SchlagwortZeile[], gesamt: number, verweise: number, filter: number, treffer: number } | null} */
	let liste = $state(null);
	let ladeFehler = $state(false);
	let suche = $state('');
	/** @type {ReturnType<typeof setTimeout> | undefined} */
	let timer;
	// Sequenznummer wie in useStudentProfile: Zwei schnell umgelegte Schalter laden die Liste
	// zweimal, und kam die ältere Antwort zuletzt, zeigte ein Schalter den alten Stand.
	let laufNr = 0;

	async function lade() {
		clearTimeout(timer);
		const meine = ++laufNr;
		const text = suche.trim();
		try {
			const frage = text ? `?suche=${encodeURIComponent(text)}` : '';
			const antwort = await apiGet(`/api/schlagworte/pflege${frage}`);
			if (meine !== laufNr) return; // eine jüngere Liste ist schon unterwegs oder da
			liste = antwort;
			ladeFehler = false;
		} catch {
			if (meine === laufNr) ladeFehler = true; // Meldung kam bereits aus apiGet.
		}
	}

	return {
		get liste() {
			return liste;
		},
		get ladeFehler() {
			return ladeFehler;
		},
		get suche() {
			return suche;
		},
		set suche(wert) {
			suche = wert;
		},
		lade,
		// 300 ms wie in der Leserdatei und der Omnibox — dieselbe Eingabe, dieselbe Wartezeit.
		angestossen() {
			clearTimeout(timer);
			timer = setTimeout(lade, 300);
		}
	};
}

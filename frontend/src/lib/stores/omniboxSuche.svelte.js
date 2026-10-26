// Die Suche der Theke: Entprellung, die Nummer des Suchlaufs, die Trefferliste und die Auswahl
// daraus, über GET /api/search. Nur der Omnibox-Store lädt diese Datei
// (frontend-hygiene-omnibox-teile.test.js); das Eingabefeld und den Scan behält er und reicht
// sie als Funktionen herein.

import { apiFetch, leseApiFehler } from '../apiFetch.js';

/**
 * @param {{
 *   leseEingabe: () => string,
 *   setzeEingabe: (text: string) => void,
 *   fuehreAktionAus: (e: Event | null, reloadProfileCb: (() => void) | null) => unknown,
 *   melde: (message: string, type?: import('./snackbars.svelte.js').ToastTyp) => void
 * }} store
 */
export function erzeugeOmniboxSuche({ leseEingabe, setzeEingabe, fuehreAktionAus, melde }) {
	let suchVerzoegerung = $state(/** @type {any} */ (null));
	let trefferlisteOffen = $state(false);
	let suchTreffer = $state({ students: [], books: [], studentsTotal: 0, booksTotal: 0 });
	let markierterTreffer = $state(-1);
	let trefferAnzahl = $derived(suchTreffer.students.length + suchTreffer.books.length);

	// Stoppt den Zeitgeber der Entprellung; der Store ruft es aus seinem stoppeZeitgeber.
	function stoppeZeitgeber() {
		if (suchVerzoegerung) clearTimeout(suchVerzoegerung);
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
		if (!leseEingabe().trim()) {
			// Auch hier hochzählen: Ein noch laufender Abruf darf die eben geleerte
			// Liste nicht wieder füllen.
			suchLauf++;
			trefferlisteOffen = false;
			suchTreffer = { students: [], books: [], studentsTotal: 0, booksTotal: 0 };
			return;
		}
		suchVerzoegerung = setTimeout(async () => {
			if (!leseEingabe().trim()) return;
			// Zwei laufende Abrufe beantwortet der Server in beliebiger Reihenfolge. Ohne die
			// Nummer tauschte eine verspätete Antwort die Trefferliste unter dem Suchtext aus,
			// und der nächste Klick buchte auf den falschen Schüler.
			const seq = ++suchLauf;
			try {
				const res = await apiFetch(`/api/search?q=${encodeURIComponent(leseEingabe().trim())}`);
				if (seq !== suchLauf) return;
				if (!res.ok) {
					// Dieselbe Gefahr beim Fehlschlag: Bliebe die Liste des vorigen Suchtextes stehen,
					// buchte der nächste Klick auf einen Treffer, der zum Feld nicht mehr passt.
					verwirfTreffer();
					const grund = await leseApiFehler(res, 'Suche fehlgeschlagen — bitte erneut versuchen.');
					if (seq === suchLauf) melde(grund, 'error');
					return;
				}
				const ergebnisse = await res.json();
				// Auch das Lesen dauert: Inzwischen kann ein neuer Lauf begonnen oder die Theke
				// geleert sein.
				if (seq !== suchLauf) return;
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
			setzeEingabe(`leser:${gewaehlterLeser.id}`);
			trefferlisteOffen = false;
			fuehreAktionAus(null, null); // Ohne Event
		} else {
			const buch = books[index - students.length];
			setzeEingabe('');
			trefferlisteOffen = false;
			if (onSelectBook) onSelectBook(buch);
		}
	}

	return /** @satisfies {object} */ ({
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
		stoppeZeitgeber,
		verarbeiteEingabe,
		waehleTreffer
	});
}

import { apiFetch, extractApiError } from '../../../../lib/apiFetch.js';

/**
 * Die Suche des Dialogs „Andere Auflage zuordnen" (docs/OFFEN.md 4.18, Stufe 2) — eigene
 * Datei wie zusammenfuehrenSuche.svelte.js und nach denselben drei Regeln: Nur die JÜNGSTE
 * Anfrage schreibt die Liste; ein Fehler heißt Fehler, nicht „nichts gefunden";
 * Zurücksetzen stoppt den Timer und entwertet die laufende Antwort.
 *
 * Gesucht wird über die Tür der Titel-Verwaltung (GET /api/books?q=), zweimal: Sie liefert
 * entweder die Titel mit Exemplaren oder mit `bestand=ohne` die ohne. Beide Hälften gehören
 * hierher — eine alte Auflage hat oft kein Exemplar mehr und steht trotzdem in der
 * Nachbestell-Liste. Angeboten werden nur Lernmittel; zusammengefasst werden nur Auflagen
 * eines Schulbuchs (repository/auflagen.go), und alles andere wiese der Server ab.
 *
 * @param {() => string[]} bekannte — IDs, die schon zu diesem Buch gehören (der Titel
 *   selbst eingeschlossen); sie sind keine Kandidaten.
 */
export function erzeugeAuflagenSuche(bekannte) {
	let suche = $state('');
	/** @type {any[]} */
	let treffer = $state([]);
	let fehler = $state('');
	/** @type {ReturnType<typeof setTimeout> | undefined} */
	let timer;
	let nr = 0;

	async function suchen() {
		const meine = ++nr;
		const q = suche.trim();
		if (q.length < 2) {
			treffer = [];
			fehler = '';
			return;
		}
		const pfad = `/api/books?q=${encodeURIComponent(q)}`;
		try {
			const [mit, ohne] = await Promise.all([apiFetch(pfad), apiFetch(`${pfad}&bestand=ohne`)]);
			const antwort = !mit.ok ? mit : !ohne.ok ? ohne : null;
			const meldung = antwort ? await extractApiError(antwort) : '';
			const liste = antwort ? [] : kandidaten([await mit.json(), await ohne.json()], bekannte());
			if (meine !== nr) return;
			treffer = liste;
			fehler = meldung;
		} catch {
			if (meine !== nr) return;
			treffer = [];
			fehler = 'Netzwerkfehler — die Suche hat den Server nicht erreicht.';
		}
	}

	return {
		get suche() {
			return suche;
		},
		set suche(wert) {
			suche = wert;
		},
		get treffer() {
			return treffer;
		},
		get fehler() {
			return fehler;
		},
		tippen() {
			clearTimeout(timer);
			timer = setTimeout(suchen, 250);
		},
		zuruecksetzen() {
			clearTimeout(timer);
			nr++;
			suche = '';
			treffer = [];
			fehler = '';
		}
	};
}

/** Höchstens so viele Treffer; wer mehr braucht, tippt weiter. */
export const AUFLAGEN_TREFFER_MAX = 20;

/**
 * Führt die beiden Antworten zusammen: nur Lernmittel, ohne die bekannten, jeder Titel
 * einmal, nach Titel und dann dem jüngsten Jahr.
 * @param {any[]} antworten — die Körper der beiden Aufrufe ({ data: [...] })
 * @param {string[]} bekannte
 */
export function kandidaten(antworten, bekannte) {
	// Liste und Objekt statt Set und Map: Die Rechnung ist ein Durchgang ohne Zustand, und
	// svelte/prefer-svelte-reactivity verlangt in .svelte.js-Dateien sonst SvelteSet/SvelteMap.
	const aus = bekannte.map((id) => id.toLowerCase());
	/** @type {Record<string, any>} */
	const gesehen = {};
	for (const antwort of antworten) {
		for (const t of antwort?.data ?? []) {
			const id = String(t.id).toLowerCase();
			if (!t.istLernmittel || aus.includes(id) || id in gesehen) continue;
			gesehen[id] = t;
		}
	}
	return Object.values(gesehen)
		.sort(
			(a, b) =>
				String(a.title).localeCompare(String(b.title), 'de') ||
				(b.erscheinungsjahr || 0) - (a.erscheinungsjahr || 0)
		)
		.slice(0, AUFLAGEN_TREFFER_MAX);
}

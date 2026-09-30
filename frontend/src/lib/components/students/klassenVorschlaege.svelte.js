import { apiFetch } from '../../apiFetch.js';

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
 * Scheitert der Abruf, bleibt die Auswahl leer, und der Platzhalter der Auswahl sagt es —
 * derselbe geduldete Fall wie die übrigen Vorschlagslisten (fehlerausgang.test.js).
 */
export function erzeugeKlassenVorschlaege() {
	/** @type {string[]} */
	let liste = $state.raw([]);

	async function lade() {
		try {
			const res = await apiFetch('/api/klassen');
			if (res.ok) {
				const daten = await res.json();
				liste = Array.isArray(daten) ? daten : [];
			}
		} catch (err) {
			console.error('Fehler beim Laden der Klassen:', err);
		}
	}

	return {
		get liste() {
			return liste;
		},
		lade
	};
}

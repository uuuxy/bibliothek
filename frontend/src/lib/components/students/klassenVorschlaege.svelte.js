import { apiFetch } from '../../apiFetch.js';

/**
 * Die Klassen der Schule als Vorschlag im Anlegen-Dialog — eigene Datei aus demselben
 * Grund wie schuelerSuche.svelte.js und ausweisdruck.svelte.js: StudentDirectory steht an
 * der Größen-Ratsche (200 Zeilen), und das Nachschlagen der Klassen hat mit dem Führen der
 * Liste nichts zu tun.
 *
 * Quelle ist GET /api/klassen (die Klassen der aktiven Schüler), wie im Druck-Center und
 * bei der LMF-Verlängerung. Bis zum 15.09.2026 las die Auswahl die Tabelle lesergruppen,
 * die kein Schreibweg füllt — angeboten wurde nur „Manuell eingeben…".
 *
 * Scheitert der Abruf, bleibt die Auswahl leer: Die Klasse lässt sich dann von Hand
 * eintragen, es geht nichts verloren — derselbe geduldete Fall wie die übrigen
 * Vorschlagslisten (fehlerausgang.test.js).
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

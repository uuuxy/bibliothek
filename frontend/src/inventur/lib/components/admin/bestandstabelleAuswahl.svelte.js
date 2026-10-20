// bestandstabelleAuswahl.svelte.js — die Auswahl der Bestandstabelle für Massenaktionen.
//
// `angleichen()` hält die Auswahl auf dem, was in der Liste steht: Überlebte sie einen Such-
// oder Filterwechsel, sagte die Werkzeugleiste nach „Alle auswählen" und „Mathe" weiter
// „Löschen (312)", und gelöscht oder einem Klassensatz zugefügt würden Titel, die niemand
// mehr sieht.

/** @typedef {{ id: string }} Zeile */

export function erzeugeAuswahl() {
	/** @type {string[]} */
	let ids = $state([]);

	return /** @satisfies {object} */ ({
		get ids() {
			return ids;
		},
		get anzahl() {
			return ids.length;
		},
		/** @param {string} id */
		enthaelt(id) {
			return ids.includes(id);
		},
		/** Alles ausgewählt? Nur wahr, wenn es überhaupt Zeilen gibt — sonst sähe eine
		 *  leere Liste wie „alles ausgewählt" aus.
		 *  @param {Zeile[]} zeilen */
		alleGewaehlt(zeilen) {
			return zeilen.length > 0 && ids.length === zeilen.length;
		},
		/** Auswahl auf die sichtbaren Zeilen beschränken (im $effect der Tabelle).
		 *  @param {Zeile[]} zeilen */
		angleichen(zeilen) {
			// Bewusst eine Liste statt eines Set: Der Wert lebt nur in dieser Funktion und
			// wird nie beobachtet — prefer-svelte-reactivity zielt auf Sammlungen im
			// Zustand, nicht auf lokale Zwischenschritte.
			const sichtbar = zeilen.map((z) => z.id);
			const rest = ids.filter((id) => sichtbar.includes(id));
			if (rest.length !== ids.length) ids = rest;
		},
		/** @param {Zeile[]} zeilen */
		alleUmschalten(zeilen) {
			ids = this.alleGewaehlt(zeilen) ? [] : zeilen.map((z) => z.id);
		},
		/** @param {string} id */
		umschalten(id) {
			ids = ids.includes(id) ? ids.filter((x) => x !== id) : [...ids, id];
		},
		leeren() {
			ids = [];
		}
	});
}

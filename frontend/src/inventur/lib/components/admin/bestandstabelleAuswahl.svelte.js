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
			// Ein Objekt als Nachschlagetabelle statt includes auf einer Liste: Der Abgleich wuchs mit
			// dem Quadrat der Zahl, bei 11.000 markierten Titeln 0,6 s je Tastendruck. Kein Set: Das
			// verlangte in einer .svelte.js ein SvelteSet (prefer-svelte-reactivity), beobachtet wird
			// es aber nie, wie in auflagenSuche.
			/** @type {Record<string, true>} */
			const sichtbar = Object.create(null);
			for (const z of zeilen) sichtbar[z.id] = true;
			const rest = ids.filter((id) => sichtbar[id] === true);
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

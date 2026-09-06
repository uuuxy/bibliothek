// bookTableAuswahl.svelte.js — die Auswahl der Bestandstabelle für Massenaktionen.
//
// Eigene Datei seit dem Rasterdurchgang am 06.09.2026: Die Tabelle steht mit gut 240
// Zeilen ohnehin über der 200-Zeilen-Regel, und der Fix des Durchgangs hätte sie weiter
// wachsen lassen. Die Ratsche lockert man dafür nicht — hier liegt jetzt die Auswahl,
// dort die Tabelle.
//
// Der Fund: Die Auswahl überlebte jeden Such- und Filterwechsel. „Alle auswählen" bei 312
// Titeln, dann „Mathe" tippen — vier Zeilen sichtbar, keine angehakt, und die
// Werkzeugleiste sagte weiter „Löschen (312)". Die Rückfrage nannte dieselbe Zahl,
// gefallen wären die 312 unsichtbaren Titel; derselbe Weg führt über „Zum Klassensatz
// hinzufügen". Deshalb hält `angleichen()` die Auswahl auf dem, was in der Liste steht.

/** @typedef {{ id: string }} Zeile */

export function erzeugeAuswahl() {
	/** @type {string[]} */
	let ids = $state([]);

	return {
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
	};
}

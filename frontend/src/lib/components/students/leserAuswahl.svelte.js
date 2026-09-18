import { SvelteSet } from 'svelte/reactivity';

/**
 * Die Markierung der Leserdatei — welche Zeilen für den Ausweis-Stapeldruck angekreuzt
 * sind.
 *
 * Eigene Datei aus demselben Grund wie ausweisdruck.svelte.js und schuelerSuche.svelte.js:
 * StudentDirectory.svelte steht an der Größen-Ratsche (200 Zeilen, docs/arc42/05-bausteinsicht.md 5.3),
 * und das Ankreuzen ist ein Stück für sich — es hat nichts damit zu tun, WELCHE Leser die
 * Liste führt.
 *
 * SvelteSet statt Set: Ein einfaches Set ist für Svelte 5 ein undurchsichtiger Wert;
 * `.add()`/`.delete()` lösten kein Neuzeichnen aus, und die Haken blieben beim Klicken
 * stehen. SvelteSet macht die Mitgliedschaft selbst reaktiv. Set statt Array, weil das
 * Ankreuzen bei jeder Zeile fragt „ist die dabei?" — das ist der Zugriff, den ein Set kann.
 *
 * @param {() => any[]} sichtbare — die aktuell ANGEZEIGTEN Leser. Als Funktion, nicht als
 *   Wert: Die Liste ändert sich mit jeder Suche, und die Markierung muss sich auf das
 *   beziehen, was gerade zu sehen ist.
 */
export function erzeugeLeserAuswahl(sichtbare) {
	const auswahl = new SvelteSet();

	const markierte = $derived(sichtbare().filter((/** @type {any} */ s) => auswahl.has(s.id)));

	// Karten ohne ableitbares Ablaufjahr würden „31.07.–" tragen. Der Balken sagt das VOR
	// dem Druck, nicht der fertige Stapel hinterher.
	const ohneDatum = $derived(
		markierte.filter((/** @type {any} */ s) => s.ausweis_gueltig_bis == null).length
	);

	return {
		get auswahl() {
			return auswahl;
		},
		get markierte() {
			return markierte;
		},
		get ohneDatum() {
			return ohneDatum;
		},

		/** @param {string} id */
		umschalten(id) {
			if (!auswahl.delete(id)) auswahl.add(id);
		},

		alleUmschalten() {
			// Bezugsgröße ist die ANGEZEIGTE Liste, nicht der Gesamtbestand: Wer nach „7H"
			// sucht und „alle" ankreuzt, meint die Treffer vor sich — nicht 875 Schüler.
			const alleSchonDrin = markierte.length === sichtbare().length;
			auswahl.clear();
			if (!alleSchonDrin) {
				for (const s of sichtbare()) auswahl.add(s.id);
			}
		},

		leeren() {
			auswahl.clear();
		},

		/** Alle sichtbaren markieren — nach dem Sprung aus dem Druck-Center. */
		alleSichtbarenMarkieren() {
			auswahl.clear();
			for (const s of sichtbare()) auswahl.add(s.id);
		}
	};
}

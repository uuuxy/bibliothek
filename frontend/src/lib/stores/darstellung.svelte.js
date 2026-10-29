// stores/darstellung.svelte.js — „Dunkles Design" an oder aus, gewählt je Rechner.
//
// Wie bei Google (Gemini-Hilfe, „Use Dark theme in Gemini Apps"): Ohne Wahl gilt das Farbschema
// des Systems, eine Wahl gilt in diesem Browser auf diesem Rechner. Sie steht deshalb im Speicher
// des Browsers und nicht am Server, wo die Einstellungen für alle Arbeitsplätze liegen. Vor dem
// ersten Bild setzt public/darstellung.js die Klasse nach derselben Regel; darstellung.test.js
// hält beide gleich.

import { MediaQuery } from 'svelte/reactivity';

export const SCHLUESSEL = 'bibliothek.darstellung';

/** @typedef {'hell' | 'dunkel' | 'system'} Wahl */

/**
 * Ein fehlender oder unbekannter Wert heißt: wie das System.
 * @param {string | null} gespeichert
 * @returns {Wahl}
 */
export function leseWahl(gespeichert) {
	return gespeichert === 'hell' || gespeichert === 'dunkel' ? gespeichert : 'system';
}

/**
 * @param {Wahl} wahl
 * @param {boolean} systemDunkel
 */
export function istDunkel(wahl, systemDunkel) {
	return wahl === 'dunkel' || (wahl === 'system' && systemDunkel);
}

function gespeichert() {
	try {
		return localStorage.getItem(SCHLUESSEL);
	} catch {
		return null;
	}
}

let wahl = $state(leseWahl(gespeichert()));
/** Erst in starte(): Außerhalb des Browsers gibt es kein matchMedia. @type {MediaQuery | undefined} */
let system;

export const darstellung = /** @satisfies {object} */ ({
	/** Steht die Seite dunkel? Ohne Wahl entscheidet das System. */
	get dunkel() {
		return istDunkel(wahl, system?.current ?? false);
	},

	/** @param {boolean} dunkel */
	schalte(dunkel) {
		wahl = dunkel ? 'dunkel' : 'hell';
		try {
			localStorage.setItem(SCHLUESSEL, wahl);
		} catch {
			// Ohne Speicher (privates Fenster) gilt die Wahl, bis das Fenster neu lädt.
		}
	},

	/** Hält die Klasse an <html> aktuell: bei einer Wahl, einer Änderung im System und einer
	 *  Wahl in einem anderen Fenster dieses Browsers. */
	starte() {
		system = new MediaQuery('(prefers-color-scheme: dark)');
		$effect.root(() => {
			$effect(() => {
				document.documentElement.classList.toggle('schema-dunkel', this.dunkel);
			});
		});
		window.addEventListener('storage', (e) => {
			if (e.key === SCHLUESSEL) wahl = leseWahl(e.newValue);
		});
	}
});

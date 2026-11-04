/**
 * Rückfrage-Dienst: ersetzt window.confirm() durch den M3-Dialog der Anwendung.
 * Browser-Dialoge sehen auf jedem Rechner anders aus, halten sich nicht an Farben, Schrift
 * und Dichte der Anwendung und werden per Enter unbeabsichtigt bestätigt. Das hier ist der
 * billige Weg zum eigenen Dialog:
 *
 *     if (!(await bestaetigen({ titel: 'Klasse 05A löschen?', aktion: 'Löschen',
 *                               gefaehrlich: true }))) return;
 *
 * Der Dialog selbst (ui/BestaetigungsDialog.svelte) hängt einmal in App.svelte neben dem
 * SnackbarAnzeige und liest diesen Store. Meldungen ohne Entscheidung (`alert`) gehen an
 * den snackbars, denn eine Meldung braucht keinen Klick.
 *
 * Eine zweite Rückfrage, während eine offen ist, beantwortet die erste mit „nein":
 * Zwei Dialoge übereinander wären genau die Unordnung, die confirm() hatte.
 */

import { AUSSONDERUNGSGRUENDE, GRUND_VORGABE } from '../utils/aussonderungsgruende.js';

/**
 * `auswahl`: Auswahlknöpfe mit Vorauswahl (M3: „Radio buttons should always have one option
 * pre-selected"). `zusatz`: ein Hinweis mit Liste und ein eigenes Kästchen; angekreuzt nennt der
 * Knopf `aktionMit`, und mit `noetig` geht er erst, wenn es angekreuzt ist (M3, Dialogs: „Disable
 * confirming actions until a choice is made.").
 * @typedef {{
 *   titel: string,
 *   text?: string,
 *   aktion?: string,
 *   abbruch?: string,
 *   gefaehrlich?: boolean,
 *   auswahl?: { beschriftung: string, optionen: { wert: string, text: string }[], vorgabe: string },
 *   zusatz?: { hinweis: string, liste?: string[], nachsatz?: string, kaestchen: string, aktionMit: string, noetig?: boolean }
 * }} Rueckfrage
 */

/** @typedef {{ grund: string, zusatz: boolean }} Wahl */

/** @type {(Rueckfrage & { resolve: (antwort: boolean | null, wahl?: Wahl) => void }) | null} */
let offen = $state(null);

export const bestaetigungStore = /** @satisfies {object} */ ({
	get anfrage() {
		return offen;
	},
	/**
	 * @param {boolean | null} antwort null: geschlossen, ohne einen Knopf zu wählen
	 * @param {Wahl} [wahl] Grund und Kästchen, wenn die Frage eine Auswahl hat
	 */
	antworten(antwort, wahl) {
		const a = offen;
		offen = null;
		a?.resolve(antwort, wahl);
	}
});

/**
 * Eine Frage, bei der auch der zweite Knopf etwas auslöst („Titel öffnen" oder „Anderes
 * Medium"). Anders als bestaetigen trennt sie den zweiten Knopf vom Schließen: Escape, der
 * Klick neben den Dialog und eine zweite Frage sind keine Antwort.
 * @param {Rueckfrage} frage
 * @returns {Promise<boolean | null>} true: die Aktion, false: der zweite Knopf, null: geschlossen
 */
export function fragen(frage) {
	return new Promise((resolve) => {
		offen?.resolve(null);
		offen = { aktion: 'Fortfahren', abbruch: 'Abbrechen', gefaehrlich: false, ...frage, resolve };
	});
}

/**
 * Stellt die Rückfrage und löst mit der Antwort auf. `aktion` ist die Beschriftung
 * des bestätigenden Knopfs — ein Verb („Löschen", „Veröffentlichen"), nie „OK".
 * @param {Rueckfrage} frage
 * @returns {Promise<boolean>}
 */
export function bestaetigen(frage) {
	return fragen(frage).then((antwort) => antwort === true);
}

/**
 * Die häufigste Rückfrage in einer Zeile: Löschen, gefährlich, Knopf „Löschen". Der
 * Erklärtext ist die M3-Formel für Unwiderrufliches; wer mehr zu sagen hat (Zahl der
 * Exemplare, Folgen für andere), gibt ihn mit.
 * @param {string} titel
 * @param {string} [text]
 */
export function loeschenBestaetigen(titel, text = 'Das lässt sich nicht rückgängig machen.') {
	return bestaetigen({ titel, text, aktion: 'Löschen', gefaehrlich: true });
}

/**
 * Die Rückfrage vor dem Löschen mit Grund: die vier Gründe, vorgewählt „Alter Bestand", und,
 * wenn die Frage es mitbringt, das eigene Kästchen für verliehene Exemplare. Gefährlich, also mit
 * dem Fokus auf „Abbrechen".
 * @param {Rueckfrage} frage
 * @returns {Promise<Wahl | null>} null: abgebrochen
 */
export function loeschenMitGrund(frage) {
	return new Promise((resolve) => {
		offen?.resolve(null);
		offen = {
			aktion: 'Löschen',
			abbruch: 'Abbrechen',
			gefaehrlich: true,
			auswahl: { beschriftung: 'Grund', optionen: AUSSONDERUNGSGRUENDE, vorgabe: GRUND_VORGABE },
			...frage,
			resolve: (antwort, wahl) => resolve(antwort === true && wahl ? wahl : null)
		};
	});
}

/**
 * Die Rückfrage, bevor ungespeicherte Eingaben einer Maske verloren gehen; der Fokus steht auf
 * „Bleiben". M3, Dialogs, „Dismissing": Wer einen Full-screen dialog verlässt, wird gefragt.
 * Der Sperrbildschirm stellt sie selbst, weil hinter ihm keine Rückfrage der Anwendung steht.
 * @param {string} [aktion] der bestätigende Knopf: beim Schließen „Verwerfen", beim Wechsel
 *   „Verwerfen und weiter", beim Abmelden „Verwerfen und abmelden"
 * @returns {Rueckfrage}
 */
export function verwerfenFrage(aktion = 'Verwerfen') {
	return {
		titel: 'Ungespeicherte Änderungen',
		text: 'Die Eingaben in der Maske sind nicht gespeichert und gehen dabei verloren.',
		aktion,
		abbruch: 'Bleiben',
		gefaehrlich: true
	};
}

/** @param {string} [aktion] wie bei verwerfenFrage */
export function verwerfenBestaetigen(aktion = 'Verwerfen') {
	return bestaetigen(verwerfenFrage(aktion));
}

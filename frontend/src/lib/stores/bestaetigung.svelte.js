/**
 * Rückfrage-Dienst: ersetzt window.confirm() durch den M3-Dialog der Anwendung.
 *
 * Bis zum 08.09.2026 stellten 19 Stellen ihre Rückfrage über `confirm()` und 15 ihre
 * Meldung über `alert()` — Browser-Dialoge, die auf jedem Rechner anders aussehen,
 * sich nicht an Farben, Schrift und Dichte der Anwendung halten und per Enter
 * unbeabsichtigt bestätigt werden. Zwei Stellen hatten sich schon einen eigenen
 * M3-Dialog gebaut (VerlustLoeschenDialog, PapierkorbLoeschenDialog); der Rest fiel
 * zurück, weil es keinen billigen Weg gab. Das hier ist der billige Weg:
 *
 *     if (!(await bestaetigen({ titel: 'Klasse 05A löschen?', aktion: 'Löschen',
 *                               gefaehrlich: true }))) return;
 *
 * Der Dialog selbst (ui/BestaetigungsDialog.svelte) hängt EINMAL in App.svelte neben
 * dem ToastContainer und liest diesen Store. Meldungen ohne Entscheidung (`alert`)
 * gehen an den toastStore — eine Meldung braucht keinen Klick.
 *
 * Eine zweite Rückfrage, während eine offen ist, beantwortet die erste mit „nein":
 * Zwei Dialoge übereinander wären genau die Unordnung, die confirm() hatte.
 */

/**
 * @typedef {{
 *   titel: string,
 *   text?: string,
 *   aktion?: string,
 *   abbruch?: string,
 *   gefaehrlich?: boolean
 * }} Rueckfrage
 */

/** @type {(Rueckfrage & { resolve: (ja: boolean) => void }) | null} */
let offen = $state(null);

export const bestaetigungStore = {
	get anfrage() {
		return offen;
	},
	/** @param {boolean} ja */
	antworten(ja) {
		const a = offen;
		offen = null;
		a?.resolve(ja);
	}
};

/**
 * Stellt die Rückfrage und löst mit der Antwort auf. `aktion` ist die Beschriftung
 * des bestätigenden Knopfs — ein Verb („Löschen", „Veröffentlichen"), nie „OK".
 * @param {Rueckfrage} frage
 * @returns {Promise<boolean>}
 */
export function bestaetigen(frage) {
	return new Promise((resolve) => {
		offen?.resolve(false);
		offen = { aktion: 'Fortfahren', abbruch: 'Abbrechen', gefaehrlich: false, ...frage, resolve };
	});
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

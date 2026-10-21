import { snackbars } from '../../stores/snackbars.svelte.js';

/**
 * Die Anfrage eines Dialogs läuft weiter, wenn er schließt: „Abbrechen" ist nie gesperrt
 * (Material 3, Dialogs: „Dismissive actions are never disabled."). Was danach ankommt, zeigt
 * nicht mehr der Dialog, sondern die Snackbar (Material 3, Snackbar: „Snackbars inform users
 * of a process that an app has performed").
 *
 * Im Aufbau des Bauteils aufrufen; `offen` liest die Prop, die den Dialog öffnet.
 * @param {() => boolean} offen
 */
export function erzeugeDialogAnfrage(offen) {
	// Zählt jedes Schließen und das Abbauen des Bauteils: Ein Dialog, der zu und wieder auf
	// ging, ist nicht mehr der, aus dem die Anfrage kam.
	let geschlossen = 0;
	$effect(() => {
		if (!offen()) geschlossen++;
	});
	$effect(() => () => geschlossen++);

	return {
		/** Vor dem Absenden aufrufen; das Ergebnis meldet die Antwort an der richtigen Stelle. */
		beginne() {
			const stand = geschlossen;
			const nochOffen = () => stand === geschlossen && offen();
			return {
				nochOffen,
				/**
				 * Steht im Dialog, solange er offen ist, sonst in der Snackbar.
				 * @param {string} text
				 * @param {(text: string) => void} imDialog
				 */
				fehler(text, imDialog) {
					if (nochOffen()) imDialog(text);
					else snackbars.addToast(text, 'error');
				},
				/**
				 * Der offene Dialog zeigt den Erfolg selbst, indem er schließt; ein geschlossener
				 * braucht die Snackbar, sonst hielte man die Sache für abgebrochen.
				 * @param {string} text
				 */
				erfolg(text) {
					if (!nochOffen()) snackbars.addToast(text, 'success');
				}
			};
		}
	};
}

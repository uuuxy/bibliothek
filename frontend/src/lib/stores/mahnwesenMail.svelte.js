import { apiFetch } from '../apiFetch.js';
import { showToast } from '../../inventur/lib/store.svelte.js';

/**
 * Massenversand: schickt je gewählte überfällige Klasse die Mahnliste an die
 * Klassenleitung — oder, mit overrideEmail, an genau diese eine Adresse.
 * Rückmeldung über die globale Snackbar, da die Aktion aus der Aktionsleiste kommt.
 *
 * Die Meldung des Servers wird durchgereicht statt selbst formuliert: Nur sie weiß,
 * an WEN die Listen tatsächlich gingen — bei einer Override-Adresse ist das die
 * entscheidende Information, und ein selbstgebautes „n versendet" verschweigt sie.
 *
 * Steht ausserhalb der Factory, weil sie keinen Zustand berührt — sie muss deshalb
 * nicht je Store-Instanz neu entstehen (SonarQube javascript:S7721).
 *
 * @param {{ klassen: string[], overrideEmail?: string }} auswahl
 */
async function sendBulkOverdueMails(auswahl) {
	// Ohne Auswahl gar nicht erst losschicken: Ein fehlendes klassen-Feld bedeutet
	// serverseitig „ALLE Klassen" — genau der Rundumschlag, den der Dialog verhindert.
	if (!auswahl?.klassen?.length) {
		showToast('Keine Klasse ausgewählt.', 'error');
		return;
	}

	try {
		const res = await apiFetch('/api/mail/send-bulk-overdue', {
			method: 'POST',
			headers: { 'Content-Type': 'application/json' },
			body: JSON.stringify({
				klassen: auswahl.klassen,
				override_email: auswahl.overrideEmail ?? ''
			})
		});
		const json = await res.json();
		if (res.ok) {
			// failed_count > 0: Teil des Laufs ist NICHT zugestellt — als Warnung, nicht
			// als Erfolg (vorher hieß jeder 200er „versendet"; Prüfung 22.08.2026, A8).
			showToast(
				json.message ?? `${json.sent_count ?? 0} Klassen-Mahnliste(n) versendet.`,
				(json.failed_count ?? 0) > 0 ? 'warning' : 'success'
			);
		} else {
			showToast(json.error ?? json.message ?? 'Versand fehlgeschlagen.', 'error');
		}
	} catch (e) {
		showToast(`Netzwerkfehler beim Mahnversand: ${e}`, 'error');
	}
}

/**
 * Der Mail-Teil des Mahnwesens. Seit dem 18.09.2026 nur noch der Massenversand: Der
 * Einzelversand je Klasse (Dialog „Mahnliste per E-Mail senden" mit eigener Route „senden")
 * hatte seit dem 21.06.2026 keinen Knopf mehr und konnte nichts, was der Massenversand
 * mit Klassenauswahl und abweichender Adresse nicht auch kann. Weg mit Dialog, Route und
 * Handler — nicht zwei Wege zum selben Versand.
 */
export function useMahnwesenMail() {
	return { sendBulkOverdueMails };
}

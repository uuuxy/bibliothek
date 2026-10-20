import { apiFetch, FRIST_SAMMELVERSAND_MS } from '../apiFetch.js';
import { melde } from '../../inventur/lib/store.svelte.js';

/**
 * Ersatz, falls der Server keine Meldung schickt; dieselben Worte wie mahnlistenMeldung im Server.
 * @param {number} n
 */
function mahnlistenVersendet(n) {
	return `Mahnlisten von ${n} ${n === 1 ? 'Klasse' : 'Klassen'} versendet`;
}

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
async function sendeMahnmails(auswahl) {
	// Ohne Auswahl gar nicht erst losschicken: Ein fehlendes klassen-Feld bedeutet
	// serverseitig „ALLE Klassen" — genau der Rundumschlag, den der Dialog verhindert.
	if (!auswahl?.klassen?.length) {
		melde('Keine Klasse ausgewählt.', 'error');
		return;
	}

	try {
		const res = await apiFetch('/api/mail/send-bulk-overdue', {
			method: 'POST',
			// Je Klasse eine Mail: Der Lauf dauert länger als die Vorgabe für Anfragen.
			timeoutMs: FRIST_SAMMELVERSAND_MS,
			headers: { 'Content-Type': 'application/json' },
			body: JSON.stringify({
				klassen: auswahl.klassen,
				override_email: auswahl.overrideEmail ?? ''
			})
		});
		const json = await res.json();
		if (res.ok) {
			// failed_count > 0: Ein Teil des Laufs ist nicht zugestellt, als Warnung, nicht als
			// Erfolg.
			melde(
				json.message ?? mahnlistenVersendet(json.sent_count ?? 0),
				(json.failed_count ?? 0) > 0 ? 'warning' : 'success'
			);
		} else {
			melde(json.error ?? json.message ?? 'Versand fehlgeschlagen.', 'error');
		}
	} catch (e) {
		melde(`Netzwerkfehler beim Mahnversand: ${e}`, 'error');
	}
}

/**
 * Der Mail-Teil des Mahnwesens: nur der Massenversand, mit Klassenauswahl und
 * abweichender Adresse. Ein Einzelversand je Klasse wäre ein zweiter Weg zum selben
 * Versand.
 */
export function useMahnwesenMail() {
	return { sendeMahnmails };
}

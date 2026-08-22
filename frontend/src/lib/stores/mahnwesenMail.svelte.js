import { apiFetch } from '../apiFetch.js';
import { showToast } from '../../inventur/lib/store.svelte.js';

/**
 * Massenversand: schickt je gewählte überfällige Klasse die Mahnliste an die
 * Klassenleitung — oder, mit overrideEmail, an genau diese eine Adresse.
 * Rückmeldung über die globale Snackbar, da die Aktion aus der Aktionsleiste kommt
 * (nicht aus dem Modal, dessen modalMsg hier nicht sichtbar wäre).
 *
 * Die Meldung des Servers wird durchgereicht statt selbst formuliert: Nur sie weiß,
 * an WEN die Listen tatsächlich gingen — bei einer Override-Adresse ist das die
 * entscheidende Information, und ein selbstgebautes „n versendet" verschweigt sie.
 *
 * Steht ausserhalb der Factory, weil sie keinen Zustand des Modals berührt — sie
 * muss deshalb nicht je Store-Instanz neu entstehen (SonarQube javascript:S7721).
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
 * Handles mailing logic for Mahnwesen.
 */
export function useMahnwesenMail() {
	let modalOpen = $state(false);
	let modalKlasse = $state('');
	let modalEmail = $state('');
	let modalSending = $state(false);
	let modalMsg = $state(/** @type {{ type: 'success'|'error', text: string }|null} */ (null));

	/**
	 * @param {string} klasse
	 * @param {string|null} [email]
	 */
	function openModal(klasse, email) {
		modalKlasse = klasse;
		modalEmail = email ?? '';
		modalMsg = null;
		modalOpen = true;
	}

	/**
	 * Closes the mail modal.
	 */
	function closeModal() {
		modalOpen = false;
		modalKlasse = '';
		modalEmail = '';
		modalMsg = null;
	}

	/**
	 * Sends the Mahnliste to the specified class email.
	 */
	async function sendMahnliste() {
		if (!modalEmail.trim()) {
			modalMsg = { type: 'error', text: 'E-Mail-Adresse angeben.' };
			return;
		}
		modalSending = true;
		modalMsg = null;
		try {
			const res = await apiFetch('/api/mahnwesen/senden', {
				method: 'POST',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify({ klasse: modalKlasse, email: modalEmail })
			});
			const json = await res.json();
			if (res.ok) {
				modalMsg = { type: 'success', text: json.message ?? 'Gesendet.' };
			} else {
				modalMsg = { type: 'error', text: json.error ?? json.message ?? 'Fehler.' };
			}
		} catch (e) {
			modalMsg = { type: 'error', text: String(e) };
		} finally {
			modalSending = false;
		}
	}

	return {
		get modalOpen() {
			return modalOpen;
		},
		get modalKlasse() {
			return modalKlasse;
		},
		get modalEmail() {
			return modalEmail;
		},
		set modalEmail(v) {
			modalEmail = v;
		},
		get modalSending() {
			return modalSending;
		},
		get modalMsg() {
			return modalMsg;
		},
		openModal,
		closeModal,
		sendMahnliste,
		sendBulkOverdueMails
	};
}

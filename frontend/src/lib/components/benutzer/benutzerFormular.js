/**
 * Formular der Benutzerverwaltung: leeres Formular, Übernahme eines Kontos, Nutzlast an den
 * Server. Eigene Datei, weil Konten.svelte an der Größen-Ratsche steht und die Abbildung
 * ohne Browser testbar sein soll.
 *
 * Kein Feld „Personenart" mehr (Migration 125): Wer jemand ist, steht an seiner Leserzeile;
 * das Konto sagt nur noch, was er darf. Die Ausweisnummer bleibt hier — sie wird in die
 * Leserzeile geschrieben, bis die Leserdatei sie übernimmt.
 *
 * Kein Passwortfeld: Das Passwort prüft der Mailserver der Schule (IMAP) oder der
 * Verzeichnisdienst (LDAP), es gibt keine lokale Passwortspalte. Welcher, sagt
 * `anmeldeverfahren`; ein LDAP-Konto trägt dazu seine `ldap_kennung`.
 */

import { nurGeaendertes } from '../../utils/geaendert.js';
import { mitVereinheitlichterNummer } from '../../utils/scanEinordnen.js';

export function leeresBenutzerFormular() {
	return {
		id: '',
		barcode_id: '',
		vorname: '',
		nachname: '',
		email: '',
		rolle: 'mitarbeiter',
		aktiv: true,
		anmeldeverfahren: 'imap',
		ldap_kennung: ''
	};
}

/**
 * Die Maske eines vorhandenen Kontos. `geladen` hält den Stand vom Öffnen fest; daran erkennt
 * das Speichern, was geändert wurde.
 * @param {any} user
 */
export function benutzerFormularAus(user) {
	const felder = {
		barcode_id: user.barcode_id || '',
		vorname: user.vorname,
		nachname: user.nachname,
		email: user.email,
		rolle: user.rolle,
		aktiv: user.aktiv,
		anmeldeverfahren: user.anmeldeverfahren || 'imap',
		ldap_kennung: user.ldap_kennung || ''
	};
	return { id: user.id, ...felder, geladen: { ...felder } };
}

/**
 * Was an den Server geht: beim neuen Konto alle Felder, beim vorhandenen nur die seit dem
 * Öffnen geänderten. Der Server schreibt nur, was der Rumpf nennt; was ein anderer Platz
 * inzwischen an den übrigen Feldern gespeichert hat, bleibt stehen.
 * @param {any} form
 * @returns {Record<string, any>}
 * @throws {Error} wenn einem vorhandenen Konto der Stand vom Öffnen fehlt
 */
export function benutzerNutzlast(form) {
	/** @type {Record<string, any>} */
	const felder = {
		barcode_id: form.barcode_id,
		vorname: form.vorname,
		nachname: form.nachname,
		email: form.email,
		rolle: form.rolle,
		aktiv: form.aktiv,
		anmeldeverfahren: form.anmeldeverfahren,
		// Ein IMAP-Konto hat keine Kennung; ein Rest aus dem Feld ginge sonst mit.
		ldap_kennung: form.anmeldeverfahren === 'ldap' ? form.ldap_kennung : ''
	};
	if (!form.id) return mitVereinheitlichterNummer(felder);
	if (!form.geladen) {
		throw new Error(
			'Der Stand vom Öffnen des Kontos fehlt. Nichts gespeichert: bitte das Konto neu öffnen.'
		);
	}
	// Erst vergleichen, dann vereinheitlichen: Eine unberührte Nummer geht so nicht mit.
	const geaendert = nurGeaendertes(form.geladen, felder);
	// Verfahren und Kennung gehen zusammen: Eine Kennung allein lehnt der Server ab, denn ob das
	// Konto LDAP ist, wüsste er sonst nicht.
	if ('anmeldeverfahren' in geaendert || 'ldap_kennung' in geaendert) {
		geaendert.anmeldeverfahren = felder.anmeldeverfahren;
		geaendert.ldap_kennung = felder.ldap_kennung;
	}
	return mitVereinheitlichterNummer(geaendert);
}

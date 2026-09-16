/**
 * Formular der Benutzerverwaltung: leeres Formular, Übernahme eines Kontos, Nutzlast an den
 * Server. Eigene Datei, weil UserManagement.svelte an der Größen-Ratsche steht und die Abbildung
 * ohne Browser testbar sein soll.
 *
 * Kein Feld „Personenart" mehr (Migration 125): Wer jemand ist, steht an seiner Leserzeile;
 * das Konto sagt nur noch, was er darf. Die Ausweisnummer bleibt hier — sie wird in die
 * Leserzeile geschrieben, bis die Leserdatei sie übernimmt.
 *
 * Kein Passwortfeld: Anmeldungen laufen über den Schul-Mailserver (IMAP), es gibt keine lokale
 * Passwortspalte.
 */

export function leeresBenutzerFormular() {
	return {
		id: '',
		barcode_id: '',
		vorname: '',
		nachname: '',
		email: '',
		rolle: 'mitarbeiter',
		aktiv: true
	};
}

/** @param {any} user */
export function benutzerFormularAus(user) {
	return {
		id: user.id,
		barcode_id: user.barcode_id || '',
		vorname: user.vorname,
		nachname: user.nachname,
		email: user.email,
		rolle: user.rolle,
		aktiv: user.aktiv
	};
}

/** @param {any} form */
export function benutzerNutzlast(form) {
	return {
		barcode_id: form.barcode_id,
		vorname: form.vorname,
		nachname: form.nachname,
		email: form.email,
		rolle: form.rolle,
		aktiv: form.aktiv
	};
}

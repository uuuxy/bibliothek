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
		aktiv: user.aktiv
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
		aktiv: form.aktiv
	};
	if (!form.id) return felder;
	if (!form.geladen) {
		throw new Error(
			'Der Stand vom Öffnen des Kontos fehlt. Nichts gespeichert: bitte das Konto neu öffnen.'
		);
	}
	return Object.fromEntries(
		Object.entries(felder).filter(([name, wert]) => wert !== form.geladen[name])
	);
}

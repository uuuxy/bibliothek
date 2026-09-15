/**
 * Formular der Benutzerverwaltung: leeres Formular, Übernahme eines Kontos, Nutzlast an den
 * Server. Eigene Datei, weil UserManagement.svelte an der Größen-Ratsche steht und die Abbildung
 * ohne Browser testbar sein soll.
 *
 * Die Personenart sagt, wer jemand im Kollegium ist (Migration 119), die Rolle, was er in der
 * Software darf. Die Nutzlast schickt die Personenart IMMER mit: Fehlt das Feld, lässt der Server
 * den alten Wert stehen — ein geleertes Feld soll aber leeren.
 *
 * Kein Passwortfeld: Anmeldungen laufen über den Schul-Mailserver (IMAP), es gibt keine lokale
 * Passwortspalte.
 */

export const PERSONENARTEN = [
	{ value: '', label: 'Keine Angabe' },
	{ value: 'lehrkraft', label: 'Lehrkraft' },
	{ value: 'liv', label: 'LiV' }
];

export function leeresBenutzerFormular() {
	return {
		id: '',
		barcode_id: '',
		vorname: '',
		nachname: '',
		email: '',
		rolle: 'mitarbeiter',
		personenart: '',
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
		personenart: user.personenart || '',
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
		personenart: form.personenart,
		aktiv: form.aktiv
	};
}

/**
 * Die Personenart entscheidet, wer als Lehrkraft ausleiht; ein Kollegiumskonto hat immer eine
 * (Migration 120). „Keine Angabe" gibt es dort nicht — die Datenbank machte daraus sonst still
 * „Lehrkraft".
 * @param {string} rolle
 */
export function personenartOptionen(rolle) {
	return rolle === 'kollegium' ? PERSONENARTEN.filter((p) => p.value !== '') : PERSONENARTEN;
}

/** @param {string | null | undefined} wert */
export function personenartLabel(wert) {
	if (!wert) return '';
	return PERSONENARTEN.find((p) => p.value === wert)?.label ?? '';
}

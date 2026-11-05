// Die Wörter der Oberfläche für die Rohwerte der beiden Protokolle. Ein unbekannter Wert
// erscheint roh: Ein neuer Code ist dann zu sehen, nur nicht übersetzt.

/** Die Tabelle eines Eintrags im Logbuch, mit dem Wort, das die Oberfläche für die Sache hat. */
/** @type {Record<string, string>} */
export const BEREICHE = {
	audit_log: 'Logbuch',
	ausleihen: 'Ausleihen',
	benutzer: 'Benutzer',
	buecher_exemplare: 'Exemplare',
	buecher_titel: 'Titel',
	class_books: 'Klassensätze',
	inventur_sessions: 'Inventur',
	klassensatz_reservierungen: 'Klassensatz-Reservierungen',
	lehrer_anliegen: 'Meldungen',
	nachbuch_meldungen: 'Nachbuchen',
	schadensersatz_bescheide: 'Bescheide',
	schadensfaelle: 'Schadensfälle',
	schueler: 'Leser',
	vormerkungen: 'Vormerkungen'
};

/** Die Aktion eines Eintrags im Verwaltungsprotokoll. */
/** @type {Record<string, string>} */
export const VERWALTUNG_AKTIONEN = {
	ABGAENGER_KONTOAUSZUG_MAIL: 'Kontoauszug an Abgänger verschickt',
	BESTAND_IMPORT: 'Bestand importiert',
	BESTELLMAIL_ERNEUT_GESENDET: 'Bestellmail erneut gesendet',
	BESTELLMAIL_VERMERK_ENTFERNT: 'Vermerk über gescheiterten Versand entfernt',
	BESTELLUNG_MITTEL_KORRIGIERT: 'Mittelherkunft einer Bestellung korrigiert',
	BULK_RECEIVE_ITEMS: 'Wareneingang gebucht',
	DELETE_STUDENT: 'Leser in den Papierkorb verschoben',
	FRIST_KLASSE_GEAENDERT: 'Rückgabefrist einer Klasse geändert',
	FRIST_OVERRIDE: 'Frist von Hand gesetzt',
	KLASSENLEITUNG_ENTFERNT: 'Klassenleitung entfernt',
	KLASSENLEITUNG_GEAENDERT: 'Klassenleitung geändert',
	KOLLEGIUMSKONTO_NACHGETRAGEN: 'Benutzer angelegt: Schul-E-Mail in der Leserakte nachgetragen',
	LESER_ENTSPERRT: 'Leser entsperrt',
	LESER_GESPERRT: 'Leser gesperrt',
	LIEFERANT_ANGELEGT: 'Lieferant angelegt',
	LIEFERANT_GEAENDERT: 'Lieferant geändert',
	LIEFERANT_GELOESCHT: 'Lieferant gelöscht',
	LITTERA_IMPORT: 'Littera-Übernahme',
	LMF_PLAN_GESPEICHERT: 'LMF-Plan gespeichert',
	LMF_PLAN_VEROEFFENTLICHT: 'LMF-Plan veröffentlicht',
	LMF_PLAN_VERWORFEN: 'LMF-Plan verworfen',
	LUSD_ID_NACHGETRAGEN: 'LUSD-ID nachgetragen',
	LUSD_IMPORT: 'LUSD-Import',
	MAILVORLAGE_GEAENDERT: 'Mailvorlage geändert',
	OVERRIDE_BLOCK: 'Sperre einmalig übergangen',
	PURGE_STUDENT: 'Leser endgültig gelöscht',
	RECHT_GEAENDERT: 'Recht geändert',
	RESTORE_STUDENT: 'Leser wiederhergestellt',
	SCHADENSERSATZ_BESCHEID_ERSTELLT: 'Bescheid erstellt',
	SCHADENSERSATZ_BESCHEID_UEBERGEBEN: 'Bescheid übergeben',
	SCHLAGWORT_GELOESCHT: 'Schlagwort gelöscht',
	SCHLAGWORT_VERWEIS: 'Schlagwort umgeleitet',
	SCHLAGWORT_ZUSAMMENGEFUEHRT: 'Schlagworte zusammengeführt',
	SCHUELER_ZUSAMMENGEFUEHRT: 'Leser zusammengeführt',
	SCHULJAHRESWECHSEL: 'Schuljahreswechsel',
	SELBSTANMELDUNG: 'Zugang über die eigene Anmeldung beantragt',
	TITEL_ZUSAMMENGEFUEHRT: 'Titel zusammengeführt',
	TRESEN_AUSKUNFT: 'Theken-Auskunft abgefragt',
	UPDATE_MAIL_SETTINGS: 'Mail-Einstellungen geändert',
	UPDATE_SETTINGS: 'Einstellungen geändert',
	USER_CREATE: 'Benutzer angelegt',
	USER_UPDATE: 'Benutzer geändert'
};

const UUID = /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i;

/**
 * Die ersten acht Zeichen einer UUID, wie Git einen Commit kurz nennt; die ganze Kennung steht
 * in der Sprechblase. Was keine UUID ist, bleibt, wie es ist.
 * @param {string} kennung
 */
export function kurzeKennung(kennung) {
	return UUID.test(kennung ?? '') ? kennung.slice(0, 8) : (kennung ?? '');
}

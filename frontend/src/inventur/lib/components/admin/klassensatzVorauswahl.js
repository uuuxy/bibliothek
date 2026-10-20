/**
 * Welche Titel einer Klasse im Dialog „Bücher verwalten" vorgewählt sind.
 *
 * Eine Klassensatz-Kachel hat zwei Quellen: die von Hand gepflegte Zuordnung (`hand`, Tabelle
 * class_books) und die live aus den laufenden Ausleihen abgeleiteten Titel (`ausleihe`, nie
 * gespeichert). Der Dialog speichert die Auswahl überschreibend: aendereKlassensaetze löscht
 * die Zuordnungen der Klasse und schreibt die gesendete Liste. Wäre ein abgeleiteter Titel
 * vorgewählt, machte der erste Klick auf Speichern ihn dauerhaft, ohne dass jemand ihn
 * zugeordnet hätte.
 *
 * Deshalb steht die Regel hier und nicht als Ausdruck in der Vorlage: als eine Stelle,
 * die ein Test festhalten kann.
 *
 * @param {{ books?: Array<{ id: any, quelle?: string }> } | null | undefined} gruppe
 * @returns {Set<any>} die IDs der handgepflegten Titel
 */
export function vorauswahlAusGruppe(gruppe) {
	return new Set((gruppe?.books ?? []).filter((b) => b.quelle !== 'ausleihe').map((b) => b.id));
}

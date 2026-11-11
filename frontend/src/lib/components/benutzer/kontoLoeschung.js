// Was die Oberfläche nach dem Löschen eines Kontos meldet. Gelöscht wird der Zugang; die Person
// dahinter bleibt in der Leserdatei, wenn sie einen Ausweis trägt oder etwas an ihrem Eintrag
// hängt (repository.LoescheKonto, leser_bleibt). Die Meldung sagt es, statt „gelöscht" zu melden.

/**
 * @param {{ leser_bleibt?: boolean } | null | undefined} antwort die Antwort von DELETE /api/benutzer/{id}
 * @returns {string}
 */
export function meldungNachLoeschung(antwort) {
	return antwort?.leser_bleibt
		? 'Zugang gelöscht. Der Eintrag der Person in der Leserdatei bleibt; dort lässt er sich löschen.'
		: 'Benutzer erfolgreich gelöscht.';
}

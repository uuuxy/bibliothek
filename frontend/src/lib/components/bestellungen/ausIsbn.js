/**
 * Die Tür der Titelsuche beim Bestellen: POST /api/buecher/aus-isbn — aus dem Katalog oder neu
 * aus der DNB. Die Antwort nennt die ISBN, wie der Katalog sie trägt: Eine zehnstellig
 * eingegebene steht dort dreizehnstellig.
 */

/**
 * Der Titel für das Staging-Fenster (OrderStaging) aus der Antwort der Tür und dem Treffer, den
 * die Titelsuche angeklickt hat.
 *
 * dnb_vorschlag_da: Nur ein eben aus der DNB angelegter Titel (exists=false) bringt den
 * Schlagwort-Vorschlag mit. Stand er schon im Katalog, hat die Tür die DNB nicht gefragt; dann
 * zeigt das Fenster den Knopf dafür (entschieden am 30.09.2026).
 *
 * @param {any} antwort die Antwort von aus-isbn
 * @param {any} treffer der DNB-Treffer der Suche
 */
export function fensterAusAntwort(antwort, treffer) {
	return {
		id: antwort.titel_id,
		titel: antwort.titel,
		autor: antwort.autor,
		isbn: antwort.isbn,
		verlag: antwort.verlag,
		cover_url: antwort.cover_url,
		// exists=false: signatur ist nur ein VORSCHLAG aus der DNB-Heuristik.
		signatur: antwort.signatur ?? '',
		// Preisvorschlag vom DNB-Treffer — über /aus-isbn ginge er sonst verloren.
		preis_vorschlag: treffer.preis_vorschlag,
		// Ein eben angelegter Titel ist noch kein Lernmittel — OrderStaging fragt nach.
		ist_lernmittel: Boolean(antwort.ist_lernmittel),
		// Schlagworte, die der DNB-Satz nennt — aus der Liste und neu; nur angeboten,
		// eingetragen wird erst im Fenster (docs/OFFEN.md 4.20, 4.25).
		schlagwort_vorschlaege: antwort.schlagwort_vorschlaege ?? [],
		schlagwort_vorschlaege_neu: antwort.schlagwort_vorschlaege_neu ?? [],
		dnb_vorschlag_da: antwort.exists === false
	};
}

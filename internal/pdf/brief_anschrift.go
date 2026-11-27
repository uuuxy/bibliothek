package pdf

import "strings"

// briefAnschrift sind die Zeilen im Anschriftfeld eines Briefs an einen Leser, für Mahnbrief,
// Bescheid und Ersatzforderung dieselbe Regel: kopf (die Zeile an die Erziehungsberechtigten,
// wenn nötig, und der Name), dann Straße mit Hausnummer und Postleitzahl mit Ort. Fehlt Straße
// oder Ort, steht „Keine Adresse hinterlegt“ da: Eine halbe oder leere Anschrift sähe aus wie ein
// Druckfehler, und der Brief ginge unzustellbar zur Post.
func briefAnschrift(kopf []string, strasse, plz, ort string) []string {
	zeilen := append([]string{}, kopf...)
	strasse, ort = strings.TrimSpace(strasse), strings.TrimSpace(ort)
	if strasse == "" || ort == "" {
		return append(zeilen, "Keine Adresse hinterlegt")
	}
	return append(zeilen, strasse, strings.TrimSpace(plz+" "+ort))
}

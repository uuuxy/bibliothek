package service

import "strings"

// ImportTrennzeichen bestimmt das Trennzeichen einer CSV anhand der Kopfzeile.
// Littera-Exporte kommen mal mit Komma, mal mit Semikolon; mit einem fest verdrahteten
// Trennzeichen läse der Import eine Zeile als eine einzige Spalte. Gewählt wird das Zeichen,
// das in der ersten Zeile häufiger vorkommt: Die Kopfzeile enthält keine Trenner in
// Anführungszeichen.
func ImportTrennzeichen(content string) rune {
	firstLine := content
	if idx := strings.IndexAny(content, "\r\n"); idx != -1 {
		firstLine = content[:idx]
	}
	if strings.Count(firstLine, ";") > strings.Count(firstLine, ",") {
		return ';'
	}
	return ','
}

// ImportKopfzeile ordnet die Spaltennamen einer Import-CSV oder -XLSX ihren Positionen zu,
// unter den Schlüsseln, die ImportDynamic liest (spaltenWert). Die Erkennung ist tolerant
// (Teilwort, klein geschrieben), damit der schlanke Littera-Export (Titel,Autor,…,Barcode)
// und die volle Bestandsdatei (…;Barcode;Zustand) über denselben Weg laufen. Zustand (sperrt
// "verliehen") und Signatur (Rücken-Etikett) dürfen fehlen.
func ImportKopfzeile(headers []string) map[string]int {
	headerMap := make(map[string]int)
	for idx, h := range headers {
		norm := strings.ToLower(strings.TrimSpace(h))
		switch {
		case strings.Contains(norm, "titel") || norm == "titelliste":
			headerMap["titel"] = idx
		case strings.Contains(norm, "autor") || norm == "verfasser":
			headerMap["autor"] = idx
		case strings.Contains(norm, "verlag"):
			headerMap["verlag"] = idx
		case strings.Contains(norm, "isbn"):
			headerMap["isbn"] = idx
		case strings.Contains(norm, "jahr") || norm == "ersch.jahr" || norm == "erscheinungsjahr":
			headerMap["jahr"] = idx
		case strings.Contains(norm, "kategorie") || strings.Contains(norm, "systematik") || norm == "fach":
			headerMap["kategorie"] = idx
		// Signatur ist das Rücken-Etikett (buecher_titel.signatur) und kein Barcode eines
		// Exemplars: Als Barcode gelesen, verdrängte sie die echten Barcodes.
		case strings.Contains(norm, "signatur"):
			headerMap["signatur"] = idx
		case strings.Contains(norm, "barcode") || strings.Contains(norm, "exemplar") || norm == "inventarnummer":
			headerMap["barcode"] = idx
		case strings.Contains(norm, "zustand") || strings.Contains(norm, "status"):
			headerMap["zustand"] = idx
		}
	}
	return headerMap
}

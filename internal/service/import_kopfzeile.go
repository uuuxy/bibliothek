package service

import "strings"

// ImportTrennzeichen bestimmt das Trennzeichen einer CSV anhand der Kopfzeile.
// Littera-Exporte kommen mal mit Komma, mal mit Semikolon; mit einem fest verdrahteten
// Trennzeichen läse der Import eine Zeile als eine einzige Spalte. Gewählt wird das Zeichen,
// das in der ersten Zeile häufiger vorkommt: Die Kopfzeile enthält keine Trenner in
// Anführungszeichen.
func ImportTrennzeichen(inhalt string) rune {
	ersteZeile := inhalt
	if ende := strings.IndexAny(inhalt, "\r\n"); ende != -1 {
		ersteZeile = inhalt[:ende]
	}
	if strings.Count(ersteZeile, ";") > strings.Count(ersteZeile, ",") {
		return ';'
	}
	return ','
}

// ImportKopfzeile ordnet die Spaltennamen einer Import-CSV oder -XLSX ihren Positionen zu,
// unter den Schlüsseln, die ImportiereZeilen liest (spaltenWert). Die Erkennung ist tolerant
// (Teilwort, klein geschrieben), damit der schlanke Littera-Export (Titel,Autor,…,Barcode)
// und die volle Bestandsdatei (…;Barcode;Zustand) über denselben Weg laufen. Zustand (sperrt
// "verliehen") und Signatur (Rücken-Etikett) dürfen fehlen.
func ImportKopfzeile(kopfzellen []string) map[string]int {
	spalten := make(map[string]int)
	for spalte, kopf := range kopfzellen {
		norm := strings.ToLower(strings.TrimSpace(kopf))
		switch {
		case strings.Contains(norm, "titel") || norm == "titelliste":
			spalten["titel"] = spalte
		case strings.Contains(norm, "autor") || norm == "verfasser":
			spalten["autor"] = spalte
		case strings.Contains(norm, "verlag"):
			spalten["verlag"] = spalte
		case strings.Contains(norm, "isbn"):
			spalten["isbn"] = spalte
		case strings.Contains(norm, "jahr") || norm == "ersch.jahr" || norm == "erscheinungsjahr":
			spalten["jahr"] = spalte
		case strings.Contains(norm, "kategorie") || strings.Contains(norm, "systematik") || norm == "fach":
			spalten["kategorie"] = spalte
		// Signatur ist das Rücken-Etikett (buecher_titel.signatur) und kein Barcode eines
		// Exemplars: Als Barcode gelesen, verdrängte sie die echten Barcodes.
		case strings.Contains(norm, "signatur"):
			spalten["signatur"] = spalte
		case strings.Contains(norm, "barcode") || strings.Contains(norm, "exemplar") || norm == "inventarnummer":
			spalten["barcode"] = spalte
		case strings.Contains(norm, "zustand") || strings.Contains(norm, "status"):
			spalten["zustand"] = spalte
		}
	}
	return spalten
}

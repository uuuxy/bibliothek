package api

import "strings"

// detectCSVDelimiter bestimmt das Trennzeichen einer CSV anhand der Kopfzeile.
// Littera-Exporte kommen mal Komma-, mal Semikolon-separiert; wer das Trennzeichen
// hart verdrahtet, liest eine Zeile fälschlich als eine einzige Spalte. Wir wählen
// das Zeichen, das in der ERSTEN Zeile häufiger vorkommt (die Kopfzeile enthält
// keine eingebetteten Trenner in Anführungszeichen und ist damit verlässlich).
func detectCSVDelimiter(content string) rune {
	firstLine := content
	if idx := strings.IndexAny(content, "\r\n"); idx != -1 {
		firstLine = content[:idx]
	}
	if strings.Count(firstLine, ";") > strings.Count(firstLine, ",") {
		return ';'
	}
	return ','
}

// buildImportHeaderMap ordnet die Spaltennamen einer Import-CSV/XLSX ihren
// Positionen zu. Die Erkennung ist bewusst tolerant (Teilstring-Matching, klein
// geschrieben), damit sowohl der schlanke Littera-Export (Titel,Autor,…,Barcode)
// als auch die volle Bestandsdatei (…;Barcode;Zustand) über denselben Pfad laufen.
// Optionale Spalten: Zustand (sperrt "verliehen") und Signatur (Rücken-Etikett).
func buildImportHeaderMap(headers []string) map[string]int {
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
		// Signatur ist das Rücken-Etikett (buecher_titel.signatur) — sie ist KEIN
		// Exemplar-Barcode. Das frühere Alias signatur→barcode hat Signaturen als
		// Barcodes importiert und die echten Barcodes verdrängt.
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

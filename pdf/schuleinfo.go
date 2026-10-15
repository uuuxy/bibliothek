package pdf

// SchuleInfo sind Name und Anschrift der Schule für den Kopf der Briefe; die Tür liest sie bei
// jeder Anfrage aus den Einstellungen.
type SchuleInfo struct {
	Name    string
	Strasse string
	PLZ     string
	Ort     string
}

// Absenderzeile liefert den Absender in einer Zeile, wie er über dem Fensterfeld steht.
func (s SchuleInfo) Absenderzeile() string {
	if s.Name == "" {
		return "Schulbibliothek"
	}
	if s.Strasse == "" || s.Ort == "" {
		return s.Name
	}
	return s.Name + " · " + s.Strasse + " · " + s.PLZ + " " + s.Ort
}

// OrtDatum liefert "Ort, den TT.MM.JJJJ" für die Datumszeile; ohne Ort nur das Datum.
func (s SchuleInfo) OrtDatum(datum string) string {
	if s.Ort == "" {
		return datum
	}
	return s.Ort + ", den " + datum
}

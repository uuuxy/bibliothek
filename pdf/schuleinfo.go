package pdf

// SchuleInfo holds configurable school identity data used in PDF letter headers.
// Values are loaded from system_einstellungen at request time.
type SchuleInfo struct {
	Name    string
	Strasse string
	PLZ     string
	Ort     string
}

// Absenderzeile returns a compact one-line sender string suitable for letterhead.
func (s SchuleInfo) Absenderzeile() string {
	if s.Name == "" {
		return "Schulbibliothek"
	}
	if s.Strasse == "" || s.Ort == "" {
		return s.Name
	}
	return s.Name + " · " + s.Strasse + " · " + s.PLZ + " " + s.Ort
}

// OrtDatum returns "Ort, den TT.MM.JJJJ" for use in date lines.
// Falls back to just the date when Ort is empty.
func (s SchuleInfo) OrtDatum(datum string) string {
	if s.Ort == "" {
		return datum
	}
	return s.Ort + ", den " + datum
}

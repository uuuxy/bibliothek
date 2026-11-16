package repository

// zeigerText macht aus einem Textfeld, das NULL sein kann, einen Text; NULL wird zu "".
func zeigerText(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// zeigerAuf liefert einen Zeiger auf s, für ein Feld, das NULL sein kann.
func zeigerAuf(s string) *string { return &s }

// leerAlsNull übersetzt den leeren Text in SQL-NULL, für eine Spalte, in der eine fehlende
// Angabe NULL ist: Als UUID ginge der leere Text nicht durch, und CopyFrom bräche die ganze
// Bestellung ab, obwohl „keine Bestellung" bei Handanlage und Import der Normalfall ist; beim
// Bestellstatus lehnt chk_exemplar_bestellstatus ihn ab; im Protokoll der Verwaltung
// (audit_logs) und beim Anlegen aus der LUSD steht eine fehlende Angabe als NULL.
func leerAlsNull(s string) any {
	if s == "" {
		return nil
	}
	return s
}

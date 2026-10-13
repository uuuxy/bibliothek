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

package strutil

// CleanISBN entfernt hochperformant Bindestriche und Leerzeichen
// und vermeidet Allokationen, wenn keine Änderungen nötig sind.
// ⚡ Bolt: Ersetzt sequenzielle strings.ReplaceAll Aufrufe.
func CleanISBN(isbn string) string {
	keep := 0
	for i := 0; i < len(isbn); i++ {
		if c := isbn[i]; c != '-' && c != ' ' {
			keep++
		}
	}
	if keep == len(isbn) {
		return isbn
	}
	b := make([]byte, keep)
	j := 0
	for i := 0; i < len(isbn); i++ {
		if c := isbn[i]; c != '-' && c != ' ' {
			b[j] = c
			j++
		}
	}
	return string(b)
}

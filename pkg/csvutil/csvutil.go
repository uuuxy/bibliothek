// Package csvutil entschärft Zellen für den Export nach CSV.
package csvutil

// entschaerfeZelle stellt einer Zelle ein Apostroph voran, wenn sie mit =, +, -, @ oder einem
// Steuerzeichen beginnt. Eine Tabellenkalkulation rechnet eine solche Zelle sonst als Formel
// aus, und ein Titel aus einem importierten Katalog könnte beim Öffnen der Datei Befehle
// ausführen oder Daten verschicken.
func entschaerfeZelle(s string) string {
	if s == "" {
		return s
	}
	switch s[0] {
	case '=', '+', '-', '@', '\t', '\r', '\n':
		return "'" + s
	}
	return s
}

// EntschaerfeZeile liefert eine neue Zeile, in der jede Zelle entschärft ist.
func EntschaerfeZeile(zeile []string) []string {
	entschaerft := make([]string, len(zeile))
	for i, zelle := range zeile {
		entschaerft[i] = entschaerfeZelle(zelle)
	}
	return entschaerft
}

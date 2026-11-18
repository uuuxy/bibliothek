package pdfzeichen

import (
	"testing"
	"unicode/utf8"
)

// Buchstaben, die die Schriften kennen, gehen unverändert aufs Papier, auch türkische, polnische,
// vietnamesische, kyrillische und griechische.
func TestErsetze_LaesstBuchstabenStehen(t *testing.T) {
	for _, text := range []string{
		"Ayşe Öztürk", "Łukasz Wiśniewski", "Ağaoğlu, İlknur", "Ștefan Țăran", "Đorđe Šimić",
		"Doǧan", "Nguyễn Văn Ưng", "Дмитрий", "Ελένη", "Müller, François", "Deutschbuch 5 (LMF)",
	} {
		if got := Ersetze(text); got != text {
			t.Errorf("%q → %q, erwartet unverändert", text, got)
		}
	}
}

// Ein Akzent, der getrennt hinter seinem Buchstaben steht (NFD), wird mit ihm zu einem Zeichen:
// Nur so misst gofpdf die Breite richtig. Die Eingabe entsteht zur Laufzeit, damit kein Editor sie
// beim Speichern zusammensetzt.
func TestErsetze_SetztAkzenteZusammen(t *testing.T) {
	zerlegt := "Nguye" + string(rune(0x0302)) + string(rune(0x0303)) + "n S" + string(rune(0x0327)) + "ahin"
	if got := Ersetze(zerlegt); got != "Nguyễn Şahin" {
		t.Errorf("zerlegte Akzente: %q, erwartet „Nguyễn Şahin“", got)
	}
}

// Ein Zeichen jenseits der Basisebene (Emoji) und kaputtes UTF-8 werden zum Ersatzzeichen; gofpdf
// verweigert sonst das ganze Dokument.
func TestErsetze_MachtAusEmojiEinErsatzzeichen(t *testing.T) {
	ersatz := string(utf8.RuneError)
	faelle := map[string]string{
		"Buch " + string(rune(0x1F600)) + " Titel": "Buch " + ersatz + " Titel",
		"Linear " + string(rune(0x10000)) + " B":   "Linear " + ersatz + " B",
		"kaputt \xff":                              "kaputt " + ersatz,
	}
	for eingabe, erwartet := range faelle {
		if got := Ersetze(eingabe); got != erwartet {
			t.Errorf("%q → %q, erwartet %q", eingabe, got, erwartet)
		}
	}
}

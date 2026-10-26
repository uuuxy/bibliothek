package api

import (
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// Gate: Eine Tür schickt ein Dokument nur dann als Anhang (Content-Disposition: attachment), wenn
// es keine Personendaten trägt. Ein Anhang landet im Download-Ordner des Arbeitsplatzes und
// bleibt dort für den Nächsten am Rechner liegen; Dokumente mit Personendaten gehen im Tab auf
// (ARCHITEKTUR.md 8.9). Für die Oberfläche hält dieselbe Regel
// frontend/src/lib/frontend-hygiene-downloads.test.js.
//
// Blindheit: ein Anhang, dessen Angabe nicht als Text unter api/ steht, etwa aus einer Konstante
// eines anderen Pakets.
func TestAnhangNurOhnePersonendaten(t *testing.T) {
	alsAnhang := map[string]string{
		"bestellung/bestellbedarf_pdf.go": "die Bestellliste",
		"bestellung/bestellbericht.go":    "der Bericht über die Bestellungen",
		"katalog/bestand_export.go":       "der Katalog als CSV",
		"lernmittel/lmf_termine.go":       "der LMF-Plan, Termine je Klasse",
		"portal/schulbuecher.go":          "die Bücher eines Fachs im Portal",
	}

	var gefunden []string
	err := filepath.WalkDir(".", func(pfad string, eintrag fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if eintrag.IsDir() || !strings.HasSuffix(pfad, ".go") || strings.HasSuffix(pfad, "_test.go") {
			return nil
		}
		inhalt, err := os.ReadFile(pfad)
		if err != nil {
			return err
		}
		if strings.Contains(string(inhalt), "attachment;") {
			gefunden = append(gefunden, filepath.ToSlash(pfad))
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	for _, pfad := range gefunden {
		if _, ok := alsAnhang[pfad]; !ok {
			t.Errorf("api/%s schickt ein Dokument als Anhang: mit Personendaten als inline schicken, "+
				"sonst mit Grund in alsAnhang eintragen", pfad)
		}
	}
	// Jeder Eintrag muss noch einen Anhang finden, sonst misst die Liste nichts mehr.
	for pfad := range alsAnhang {
		if !slices.Contains(gefunden, pfad) {
			t.Errorf("api/%s steht in alsAnhang, schickt aber keinen Anhang mehr: austragen", pfad)
		}
	}
}

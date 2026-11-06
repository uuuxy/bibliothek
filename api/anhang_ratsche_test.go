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
// Einen Anhang erkennt der Test an tuer.Speichern, dem Wert für tuer.AntworteDatei und
// tuer.SetzeDateiKopf, oder am Text "attachment;". Blindheit: ein Anhang aus einer Konstante
// eines anderen Pakets und eine Tür im Paket tuer selbst.
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
		if istAnhang(string(inhalt)) {
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

// istAnhang erkennt eine Tür, die ein Dokument als Anhang schickt.
func istAnhang(text string) bool {
	return strings.Contains(text, "tuer.Speichern") || strings.Contains(text, "attachment;")
}

// setztDateiKopf erkennt den Kopf einer Datei in jeder Schreibweise, auch über die Konstante.
func setztDateiKopf(text string) bool {
	text = strings.ToLower(text)
	return strings.Contains(text, "content-disposition") || strings.Contains(text, "headercontentdisposition")
}

// Gate: Den Kopf einer Datei (Content-Disposition) setzt unter api/ nur das Paket tuer. Von Hand
// geschrieben fehlte ihm die Kodierung des Namens nach RFC 6266, und TestAnhangNurOhnePersonendaten
// sähe einen Anhang nur, wenn er als Text dasteht.
//
// Blindheit: ein Kopf, dessen Name nicht als Text in der Datei steht, etwa aus einer Variablen.
func TestDateiKopfNurUeberTuer(t *testing.T) {
	// Der Erkenner muss den Kopf im Helfer selbst finden, sonst misst der Test nichts.
	helfer, err := os.ReadFile(filepath.Join("tuer", "datei.go"))
	if err != nil {
		t.Fatal(err)
	}
	if !setztDateiKopf(string(helfer)) {
		t.Fatal("setztDateiKopf erkennt den Kopf in tuer/datei.go nicht")
	}

	err = filepath.WalkDir(".", func(pfad string, eintrag fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if eintrag.IsDir() {
			if pfad == "tuer" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(pfad, ".go") || strings.HasSuffix(pfad, "_test.go") {
			return nil
		}
		inhalt, err := os.ReadFile(pfad)
		if err != nil {
			return err
		}
		if setztDateiKopf(string(inhalt)) {
			t.Errorf("api/%s setzt Content-Disposition selbst: tuer.AntworteDatei oder tuer.SetzeDateiKopf nehmen",
				filepath.ToSlash(pfad))
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

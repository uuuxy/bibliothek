package api

import (
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"
)

// Schichtung des Backends: Was in api/ liegt, ist eine Tür, auch in einem Unterordner. Der
// Bestand dateienOhneTuer nennt, was ohne Tür in api/ liegt, und kann nur kleiner werden. Dass
// kein Paket außerhalb der Datenbankschicht SQL formuliert, hält schichtung_ratsche_test.go im
// Wurzelpaket.
//
// Blindheit: Regeln in einer Datei, die auch eine Tür trägt; eine Datei, die net/http nur für
// eine Konstante einbindet.

// produktivDateien nennt die Go-Dateien unter der Wurzel ohne Tests, mit ihrem Pfad ab der
// Wurzel. Sie steigt in jeden Unterordner: Eine Datei in einem Themenordner zählt wie eine
// daneben. Abgelegte Bilder und Testdaten tragen keinen Quelltext.
func produktivDateien(t *testing.T, wurzel string) []string {
	t.Helper()
	var pfade []string
	err := filepath.WalkDir(wurzel, func(pfad string, eintrag fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		name := eintrag.Name()
		if eintrag.IsDir() {
			if pfad != wurzel && (strings.HasPrefix(name, ".") || name == "testdata" || name == "uploads") {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			return nil
		}
		rel, err := filepath.Rel(wurzel, pfad)
		if err != nil {
			return err
		}
		pfade = append(pfade, filepath.ToSlash(rel))
		return nil
	})
	if err != nil {
		t.Fatalf("%s nicht lesbar: %v", wurzel, err)
	}
	return pfade
}

// apiDateien nennt die Go-Dateien von api/ samt Unterordnern. Die Untergrenze hält einen
// umbenannten Ordner davon ab, die Prüfungen still grün zu stellen, die hier lesen.
func apiDateien(t *testing.T) []string {
	t.Helper()
	pfade := produktivDateien(t, ".")
	if len(pfade) < 50 {
		t.Fatalf("nur %d Go-Dateien in api/ gefunden — der Test misst offenbar nichts mehr", len(pfade))
	}
	return pfade
}

// dateienOhneTuer: Dateien in api/, die net/http nicht einbinden, mit ihrem Pfad ab api/. Sie
// nehmen keine Anfrage an und geben keine Antwort; was davon Regel oder Erzeuger ist, zieht in
// ein eigenes Paket und fällt hier weg.
var dateienOhneTuer = []string{
	"benutzer/benutzer.go",
	"bestellung/bestellmail_anhaenge.go",
	"bestellung/bestellmail_versand.go",
	"bestellung/bestellung.go",
	"druck/druck.go",
	"inventurlauf/inventurlauf.go",
	"katalog/katalog.go",
	"lernmittel/lernmittel.go",
	"lernmittel/lmf_plan_live.go",
	"leser/leser.go",
	"leser/leser_klasse_regel.go",
	"mahnwesen/abgaenger_fenster.go",
	"mahnwesen/mahnwesen.go",
	"mahnwesen/mahnwesen_mail.go",
	"portal/portal.go",
	"schaden/bescheid_schnappschuss.go",
	"schaden/schaden.go",
	"system/betriebsbereitschaft_alarm.go",
	"system/system.go",
	"theke/eingabe_anfrage_antwort.go",
	"theke/theke.go",
	"theke/theken_leser.go",
	"tuer/tuer.go",
	"tuer/verwaltung_protokoll.go",
}

// bindetHTTPEin sagt, ob die Quelle net/http einbindet, unter welchem Namen auch immer.
func bindetHTTPEin(t *testing.T, name string, quelle any) bool {
	t.Helper()
	datei, err := parser.ParseFile(token.NewFileSet(), name, quelle, parser.ImportsOnly)
	if err != nil {
		t.Fatalf("%s nicht lesbar: %v", name, err)
	}
	for _, imp := range datei.Imports {
		if pfad, err := strconv.Unquote(imp.Path.Value); err == nil && pfad == "net/http" {
			return true
		}
	}
	return false
}

func TestNeueDateiInAPITraegtEineTuer(t *testing.T) {
	var gefunden []string
	for _, name := range apiDateien(t) {
		if !bindetHTTPEin(t, name, nil) {
			gefunden = append(gefunden, name)
		}
	}
	for _, name := range gefunden {
		if !slices.Contains(dateienOhneTuer, name) {
			t.Errorf("api/%s trägt keine Tür (bindet net/http nicht ein). Regeln, Leser fremder "+
				"Dateien und Erzeuger von Dokumenten gehören in ein eigenes Paket unter internal/, "+
				"pkg/ oder pdf/; api/ ruft sie auf. Gehört die Datei zu den Türen (Typen einer "+
				"Anfrage, Helfer mehrerer Handler), trage sie in dateienOhneTuer ein.", name)
		}
	}
	for _, name := range dateienOhneTuer {
		if !slices.Contains(gefunden, name) {
			t.Errorf("api/%s liegt nicht mehr ohne Tür in api/ — bitte aus dateienOhneTuer "+
				"entfernen, damit die Ratsche greift.", name)
		}
	}
}

// Der Detektor liest die Einbindungen, nicht den Text: Ein Wort im Kommentar oder ein
// Nachbarpaket darf er nicht für die Tür halten, einen umbenannten Import muss er erkennen.
func TestBindetHTTPEin_ErkenntJedeForm(t *testing.T) {
	mit := map[string]string{
		"einzeln":     "package x\nimport \"net/http\"\n",
		"im Block":    "package x\nimport (\n\t\"fmt\"\n\t\"net/http\"\n)\n",
		"umbenannt":   "package x\nimport web \"net/http\"\n",
		"mit Punkt":   "package x\nimport . \"net/http\"\n",
		"zwei Blöcke": "package x\nimport \"fmt\"\nimport (\n\t\"net/http\"\n)\n",
	}
	for name, quelle := range mit {
		if !bindetHTTPEin(t, name+".go", quelle) {
			t.Errorf("%s: die Einbindung von net/http wird nicht erkannt", name)
		}
	}
	ohne := map[string]string{
		"keine Einbindung": "package x\n",
		"Nachbarpaket":     "package x\nimport (\n\t\"net/http/httptest\"\n\t\"net/url\"\n)\n",
		"nur im Kommentar": "package x\n// braucht net/http nicht\nimport \"fmt\"\n",
		"nur als Text":     "package x\nimport \"fmt\"\nvar s = \"net/http\"\n",
	}
	for name, quelle := range ohne {
		if bindetHTTPEin(t, name+".go", quelle) {
			t.Errorf("%s: gilt als Tür, obwohl net/http nicht eingebunden ist", name)
		}
	}
}

// Eine Datei in einem Unterordner zählt mit: Sonst verließe sie mit dem Umzug in einen
// Themenordner die Prüfungen, die api/ lesen.
func TestProduktivDateien_SteigtInUnterordner(t *testing.T) {
	wurzel := t.TempDir()
	for pfad, inhalt := range map[string]string{
		"tuer.go":                   "package x\n",
		"tuer_test.go":              "package x\n",
		"notiz.md":                  "kein Quelltext\n",
		"thema/regel.go":            "package thema\n",
		"thema/regel_test.go":       "package thema\n",
		"thema/tiefer/helfer.go":    "package tiefer\n",
		"thema/testdata/muster.go":  "package muster\n",
		"uploads/abgelegt.go":       "package uploads\n",
		".arbeitskopie/api/tuer.go": "package x\n",
	} {
		ziel := filepath.Join(wurzel, filepath.FromSlash(pfad))
		if err := os.MkdirAll(filepath.Dir(ziel), 0o750); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(ziel, []byte(inhalt), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	gefunden := produktivDateien(t, wurzel)
	slices.Sort(gefunden)
	erwartet := []string{"thema/regel.go", "thema/tiefer/helfer.go", "tuer.go"}
	if !slices.Equal(gefunden, erwartet) {
		t.Errorf("gefunden %v, erwartet %v", gefunden, erwartet)
	}
}

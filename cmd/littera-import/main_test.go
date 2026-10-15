package main

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"bibliothek/internal/pgtest"
	"bibliothek/pkg/isbnutil"
)

// Die Hülle des Imports eines Katalogisats; sie läuft einmal gegen den echten Bestand. Der
// Import selbst ist im Dienst geprüft (internal/service, internal/littera); hier geht es um das,
// was die Hülle tut: Argumente prüfen, die Datei öffnen, den Dienst mit echter Datenbank aufrufen
// und das Ergebnis melden.

func keineUmgebung(string) string { return "" }

func TestFuehreAus_OhneDateiIstEinFehlerVorJederVerbindung(t *testing.T) {
	var out bytes.Buffer
	err := fuehreAus(nil, keineUmgebung, &out)
	if err == nil || !strings.Contains(err.Error(), "-file") {
		t.Fatalf("erwartet den Hinweis auf -file, bekam %v", err)
	}
}

func TestFuehreAus_OhneDatenbankIstEinFehler(t *testing.T) {
	var out bytes.Buffer
	err := fuehreAus([]string{"-file", "egal.xml"}, keineUmgebung, &out)
	if err == nil || !strings.Contains(err.Error(), "DATABASE_URL") {
		t.Fatalf("erwartet den Hinweis auf DATABASE_URL, bekam %v", err)
	}
}

// Ein Tippfehler im Pfad scheitert an der Datei, nicht erst an der Datenbank. Die Adresse hier
// ist absichtlich unlesbar: pgxpool verbindet erst beim ersten Zugriff, eine nur falsche Adresse
// fiele also gar nicht auf, und der Test wäre auch bei vertauschter Reihenfolge grün. Eine
// unlesbare scheitert schon beim Lesen; steht die Datenbank vor der Datei, kommt dieser Fehler
// statt des Fehlers der Datei.
func TestFuehreAus_FehlendeDateiScheitertVorDerDatenbank(t *testing.T) {
	var out bytes.Buffer
	err := fuehreAus([]string{"-file", filepath.Join(t.TempDir(), "nirgends.xml"), "-db", "::unlesbar::"}, keineUmgebung, &out)
	if err == nil || !strings.Contains(err.Error(), "XML-Datei konnte nicht geöffnet werden") {
		t.Fatalf("erwartet den Dateifehler, bekam %v", err)
	}
}

// Der ganze Weg am echten Postgres: ein Katalogisat mit einem Titel, danach steht er im
// Bestand und die Ausgabe nennt die Zahl.
func TestFuehreAus_ImportiertEinKatalogisat(t *testing.T) {
	pool := pgtest.Pool(t) // lädt das Schema und überspringt ohne TEST_DATABASE_URL
	ctx := context.Background()
	const isbn = "978-3-16-148410-0"
	// Der Import legt die ISBN bereinigt ab (isbnutil.OhneTrenner: ohne Bindestriche) —
	// gesucht und aufgeräumt wird deshalb in derselben Schreibweise.
	gespeichert := isbnutil.OhneTrenner(isbn)
	t.Cleanup(func() {
		if _, err := pool.Exec(context.Background(), `DELETE FROM buecher_titel WHERE isbn = $1`, gespeichert); err != nil {
			t.Logf("Aufräumen: %v", err)
		}
	})

	pfad := filepath.Join(t.TempDir(), "katalogisat.xml")
	xml := `<?xml version="1.0" encoding="UTF-8"?>
<Katalogisate>
  <Katalogisat>
    <Feld MAB="100 ">Lergenmüller, Arno</Feld>
    <Feld MAB="310 ">Mathematik Neue Wege 9 (Importprobe)</Feld>
    <Feld MAB="540 ">` + isbn + `</Feld>
  </Katalogisat>
</Katalogisate>
`
	if err := os.WriteFile(pfad, []byte(xml), 0o600); err != nil {
		t.Fatal(err)
	}

	var out bytes.Buffer
	if err := fuehreAus([]string{"-file", pfad, "-db", os.Getenv(pgtest.Variable)}, keineUmgebung, &out); err != nil {
		t.Fatalf("run: %v", err)
	}
	if !strings.Contains(out.String(), `"verarbeitete_titel":1`) {
		t.Errorf("Ausgabe nennt die Zahl nicht:\n%s", out.String())
	}

	var titel string
	if err := pool.QueryRow(ctx, `SELECT titel FROM buecher_titel WHERE isbn = $1`, gespeichert).Scan(&titel); err != nil {
		t.Fatalf("Titel steht nach dem Import nicht im Bestand: %v", err)
	}
	if titel != "Mathematik Neue Wege 9 (Importprobe)" {
		t.Errorf("Titel im Bestand: %q", titel)
	}
}

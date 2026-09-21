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

// Die Hülle des Katalogisat-Imports: Sie läuft einmal gegen den echten Bestand und
// hatte bis zum 21.09.2026 keinen Test (OFFEN.md 5.10). Der Import selbst ist im Service
// geprüft (internal/service, internal/littera); hier geht es um das, was die Hülle tut:
// Argumente prüfen, die Datei öffnen, den Service mit echter Datenbank aufrufen und das
// Ergebnis melden.

func keineUmgebung(string) string { return "" }

func TestRun_OhneDateiIstEinFehlerVorJederVerbindung(t *testing.T) {
	var out bytes.Buffer
	err := run(nil, keineUmgebung, &out)
	if err == nil || !strings.Contains(err.Error(), "-file") {
		t.Fatalf("erwartet den Hinweis auf -file, bekam %v", err)
	}
}

func TestRun_OhneDatenbankIstEinFehler(t *testing.T) {
	var out bytes.Buffer
	err := run([]string{"-file", "egal.xml"}, keineUmgebung, &out)
	if err == nil || !strings.Contains(err.Error(), "DATABASE_URL") {
		t.Fatalf("erwartet den Hinweis auf DATABASE_URL, bekam %v", err)
	}
}

// Ein Tippfehler im Pfad scheitert an der Datei — nicht erst an der Datenbank. Die
// Adresse hier ist bewusst UNLESBAR: pgxpool verbindet erst beim ersten Zugriff, eine
// nur falsche Adresse fiele also gar nicht auf, und der Test wäre auch bei vertauschter
// Reihenfolge grün gewesen (so gesehen am 21.09.2026). Eine unlesbare scheitert schon
// beim Parsen — steht die Datenbank vor der Datei, kommt dieser Fehler statt des Dateifehlers.
func TestRun_FehlendeDateiScheitertVorDerDatenbank(t *testing.T) {
	var out bytes.Buffer
	err := run([]string{"-file", filepath.Join(t.TempDir(), "nirgends.xml"), "-db", "::unlesbar::"}, keineUmgebung, &out)
	if err == nil || !strings.Contains(err.Error(), "XML-Datei konnte nicht geöffnet werden") {
		t.Fatalf("erwartet den Dateifehler, bekam %v", err)
	}
}

// Der ganze Weg am echten Postgres: ein Katalogisat mit einem Titel, danach steht er im
// Bestand und die Ausgabe nennt die Zahl.
func TestRun_ImportiertEinKatalogisat(t *testing.T) {
	pool := pgtest.Pool(t) // lädt das Schema und überspringt ohne TEST_DATABASE_URL
	ctx := context.Background()
	const isbn = "978-3-16-148410-0"
	// Der Import legt die ISBN bereinigt ab (isbnutil.CleanISBN: ohne Bindestriche) —
	// gesucht und aufgeräumt wird deshalb in derselben Schreibweise.
	gespeichert := isbnutil.CleanISBN(isbn)
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
	if err := run([]string{"-file", pfad, "-db", os.Getenv(pgtest.EnvVar)}, keineUmgebung, &out); err != nil {
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

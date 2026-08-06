package inventur

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// imTestVerzeichnis wechselt für die Dauer des Tests in ein leeres Verzeichnis.
// Die uploads-Helfer arbeiten relativ zum Arbeitsverzeichnis des Prozesses.
func imTestVerzeichnis(t *testing.T) string {
	t.Helper()
	verzeichnis := t.TempDir()
	t.Chdir(verzeichnis)
	return verzeichnis
}

func TestSchreibeUploadDateiLegtDateiAn(t *testing.T) {
	verzeichnis := imTestVerzeichnis(t)

	if err := schreibeUploadDatei("cover_abc_1.jpg", []byte("bilddaten")); err != nil {
		t.Fatalf("schreiben: %v", err)
	}

	inhalt, err := os.ReadFile(filepath.Join(verzeichnis, "uploads", "cover_abc_1.jpg"))
	if err != nil {
		t.Fatalf("lesen: %v", err)
	}
	if string(inhalt) != "bilddaten" {
		t.Errorf("Inhalt %q, erwartet %q", inhalt, "bilddaten")
	}
}

// Der eigentliche Punkt der Umstellung: Ein Name, der aus uploads/ herausführt, wird
// vom Betriebssystem abgewiesen — nicht von einem Zeichenkettenvergleich im Programm.
func TestSchreibeUploadDateiVerlaesstUploadsNicht(t *testing.T) {
	verzeichnis := imTestVerzeichnis(t)

	faelle := map[string]string{
		"Elternverzeichnis":  "../entwischt.txt",
		"tief verschachtelt": "../../entwischt.txt",
		"absoluter Pfad":     "/tmp/entwischt-bibliothek-test.txt",
		"getarnt":            "uploads/../../entwischt.txt",
	}

	for name, dateiname := range faelle {
		t.Run(name, func(t *testing.T) {
			err := schreibeUploadDatei(dateiname, []byte("x"))
			if err == nil {
				t.Fatalf("%q wurde geschrieben — der Schreibzugriff verlässt uploads/", dateiname)
			}

			// Und nichts ist außerhalb gelandet.
			if _, statErr := os.Stat(filepath.Join(verzeichnis, "entwischt.txt")); statErr == nil {
				t.Error("Datei liegt neben uploads/ statt darin")
			}
		})
	}
}

// Symlink-Fall: Genau hier reichte die frühere Präfixprüfung nicht. Sie sah nur den
// zusammengesetzten Pfad ("uploads/raus.jpg" — beginnt mit "uploads/", also erlaubt)
// und nicht, wohin dieser Name im Dateisystem tatsächlich zeigt.
func TestSchreibeUploadDateiFolgtKeinemSymlinkAusUploadsHeraus(t *testing.T) {
	verzeichnis := imTestVerzeichnis(t)

	if err := os.MkdirAll("uploads", 0750); err != nil {
		t.Fatalf("uploads anlegen: %v", err)
	}
	ausserhalb := filepath.Join(verzeichnis, "geheim")
	if err := os.MkdirAll(ausserhalb, 0750); err != nil {
		t.Fatalf("Zielverzeichnis anlegen: %v", err)
	}
	if err := os.Symlink(filepath.Join(ausserhalb, "beute.txt"), filepath.Join("uploads", "raus.jpg")); err != nil {
		t.Skipf("Symlinks nicht verfügbar: %v", err)
	}

	err := schreibeUploadDatei("raus.jpg", []byte("beute"))
	if err == nil {
		t.Error("dem Symlink wurde gefolgt — der Schreibzugriff landete außerhalb von uploads/")
	}
	if _, statErr := os.Stat(filepath.Join(ausserhalb, "beute.txt")); statErr == nil {
		t.Error("Datei wurde außerhalb von uploads/ angelegt")
	}
}

func TestLoescheUploadDatei(t *testing.T) {
	verzeichnis := imTestVerzeichnis(t)

	if err := schreibeUploadDatei("weg.jpg", []byte("x")); err != nil {
		t.Fatalf("schreiben: %v", err)
	}
	if err := loescheUploadDatei("weg.jpg"); err != nil {
		t.Fatalf("löschen: %v", err)
	}
	if _, err := os.Stat(filepath.Join(verzeichnis, "uploads", "weg.jpg")); !os.IsNotExist(err) {
		t.Error("Datei existiert noch")
	}

	// Eine nicht vorhandene Datei ist kein Fehler — der Aufrufer räumt nur auf.
	if err := loescheUploadDatei("gabesnie.jpg"); err != nil {
		t.Errorf("Löschen einer nicht vorhandenen Datei meldete Fehler: %v", err)
	}
}

func TestLoescheUploadDateiVerlaesstUploadsNicht(t *testing.T) {
	verzeichnis := imTestVerzeichnis(t)

	opfer := filepath.Join(verzeichnis, "wichtig.txt")
	if err := os.WriteFile(opfer, []byte("nicht anfassen"), 0600); err != nil {
		t.Fatalf("Opferdatei anlegen: %v", err)
	}

	if err := loescheUploadDatei("../wichtig.txt"); err == nil {
		t.Error("Löschen außerhalb von uploads/ wurde nicht abgewiesen")
	}
	if _, err := os.Stat(opfer); err != nil {
		t.Errorf("Datei außerhalb von uploads/ wurde gelöscht: %v", err)
	}
}

// Belegt, dass der Ablehnungsgrund vom Dateisystem kommt und nicht von einer
// Eingabeprüfung, die jemand später "vereinfachen" könnte.
func TestFehlermeldungNenntDenDateinamen(t *testing.T) {
	imTestVerzeichnis(t)

	err := schreibeUploadDatei("../raus.jpg", []byte("x"))
	if err == nil {
		t.Fatal("kein Fehler")
	}
	if !strings.Contains(err.Error(), "raus.jpg") {
		t.Errorf("Fehlermeldung nennt die Datei nicht: %v", err)
	}
}

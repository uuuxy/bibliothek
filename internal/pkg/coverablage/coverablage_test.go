package coverablage

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// bereiteWurzel legt ein Wegwerf-Arbeitsverzeichnis mit „uploads/" an und wechselt hinein;
// sonst liefe der Test gegen das Upload-Verzeichnis des Entwicklungsrechners.
func bereiteWurzel(t *testing.T) string {
	t.Helper()
	basis := t.TempDir()
	t.Chdir(basis)
	if err := os.Mkdir(Wurzel, 0o755); err != nil {
		t.Fatalf("uploads anlegen: %v", err)
	}
	return basis
}

func schreibeDatei(t *testing.T, pfad string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(pfad), 0o755); err != nil {
		t.Fatalf("Verzeichnis anlegen: %v", err)
	}
	if err := os.WriteFile(pfad, []byte("bild"), 0o644); err != nil {
		t.Fatalf("Datei schreiben: %v", err)
	}
}

// Loesche entfernt nur Dateien direkt im Upload-Verzeichnis. Die Cover-URL eines Titels kommt
// aus einer Maske: Zeigte sie in einen Unterordner (Ausweisfotos, Zwischenspeicher) oder nach
// außen, nähme das Löschen des Titels eine fremde Datei mit.
func TestLoescheNimmtNurDateienDirektInDerWurzel(t *testing.T) {
	basis := bereiteWurzel(t)
	schreibeDatei(t, filepath.Join(Wurzel, "cover.png"))
	schreibeDatei(t, filepath.Join(Wurzel, "fotos", "ausweis.png"))
	if err := os.WriteFile(filepath.Join(basis, "geheim.txt"), []byte("geheim"), 0o644); err != nil {
		t.Fatalf("Datei außerhalb schreiben: %v", err)
	}

	for _, coverURL := range []string{
		"/uploads/cover.png",
		"/uploads/fotos/ausweis.png",
		"/uploads/fotos",
		"/uploads/../geheim.txt",
		"/uploads/",
		"/uploads/gibtesnicht.png",
		"https://portal.dnb.de/opac/mvb/cover?isbn=9783551551672",
		"",
	} {
		if err := Loesche(coverURL); err != nil {
			t.Errorf("Loesche(%q): %v", coverURL, err)
		}
	}

	if _, err := os.Stat(filepath.Join(Wurzel, "cover.png")); !os.IsNotExist(err) {
		t.Errorf("das Cover liegt noch da (%v)", err)
	}
	if err := os.Mkdir(filepath.Join(Wurzel, "leer"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := Loesche("/uploads/leer"); err != nil {
		t.Errorf("Loesche auf einem leeren Ordner: %v", err)
	}
	for _, bleibt := range []string{filepath.Join(Wurzel, "fotos", "ausweis.png"), filepath.Join(basis, "geheim.txt"), filepath.Join(Wurzel, "leer")} {
		if _, err := os.Stat(bleibt); err != nil {
			t.Errorf("%s ist fort: %v", bleibt, err)
		}
	}
}

// leeresArbeitsverzeichnis wechselt für die Dauer des Tests in ein leeres Verzeichnis ohne
// uploads/: Schreibe legt die Wurzel selbst an und arbeitet relativ zum Arbeitsverzeichnis.
func leeresArbeitsverzeichnis(t *testing.T) string {
	t.Helper()
	verzeichnis := t.TempDir()
	t.Chdir(verzeichnis)
	return verzeichnis
}

func TestSchreibeLegtDateiAn(t *testing.T) {
	verzeichnis := leeresArbeitsverzeichnis(t)

	if err := Schreibe("cover_abc_1.jpg", []byte("bilddaten")); err != nil {
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
func TestSchreibeVerlaesstDieWurzelNicht(t *testing.T) {
	verzeichnis := leeresArbeitsverzeichnis(t)

	faelle := map[string]string{
		"Elternverzeichnis":  "../entwischt.txt",
		"tief verschachtelt": "../../entwischt.txt",
		"absoluter Pfad":     "/tmp/entwischt-bibliothek-test.txt",
		"getarnt":            "uploads/../../entwischt.txt",
	}

	for name, dateiname := range faelle {
		t.Run(name, func(t *testing.T) {
			err := Schreibe(dateiname, []byte("x"))
			if err == nil {
				t.Fatalf("%q wurde geschrieben — der Schreibzugriff verlässt uploads/", dateiname)
			}

			// Und nichts ist außerhalb gelandet.
			if _, statFehler := os.Stat(filepath.Join(verzeichnis, "entwischt.txt")); statFehler == nil {
				t.Error("Datei liegt neben uploads/ statt darin")
			}
		})
	}
}

// Symlink-Fall: Genau hier reichte die frühere Präfixprüfung nicht. Sie sah nur den
// zusammengesetzten Pfad ("uploads/raus.jpg" — beginnt mit "uploads/", also erlaubt)
// und nicht, wohin dieser Name im Dateisystem tatsächlich zeigt.
func TestSchreibeFolgtKeinerVerknuepfungHinaus(t *testing.T) {
	verzeichnis := leeresArbeitsverzeichnis(t)

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

	err := Schreibe("raus.jpg", []byte("beute"))
	if err == nil {
		t.Error("dem Symlink wurde gefolgt — der Schreibzugriff landete außerhalb von uploads/")
	}
	if _, statFehler := os.Stat(filepath.Join(ausserhalb, "beute.txt")); statFehler == nil {
		t.Error("Datei wurde außerhalb von uploads/ angelegt")
	}
}

// Belegt, dass der Ablehnungsgrund vom Dateisystem kommt und nicht von einer
// Eingabeprüfung, die jemand später "vereinfachen" könnte.
func TestFehlermeldungNenntDenDateinamen(t *testing.T) {
	leeresArbeitsverzeichnis(t)

	err := Schreibe("../raus.jpg", []byte("x"))
	if err == nil {
		t.Fatal("kein Fehler")
	}
	if !strings.Contains(err.Error(), "raus.jpg") {
		t.Errorf("Fehlermeldung nennt die Datei nicht: %v", err)
	}
}

// IstVorhanden zählt nur eine Datei direkt in der Wurzel: Ein Ordner gleichen Namens ist kein
// abgelegtes Cover, und der Abruf legte sonst keines an.
func TestIstVorhandenNurFuerDateien(t *testing.T) {
	leeresArbeitsverzeichnis(t)
	if IstVorhanden("cover.jpg") {
		t.Fatal("ohne Datei gemeldet")
	}
	if err := Schreibe("cover.jpg", []byte("x")); err != nil {
		t.Fatalf("schreiben: %v", err)
	}
	if !IstVorhanden("cover.jpg") {
		t.Error("abgelegte Datei nicht gemeldet")
	}
	if err := os.Mkdir(filepath.Join(Wurzel, "ordner.jpg"), 0o755); err != nil {
		t.Fatal(err)
	}
	if IstVorhanden("ordner.jpg") {
		t.Error("Ordner als Datei gemeldet")
	}
}

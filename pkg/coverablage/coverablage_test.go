package coverablage

import (
	"os"
	"path/filepath"
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

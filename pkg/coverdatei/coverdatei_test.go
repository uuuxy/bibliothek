package coverdatei

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

// bereiteWurzel legt ein Wegwerf-Arbeitsverzeichnis mit „uploads/" an und wechselt hinein.
// Wurzel ist relativ zum Arbeitsverzeichnis des Servers; ohne den Wechsel würde der Test
// gegen das echte Upload-Verzeichnis des Entwicklungsrechners laufen.
func bereiteWurzel(t *testing.T) string {
	t.Helper()
	basis := t.TempDir()
	t.Chdir(basis)
	if err := os.Mkdir(Wurzel, 0o755); err != nil {
		t.Fatalf("uploads anlegen: %v", err)
	}
	return basis
}

// schreibeBild legt ein winziges, echt dekodierbares PNG ab — AlsJPEG soll an der
// Pfadprüfung scheitern, nicht am Bildinhalt.
func schreibeBild(t *testing.T, pfad string) {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 2, 2))
	img.Set(0, 0, color.RGBA{R: 200, G: 30, B: 30, A: 255})
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("PNG erzeugen: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(pfad), 0o755); err != nil {
		t.Fatalf("Verzeichnis anlegen: %v", err)
	}
	if err := os.WriteFile(pfad, buf.Bytes(), 0o644); err != nil {
		t.Fatalf("Bild schreiben: %v", err)
	}
}

func TestPfadNimmtNurCoverInnerhalbDerWurzel(t *testing.T) {
	basis := bereiteWurzel(t)
	schreibeBild(t, filepath.Join(Wurzel, "cover.png"))
	schreibeBild(t, filepath.Join(Wurzel, "unter", "cover.png"))
	if err := os.WriteFile(filepath.Join(basis, "geheim.txt"), []byte("geheim"), 0o644); err != nil {
		t.Fatalf("Datei außerhalb schreiben: %v", err)
	}
	if err := os.Mkdir(filepath.Join(Wurzel, "ordner"), 0o755); err != nil {
		t.Fatalf("Ordner anlegen: %v", err)
	}

	faelle := []struct {
		name     string
		coverURL string
		will     string
	}{
		{"gewöhnliches Cover", "/uploads/cover.png", filepath.Join(Wurzel, "cover.png")},
		{"Unterverzeichnis", "/uploads/unter/cover.png", filepath.Join(Wurzel, "unter", "cover.png")},
		{"Punkt im Weg bleibt drinnen", "/uploads/unter/./cover.png", filepath.Join(Wurzel, "unter", "cover.png")},
		{"Umweg bleibt drinnen", "/uploads/unter/../cover.png", filepath.Join(Wurzel, "cover.png")},
		{"leer", "", ""},
		{"fremde Herkunft", "https://example.org/cover.png", ""},
		{"fremdes Verzeichnis mit gleichem Anfang", "/uploadsX/cover.png", ""},
		{"ohne Dateinamen", "/uploads/", ""},
		{"gibt es nicht", "/uploads/fehlt.png", ""},
		{"ist ein Verzeichnis", "/uploads/ordner", ""},
		{"Ausbruch nach oben", "/uploads/../geheim.txt", ""},
		{"Ausbruch mit zwei Stufen", "/uploads/../../etc/passwd", ""},
	}
	for _, f := range faelle {
		t.Run(f.name, func(t *testing.T) {
			if hat := Pfad(f.coverURL); hat != f.will {
				t.Errorf("Pfad(%q) = %q, erwartet %q", f.coverURL, hat, f.will)
			}
			// AlsJPEG muss dieselbe Grenze ziehen — sonst hinge der Schutz allein an
			// der Höflichkeit der Aufrufer, vorher Pfad zu fragen.
			_, _, ok := AlsJPEG(f.coverURL)
			if ok != (f.will != "") {
				t.Errorf("AlsJPEG(%q) ok = %v, erwartet %v", f.coverURL, ok, f.will != "")
			}
		})
	}
}

// Die Verknüpfung liegt INNERHALB von uploads und zeigt nach draußen. Die Textprüfung
// kann das nicht sehen — der Pfad ist sauber; nur os.Root verweigert ihr zu folgen.
func TestPfadFolgtKeinerVerknuepfungNachDraussen(t *testing.T) {
	basis := bereiteWurzel(t)
	if err := os.WriteFile(filepath.Join(basis, "geheim.txt"), []byte("geheim"), 0o644); err != nil {
		t.Fatalf("Datei außerhalb schreiben: %v", err)
	}
	if err := os.Symlink(filepath.Join(basis, "geheim.txt"), filepath.Join(Wurzel, "cover.png")); err != nil {
		t.Skipf("Verknüpfungen nicht möglich: %v", err)
	}

	if hat := Pfad("/uploads/cover.png"); hat != "" {
		t.Errorf("Pfad folgte der Verknüpfung nach draußen: %q", hat)
	}
	if _, _, ok := AlsJPEG("/uploads/cover.png"); ok {
		t.Error("AlsJPEG folgte der Verknüpfung nach draußen")
	}
}

// Innerhalb der Wurzel darf eine Verknüpfung weiter funktionieren: Der Schutz soll
// Ausbrüche verhindern, nicht die Ablage einschränken.
func TestPfadNimmtVerknuepfungInnerhalbDerWurzel(t *testing.T) {
	bereiteWurzel(t)
	schreibeBild(t, filepath.Join(Wurzel, "echt.png"))
	if err := os.Symlink("echt.png", filepath.Join(Wurzel, "zeiger.png")); err != nil {
		t.Skipf("Verknüpfungen nicht möglich: %v", err)
	}

	if hat := Pfad("/uploads/zeiger.png"); hat != filepath.Join(Wurzel, "zeiger.png") {
		t.Errorf("Pfad(/uploads/zeiger.png) = %q", hat)
	}
	if _, _, ok := AlsJPEG("/uploads/zeiger.png"); !ok {
		t.Error("AlsJPEG konnte die Verknüpfung innerhalb der Wurzel nicht lesen")
	}
}

package oberflaechenstand

import (
	"os"
	"path/filepath"
	"testing"
)

// oeffne legt index.html in einem eigenen Ordner an und öffnet ihn wie der Router frontend/dist.
func oeffne(t *testing.T, index string) *os.Root {
	t.Helper()
	ordner := t.TempDir()
	if err := os.WriteFile(filepath.Join(ordner, "index.html"), []byte(index), 0o600); err != nil {
		t.Fatalf("index.html anlegen: %v", err)
	}
	root, err := os.OpenRoot(ordner)
	if err != nil {
		t.Fatalf("OpenRoot: %v", err)
	}
	t.Cleanup(func() {
		if err := root.Close(); err != nil {
			t.Errorf("Root schließen: %v", err)
		}
	})
	return root
}

// Die Form, die vite build schreibt: Das Programm ist das Modul, nicht das Stylesheet daneben.
func TestLese_ProgrammdateiWieVomBau(t *testing.T) {
	root := oeffne(t, `<head>
		<script type="module" crossorigin src="/assets/index-BQZzTtp2.js"></script>
		<link rel="stylesheet" crossorigin href="/assets/index-D7RhG9_i.css">
	</head>`)
	if got := Lese(root); got != "/assets/index-BQZzTtp2.js" {
		t.Errorf("Stand %q, erwartet /assets/index-BQZzTtp2.js", got)
	}
}

// Nennt index.html keine Programmdatei, bleibt der Stand leer, statt einen Teil der Seite zu nehmen.
func TestLese_OhneProgrammdatei(t *testing.T) {
	root := oeffne(t, `<!doctype html><script src="/alt.js"></script><title>ohne Modul</title>`)
	if got := Lese(root); got != "" {
		t.Errorf("Stand %q, erwartet leer", got)
	}
}

func TestLese_OhneOberflaeche(t *testing.T) {
	if got := Lese(nil); got != "" {
		t.Errorf("Stand %q ohne Oberfläche, erwartet leer", got)
	}
	leer, err := os.OpenRoot(t.TempDir())
	if err != nil {
		t.Fatalf("OpenRoot: %v", err)
	}
	t.Cleanup(func() {
		if err := leer.Close(); err != nil {
			t.Errorf("Root schließen: %v", err)
		}
	})
	if got := Lese(leer); got != "" {
		t.Errorf("Stand %q ohne index.html, erwartet leer", got)
	}
}

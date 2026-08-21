package uebernahme

import (
	"os"
	"strings"
	"testing"
)

// TestProtokollNenntDenQuellschluessel: ohne die Quell-ID in der Zeile ist das Protokoll
// wertlos — man findet den Datensatz in der Quelldatei nicht wieder.
func TestProtokollNenntDenQuellschluessel(t *testing.T) {
	pfad := t.TempDir() + "/err.log"
	p, err := NeuesProtokoll(pfad, "littera_id")
	if err != nil {
		t.Fatalf("Protokoll: %v", err)
	}
	p.Fehler("4711", "B-00042", "Barcode belegt")
	p.Warnung("4712", "", "Titel gekürzt")
	p.Schliessen()

	inhalt := leseDatei(t, pfad)
	for _, erwartet := range []string{"FEHLER", "littera_id=4711", "B-00042", "WARNUNG", "littera_id=4712"} {
		if !strings.Contains(inhalt, erwartet) {
			t.Errorf("Protokoll nennt %q nicht:\n%s", erwartet, inhalt)
		}
	}
	if p.Warnungen() != 1 || p.FehlerAnzahl() != 1 {
		t.Errorf("1/1 erwartet, gezählt: %d/%d", p.Warnungen(), p.FehlerAnzahl())
	}
}

func leseDatei(t *testing.T, pfad string) string {
	t.Helper()
	b, err := os.ReadFile(pfad) // #nosec G304 - Pfad aus t.TempDir()
	if err != nil {
		t.Fatalf("Protokoll lesen: %v", err)
	}
	return string(b)
}

func TestNeuesProtokoll(t *testing.T) {
	t.Run("erfolgreich", func(t *testing.T) {
		pfad := t.TempDir() + "/test.log"
		p, err := NeuesProtokoll(pfad, "id")
		if err != nil {
			t.Fatalf("unerwarteter Fehler: %v", err)
		}
		if p == nil {
			t.Fatal("erwartete Protokoll-Instanz, bekam nil")
		}
		p.Schliessen()

		if _, err := os.Stat(pfad); err != nil {
			t.Errorf("Protokolldatei wurde nicht angelegt: %v", err)
		}
	})

	t.Run("verwirft existierende Inhalte", func(t *testing.T) {
		pfad := t.TempDir() + "/test.log"
		err := os.WriteFile(pfad, []byte("altes zeug"), 0644)
		if err != nil {
			t.Fatalf("konnte Testdatei nicht anlegen: %v", err)
		}

		p, err := NeuesProtokoll(pfad, "id")
		if err != nil {
			t.Fatalf("unerwarteter Fehler: %v", err)
		}
		p.Schliessen()

		inhalt, err := os.ReadFile(pfad)
		if err != nil {
			t.Fatalf("konnte Datei nicht lesen: %v", err)
		}
		if len(inhalt) > 0 {
			t.Errorf("erwartete leere Datei nach NeuesProtokoll, fand %q", inhalt)
		}
	})

	t.Run("Fehler bei ungueltigem Pfad", func(t *testing.T) {
		pfad := t.TempDir() + "/gibt-es-nicht/test.log"
		p, err := NeuesProtokoll(pfad, "id")
		if err == nil {
			t.Error("erwartete Fehler für ungültigen Pfad, bekam nil")
			p.Schliessen()
		}
		if err != nil && !strings.Contains(err.Error(), "konnte die Protokolldatei") {
			t.Errorf("Fehlermeldung sollte 'konnte die Protokolldatei' enthalten, war: %v", err)
		}
	})
}

func TestProtokoll_Leeren(t *testing.T) {
	pfad := t.TempDir() + "/test_leeren.log"
	p, err := NeuesProtokoll(pfad, "test_id")
	if err != nil {
		t.Fatalf("unerwarteter Fehler: %v", err)
	}
	defer p.Schliessen()

	// Schreibe eine Warnung
	p.Warnung("123", "X", "Grund Y")

	// Puffer leeren, damit es in die Datei geschrieben wird, bevor wir schließen
	if err := p.Leeren(); err != nil {
		t.Fatalf("Leeren fehlgeschlagen: %v", err)
	}

	// Lese die Datei
	inhalt, err := os.ReadFile(pfad)
	if err != nil {
		t.Fatalf("Datei lesen: %v", err)
	}
	if len(inhalt) == 0 {
		t.Error("erwartete Inhalt in der Datei nach Leeren, fand nichts")
	}
	if !strings.Contains(string(inhalt), "WARNUNG") {
		t.Errorf("erwartete WARNUNG in Datei, fand: %q", string(inhalt))
	}
}

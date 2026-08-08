package uebernahme

import (
	"os"
	"strings"
	"testing"
)

// TestKuerzeZaehltZeichen: Postgres begrenzt varchar in ZEICHEN. Ein byteweiser Schnitt
// zerlegte deutsche Umlaute und schriebe ungültiges UTF-8 in die Datenbank.
func TestKuerzeZaehltZeichen(t *testing.T) {
	p := testProtokoll(t)

	lang := strings.Repeat("ä", 300)
	got := Kuerze(p, "1", "", "titel", lang, MaxFreitext)

	if r := []rune(got); len(r) != MaxFreitext {
		t.Fatalf("%d Zeichen erwartet, geliefert: %d", MaxFreitext, len(r))
	}
	if strings.ContainsRune(got, '�') {
		t.Error("die Kürzung hat ein Zeichen zerschnitten")
	}
	if p.Warnungen() != 1 {
		t.Errorf("die Kürzung muss protokolliert werden, gezählt: %d", p.Warnungen())
	}
}

func TestKuerzeLaesstPassendeWerteUnberuehrt(t *testing.T) {
	p := testProtokoll(t)

	grenzwert := strings.Repeat("ö", MaxFreitext)
	if got := Kuerze(p, "1", "", "titel", grenzwert, MaxFreitext); got != grenzwert {
		t.Error("ein Wert exakt auf Spaltenbreite darf nicht angetastet werden")
	}
	if p.Warnungen() != 0 {
		t.Errorf("keine Warnung erwartet, gezählt: %d", p.Warnungen())
	}
}

func TestKuerzeKurzerString(t *testing.T) {
	p := testProtokoll(t)

	kurz := "Hallo"
	if got := Kuerze(p, "1", "", "titel", kurz, MaxFreitext); got != kurz {
		t.Error("ein kurzer Wert darf nicht angetastet werden")
	}
	if p.Warnungen() != 0 {
		t.Errorf("keine Warnung erwartet, gezählt: %d", p.Warnungen())
	}
}

// TestKlaereISBNWertetAbStattZuVerwerfen hält die Grundhaltung fest: eine kaputte oder
// doppelte ISBN kostet die ISBN, nicht das Buch.
func TestKlaereISBNWertetAbStattZuVerwerfen(t *testing.T) {
	t.Run("ungültige Prüfziffer", func(t *testing.T) {
		p := testProtokoll(t)
		if got := KlaereISBN(p, "1", "9783161484101", map[string]string{}); got != "" {
			t.Errorf("leere ISBN erwartet, geliefert: %q", got)
		}
		if p.Warnungen() != 1 || p.FehlerAnzahl() != 0 {
			t.Errorf("WARNUNG erwartet, gezählt: %d/%d", p.Warnungen(), p.FehlerAnzahl())
		}
	})

	t.Run("Dublette im selben Lauf", func(t *testing.T) {
		p := testProtokoll(t)
		gesehen := map[string]string{"9783161484100": "7"}
		if got := KlaereISBN(p, "9", "978-3-16-148410-0", gesehen); got != "" {
			t.Errorf("leere ISBN erwartet, geliefert: %q", got)
		}
		if gesehen["9783161484100"] != "7" {
			t.Error("die bestehende Reservierung darf nicht überschrieben werden")
		}
	})

	t.Run("gültige ISBN wird reserviert", func(t *testing.T) {
		p := testProtokoll(t)
		gesehen := map[string]string{}
		if got := KlaereISBN(p, "3", "978-3-16-148410-0", gesehen); got != "9783161484100" {
			t.Errorf("normalisierte ISBN erwartet, geliefert: %q", got)
		}
		if gesehen["9783161484100"] != "3" {
			t.Error("die ISBN muss für den laufenden Import reserviert werden")
		}
		if p.Warnungen() != 0 {
			t.Errorf("keine Warnung erwartet, gezählt: %d", p.Warnungen())
		}
	})
}

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

func testProtokoll(t *testing.T) *Protokoll {
	t.Helper()
	p, err := NeuesProtokoll(t.TempDir()+"/err.log", "quell_id")
	if err != nil {
		t.Fatalf("Protokoll: %v", err)
	}
	t.Cleanup(p.Schliessen)
	return p
}

func leseDatei(t *testing.T, pfad string) string {
	t.Helper()
	b, err := os.ReadFile(pfad) // #nosec G304 - Pfad aus t.TempDir()
	if err != nil {
		t.Fatalf("Protokoll lesen: %v", err)
	}
	return string(b)
}

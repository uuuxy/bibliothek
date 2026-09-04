package uebernahme

import (
	"strings"
	"testing"
)

// TestKuerzeZaehltZeichen: Postgres begrenzt varchar in ZEICHEN. Ein byteweiser Schnitt
// zerlegte deutsche Umlaute und schriebe ungültiges UTF-8 in die Datenbank.
func TestKuerzeZaehltZeichen(t *testing.T) {
	p := testProtokoll(t)

	lang := strings.Repeat("ä", 300)
	got := Kuerze(p, FeldKontext{QuellID: "1", Kennung: "", Feld: "titel", Wert: lang, Max: MaxFreitext})

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
	if got := Kuerze(p, FeldKontext{QuellID: "1", Kennung: "", Feld: "titel", Wert: grenzwert, Max: MaxFreitext}); got != grenzwert {
		t.Error("ein Wert exakt auf Spaltenbreite darf nicht angetastet werden")
	}
	if p.Warnungen() != 0 {
		t.Errorf("keine Warnung erwartet, gezählt: %d", p.Warnungen())
	}
}

func TestKuerzeKurzerString(t *testing.T) {
	p := testProtokoll(t)

	kurz := "Hallo"
	if got := Kuerze(p, FeldKontext{QuellID: "1", Kennung: "", Feld: "titel", Wert: kurz, Max: MaxFreitext}); got != kurz {
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

func testProtokoll(t *testing.T) *Protokoll {
	t.Helper()
	p, err := NeuesProtokoll(t.TempDir()+"/err.log", "quell_id")
	if err != nil {
		t.Fatalf("Protokoll: %v", err)
	}
	t.Cleanup(p.Schliessen)
	return p
}

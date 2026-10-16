package uebernahme

import (
	"strings"
	"testing"
)

// Postgres begrenzt varchar in Zeichen. Ein Schnitt nach Bytes zerlegte einen Umlaut und
// schriebe ungültiges UTF-8 in die Datenbank.
func TestKuerzeZaehltZeichen(t *testing.T) {
	p := testProtokoll(t)

	lang := strings.Repeat("ä", 300)
	got := Kuerze(Kuerzung{Protokoll: p, QuellID: "1", Kennung: "", Feld: "titel", Wert: lang, Max: MaxFreitext})

	if r := []rune(got); len(r) != MaxFreitext {
		t.Fatalf("len(r) = %d, erwartet %d Zeichen", len(r), MaxFreitext)
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
	if got := Kuerze(Kuerzung{Protokoll: p, QuellID: "1", Kennung: "", Feld: "titel", Wert: grenzwert, Max: MaxFreitext}); got != grenzwert {
		t.Error("ein Wert exakt auf Spaltenbreite darf nicht angetastet werden")
	}
	if p.Warnungen() != 0 {
		t.Errorf("keine Warnung erwartet, gezählt: %d", p.Warnungen())
	}
}

func TestKuerzeKurzerString(t *testing.T) {
	p := testProtokoll(t)

	kurz := "Hallo"
	if got := Kuerze(Kuerzung{Protokoll: p, QuellID: "1", Kennung: "", Feld: "titel", Wert: kurz, Max: MaxFreitext}); got != kurz {
		t.Error("ein kurzer Wert darf nicht angetastet werden")
	}
	if p.Warnungen() != 0 {
		t.Errorf("keine Warnung erwartet, gezählt: %d", p.Warnungen())
	}
}

// Eine kaputte oder doppelte ISBN kostet die ISBN und nicht das Buch.
func TestKlaereISBNWertetAbStattZuVerwerfen(t *testing.T) {
	t.Run("ungültige Prüfziffer", func(t *testing.T) {
		p := testProtokoll(t)
		if got := KlaereISBN(p, "1", "9783161484101", map[string]string{}); got != "" {
			t.Errorf("KlaereISBN(…) = %q, erwartet leere ISBN", got)
		}
		if p.Warnungen() != 1 || p.FehlerAnzahl() != 0 {
			t.Errorf("WARNUNG erwartet, gezählt: %d/%d", p.Warnungen(), p.FehlerAnzahl())
		}
	})

	t.Run("Dublette im selben Lauf", func(t *testing.T) {
		p := testProtokoll(t)
		gesehen := map[string]string{"9783161484100": "7"}
		if got := KlaereISBN(p, "9", "978-3-16-148410-0", gesehen); got != "" {
			t.Errorf("KlaereISBN(…) = %q, erwartet leere ISBN", got)
		}
		if gesehen["9783161484100"] != "7" {
			t.Error("die bestehende Reservierung darf nicht überschrieben werden")
		}
	})

	// Die Datenbank speichert eine zehnstellige ISBN dreizehnstellig. Die Vormerkung muss
	// dieselbe Form führen, sonst trifft der zweite Titel erst den UNIQUE-Index.
	t.Run("Dublette in der anderen Länge", func(t *testing.T) {
		for name, f := range map[string]struct{ vorgemerkt, kommt string }{
			"dreizehn vorgemerkt, zehn kommt": {"978-3-16-148410-0", "3-16-148410-X"},
			"zehn vorgemerkt, dreizehn kommt": {"3-16-148410-x", "9783161484100"},
		} {
			p := testProtokoll(t)
			gesehen := map[string]string{}
			if got := KlaereISBN(p, "7", f.vorgemerkt, gesehen); got != "9783161484100" {
				t.Errorf("%s: der erste Titel bekommt %q, erwartet 9783161484100", name, got)
			}
			if got := KlaereISBN(p, "9", f.kommt, gesehen); got != "" {
				t.Errorf("%s: der zweite Titel bekommt %q, erwartet keine ISBN", name, got)
			}
			if p.Warnungen() != 1 {
				t.Errorf("%s: eine Warnung erwartet, gezählt: %d", name, p.Warnungen())
			}
		}
	})

	t.Run("gültige ISBN wird reserviert", func(t *testing.T) {
		p := testProtokoll(t)
		gesehen := map[string]string{}
		if got := KlaereISBN(p, "3", "978-3-16-148410-0", gesehen); got != "9783161484100" {
			t.Errorf("KlaereISBN(…) = %q, erwartet normalisierte ISBN", got)
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
	p, err := NewProtokoll(t.TempDir()+"/err.log", "quell_id")
	if err != nil {
		t.Fatalf("Protokoll: %v", err)
	}
	t.Cleanup(p.Schliesse)
	return p
}

package api

import (
	"strings"
	"testing"
	"unicode/utf8"
)

// Diese Tests halten eine Bugklasse fest, die in dieser Datenbasis garantiert auftritt:
// Kürzen nach Bytes statt nach Zeichen.
//
// len(s) und s[:n] rechnen in Go in Bytes; ein Umlaut belegt in UTF-8 zwei. An fünf
// Stellen (Mahnliste, Erinnerungsbrief, Etikettendruck) wurde genau so gekürzt — ein
// Schnitt mitten durch „ä" hinterließ ein halbes Zeichen, aus dem der Unicode-Übersetzer
// von gofpdf sichtbaren Zeichensalat machte. Auf Schriftstücken, die an Eltern gehen.

func TestKuerzeAufZeichenLaesstKurzeWerteUnberuehrt(t *testing.T) {
	for _, s := range []string{"", "kurz", "Müller", strings.Repeat("ö", 10)} {
		if got := kuerzeAufZeichen(s, 10); got != s {
			t.Errorf("kuerzeAufZeichen(%q, 10) = %q — kurze Werte dürfen nicht angetastet werden", s, got)
		}
	}
}

func TestKuerzeAufZeichenHaeltDieObergrenze(t *testing.T) {
	faelle := []struct {
		name string
		ein  string
		max  int
	}{
		{"reines ASCII", strings.Repeat("a", 100), 38},
		{"nur Umlaute", strings.Repeat("ä", 100), 38},
		{"gemischt", strings.Repeat("aä", 50), 19},
		{"Emoji (4 Byte je Zeichen)", strings.Repeat("📚", 50), 20},
		{"scharfes S", strings.Repeat("ß", 100), 30},
	}

	for _, f := range faelle {
		t.Run(f.name, func(t *testing.T) {
			got := kuerzeAufZeichen(f.ein, f.max)

			if n := utf8.RuneCountInString(got); n != f.max {
				t.Errorf("%d Zeichen erwartet, geliefert: %d", f.max, n)
			}
			// Der eigentliche Punkt: Ein byteweiser Schnitt hinterlässt ungültiges UTF-8.
			if !utf8.ValidString(got) {
				t.Errorf("Ergebnis ist kein gültiges UTF-8 — ein Zeichen wurde zerschnitten: %q", got)
			}
			if !strings.HasSuffix(got, "…") {
				t.Errorf("gekürzter Wert muss auf … enden, geliefert: %q", got)
			}
		})
	}
}

// TestKuerzeAufZeichenGrenzwert: genau auf der Grenze wird nicht gekürzt, ein Zeichen
// darüber schon. Ein Off-by-one hier verschiebt jede Spaltenbreite im PDF.
func TestKuerzeAufZeichenGrenzwert(t *testing.T) {
	genau := strings.Repeat("ü", 20)
	if got := kuerzeAufZeichen(genau, 20); got != genau {
		t.Errorf("exakt auf der Grenze darf nicht gekürzt werden, geliefert: %q", got)
	}

	einsZuViel := strings.Repeat("ü", 21)
	got := kuerzeAufZeichen(einsZuViel, 20)
	if utf8.RuneCountInString(got) != 20 {
		t.Errorf("ein Zeichen über der Grenze muss auf 20 kürzen, geliefert: %d",
			utf8.RuneCountInString(got))
	}
	if got != strings.Repeat("ü", 19)+"…" {
		t.Errorf("unerwartetes Ergebnis: %q", got)
	}
}

// TestByteweisesKuerzenWaereKaputt dokumentiert den alten Zustand und begründet damit,
// warum es diese Funktion überhaupt gibt. Schlüge dieser Test eines Tages fehl, wäre die
// Annahme „Umlaute belegen mehrere Bytes" widerlegt — dann darf auch die Funktion weg.
func TestByteweisesKuerzenWaereKaputt(t *testing.T) {
	titel := strings.Repeat("ä", 30) // 60 Bytes, 30 Zeichen

	// Das alte Muster: len(...) > 40 { titel[:37] + "…" }
	if len(titel) <= 40 {
		t.Fatalf("Vorbedingung verfehlt: %d Bytes", len(titel))
	}
	alt := titel[:37] + "…"
	if utf8.ValidString(alt) {
		t.Fatal("erwartet war ungültiges UTF-8 — der byteweise Schnitt trifft hier keine Zeichengrenze")
	}

	neu := kuerzeAufZeichen(titel, 38)
	if !utf8.ValidString(neu) {
		t.Errorf("kuerzeAufZeichen liefert ungültiges UTF-8: %q", neu)
	}
	// 30 Zeichen liegen unter der Grenze von 38 — es wird gar nicht gekürzt.
	if neu != titel {
		t.Errorf("30 Zeichen liegen unter der Grenze und dürfen unverändert bleiben, geliefert: %q", neu)
	}
}

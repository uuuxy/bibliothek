package littera

import (
	"os"
	"strings"
	"testing"

	"bibliothek/internal/uebernahme"
)

func TestLowerTrim(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"", ""},
		{"   ", ""},
		{"TEST@EXAMPLE.COM", "test@example.com"},
		{"  Test@Example.com  ", "test@example.com"},
		{"test@example.com", "test@example.com"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := lowerTrim(tt.input); got != tt.expected {
				t.Errorf("lowerTrim(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestPersonenlaufAusweis(t *testing.T) {
	prot, err := uebernahme.NeuesProtokoll(t.TempDir()+"/prot.log", "id")
	if err != nil {
		t.Fatalf("NeuesProtokoll: %v", err)
	}
	t.Cleanup(func() { prot.Schliessen() })

	p := &personenlauf{
		s:               &Schreiber{prot: prot},
		belegteAusweise: make(map[string]bool),
	}

	// 1. Regular case: Lesernummer exists and is not used
	l1 := Leser{ID: "1", Lesernummer: "101"}
	if got := p.ausweis(l1); got != "101" {
		t.Errorf("expected 101, got %q", got)
	}
	if !p.belegteAusweise["101"] {
		t.Errorf("101 should be marked as used")
	}

	// 2. Collision case: Lesernummer already used
	l2 := Leser{ID: "2", Lesernummer: "101"}
	if got := p.ausweis(l2); got != "L-2" {
		t.Errorf("expected L-2 due to collision, got %q", got)
	}
	if !p.belegteAusweise["L-2"] {
		t.Errorf("L-2 should be marked as used")
	}

	// 3. Empty Lesernummer
	l3 := Leser{ID: "3", Lesernummer: ""}
	if got := p.ausweis(l3); got != "L-3" {
		t.Errorf("expected L-3 due to empty lesernummer, got %q", got)
	}
	if !p.belegteAusweise["L-3"] {
		t.Errorf("L-3 should be marked as used")
	}
}

func TestPersonenlaufMailadresse(t *testing.T) {
	prot, err := uebernahme.NeuesProtokoll(t.TempDir()+"/prot.log", "id")
	if err != nil {
		t.Fatalf("NeuesProtokoll: %v", err)
	}
	t.Cleanup(func() { prot.Schliessen() })

	p := &personenlauf{
		s:            &Schreiber{prot: prot},
		belegteMails: make(map[string]bool),
	}

	// 1. Regular case: Email exists and is not used
	l1 := Leser{ID: "1", EMail: "Test@Example.com"}
	if got := p.mailadresse(l1); got != "test@example.com" {
		t.Errorf("expected test@example.com, got %q", got)
	}
	if !p.belegteMails["test@example.com"] {
		t.Errorf("test@example.com should be marked as used")
	}

	// 2. Collision case: Email already used
	l2 := Leser{ID: "2", EMail: "test@example.com"}
	expected2 := "littera-2@littera.invalid"
	if got := p.mailadresse(l2); got != expected2 {
		t.Errorf("expected %q, got %q", expected2, got)
	}
	if !p.belegteMails[expected2] {
		t.Errorf("%q should be marked as used", expected2)
	}

	// 3. Empty Email
	l3 := Leser{ID: "3", EMail: ""}
	expected3 := "littera-3@littera.invalid"
	if got := p.mailadresse(l3); got != expected3 {
		t.Errorf("expected %q, got %q", expected3, got)
	}
	if !p.belegteMails[expected3] {
		t.Errorf("%q should be marked as used", expected3)
	}
}

func TestPersonenlaufAbgangsjahr(t *testing.T) {
	p := &personenlauf{
		s: &Schreiber{
			opt: Optionen{SchuljahrEnde: 2027},
		},
	}

	// 1. Valid class with calculable year
	l1 := Leser{Klasse: "10R1"}
	jahr, ok := p.abgangsjahr(l1)
	if !ok || jahr != 2027 {
		t.Errorf("expected 2027, true; got %d, %v", jahr, ok)
	}

	// 2. Uncalculable class, but ArtAbgegangen
	l2 := Leser{Klasse: "Ab", Art: ArtAbgegangen}
	jahr, ok = p.abgangsjahr(l2)
	if !ok || jahr != 2027 {
		t.Errorf("expected 2027, true; got %d, %v", jahr, ok)
	}

	// 3. Uncalculable class, NOT ArtAbgegangen
	l3 := Leser{Klasse: "Sonderklasse", Art: ArtSchueler}
	jahr, ok = p.abgangsjahr(l3)
	if ok || jahr != 0 {
		t.Errorf("expected 0, false; got %d, %v", jahr, ok)
	}
}

func TestPersonenlaufKuerze(t *testing.T) {
	prot, err := uebernahme.NeuesProtokoll(t.TempDir()+"/prot.log", "id")
	if err != nil {
		t.Fatalf("NeuesProtokoll: %v", err)
	}
	t.Cleanup(func() { prot.Schliessen() })

	p := &personenlauf{
		s: &Schreiber{prot: prot},
	}

	l := Leser{ID: "1", Lesernummer: "101"}

	// Value within limits
	val1 := "Short"
	if got := p.kuerze(l, "feld", val1, 10); got != val1 {
		t.Errorf("expected %q, got %q", val1, got)
	}

	// Value exceeding limits
	val2 := "VeryLongString"
	expected2 := "VeryLongSt"
	if got := p.kuerze(l, "feld", val2, 10); got != expected2 {
		t.Errorf("expected %q, got %q", expected2, got)
	}
}

// TestPersonenlaufMailadresse_ProtokollOhneEchteAdresse ist das Gegenstück zum
// Datenschutz-Gate in internal/uebernahme: Dort geht es um den Wert, den Postgres
// meldet, hier um den, den wir selbst hineinschreiben. Bis zum 23.08.2026 stand die
// kollidierende Adresse als Kennung in `littera_import.log` — unverschlüsselt, ohne
// Frist. Die Lesernummer reicht, um die Zeile in der Quelldatei wiederzufinden.
func TestPersonenlaufMailadresse_ProtokollOhneEchteAdresse(t *testing.T) {
	pfad := t.TempDir() + "/prot.log"
	prot, err := uebernahme.NeuesProtokoll(pfad, "littera_id")
	if err != nil {
		t.Fatalf("NeuesProtokoll: %v", err)
	}

	p := &personenlauf{s: &Schreiber{prot: prot}, belegteMails: make(map[string]bool)}

	adresse := "erika.mustermann@philipp-reis-schule.de"
	// Erste Person belegt die Adresse, die zweite kollidiert damit.
	p.mailadresse(Leser{ID: "1", Lesernummer: "L-4711", EMail: adresse})
	p.mailadresse(Leser{ID: "2", Lesernummer: "L-4712", EMail: adresse})
	prot.Schliessen()

	roh, err := os.ReadFile(pfad)
	if err != nil {
		t.Fatalf("Protokoll lesen: %v", err)
	}
	inhalt := string(roh)

	if strings.Contains(inhalt, adresse) {
		t.Errorf("die echte Adresse steht im Protokoll:\n%s", inhalt)
	}
	if strings.Contains(inhalt, "mustermann") {
		t.Errorf("ein Bestandteil der Adresse steht im Protokoll:\n%s", inhalt)
	}
	// Die Zeile muss reparierbar bleiben: Quell-ID und Lesernummer gehören hinein.
	for _, erwartet := range []string{"littera_id=2", "L-4712", "bereits vergeben"} {
		if !strings.Contains(inhalt, erwartet) {
			t.Errorf("Protokollzeile nennt %q nicht: %s", erwartet, inhalt)
		}
	}
}

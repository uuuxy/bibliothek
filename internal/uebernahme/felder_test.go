package uebernahme

import (
	"path/filepath"
	"testing"
)

func TestKuerze(t *testing.T) {
	tmpDir := t.TempDir()
	protokollPfad := filepath.Join(tmpDir, "protokoll.log")
	p, err := NeuesProtokoll(protokollPfad, "id")
	if err != nil {
		t.Fatalf("NeuesProtokoll schlug fehl: %v", err)
	}
	defer p.Schliessen()

	tests := []struct {
		name     string
		wert     string
		max      int
		erwartet string
		warnung  bool
	}{
		{"kürzer als max", "abc", 5, "abc", false},
		{"genau max", "abc", 3, "abc", false},
		{"länger als max (ASCII)", "abcd", 3, "abc", true},
		{"länger als max (Umlaute/Runen)", "äöüß", 3, "äöü", true},
		{"leer", "", 5, "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			startWarnungen := p.Warnungen()
			got := Kuerze(p, "1", "kennung", "feld", tt.wert, tt.max)
			if got != tt.erwartet {
				t.Errorf("Kuerze() = %q, erwartet %q", got, tt.erwartet)
			}
			warnungAusgeloest := (p.Warnungen() - startWarnungen) > 0
			if warnungAusgeloest != tt.warnung {
				t.Errorf("Kuerze() Warnungen = %v, erwartet: %v", warnungAusgeloest, tt.warnung)
			}
		})
	}
}

func TestKuerzeNullbar(t *testing.T) {
	tmpDir := t.TempDir()
	protokollPfad := filepath.Join(tmpDir, "protokoll.log")
	p, err := NeuesProtokoll(protokollPfad, "id")
	if err != nil {
		t.Fatalf("NeuesProtokoll schlug fehl: %v", err)
	}
	defer p.Schliessen()

	t.Run("leer ergibt nil", func(t *testing.T) {
		if got := KuerzeNullbar(p, "1", "kennung", "feld", "", 5); got != nil {
			t.Errorf("KuerzeNullbar() erwartet nil, bekam %v", *got)
		}
	})

	t.Run("nicht leer", func(t *testing.T) {
		got := KuerzeNullbar(p, "1", "kennung", "feld", "abcde", 3)
		if got == nil {
			t.Fatal("KuerzeNullbar() erwartet nicht nil")
		}
		if *got != "abc" {
			t.Errorf("KuerzeNullbar() = %q, erwartet %q", *got, "abc")
		}
	})
}

func TestNullbar(t *testing.T) {
	t.Run("leer ergibt nil", func(t *testing.T) {
		if got := Nullbar(""); got != nil {
			t.Errorf("Nullbar() erwartet nil, bekam %v", *got)
		}
	})

	t.Run("nicht leer", func(t *testing.T) {
		got := Nullbar("abc")
		if got == nil {
			t.Fatal("Nullbar() erwartet nicht nil")
		}
		if *got != "abc" {
			t.Errorf("Nullbar() = %q, erwartet %q", *got, "abc")
		}
	})
}

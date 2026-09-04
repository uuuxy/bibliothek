package uebernahme

import (
	"path/filepath"
	"testing"
)

func TestKuerzeNullbar(t *testing.T) {
	tmpDir := t.TempDir()
	protokollPfad := filepath.Join(tmpDir, "protokoll.log")
	p, err := NeuesProtokoll(protokollPfad, "id")
	if err != nil {
		t.Fatalf("NeuesProtokoll schlug fehl: %v", err)
	}
	defer p.Schliessen()

	t.Run("leer ergibt nil", func(t *testing.T) {
		if got := KuerzeNullbar(p, FeldKontext{QuellID: "1", Kennung: "kennung", Feld: "feld", Wert: "", Max: 5}); got != nil {
			t.Errorf("KuerzeNullbar() erwartet nil, bekam %v", *got)
		}
	})

	t.Run("nicht leer", func(t *testing.T) {
		got := KuerzeNullbar(p, FeldKontext{QuellID: "1", Kennung: "kennung", Feld: "feld", Wert: "abcde", Max: 3})
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

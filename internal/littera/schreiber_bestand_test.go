package littera

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"

	"bibliothek/internal/uebernahme"
)

func TestJahrOderNil(t *testing.T) {
	if got := jahrOderNil(0); got != nil {
		t.Errorf("jahrOderNil(0) = %v, want nil", got)
	}

	if got := jahrOderNil(2023); got == nil || *got != 2023 {
		if got == nil {
			t.Errorf("jahrOderNil(2023) = nil, want 2023")
		} else {
			t.Errorf("jahrOderNil(2023) = %v, want 2023", *got)
		}
	}
}

func TestJsonEigenschaften(t *testing.T) {
	werte := map[string]string{
		"quelle": "littera",
		"leer":   "",
		"zahl":   "123",
	}

	got, err := jsonEigenschaften(werte)
	if err != nil {
		t.Fatalf("jsonEigenschaften error: %v", err)
	}

	var parsed map[string]string
	if err := json.Unmarshal([]byte(got), &parsed); err != nil {
		t.Fatalf("invalid JSON returned: %v", err)
	}

	if len(parsed) != 2 {
		t.Errorf("expected 2 keys, got %d", len(parsed))
	}
	if parsed["quelle"] != "littera" {
		t.Errorf("expected quelle=littera, got %v", parsed["quelle"])
	}
	if parsed["zahl"] != "123" {
		t.Errorf("expected zahl=123, got %v", parsed["zahl"])
	}
	if _, ok := parsed["leer"]; ok {
		t.Errorf("did not expect empty value to be included")
	}
}

func TestErworbenAm(t *testing.T) {
	ersatz := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)

	// Valid date
	e1 := Exemplar{Zugangsdatum: "05/03/07"}
	got1 := erworbenAm(e1, ersatz)
	expected1 := time.Date(2007, 5, 3, 0, 0, 0, 0, time.UTC)
	if got1.Year() != expected1.Year() || got1.Month() != expected1.Month() || got1.Day() != expected1.Day() {
		t.Errorf("erworbenAm valid date = %v, want %v", got1, expected1)
	}

	// Invalid date
	e2 := Exemplar{Zugangsdatum: "ungültig"}
	got2 := erworbenAm(e2, ersatz)
	if !got2.Equal(ersatz) {
		t.Errorf("erworbenAm invalid date = %v, want %v", got2, ersatz)
	}

	// Empty date
	e3 := Exemplar{Zugangsdatum: ""}
	got3 := erworbenAm(e3, ersatz)
	if !got3.Equal(ersatz) {
		t.Errorf("erworbenAm empty date = %v, want %v", got3, ersatz)
	}
}

func TestFelder(t *testing.T) {
	pfad := t.TempDir() + "/protokoll.md"
	prot, err := uebernahme.NeuesProtokoll(pfad, "Littera-ID")
	if err != nil {
		t.Fatalf("NeuesProtokoll: %v", err)
	}

	s := &Schreiber{prot: prot}
	ab := &Altbestand{
		Verlage:     map[string]string{"V1": "Ein Verlag"},
		Signaturen:  map[string]string{"T1": "Sig 123"},
		Medienarten: map[string]string{"M1": "Zeitschrift"},
	}
	l := &bestandslauf{s: s, ab: ab}

	titel := Titel{
		ID:          "T1",
		Haupttitel:  "Mein Buch",
		Untertitel:  "Band 1",
		Autor:       "Autor A",
		VerlagID:    "V1",
		MedienartID: "M1",
	}

	f := l.felder(titel)

	if f.titel != "Mein Buch" {
		t.Errorf("titel = %v, want Mein Buch", f.titel)
	}
	if f.untertitel == nil || *f.untertitel != "Band 1" {
		t.Errorf("untertitel = %v, want Band 1", f.untertitel)
	}
	if f.autor == nil || *f.autor != "Autor A" {
		t.Errorf("autor = %v, want Autor A", f.autor)
	}
	if f.verlag == nil || *f.verlag != "Ein Verlag" {
		t.Errorf("verlag = %v, want Ein Verlag", f.verlag)
	}
	if f.signatur == nil || *f.signatur != "Sig 123" {
		t.Errorf("signatur = %v, want Sig 123", f.signatur)
	}
	if f.medientyp != "Zeitschrift" {
		t.Errorf("medientyp = %v, want Zeitschrift", f.medientyp)
	}
	prot.Schliessen()
}

func TestFelder_LeererHaupttitel(t *testing.T) {
	pfad := t.TempDir() + "/protokoll.md"
	prot, err := uebernahme.NeuesProtokoll(pfad, "Littera-ID")
	if err != nil {
		t.Fatalf("NeuesProtokoll: %v", err)
	}

	s := &Schreiber{prot: prot}
	ab := &Altbestand{}
	l := &bestandslauf{s: s, ab: ab}

	titel := Titel{
		ID:         "T2",
		Haupttitel: "",
	}

	f := l.felder(titel)

	expectedTitel := "[ohne Titel, Littera T2]"
	if f.titel != expectedTitel {
		t.Errorf("titel = %v, want %v", f.titel, expectedTitel)
	}

	prot.Schliessen()
	data, err := os.ReadFile(pfad)
	if err != nil {
		t.Fatalf("protokoll lesen: %v", err)
	}
	if !strings.Contains(string(data), "WARNUNG") || !strings.Contains(string(data), "Haupttitel ist leer") {
		t.Errorf("expected warning in protokoll, got: %s", string(data))
	}
}

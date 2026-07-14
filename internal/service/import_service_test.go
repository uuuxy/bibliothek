package service

import (
	"context"
	"errors"
	"strings"
	"testing"
)

// Falsche XML-Dateien (z. B. der Schlagwort-Export systematik+.xml mit
// <Schlagworte>-Wurzel) sind Nutzer-Formatfehler: ParseLitteraXML muss sie als
// ErrKeinKatalogisat melden, damit der Handler 400 statt 500 antwortet.
func TestParseLitteraXML_FalscheWurzelIstKeinKatalogisat(t *testing.T) {
	tests := []struct {
		name string
		xml  string
	}{
		{"Schlagwort-Export", `<?xml version="1.0"?><Schlagworte><SW>Test</SW></Schlagworte>`},
		{"kaputtes XML", `<?xml version="1.0"?><Katalogisate><Katalogisat>`},
		{"leere Katalogisate", `<?xml version="1.0"?><Katalogisate></Katalogisate>`},
		{"gar kein XML", `Titel,Autor,Barcode`},
	}

	svc := NewImportService(&stubBookRepo{}, nil)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := svc.ParseLitteraXML(context.Background(), strings.NewReader(tt.xml))
			if !errors.Is(err, ErrKeinKatalogisat) {
				t.Errorf("err = %v, want ErrKeinKatalogisat", err)
			}
		})
	}
}

func TestBereinigeImportTitel(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		// Streu-Anführungszeichen aus der PDF-konvertierten Bestands-CSV
		{`"Elemente Chemie 1`, "Elemente Chemie 1"},
		{`"Mensch und Politik Sekundarstufe 1"`, "Mensch und Politik Sekundarstufe 1"},
		{`  Faust  `, "Faust"},
		{"Faust", "Faust"},
		// Anführungszeichen IM Titel bleiben erhalten
		{`Das "besondere" Buch`, `Das "besondere" Buch`},
	}
	for _, tt := range tests {
		if got := bereinigeImportTitel(tt.in); got != tt.want {
			t.Errorf("bereinigeImportTitel(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestTitelZeilenFelder_LMFAusKategorieUndSignatur(t *testing.T) {
	headerMap := map[string]int{"titel": 0, "kategorie": 1, "signatur": 2}

	// LMF-Token in der Kategorie (Bestands-CSV-Fall)
	titel, signatur, kategorie := titelZeilenFelder([]string{"Deutschbuch 9", "Buch LMF Ma 6", "De 9"}, headerMap)
	if titel != "LMF-Deutschbuch 9" || kategorie != "Buch Ma 6" || signatur != "De 9" {
		t.Errorf("Kategorie-LMF: titel=%q kategorie=%q signatur=%q", titel, kategorie, signatur)
	}

	// LMF-Präfix in der Signatur (Littera-Konvention wie im XML-Feld 700)
	titel, signatur, _ = titelZeilenFelder([]string{"Biologie heute 7", "Buch", "LMF Bio 7"}, headerMap)
	if titel != "LMF-Biologie heute 7" || signatur != "Bio 7" {
		t.Errorf("Signatur-LMF: titel=%q signatur=%q", titel, signatur)
	}

	// Ohne LMF bleibt alles unangetastet
	titel, signatur, _ = titelZeilenFelder([]string{"Harry Potter", "Roman", "JF"}, headerMap)
	if titel != "Harry Potter" || signatur != "JF" {
		t.Errorf("ohne LMF: titel=%q signatur=%q", titel, signatur)
	}
}

// sammleSignaturUpdates trägt Signaturen für BESTEHENDE Titel nach — aber nur,
// wenn die Datei überhaupt eine Signatur-Spalte hat und der Wert nicht leer ist
// (das Rücken-Etikett gewinnt, leer überschreibt nie).
func TestSammleSignaturUpdates(t *testing.T) {
	headerMap := map[string]int{"titel": 0, "signatur": 1, "barcode": 2}
	rows := [][]string{
		{"Titel", "Signatur", "Barcode"},
		{"Faust", "De GOE", "1001"},
		{"Unbekanntes Buch", "Xy 1", "1002"}, // nicht im Bestand → kein Update
		{"Faust", "", "1003"},                // leer → darf "De GOE" nicht verdrängen
	}
	titelToID := map[string]string{"Faust": "id-faust"}

	updates := sammleSignaturUpdates(rows, headerMap, map[string]string{}, titelToID)
	if len(updates) != 1 || updates["id-faust"] != "De GOE" {
		t.Errorf("updates = %v, want {id-faust: De GOE}", updates)
	}

	// Ohne Signatur-Spalte: gar keine Updates (nil)
	if u := sammleSignaturUpdates(rows, map[string]int{"titel": 0}, map[string]string{}, titelToID); u != nil {
		t.Errorf("ohne Signatur-Spalte: updates = %v, want nil", u)
	}
}

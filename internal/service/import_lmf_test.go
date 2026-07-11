package service

import (
	"context"
	"strings"
	"testing"

	"bibliothek/repository"
)

func TestHatLMFKennung(t *testing.T) {
	tests := []struct {
		wert string
		want bool
	}{
		// Echte Littera-Werte aus dem Schulbestand
		{"LMF Bio 7", true},
		{"LMF", true},
		{"LMF/Bibliothek", true},
		{"Buch LMF Ma 6/Gri 213 Exemplare", true},
		{"lmf eng 9", true},
		// Negativfälle: LMF nur als Token, nie als Teilwort
		{"Filmfest", false},
		{"Elmshorn", false},
		{"BIOLMF", false},
		{"", false},
		{"Buch", false},
		{"8 G (G9) / Sèrie verte Stuttgart: Klett 1998", false},
	}

	for _, tt := range tests {
		if got := hatLMFKennung(tt.wert); got != tt.want {
			t.Errorf("hatLMFKennung(%q) = %v, want %v", tt.wert, got, tt.want)
		}
	}
}

func TestEntferneLMFToken(t *testing.T) {
	tests := []struct {
		wert string
		want string
	}{
		{"LMF Bio 7", "Bio 7"},
		{"LMF Ma 6/Gri", "Ma 6/Gri"},
		{"Buch LMF Ma 6/Gri 213 Exemplare", "Buch Ma 6/Gri 213 Exemplare"},
		{"LMF/Bibliothek", "Bibliothek"},
		{"LMF", ""},
		// Ohne Token bleibt der Wert (bis auf Whitespace-Normalisierung) unverändert
		{"Bio 7", "Bio 7"},
		{"Filmfest", "Filmfest"},
	}

	for _, tt := range tests {
		if got := entferneLMFToken(tt.wert); got != tt.want {
			t.Errorf("entferneLMFToken(%q) = %q, want %q", tt.wert, got, tt.want)
		}
	}
}

func TestFlaggeAlsSchulbuch(t *testing.T) {
	tests := []struct {
		titel string
		want  string
	}{
		{"Biologie heute SII", "LMF-Biologie heute SII"},
		// Bereits geflaggte Titel werden nicht doppelt markiert
		{"LMF-Mathe 9", "LMF-Mathe 9"},
		{"lmf-mathe 9", "lmf-mathe 9"},
		// Leerzeichen-Schreibweise wird auf die Bindestrich-Konvention vereinheitlicht
		{"LMF Deutschbuch 8", "LMF-Deutschbuch 8"},
	}

	for _, tt := range tests {
		if got := flaggeAlsSchulbuch(tt.titel); got != tt.want {
			t.Errorf("flaggeAlsSchulbuch(%q) = %q, want %q", tt.titel, got, tt.want)
		}
	}
}

// stubBookRepo zeichnet Upserts auf; alle übrigen Interface-Methoden
// stammen aus dem eingebetteten Nil-Interface und dürfen nicht aufgerufen werden.
type stubBookRepo struct {
	repository.BookRepository
	titles []repository.BookTitle
}

func (s *stubBookRepo) BulkUpsertBookTitles(_ context.Context, titles []repository.BookTitle) (int, error) {
	s.titles = append(s.titles, titles...)
	return len(titles), nil
}

func TestParseLitteraXML_LMFUndSignatur(t *testing.T) {
	// Drei Katalogisate wie im echten Littera-4.5-Export:
	// 1. LMF-Kennung im Standort-Feld 108a, Signatur ohne Präfix
	// 2. LMF-Kennung als Signatur-Präfix im Feld 700
	// 3. Regulärer Bibliotheksbestand ohne LMF
	xmlDaten := `<?xml version="1.0"?>
<Katalogisate>
  <Katalogisat>
    <Feld MAB="108a">LMF</Feld>
    <Feld MAB="310 ">Physik Oberstufe</Feld>
    <Feld MAB="540 ">978-3-464-03440-8</Feld>
    <Feld MAB="700 " Reihung="1">Uc</Feld>
  </Katalogisat>
  <Katalogisat>
    <Feld MAB="310 ">Biologie heute 7</Feld>
    <Feld MAB="540 ">978-3-507-87301-1</Feld>
    <Feld MAB="700 " Reihung="1">LMF Bio 7</Feld>
  </Katalogisat>
  <Katalogisat>
    <Feld MAB="100 ">Rowling, Joanne K.</Feld>
    <Feld MAB="310 ">Harry Potter und der Stein der Weisen</Feld>
    <Feld MAB="540 ">978-3-551-35401-3</Feld>
    <Feld MAB="700 " Reihung="1">JF</Feld>
  </Katalogisat>
</Katalogisate>`

	repo := &stubBookRepo{}
	svc := NewImportService(repo, nil)

	count, err := svc.ParseLitteraXML(context.Background(), strings.NewReader(xmlDaten))
	if err != nil {
		t.Fatalf("ParseLitteraXML: %v", err)
	}
	if count != 3 {
		t.Fatalf("importierte Titel = %d, want 3", count)
	}

	want := []struct {
		titel    string
		signatur string
	}{
		{"LMF-Physik Oberstufe", "Uc"},
		{"LMF-Biologie heute 7", "Bio 7"},
		{"Harry Potter und der Stein der Weisen", "JF"},
	}
	for i, w := range want {
		if repo.titles[i].Titel != w.titel {
			t.Errorf("Titel[%d] = %q, want %q", i, repo.titles[i].Titel, w.titel)
		}
		if repo.titles[i].Signatur != w.signatur {
			t.Errorf("Signatur[%d] = %q, want %q", i, repo.titles[i].Signatur, w.signatur)
		}
	}
}

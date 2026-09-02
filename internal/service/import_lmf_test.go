package service

import (
	"context"
	"strings"
	"testing"

	"bibliothek/pkg/lmf"
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

func TestZerlegeLMFTeil(t *testing.T) {
	z, ok := zerlegeLMFTeil("Buch LMF Ma 6/Gri")
	if !ok || z.Fach != lmf.FachMathematik || z.JahrgangVon != 6 || z.JahrgangBis != 6 {
		t.Errorf("Buch LMF Ma 6/Gri → %+v ok=%v", z, ok)
	}
	if _, ok := zerlegeLMFTeil("Buch"); ok {
		t.Error("ohne LMF-Token darf nichts zerlegt werden")
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

// Der Katalogisat-Import lässt Titel und Signatur, wie Littera sie liefert, und setzt
// das Feld (Migration 093). Bis dahin schnitt er „LMF" aus der Signatur und stellte
// es dem Titel voran — „LMF-Biologie heute 7" auf jeder Cover-Kachel.
func TestParseLitteraXML_LMFUndSignatur(t *testing.T) {
	// Vier Katalogisate wie im echten Littera-4.5-Export:
	// 1. LMF-Kennung im Standort-Feld 108a, Signatur ist eine Regaladresse; Fach aus
	//    den Schlagwörtern, Jahrgang aus der Zielgruppe
	// 2. LMF-Kennung als Signatur-Präfix im Feld 700 → Fach und Jahrgang daraus
	// 3. Regulärer Bibliotheksbestand ohne LMF, mit Fach-Schlagwort
	// 4. Bibliotheksbestand mit zwei Fach-Schlagwörtern → Fach bleibt offen
	xmlDaten := `<?xml version="1.0"?>
<Katalogisate>
  <Katalogisat>
    <Feld MAB="108a">LMF</Feld>
    <Feld MAB="310 ">Physik Oberstufe</Feld>
    <Feld MAB="540 ">978-3-464-03440-8</Feld>
    <Feld MAB="700 " Reihung="1">Uc</Feld>
    <Feld MAB="710 ">Schulbuch</Feld>
    <Feld MAB="710 ">Naturwissenschaften</Feld>
    <Feld MAB="710 ">Physik</Feld>
    <Feld MAB="070b">Sekundarstufe 2</Feld>
  </Katalogisat>
  <Katalogisat>
    <Feld MAB="310 ">Biologie heute 7</Feld>
    <Feld MAB="540 ">978-3-507-87301-1</Feld>
    <Feld MAB="700 " Reihung="1">LMF Bio 7</Feld>
  </Katalogisat>
  <Katalogisat>
    <Feld MAB="100 ">Neebe, Reinhard</Feld>
    <Feld MAB="310 ">¬Die¬ Republik von Weimar 1918-1933</Feld>
    <Feld MAB="540 ">978-3-12-490250-4</Feld>
    <Feld MAB="700 " Reihung="1">Ev</Feld>
    <Feld MAB="710 ">Weimarer Republik</Feld>
    <Feld MAB="710 ">Geschichte</Feld>
    <Feld MAB="070b">Lehrer</Feld>
  </Katalogisat>
  <Katalogisat>
    <Feld MAB="310 ">Bilingual unterrichten</Feld>
    <Feld MAB="540 ">978-3-551-35401-3</Feld>
    <Feld MAB="700 " Reihung="1">Pg</Feld>
    <Feld MAB="710 ">Deutsch</Feld>
    <Feld MAB="710 ">Englisch</Feld>
  </Katalogisat>
</Katalogisate>`

	repo := &stubBookRepo{}
	svc := NewImportService(repo, nil)

	count, err := svc.ParseLitteraXML(context.Background(), strings.NewReader(xmlDaten))
	if err != nil {
		t.Fatalf("ParseLitteraXML: %v", err)
	}
	if count != 4 {
		t.Fatalf("importierte Titel = %d, want 4", count)
	}

	want := []repository.BookTitle{
		{Titel: "Physik Oberstufe", Signatur: "Uc", IstLernmittel: true, Fach: lmf.FachPhysik, JahrgangVon: 11, JahrgangBis: 13},
		{Titel: "Biologie heute 7", Signatur: "LMF Bio 7", IstLernmittel: true, Fach: lmf.FachBiologie, JahrgangVon: 7, JahrgangBis: 7},
		{Titel: "Die Republik von Weimar 1918-1933", Signatur: "Ev", IstLernmittel: false, Fach: lmf.FachGeschichte},
		{Titel: "Bilingual unterrichten", Signatur: "Pg", IstLernmittel: false, Fach: ""},
	}
	for i, w := range want {
		g := repo.titles[i]
		if g.Titel != w.Titel || g.Signatur != w.Signatur || g.IstLernmittel != w.IstLernmittel ||
			g.Fach != w.Fach || g.JahrgangVon != w.JahrgangVon || g.JahrgangBis != w.JahrgangBis {
			t.Errorf("Titel[%d] = {%q %q lernmittel=%v fach=%q %d–%d}, want {%q %q lernmittel=%v fach=%q %d–%d}",
				i, g.Titel, g.Signatur, g.IstLernmittel, g.Fach, g.JahrgangVon, g.JahrgangBis,
				w.Titel, w.Signatur, w.IstLernmittel, w.Fach, w.JahrgangVon, w.JahrgangBis)
		}
	}
}

// Ein Bibliotheksbuch bekommt sein Fach nur aus den Schlagwörtern, nie aus dem
// Titeltext — „Sport und Spiel im Kindergarten" ist kein Sportbuch der Schule.
func TestKategorisiere_TitelheuristikNurBeiLernmitteln(t *testing.T) {
	f := litteraFelder{titel: "Chemie der Gefühle", signatur: "Pg"}
	if fach, _, _ := kategorisiere(f, false); fach != "" {
		t.Errorf("Bibliotheksbuch: Fach aus dem Titel geraten (%q)", fach)
	}
	if fach, _, _ := kategorisiere(f, true); fach != lmf.FachChemie {
		t.Errorf("Lernmittel: Titel-Heuristik erwartet %q, war %q", lmf.FachChemie, fach)
	}
}

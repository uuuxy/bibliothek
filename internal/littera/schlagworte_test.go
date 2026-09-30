package littera

import (
	"slices"
	"strings"
	"testing"
)

// Die Schlagworte aus Littera (docs/OFFEN.md 4.20, Stufe 2): Lesen und die Herkunft des Fachs,
// ohne Datenbank.

const (
	schlagworteCSV = `Buchungsdatum,Buchungsnummer,SuchWort,Flags,Zugriff_art,Zugriff_von,Zugriff_terminal
"01/08/09 1",11,"Weimarer Republik",0,ANLAGE,,
"01/08/09 1",12,"Geschichte",0,ANLAGE,,
"01/08/09 1",13,"",0,ANLAGE,,
`
	schlagZuordCSV = `Buchungsdatum,Buchungsnummer,Titel,Schlagwort,Sortierung,Flags,Zugriff_art,Zugriff_von,Zugriff_terminal
"01/08/09 1",102,7,11,0,1,ANLAGE,,
"01/08/09 1",101,7,12,0,1,ANLAGE,,
"01/08/09 1",103,7,13,0,1,ANLAGE,,
"01/08/09 1",104,8,99,0,1,ANLAGE,,
"01/08/09 1",105,9,11,2,1,ANLAGE,,
"01/08/09 1",100,9,12,1,1,ANLAGE,,
`
)

func TestSchlagworteJeTitel_ReihenfolgeUndLuecken(t *testing.T) {
	woerter, err := LeseSchlagworte(strings.NewReader(schlagworteCSV))
	if err != nil {
		t.Fatal(err)
	}
	q, err := SchlagworteJeTitel(woerter, strings.NewReader(schlagZuordCSV))
	if err != nil {
		t.Fatal(err)
	}
	// Titel 7: Erfassungsreihenfolge (Buchungsnummer der Zuordnung); das leere Wort kommt mit,
	// weglassen ist Sache des Schreibpfads. Titel 9: Sortierung vor Erfassung.
	if got, will := q.JeTitel["7"], []string{"Geschichte", "Weimarer Republik", ""}; !slices.Equal(got, will) {
		t.Errorf("Titel 7: %q, erwartet %q", got, will)
	}
	if got, will := q.JeTitel["9"], []string{"Geschichte", "Weimarer Republik"}; !slices.Equal(got, will) {
		t.Errorf("Titel 9: %q, erwartet %q", got, will)
	}
	if _, da := q.JeTitel["8"]; da || q.OhneWort != 1 || q.Zuordnungen != 6 {
		t.Errorf("Zuordnung auf ein unbekanntes Wort: JeTitel[8] da=%v, OhneWort=%d, Zuordnungen=%d — "+
			"erwartet nicht da, 1 und 6", da, q.OhneWort, q.Zuordnungen)
	}
}

// Heißt eine Spalte anders (etwa in einer anders ausgegebenen Sicherung), bricht das Lesen ab.
// Still leer gelesen, fiele jedes Wort als leer weg, und der Lauf meldete Erfolg.
func TestSchlagworte_FehlendeSpalteBrichtAb(t *testing.T) {
	if _, err := LeseSchlagworte(strings.NewReader("Buchungsnummer,Suchwort\n11,Weimar\n")); err == nil ||
		!strings.Contains(err.Error(), "SuchWort") {
		t.Errorf("Spalte Suchwort statt SuchWort: Fehler %v, erwartet einer, der SuchWort nennt", err)
	}
	if _, err := SchlagworteJeTitel(map[string]string{}, strings.NewReader("Buchungsnummer,Titel\n1,7\n")); err == nil {
		t.Error("Schlag_zuord ohne Spalte Schlagwort: kein Fehler")
	}
}

func TestLernmittelUndFach_SignaturVorSchlagworten(t *testing.T) {
	for _, f := range []struct {
		name, signatur      string
		schlagworte         []string
		fach                string
		ausSchlagworten, lm bool
	}{
		{"Lernmittel-Signatur nennt das Fach", "LMF Bio 7", []string{"Geschichte"}, "Biologie", false, true},
		{"sonst die Schlagworte", "Ea 1 / Xyz", []string{"Jugendbuch", "Geschichte"}, "Geschichte", true, false},
		{"zwei Fächer: lieber leer als falsch", "Ea 1 / Xyz", []string{"Geschichte", "Religion"}, "", false, false},
		{"kein Fach", "", []string{"Jugendbuch"}, "", false, false},
	} {
		lern, aus := lernmittelUndFach(f.signatur, f.schlagworte)
		if lern.Fach != f.fach || aus != f.ausSchlagworten || lern.IstLernmittel != f.lm {
			t.Errorf("%s: Fach %q aus Schlagworten %v Lernmittel %v, erwartet %q %v %v",
				f.name, lern.Fach, aus, lern.IstLernmittel, f.fach, f.ausSchlagworten, f.lm)
		}
	}
}

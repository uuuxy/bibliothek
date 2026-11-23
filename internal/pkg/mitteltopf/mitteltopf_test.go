package mitteltopf

import (
	"reflect"
	"slices"
	"strconv"
	"strings"
	"testing"
)

func TestGueltig(t *testing.T) {
	for _, m := range []string{Land, Schultraeger, Spende} {
		if !IstGueltig(m) {
			t.Errorf("%q gehört zum Vokabular", m)
		}
	}
	for _, m := range []string{"", "kreis", "Land", "LAND", "schultraeger "} {
		if IstGueltig(m) {
			t.Errorf("%q darf nicht gültig sein — die Tür muss 400 sagen", m)
		}
	}
}

// Jeder Topf trägt jeden Text, und keiner gleicht dem des anderen Topfs: Ein leerer Vermerk
// ergäbe ein Anschreiben ohne Vermerk, zwei gleiche Wörter zwei Töpfe, die niemand
// unterscheidet. Über die Felder des Typs, damit ein später ergänztes mitgeprüft wird.
func TestTexteFuer_JederTopfHatEigeneTexte(t *testing.T) {
	toepfe := []string{Land, Schultraeger, Spende}
	werte := make([]reflect.Value, len(toepfe))
	for i, m := range toepfe {
		texte, err := TexteFuer(m)
		if err != nil {
			t.Fatalf("%s: %v", m, err)
		}
		werte[i] = reflect.ValueOf(texte)
	}
	if werte[0].NumField() < 4 {
		t.Fatalf("nur %d Felder am Typ Texte gefunden, der Test misst offenbar nichts mehr", werte[0].NumField())
	}
	for i := range werte[0].NumField() {
		feld := werte[0].Type().Field(i).Name
		for a := range toepfe {
			if werte[a].Field(i).String() == "" {
				t.Errorf("%s ist für %s leer", feld, toepfe[a])
			}
			for b := a + 1; b < len(toepfe); b++ {
				if werte[a].Field(i).String() == werte[b].Field(i).String() {
					t.Errorf("%s ist für %s und %s gleich: %q", feld, toepfe[a], toepfe[b], werte[a].Field(i).String())
				}
			}
		}
	}
}

// Ein Wert außerhalb des Vokabulars ergibt einen Fehler, der ihn nennt, und keine Texte.
func TestTexteFuer_UnbekannterTopfIstEinFehler(t *testing.T) {
	for _, m := range []string{"", "kreis", "Land"} {
		texte, err := TexteFuer(m)
		if err == nil {
			t.Errorf("%q: kein Fehler, aber Texte %+v", m, texte)
			continue
		}
		if texte != (Texte{}) {
			t.Errorf("%q: neben dem Fehler stehen Texte %+v", m, texte)
		}
		if !strings.Contains(err.Error(), strconv.Quote(m)) {
			t.Errorf("%q: der Fehler nennt den Wert nicht: %v", m, err)
		}
	}
}

func TestTraeger(t *testing.T) {
	faelle := map[string]string{Land: "Land", Schultraeger: "Schulträger", Spende: "ohne Topf", "": "", "kreis": ""}
	for mittel, will := range faelle {
		if ist := Traeger(mittel); ist != will {
			t.Errorf("Traeger(%q) = %q, erwartet %q", mittel, ist, will)
		}
	}
}

func TestBeschriftung(t *testing.T) {
	faelle := map[string]string{
		Land:         "Lernmittelfreiheit · Land",
		Schultraeger: "Schülerbücherei · Schulträger",
		Spende:       "Spende · ohne Topf",
		"":           "ohne Zuordnung",
		"kreis":      "",
	}
	for mittel, will := range faelle {
		if ist := Beschriftung(mittel); ist != will {
			t.Errorf("Beschriftung(%q) = %q, erwartet %q", mittel, ist, will)
		}
	}
	if Beschriftung("") != OhneZuordnung {
		t.Errorf("der leere Wert heißt %q, die Konstante %q", Beschriftung(""), OhneZuordnung)
	}
}

// Jeder Topf des Vokabulars steht einmal in der Reihenfolge, der leere Wert zuletzt. Jeder
// Aufruf bekommt eine eigene Liste: Sonst verstellte ein Aufrufer die Reihenfolge aller Berichte.
func TestReihenfolge(t *testing.T) {
	r := Reihenfolge()
	if will := []string{Land, Schultraeger, Spende, ""}; !slices.Equal(r, will) {
		t.Fatalf("Reihenfolge %q, erwartet %q", r, will)
	}
	r[0] = "verstellt"
	if erster := Reihenfolge()[0]; erster != Land {
		t.Errorf("nach einer Änderung an der gelieferten Liste beginnt die Reihenfolge mit %q", erster)
	}
}

func TestHatGrossesLernmittelEtikett(t *testing.T) {
	faelle := map[string]bool{Land: true, Schultraeger: false, Spende: false, "": true}
	for mittel, will := range faelle {
		if ist := HatGrossesLernmittelEtikett(mittel); ist != will {
			t.Errorf("HatGrossesLernmittelEtikett(%q) = %v, erwartet %v", mittel, ist, will)
		}
	}
}

// Die Spende gilt beim Händler, auf dem Etikett und beim Ersatz wie die Schülerbücherei; das Land
// und eine Bestellung ohne Zuordnung nicht.
func TestWieSchuelerbuecherei(t *testing.T) {
	faelle := map[string]bool{Land: false, Schultraeger: true, Spende: true, "": false, "kreis": false}
	for mittel, will := range faelle {
		if ist := WieSchuelerbuecherei(mittel); ist != will {
			t.Errorf("WieSchuelerbuecherei(%q) = %v, erwartet %v", mittel, ist, will)
		}
	}
}

// Die Ablehnung nennt jeden Wert des Vokabulars mit seinem Wort, im Fehlertext wie in der
// Meldung der Türen: Wer sie liest, sieht, was erlaubt ist.
func TestErrUngueltig_NenntJedenWertMitSeinemWort(t *testing.T) {
	for _, mittel := range []string{Land, Schultraeger, Spende} {
		texte, err := TexteFuer(mittel)
		if err != nil {
			t.Fatalf("%s: %v", mittel, err)
		}
		if will := "'" + mittel + "' (" + texte.Kurz + ")"; !strings.Contains(ErrUngueltig.Error(), will) {
			t.Errorf("der Fehlertext %q nennt %s nicht", ErrUngueltig.Error(), will)
		}
		if will := "„" + mittel + "“ für die " + texte.Kurz; !strings.Contains(MeldungUngueltig, will) {
			t.Errorf("die Meldung %q nennt %s nicht", MeldungUngueltig, will)
		}
	}
}

package mitteltopf

import (
	"reflect"
	"slices"
	"strconv"
	"strings"
	"testing"
)

func TestGueltig(t *testing.T) {
	for _, m := range []string{Land, Schultraeger} {
		if !Gueltig(m) {
			t.Errorf("%q gehört zum Vokabular", m)
		}
	}
	for _, m := range []string{"", "kreis", "Land", "LAND", "schultraeger "} {
		if Gueltig(m) {
			t.Errorf("%q darf nicht gültig sein — die Tür muss 400 sagen", m)
		}
	}
}

// Jeder Topf trägt jeden Text, und keiner gleicht dem des anderen Topfs: Ein leerer Vermerk
// ergäbe ein Anschreiben ohne Vermerk, zwei gleiche Wörter zwei Töpfe, die niemand
// unterscheidet. Über die Felder des Typs, damit ein später ergänztes mitgeprüft wird.
func TestTexteFuer_JederTopfHatEigeneTexte(t *testing.T) {
	land, err := TexteFuer(Land)
	if err != nil {
		t.Fatalf("%s: %v", Land, err)
	}
	traeger, err := TexteFuer(Schultraeger)
	if err != nil {
		t.Fatalf("%s: %v", Schultraeger, err)
	}
	l, s := reflect.ValueOf(land), reflect.ValueOf(traeger)
	if l.NumField() < 4 {
		t.Fatalf("nur %d Felder am Typ Texte gefunden, der Test misst offenbar nichts mehr", l.NumField())
	}
	for i := range l.NumField() {
		feld := l.Type().Field(i).Name
		if l.Field(i).String() == "" || s.Field(i).String() == "" {
			t.Errorf("%s ist für einen Topf leer: %q und %q", feld, l.Field(i).String(), s.Field(i).String())
		}
		if l.Field(i).String() == s.Field(i).String() {
			t.Errorf("%s ist für beide Töpfe gleich: %q", feld, l.Field(i).String())
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
	faelle := map[string]string{Land: "Land", Schultraeger: "Schulträger", "": "", "kreis": ""}
	for mittel, will := range faelle {
		if ist := Traeger(mittel); ist != will {
			t.Errorf("Traeger(%q) = %q, erwartet %q", mittel, ist, will)
		}
	}
}

func TestBeschriftung(t *testing.T) {
	faelle := map[string]string{
		Land:         "Lernmittelfreiheit (Land)",
		Schultraeger: "Schülerbücherei (Schulträger)",
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
	if will := []string{Land, Schultraeger, ""}; !slices.Equal(r, will) {
		t.Fatalf("Reihenfolge %q, erwartet %q", r, will)
	}
	r[0] = "verstellt"
	if erster := Reihenfolge()[0]; erster != Land {
		t.Errorf("nach einer Änderung an der gelieferten Liste beginnt die Reihenfolge mit %q", erster)
	}
}

func TestGrossesLernmittelEtikettFuer(t *testing.T) {
	faelle := map[string]bool{Land: true, Schultraeger: false, "": true}
	for mittel, will := range faelle {
		if ist := GrossesLernmittelEtikettFuer(mittel); ist != will {
			t.Errorf("GrossesLernmittelEtikettFuer(%q) = %v, erwartet %v", mittel, ist, will)
		}
	}
}

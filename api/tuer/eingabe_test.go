package tuer

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// pruefPosition steht in einer Liste der Anfrage.
type pruefPosition struct {
	TitelID string `json:"titel_id" validate:"required"`
}

// pruefAnfrage trägt je Regel, die die Türen nutzen, ein Feld.
type pruefAnfrage struct {
	Name       string          `json:"name" validate:"required"`
	TitelID    string          `json:"titel_id" validate:"omitempty,uuid_oder_leer"`
	Email      string          `json:"email" validate:"omitempty,email"`
	Absicht    string          `json:"absicht" validate:"omitempty,oneof=ausleihe rueckgabe"`
	Eintraege  []string        `json:"eintraege" validate:"omitempty,min=2,max=3"`
	Prozent    *int            `json:"prozent" validate:"omitempty,min=0,max=100"`
	Positionen []pruefPosition `json:"items" validate:"dive"`
}

// Jede Regel bekommt einen deutschen Satz, der das Feld mit seinem Namen in der Anfrage nennt.
func TestPruefmeldung_SatzJeRegel(t *testing.T) {
	gueltig := func() pruefAnfrage { return pruefAnfrage{Name: "x"} }
	zuViel := 101
	faelle := []struct {
		name    string
		anfrage func() pruefAnfrage
		meldung string
	}{
		{"required", func() pruefAnfrage { return pruefAnfrage{} }, "Bitte das Feld „name“ ausfüllen"},
		{"Kennung", func() pruefAnfrage { a := gueltig(); a.TitelID = "x"; return a }, "Ungültige Kennung im Feld „titel_id“"},
		{"email", func() pruefAnfrage { a := gueltig(); a.Email = "peter@"; return a }, "„peter@“ ist keine E-Mail-Adresse"},
		{"oneof", func() pruefAnfrage { a := gueltig(); a.Absicht = "x"; return a },
			"Im Feld „absicht“ ist nur einer dieser Werte möglich: ausleihe, rueckgabe"},
		{"min an einer Liste", func() pruefAnfrage { a := gueltig(); a.Eintraege = []string{"a"}; return a },
			"Das Feld „eintraege“ braucht mindestens 2 Einträge"},
		{"max an einer Liste", func() pruefAnfrage { a := gueltig(); a.Eintraege = []string{"a", "b", "c", "d"}; return a },
			"Das Feld „eintraege“ hat höchstens 3 Einträge"},
		{"max an einer Zahl", func() pruefAnfrage { a := gueltig(); a.Prozent = &zuViel; return a },
			"Das Feld „prozent“ darf höchstens 100 sein"},
		{"Feld in einer Liste", func() pruefAnfrage {
			a := gueltig()
			a.Positionen = []pruefPosition{{TitelID: "a"}, {}}
			return a
		}, "Bitte das Feld „items[1].titel_id“ ausfüllen"},
	}
	for _, f := range faelle {
		t.Run(f.name, func(t *testing.T) {
			err := Pruefer.Struct(f.anfrage())
			if err == nil {
				t.Fatal("Pruefer.Struct: kein Fehler, erwartet einer")
			}
			if got := Pruefmeldung(err); got != f.meldung {
				t.Errorf("Pruefmeldung = %q, erwartet %q", got, f.meldung)
			}
		})
	}
}

// Ein Rumpf, der kein passendes JSON ist, und ein abgelehntes Feld antworten mit 400 und einem
// Satz; der Text aus encoding/json geht nicht hinaus.
func TestLeseAnfrage_MeldungStattFehlertext(t *testing.T) {
	faelle := []struct {
		name    string
		rumpf   string
		streng  bool
		meldung string
	}{
		{"kein JSON", `{"name":`, false, MeldungAnfrageUnlesbar},
		{"falscher Typ", `{"name":1}`, false, MeldungAnfrageUnlesbar},
		{"unbekanntes Feld, streng", `{"name":"x","gibtsnicht":1}`, true, MeldungAnfrageUnlesbar},
		{"Feld fehlt", `{}`, false, "Bitte das Feld „name“ ausfüllen"},
	}
	for _, f := range faelle {
		t.Run(f.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPost, "/x", strings.NewReader(f.rumpf))
			var ziel pruefAnfrage
			var ok bool
			if f.streng {
				ok = LeseAnfrageStreng(rec, req, &ziel)
			} else {
				ok = LeseAnfrage(rec, req, &ziel)
			}
			if ok || rec.Code != http.StatusBadRequest {
				t.Fatalf("ok = %v, Status %d, erwartet false und 400", ok, rec.Code)
			}
			var antwort map[string]string
			if err := json.Unmarshal(rec.Body.Bytes(), &antwort); err != nil {
				t.Fatalf("json.Unmarshal(%q): %v", rec.Body.String(), err)
			}
			if antwort["error"] != f.meldung {
				t.Errorf("Meldung = %q, erwartet %q", antwort["error"], f.meldung)
			}
		})
	}
}

// Eine Kennung im Parameter, die keine UUID ist, wird ein Bedienfehler mit Meldung.
func TestLeseUUIDParameter_KeineKennung(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/x?titel_id=abc", nil)
	if _, err := LeseUUIDParameter(req, "titel_id"); err == nil || err.Error() != "In der Adresse ist „titel_id“ keine gültige Kennung" {
		t.Errorf("LeseUUIDParameter: Fehler %v, erwartet die Meldung zur Kennung", err)
	}
	req = httptest.NewRequest(http.MethodGet, "/x", nil)
	if wert, err := LeseUUIDParameter(req, "titel_id"); err != nil || wert != "" {
		t.Errorf("LeseUUIDParameter ohne Parameter = %q, %v, erwartet leer ohne Fehler", wert, err)
	}
}

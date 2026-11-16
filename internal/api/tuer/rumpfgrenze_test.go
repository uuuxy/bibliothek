package tuer

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// zaehlLeser reicht einen Rumpf durch und zählt, wie viel davon gelesen wurde.
type zaehlLeser struct {
	quelle  io.Reader
	gelesen int
}

func (z *zaehlLeser) Read(p []byte) (int, error) {
	n, err := z.quelle.Read(p)
	z.gelesen += n
	return n, err
}

// rumpfMitName ist {"name":"AAA…"} mit einem Namen der genannten Länge.
func rumpfMitName(laenge int) *zaehlLeser {
	return &zaehlLeser{quelle: io.MultiReader(strings.NewReader(`{"name":"`),
		strings.NewReader(strings.Repeat("A", laenge)), strings.NewReader(`"}`))}
}

// Ein Rumpf über der Grenze wird nicht zu Ende gelesen, und die Antwort nennt die Grenze. Ohne
// sie hielt der Decoder jeden Rumpf bis zur Grenze aller Anfragen (100 MB) im Speicher.
func TestLeseAnfrage_HoertAnDerGrenzeAuf(t *testing.T) {
	for _, streng := range []bool{false, true} {
		rumpf := rumpfMitName(2 * maxRumpfJSON)
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/x", rumpf)
		var ziel pruefAnfrage
		var ok bool
		if streng {
			ok = LeseAnfrageStreng(rec, req, &ziel)
		} else {
			ok = LeseAnfrage(rec, req, &ziel)
		}
		if ok || rec.Code != http.StatusRequestEntityTooLarge {
			t.Fatalf("streng %v: ok = %v, Status %d, erwartet false und 413", streng, ok, rec.Code)
		}
		if grenze := maxRumpfJSON + 64<<10; rumpf.gelesen > grenze {
			t.Errorf("streng %v: %d Bytes gelesen, erwartet höchstens %d", streng, rumpf.gelesen, grenze)
		}
		var antwort map[string]string
		if err := json.Unmarshal(rec.Body.Bytes(), &antwort); err != nil {
			t.Fatalf("json.Unmarshal(%q): %v", rec.Body.String(), err)
		}
		if antwort["error"] != meldungAnfrageZuGross() {
			t.Errorf("streng %v: Meldung = %q, erwartet %q", streng, antwort["error"], meldungAnfrageZuGross())
		}
	}
}

// Ein Ausweisfoto als Base64 ist gut 1 MB groß; ein Rumpf von 2 MB geht durch.
func TestLeseAnfrage_GrosserRumpfUnterDerGrenze(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/x", rumpfMitName(2<<20))
	var ziel pruefAnfrage
	if !LeseAnfrage(rec, req, &ziel) || len(ziel.Name) != 2<<20 {
		t.Errorf("LeseAnfrage = Status %d, Name mit %d Zeichen, erwartet %d", rec.Code, len(ziel.Name), 2<<20)
	}
}

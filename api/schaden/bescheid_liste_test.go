package schaden

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"bibliothek/repository"
)

// bescheidListeAttrappe beantwortet LadeListe und merkt sich, ob nach den offenen Bescheiden
// gefragt wurde.
type bescheidListeAttrappe struct {
	nurOffen *bool
	fehler   error
}

func (a bescheidListeAttrappe) LadeListe(_ context.Context, nurOffen bool) ([]repository.Bescheid, error) {
	*a.nurOffen = nurOffen
	if a.fehler != nil {
		return nil, a.fehler
	}
	return []repository.Bescheid{{ID: "b1"}}, nil
}

// Die Liste der Bescheide nimmt status=offen als „nur nicht übergebene", so beschreibt es die
// Tür; ein anderer Wert oder keiner liefert alle. Die PG-Tests rufen LadeListe direkt und
// sehen diese Zuordnung nicht. Scheitert die Abfrage, antwortet die Tür mit 500 und nicht mit
// einer leeren Liste, die wie „keine Bescheide" aussähe.
func TestBescheide_StatusOffenUndFehler(t *testing.T) {
	faelle := []struct {
		ziel     string
		fehler   error
		nurOffen bool
		status   int
	}{
		{ziel: "/api/bescheide?status=offen", nurOffen: true, status: http.StatusOK},
		{ziel: "/api/bescheide", nurOffen: false, status: http.StatusOK},
		{ziel: "/api/bescheide?status=alle", nurOffen: false, status: http.StatusOK},
		{ziel: "/api/bescheide", fehler: errors.New("verbindung abgebrochen"), status: http.StatusInternalServerError},
	}

	for _, f := range faelle {
		var nurOffen bool
		rec := httptest.NewRecorder()
		(&Tueren{}).Bescheide(bescheidListeAttrappe{nurOffen: &nurOffen, fehler: f.fehler})(
			rec, httptest.NewRequest(http.MethodGet, f.ziel, nil))

		if rec.Code != f.status {
			t.Errorf("GET %s = %d, erwartet %d", f.ziel, rec.Code, f.status)
			continue
		}
		if nurOffen != f.nurOffen {
			t.Errorf("GET %s fragt LadeListe(nurOffen = %v), erwartet %v", f.ziel, nurOffen, f.nurOffen)
		}
		if f.status != http.StatusOK {
			continue
		}
		var liste []struct {
			ID string `json:"id"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &liste); err != nil {
			t.Fatalf("GET %s: Antwort unlesbar: %v", f.ziel, err)
		}
		if len(liste) != 1 || liste[0].ID != "b1" {
			t.Errorf("GET %s liefert %s, erwartet den einen Bescheid b1", f.ziel, rec.Body.String())
		}
	}
}

package system

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"bibliothek/api/tuer"
)

// Der Lernmittel-Stichtag trägt jede Lernmittel-Frist und den Rückweg des LMF-Plans.
// Prüfte ihn der Server nicht, würde „30.06." gespeichert und angezeigt, gerechnet aber still
// mit 07-31; „06-31" würde zum 1. Juli, „02-30" zum 2. März. Die Nachbarn (Sommerferien,
// Eingangsjahrgänge) prüft der Server ebenso.
func TestEinstellungenSchreiben_LMFStichtagIstEinKalendertag(t *testing.T) {
	faelle := []struct {
		wert    string
		erwarte int
	}{
		{"07-31", http.StatusOK},
		{"08-15", http.StatusOK},
		{"", http.StatusOK}, // leer = Vorgabe (repository/einstellungen_patch.go)
		{"06-31", http.StatusBadRequest},
		{"02-30", http.StatusBadRequest},
		{"13-01", http.StatusBadRequest},
		{"30.06.", http.StatusBadRequest},
		{"Juli", http.StatusBadRequest},
	}
	for _, f := range faelle {
		t.Run(f.wert, func(t *testing.T) {
			repo := &einstellungsAttrappe{}
			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodPut, "/api/einstellungen",
				strings.NewReader(`{"lmf_stichtag": "`+f.wert+`"}`))
			(&Tueren{Basis: &tuer.Basis{}}).EinstellungenSchreiben(repo).ServeHTTP(w, r)
			if w.Code != f.erwarte {
				t.Errorf("Stichtag %q: Status %d, want %d — %s", f.wert, w.Code, f.erwarte, w.Body.String())
			}
			if f.erwarte == http.StatusBadRequest && repo.gespeichert {
				t.Errorf("Stichtag %q abgelehnt, aber gespeichert", f.wert)
			}
		})
	}
}

package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// Der Lernmittel-Stichtag trägt jede Lernmittel-Frist und den Rückweg des LMF-Plans.
// Bis zum 10.09.2026 prüfte ihn der Server nicht: „30.06." wurde gespeichert und
// angezeigt, gerechnet wurde still mit 07-31; „06-31" wurde zum 1. Juli, „02-30" zum
// 2. März. Die beiden Nachbarn (Sommerferien, Eingangsjahrgänge) wurden am 06.09. genau
// dafür geprüft — der Stichtag nicht, obwohl ein Kommentar das behauptete
// (Bestands-Durchgang, Bugklasse „Schutz, den nur ein Kommentar behauptet").
func TestUpdateSettings_LmfStichtagIstEinKalendertag(t *testing.T) {
	faelle := []struct {
		wert    string
		erwarte int
	}{
		{"07-31", http.StatusOK},
		{"08-15", http.StatusOK},
		{"", http.StatusOK}, // leer = Vorgabe (system_settings_patch.go)
		{"06-31", http.StatusBadRequest},
		{"02-30", http.StatusBadRequest},
		{"13-01", http.StatusBadRequest},
		{"30.06.", http.StatusBadRequest},
		{"Juli", http.StatusBadRequest},
	}
	for _, f := range faelle {
		t.Run(f.wert, func(t *testing.T) {
			repo := &attrappeSettingsRepo{}
			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodPut, "/api/einstellungen",
				strings.NewReader(`{"lmf_stichtag": "`+f.wert+`"}`))
			(&Server{}).UpdateSettingsHandler(repo).ServeHTTP(w, r)
			if w.Code != f.erwarte {
				t.Errorf("Stichtag %q: Status %d, want %d — %s", f.wert, w.Code, f.erwarte, w.Body.String())
			}
			if f.erwarte == http.StatusBadRequest && repo.gespeichert {
				t.Errorf("Stichtag %q abgelehnt, aber gespeichert", f.wert)
			}
		})
	}
}

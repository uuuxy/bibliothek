package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// Eine Zahl außerhalb ihrer Spanne wird abgewiesen, nicht still ersetzt.
//
// Gemessen am 16.09.2026 über genau diese Tür (Rasterdurchgang, OFFEN.md 5.18):
//
//   - `{"max_ausleihen_schueler": 0}` wurde als 5 GESPEICHERT. Die Antwort war 200, die
//     Oberfläche zeigte danach 5, und im Prüfprotokoll stand die 0 aus dem Request —
//     drei Stellen, drei verschiedene Wahrheiten über denselben Klick.
//   - `{"max_overdue_items": 999999}` wurde unverändert übernommen. Damit ist die
//     Sperr-Automatik aus der Oberfläche abschaltbar, ohne dass irgendwo „aus" steht.
//
// Dieselbe Entscheidung wie beim LMF-Stichtag und den Sommerferien: Unlesbares wird
// abgelehnt. Der Test steht an der Tür und nicht am Sammler, weil genau dort gemessen
// wurde — und weil „gespeichert" die Antwort des Servers ist, nicht die einer Funktion.
func TestUpdateSettings_ZahlAusserhalbDerSpanneWirdAbgelehnt(t *testing.T) {
	faelle := []struct {
		name    string
		rumpf   string
		erwarte int
	}{
		{"Ausleihen 5", `{"max_ausleihen_schueler": 5}`, http.StatusOK},
		{"Ausleihen 0", `{"max_ausleihen_schueler": 0}`, http.StatusBadRequest},
		{"Ausleihen 101", `{"max_ausleihen_schueler": 101}`, http.StatusBadRequest},
		{"Überfällige 1", `{"max_overdue_items": 1}`, http.StatusOK},
		{"Überfällige 999999", `{"max_overdue_items": 999999}`, http.StatusBadRequest},
		{"Karenz 0 ist ein Wert", `{"abgaenger_karenz_tage": 0}`, http.StatusOK},
		{"Karenz negativ", `{"abgaenger_karenz_tage": -5}`, http.StatusBadRequest},
		{"Lesehistorie 0 ist aus", `{"lesehistorie_tage": 0}`, http.StatusOK},
		{"Lesehistorie 11 Jahre", `{"lesehistorie_tage": 4000}`, http.StatusBadRequest},
		{"Prüfprotokoll unter der Grenze", `{"audit_aufbewahrung_monate": 3}`, http.StatusBadRequest},
		{"Theke leeren nach 2 Tagen", `{"theke_leeren_minuten": 2881}`, http.StatusBadRequest},
	}
	for _, f := range faelle {
		t.Run(f.name, func(t *testing.T) {
			repo := &attrappeSettingsRepo{}
			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodPut, "/api/einstellungen", strings.NewReader(f.rumpf))
			(&Server{}).UpdateSettingsHandler(repo).ServeHTTP(w, r)
			if w.Code != f.erwarte {
				t.Errorf("Status %d, want %d — %s", w.Code, f.erwarte, w.Body.String())
			}
			if f.erwarte == http.StatusBadRequest && repo.gespeichert {
				t.Error("abgelehnt, aber gespeichert — genau das war der Fund: die Antwort und die Datenbank sagten Verschiedenes")
			}
		})
	}
}

// Die Meldung nennt das Feld beim Namen, unter dem es in der Oberfläche steht, und
// sagt, dass nichts gespeichert wurde. Eine 400 ohne Feldnamen schickt den Menschen
// in ein Formular mit fünfzehn Zahlen.
func TestUpdateSettings_MeldungNenntFeldUndSpanne(t *testing.T) {
	repo := &attrappeSettingsRepo{}
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPut, "/api/einstellungen",
		strings.NewReader(`{"max_overdue_items": 999999, "frist_buch_tage": 0}`))
	(&Server{}).UpdateSettingsHandler(repo).ServeHTTP(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("Status %d, want 400", w.Code)
	}
	rumpf := w.Body.String()
	for _, teil := range []string{"Max. überfällige Titel", "Leihfrist Bücher", "Nichts gespeichert"} {
		if !strings.Contains(rumpf, teil) {
			t.Errorf("die Meldung enthält %q nicht: %s", teil, rumpf)
		}
	}
}

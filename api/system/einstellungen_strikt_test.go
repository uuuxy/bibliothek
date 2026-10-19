package system

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"bibliothek/api/tuer"
	"bibliothek/repository"
)

// Der Einstellungs-Endpunkt dekodiert streng: Ein Feld, das der Patch nicht kennt, ist
// ein Fehler (400) und kein stiller Verlust.
//
// Ein eigener Test und nicht der Paritäts-Ratsche nebenan überlassen: Sie prüft den Code von heute
// (Kategorien gegen Struct) und sagt nichts über einen anderen Absender, eine alte Oberfläche im
// Browser-Cache, ein Skript, eine zweite Oberfläche. Genau deren stiller Verlust ist die
// teuerste Fehlerklasse dieses Projekts: 200 zurück, nichts gespeichert, „erfolgreich" auf dem
// Bildschirm.
func TestEinstellungenSchreiben_UnbekanntesFeldWirdAbgelehnt(t *testing.T) {
	faelle := []struct {
		name    string
		rumpf   string
		erwarte int
	}{
		{
			name:    "bekanntes Feld",
			rumpf:   `{"lesehistorie_tage": 120}`,
			erwarte: http.StatusOK,
		},
		{
			name:    "unbekanntes Feld allein",
			rumpf:   `{"lesehistorie_tag": 120}`,
			erwarte: http.StatusBadRequest,
		},
		{
			// Der gefährliche Fall: Ein Teil kommt an, ein Teil verschwindet. Ohne
			// strenge Prüfung wäre das eine 200 mit halb gespeicherten Einstellungen.
			name:    "bekanntes und unbekanntes Feld gemischt",
			rumpf:   `{"lesehistorie_tage": 120, "smtp_host": "mail.example.org"}`,
			erwarte: http.StatusBadRequest,
		},
	}

	for _, f := range faelle {
		t.Run(f.name, func(t *testing.T) {
			repo := &einstellungsAttrappe{}
			s := &Tueren{Basis: &tuer.Basis{}}
			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodPut, "/api/einstellungen", strings.NewReader(f.rumpf))

			s.EinstellungenSchreiben(repo).ServeHTTP(w, r)

			if w.Code != f.erwarte {
				t.Errorf("Status %d, erwartet %d — Antwort: %s", w.Code, f.erwarte, w.Body.String())
			}
			if f.erwarte == http.StatusBadRequest && repo.gespeichert {
				t.Error("bei abgelehntem Rumpf darf nichts gespeichert werden")
			}
		})
	}
}

// einstellungsAttrappe merkt sich nur, ob gespeichert wurde.
type einstellungsAttrappe struct {
	gespeichert bool
}

func (a *einstellungsAttrappe) Schreibe(ctx context.Context, p *repository.EinstellungenPatch) error {
	a.gespeichert = true
	return nil
}

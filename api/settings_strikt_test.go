package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"bibliothek/repository"
)

// Der Einstellungs-Endpunkt dekodiert streng: Ein Feld, das der Patch nicht kennt, ist
// ein Fehler (400) und kein stiller Verlust.
//
// Warum das ein eigener Test ist und nicht der Paritäts-Ratsche nebenan überlassen
// bleibt: Die Ratsche prüft den Code von heute (Kategorien gegen Struct). Sie sagt
// nichts über einen ANDEREN Absender — eine alte Oberfläche im Browser-Cache, ein
// Skript, ein zweiter Client. Genau deren stiller Verlust ist die teuerste Klasse
// dieses Projekts: 200 zurück, nichts gespeichert, "erfolgreich" auf dem Bildschirm
// (23.08.2026, Commit 488f51d9).
func TestUpdateSettings_UnbekanntesFeldWirdAbgelehnt(t *testing.T) {
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
			repo := &attrappeSettingsRepo{}
			s := &Server{}
			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodPut, "/api/einstellungen", strings.NewReader(f.rumpf))

			s.UpdateSettingsHandler(repo).ServeHTTP(w, r)

			if w.Code != f.erwarte {
				t.Errorf("Status %d, erwartet %d — Antwort: %s", w.Code, f.erwarte, w.Body.String())
			}
			if f.erwarte == http.StatusBadRequest && repo.gespeichert {
				t.Error("bei abgelehntem Rumpf darf nichts gespeichert werden")
			}
		})
	}
}

// attrappeSettingsRepo merkt sich nur, ob gespeichert wurde.
type attrappeSettingsRepo struct {
	gespeichert bool
}

func (a *attrappeSettingsRepo) GetSettings(ctx context.Context) (*repository.SystemEinstellungen, error) {
	return &repository.SystemEinstellungen{}, nil
}

func (a *attrappeSettingsRepo) SaveSettings(ctx context.Context, p *repository.EinstellungenPatch) error {
	a.gespeichert = true
	return nil
}

package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"bibliothek/db"

	"github.com/pashagolub/pgxmock/v4"
)

// reorderSpalten spiegelt die Projektion aus queryReorders.
func reorderSpalten() []string {
	return []string{"id", "titel", "autor", "isbn", "verlag", "signatur",
		"erscheinungsjahr", "cover_url", "meldebestand", "verfuegbar", "gesamt"}
}

func TestQueryReorders(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("Mock-Pool konnte nicht erstellt werden: %v", err)
	}
	defer mock.Close()

	server := &Server{DB: &db.Database{Pool: mock}}

	mock.ExpectQuery("SELECT t.id, t.titel, coalesce").
		WillReturnRows(pgxmock.NewRows(reorderSpalten()).
			AddRow("1", "LMF-Mathe 7", "Verlag", "12345", "Klett", "Ma 7", 2023, "", 5, 2, 30))

	results, err := server.queryReorders(context.Background(), "")
	if err != nil {
		t.Fatalf("queryReorders: unerwarteter Fehler: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("erwartet 1 Treffer, waren %d", len(results))
	}

	got := results[0]
	if got.Titel != "LMF-Mathe 7" {
		t.Errorf("Titel: erwartet 'LMF-Mathe 7', war %q", got.Titel)
	}
	if got.VerfuegbarBestand != 2 {
		t.Errorf("VerfuegbarBestand: erwartet 2, war %d", got.VerfuegbarBestand)
	}
	// Beide Bestandszahlen müssen ankommen: 2 verfügbar bei 30 vorhandenen ist ein
	// verliehener Klassensatz, kein Bestellgrund — das kann die Ansicht nur
	// unterscheiden, wenn sie den Gesamtbestand kennt.
	if got.GesamtBestand != 30 {
		t.Errorf("GesamtBestand: erwartet 30, war %d", got.GesamtBestand)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("offene Mock-Erwartungen: %v", err)
	}
}

// TestReorderFilterDefaultIstLMF sichert die fachliche Vorauswahl ab: Nachbestellt
// werden Lernmittel; der Freihandbestand besteht überwiegend aus bewussten
// Einzelstücken (Prüf-/Leseexemplare). Ohne diesen Default enthielt die Liste
// praktisch den gesamten Katalog und war unbenutzbar.
func TestReorderFilterDefaultIstLMF(t *testing.T) {
	faelle := []struct {
		name, query, wantFragment string
	}{
		{"ohne Parameter", "", "AND LOWER(t.titel) ~ '^lmf[ -]'"},
		{"type=lmf", "?type=lmf", "AND LOWER(t.titel) ~ '^lmf[ -]'"},
		{"type=freihand", "?type=freihand", "AND NOT (LOWER(t.titel) ~ '^lmf[ -]')"},
		{"type=alle", "?type=alle", ""},
		{"unbekannter Wert", "?type=kaputt", ""},
	}

	for _, f := range faelle {
		t.Run(f.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodGet, "/api/bestellungen"+f.query, nil)
			if got := reorderFilter(r); got != f.wantFragment {
				t.Errorf("erwartet %q, war %q", f.wantFragment, got)
			}
		})
	}
}

// TestGetReordersLeereListeIstArray: wie bei der Schülerliste darf eine leere Liste
// nicht als null herausgehen — das Frontend ruft .length darauf auf.
func TestGetReordersLeereListeIstArray(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("Mock-Pool konnte nicht erstellt werden: %v", err)
	}
	defer mock.Close()

	mock.ExpectQuery("SELECT t.id, t.titel, coalesce").
		WillReturnRows(pgxmock.NewRows(reorderSpalten()))

	server := &Server{DB: &db.Database{Pool: mock}}
	rec := httptest.NewRecorder()
	server.GetReordersHandler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/bestellungen", nil))

	if body := strings.TrimSpace(rec.Body.String()); body != "[]" {
		t.Errorf("erwartet [], war: %s", body)
	}
}

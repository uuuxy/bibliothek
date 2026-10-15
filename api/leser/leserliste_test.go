package leser

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"bibliothek/api/tuer"
	"bibliothek/repository"

	"github.com/pashagolub/pgxmock/v5"
)

// TestLeserlisteLeereListeIstArray: Eine leere Leserliste kommt als leeres Array, sonst
// stürzt die Schülerdatei bei einer frischen Installation ab.
//
// Der Ablauf: frische Installation -> noch keine Schüler -> das Repository gibt eine
// nil-Slice zurück -> JSON "null" -> im Frontend "students = null" -> students.length
// wirft "Cannot read properties of null (reading 'length')" und reißt die ganze Ansicht ab
// (Schuelerdatei.svelte).
//
// Der Test prüft den echten HTTP-Pfad, nicht nur die Kodierfunktion: Genau hier ist die
// Zusicherung "eine Liste ist immer ein Array" für die Oberfläche sichtbar.
func TestLeserlisteLeereListeIstArray(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("Mock-Pool konnte nicht erstellt werden: %v", err)
	}
	defer mock.Close()

	// Keine Zeilen — der Zustand einer frisch aufgesetzten Installation.
	mock.ExpectQuery("SELECT").
		WillReturnRows(pgxmock.NewRows([]string{
			"id", "vorname", "nachname", "klasse", "barcode_id", "gesperrt",
			"lusd_id", "geburtsdatum", "foto_url", "aktive_ausleihen", "offene_gebuehren",
		}))

	server := &Tueren{Basis: &tuer.Basis{}}
	// lmfAbfragen = nil: Die Türen haben in diesem Test keine Datenbank, und ohne
	// Jahrgangsfilter darf die Klassenliste gar nicht erst angefasst werden. Holte die Tür
	// sie sich über t.DB.Pool, wäre das ohne Datenbank ein nil-Zeiger, und dieser Test
	// stürzte mit SIGSEGV ab statt zu prüfen, was er prüfen soll. Dass hier nil stehen darf,
	// ist die Zusicherung.
	handler := server.Leserliste(repository.NewLeserAbfragen(mock), nil)

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/schueler", nil))

	body := strings.TrimSpace(rec.Body.String())
	if body == "null" {
		t.Fatal("leere Schülerliste kam als null zurück — das Frontend bricht darauf ab (.length auf null)")
	}
	if body != "[]" {
		t.Errorf("erwartet [], war: %s", body)
	}
}

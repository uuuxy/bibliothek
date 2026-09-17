package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"bibliothek/repository"

	"github.com/pashagolub/pgxmock/v4"
)

// TestListStudentsLeereListeIstArray hält den Fehler fest, an dem die Schülerdatei
// beim Erst-Deployment abgestürzt ist.
//
// Ablauf damals: frische Installation -> noch keine Schüler -> das Repository gibt eine
// nil-Slice zurück -> JSON "null" -> im Frontend "students = null" -> students.length
// wirft "Cannot read properties of null (reading 'length')" und reisst die ganze
// Ansicht ab (StudentDirectory.svelte, totalCount/filteredCount).
//
// Der Test prüft den echten HTTP-Pfad, nicht nur die Kodierfunktion: Genau hier ist
// die Zusicherung "eine Liste ist immer ein Array" für den Client sichtbar.
func TestListStudentsLeereListeIstArray(t *testing.T) {
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

	server := &Server{}
	// lmfRepo = nil: Der Server hat in diesem Test keine Datenbank, und ohne
	// Jahrgangsfilter darf die Klassenliste gar nicht erst angefasst werden. Bis zum
	// 17.09.2026 holte der Handler sie sich über s.DB.Pool — bei einem Server ohne DB
	// ein nil-Zeiger, und dieser Test stürzte mit SIGSEGV ab statt zu prüfen, was er
	// prüfen soll. Dass hier nil stehen DARF, ist die Zusicherung.
	handler := server.ListStudentsHandler(repository.NewStudentRepository(mock), nil)

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

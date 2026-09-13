package inventur

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/pashagolub/pgxmock/v4"
)

// Verhalten zu uuid_eingaben_test.go (Root-Paket): Eine Titel-Kennung, die keine UUID
// ist, wird mit 400 abgewiesen, bevor sie die Datenbank erreicht. Die Listen gehen als
// `= ANY($1::uuid[])` bzw. als class_books.book_id (UUID) an Postgres; ohne Prüfung kam
// `invalid input syntax for type uuid` (22P02) als 500 zurück.
//
// Das Mock hat keine einzige Erwartung: Jeder Datenbankzugriff wäre ein Fehler, und der
// Handler antwortete dann mit 500 statt 400.
func TestUngueltigeTitelKennungIst400VorDerDatenbank(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock: %v", err)
	}
	defer mock.Close()
	handler := &APIHandler{repo: NewBookRepository(mock)}

	faelle := []struct {
		name    string
		aufruf  func(http.ResponseWriter, *http.Request)
		methode string
		rumpf   string
	}{
		{"Bücher löschen", handler.BearbeiteBuecherLoeschen, http.MethodDelete, `{"ids":["x"]}`},
		{"Bücher umsortieren", handler.handleReorderBooks, http.MethodPut, `{"bookIds":["x"]}`},
		{"Klassenbücher hinzufügen", handler.handleAddClassBooks, http.MethodPost, `{"classNames":["05G1"],"bookIds":["x"]}`},
		{"Klassenbücher ändern", handler.handleUpdateClassBooks, http.MethodPut, `{"className":"05G1","bookIds":["x"]}`},
		{"Cover neu laden", handler.handleRetryExternalCovers, http.MethodPost, `{"ids":["x"]}`},
	}
	for _, f := range faelle {
		t.Run(f.name, func(t *testing.T) {
			req := httptest.NewRequest(f.methode, "/api/books", strings.NewReader(f.rumpf))
			rec := httptest.NewRecorder()
			f.aufruf(rec, req)
			if rec.Code != http.StatusBadRequest {
				t.Fatalf("Status %d, erwartet 400: %s", rec.Code, rec.Body.String())
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatalf("Datenbank wurde angesprochen: %v", err)
			}
		})
	}
}

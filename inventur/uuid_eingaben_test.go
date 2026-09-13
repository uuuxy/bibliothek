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
		// Besteht uuid.Validate, Postgres weist sie ab (api/uuid_eingaben_test.go, urnForm).
		{"Bücher löschen, urn-Form", handler.BearbeiteBuecherLoeschen, http.MethodDelete, `{"ids":["urn:uuid:7c9e6679-7425-40de-944b-e07fc1f90ae7"]}`},
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

// Der Import-Schlüssel geht als UUID in idempotency_keys. uuid.Parse nahm die urn-Form an,
// Postgres nicht: Der Fehler käme erst beim Eintragen, als 500.
func TestImportSchluesselInUrnFormIst400(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/books/import", nil)
	req.Header.Set(importSchluesselKopf, "urn:uuid:7c9e6679-7425-40de-944b-e07fc1f90ae7")
	rec := httptest.NewRecorder()
	if _, ok := importSchluessel(rec, req); ok || rec.Code != http.StatusBadRequest {
		t.Fatalf("urn-Form angenommen: ok=%v, Status %d", ok, rec.Code)
	}
}

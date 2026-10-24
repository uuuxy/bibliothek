package inventur

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/pashagolub/pgxmock/v5"

	"bibliothek/repository"
)

// coverQueryRegex matcht die SELECT-Query von handleListExternalCovers (LIMIT $1).
const coverQueryRegex = `(?s)SELECT id, COALESCE\(isbn, ''\) AS isbn, titel AS title, COALESCE\(cover_url, ''\) AS cover_url.*FROM buecher_titel.*WHERE cover_url LIKE 'http%'.*ORDER BY id ASC.*LIMIT \$1`

func TestFremdeCoverListe(t *testing.T) {
	// Subtests in eigene Top-Level-Funktionen ausgelagert, damit die Assertions
	// nicht in der t.Run-Closure verschachtelt liegen (S3776). Namen unverändert.
	t.Run("Success", testListExternalCoversSuccess)
	t.Run("DatabaseError", testListExternalCoversDatabaseError)
}

func testListExternalCoversSuccess(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("failed to create pgxmock: %v", err)
	}
	defer mock.Close()

	tueren := &Tueren{abfragen: repository.NewTitelAbfragen(mock)}

	mock.ExpectQuery(coverQueryRegex).
		WithArgs(300).
		WillReturnRows(pgxmock.NewRows([]string{"id", "isbn", "title", "cover_url"}).
			AddRow("1", "1234567890", "Test Book 1", "http://example.com/cover1.jpg").
			AddRow("2", "0987654321", "Test Book 2", "https://example.com/cover2.jpg"))

	anfrage := httptest.NewRequest(http.MethodGet, "/api/covers/external", nil)
	w := httptest.NewRecorder()

	tueren.FremdeCoverListe(w, anfrage)

	if w.Code != http.StatusOK {
		t.Errorf("w.Code = %d, erwartet %d", w.Code, http.StatusOK)
	}

	var antwort map[string][]repository.KatalogTitel
	if err := json.Unmarshal(w.Body.Bytes(), &antwort); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	eintraege, ok := antwort["data"]
	if !ok {
		t.Fatalf("Antwort ohne Schlüssel 'data'")
	}

	if len(eintraege) != 2 {
		t.Errorf("len(eintraege) = %d, erwartet 2 Titel", len(eintraege))
	}
	if eintraege[0].ID != "1" || eintraege[1].ID != "2" {
		t.Errorf("Unexpected book IDs returned")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}

func testListExternalCoversDatabaseError(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("failed to create pgxmock: %v", err)
	}
	defer mock.Close()

	tueren := &Tueren{abfragen: repository.NewTitelAbfragen(mock)}

	mock.ExpectQuery(coverQueryRegex).
		WithArgs(300).
		WillReturnError(errTest)

	anfrage := httptest.NewRequest(http.MethodGet, "/api/covers/external", nil)
	w := httptest.NewRecorder()

	tueren.FremdeCoverListe(w, anfrage)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("w.Code = %d, erwartet %d", w.Code, http.StatusInternalServerError)
	}

	var antwort map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &antwort); err != nil {
		t.Fatalf("Failed to parse error response: %v", err)
	}

	if antwort["error"] != "Ein interner Serverfehler ist aufgetreten" {
		t.Errorf("Unexpected error message: %s", antwort["error"])
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}

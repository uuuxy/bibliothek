package katalog

import (
	"bytes"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/pashagolub/pgxmock/v5"

	"bibliothek/api/tuer"
	"bibliothek/db"
	"bibliothek/internal/katalogdienste"
)

// offlineMetadatenClient baut einen MetadatenClient, dessen Transport jede
// Anfrage mit 404 beantwortet. Die Import-Tests duerfen nie ins echte Netz
// (ergaenzeMetadaten wuerde sonst pro CSV-Zeile DNB/Google/OpenLibrary
// abfragen und gefundene Cover real nach uploads/ schreiben).
func offlineMetadatenClient() *katalogdienste.MetadatenClient {
	return metadatenUeber(&http.Client{Transport: &mockTransportCover{
		roundTripFunc: func(*http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusNotFound,
				Body:       io.NopCloser(strings.NewReader("{}")),
				Header:     make(http.Header),
			}, nil
		},
	}})
}

func TestExcelImport(t *testing.T) {
	t.Run("Missing File", func(t *testing.T) {
		rumpf := new(bytes.Buffer)
		schreiber := multipart.NewWriter(rumpf)
		err := schreiber.Close()
		if err != nil {
			t.Fatalf("schreiber.Close: %v", err)
		}

		anfrage := httptest.NewRequest(http.MethodPost, "/api/books/import", rumpf)
		anfrage.Header.Set("Content-Type", schreiber.FormDataContentType())

		w := httptest.NewRecorder()

		tueren := &Tueren{}
		tueren.ExcelImport()(w, anfrage)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Status = %v, erwartet %v", w.Code, http.StatusBadRequest)
		}

		var antwort map[string]any
		err = json.NewDecoder(w.Body).Decode(&antwort)
		if err != nil {
			t.Fatalf("json.NewDecoder(w.Body).Decode: %v", err)
		}
		if antwort["error"] != "Bitte eine Datei auswählen" {
			t.Errorf("antwort[\"error\"] = %v, erwartet %v", antwort["error"], "Bitte eine Datei auswählen")
		}
	})

	t.Run("Missing ISBN column", func(t *testing.T) {
		rumpf := new(bytes.Buffer)
		schreiber := multipart.NewWriter(rumpf)
		teil, err := schreiber.CreateFormFile("file", "test.csv")
		if err != nil {
			t.Fatalf("schreiber.CreateFormFile: %v", err)
		}
		_, err = teil.Write([]byte("titel,autor\nBuch1,Autor1"))
		if err != nil {
			t.Fatalf("teil.Write: %v", err)
		}
		err = schreiber.Close()
		if err != nil {
			t.Fatalf("schreiber.Close: %v", err)
		}

		anfrage := httptest.NewRequest(http.MethodPost, "/api/books/import", rumpf)
		anfrage.Header.Set("Content-Type", schreiber.FormDataContentType())

		w := httptest.NewRecorder()

		tueren := &Tueren{}
		tueren.ExcelImport()(w, anfrage)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Status = %v, erwartet %v", w.Code, http.StatusBadRequest)
		}
		var antwort map[string]any
		err = json.NewDecoder(w.Body).Decode(&antwort)
		if err != nil {
			t.Fatalf("json.NewDecoder(w.Body).Decode: %v", err)
		}
		if got, ok := antwort["error"].(string); !ok || !strings.Contains(got, "Die Spalte ISBN fehlt") {
			t.Errorf("antwort[\"error\"] = %v, erwartet darin %q", antwort["error"], "Die Spalte ISBN fehlt")
		}
	})

	t.Run("Successful Import", func(t *testing.T) {
		mock, err := pgxmock.NewPool(pgxmock.QueryMatcherOption(pgxmock.QueryMatcherRegexp))
		if err != nil {
			t.Fatalf("pgxmock.NewPool: %v", err)
		}
		defer mock.Close()

		rumpf := new(bytes.Buffer)
		schreiber := multipart.NewWriter(rumpf)
		teil, err := schreiber.CreateFormFile("file", "test.csv")
		if err != nil {
			t.Fatalf("schreiber.CreateFormFile: %v", err)
		}
		_, err = teil.Write([]byte("isbn,titel,autor,bestand\n9781234567890,Buch1,Autor1,5"))
		if err != nil {
			t.Fatalf("teil.Write: %v", err)
		}
		err = schreiber.Close()
		if err != nil {
			t.Fatalf("schreiber.Close: %v", err)
		}

		anfrage := httptest.NewRequest(http.MethodPost, "/api/books/import", rumpf)
		anfrage.Header.Set("Content-Type", schreiber.FormDataContentType())

		w := httptest.NewRecorder()

		tueren := &Tueren{Basis: &tuer.Basis{DB: &db.Verbindung{Pool: mock}}, Katalogdienste: offlineMetadatenClient()}

		// Kein Nachschlagen des Fachs: Die Testdatei hat keine Spalte für das Fach, und daraus wird ein
		// leeres Fach, nicht das wörtliche Fach „Unbekannt" (das eine Kategorie dieses Namens anlegte).
		// StelleFaecherSicher überspringt leere Eingaben. Der Batch (Titel und Exemplare) läuft in einer
		// Transaktion.
		mock.ExpectBegin()
		mock.ExpectExec(".*").WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg()).WillReturnResult(pgxmock.NewResult("INSERT", 1))
		erwarteBarcodeVergabe(mock, 5) // bestand 5 in der Testdatei
		mock.ExpectExec("INSERT INTO buecher_exemplare").WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg()).WillReturnResult(pgxmock.NewResult("INSERT", 0))
		mock.ExpectCommit()

		tueren.ExcelImport()(w, anfrage)

		if w.Code != http.StatusOK {
			t.Errorf("Status = %v, erwartet %v", w.Code, http.StatusOK)
		}
		var antwort map[string]any
		err = json.NewDecoder(w.Body).Decode(&antwort)
		if err != nil {
			t.Fatalf("json.NewDecoder(w.Body).Decode: %v", err)
		}
		if antwort["message"] != "1 Titel importiert" {
			t.Errorf("antwort[\"message\"] = %v, erwartet %v", antwort["message"], "1 Titel importiert")
		}
		if antwort["imported"] != float64(1) {
			t.Errorf("antwort[\"imported\"] = %v, erwartet %v", antwort["imported"], float64(1))
		}
		if antwort["failed"] != float64(0) {
			t.Errorf("antwort[\"failed\"] = %v, erwartet %v", antwort["failed"], float64(0))
		}
	})

	t.Run("Import failed completely", func(t *testing.T) {
		mock, err := pgxmock.NewPool(pgxmock.QueryMatcherOption(pgxmock.QueryMatcherRegexp))
		if err != nil {
			t.Fatalf("pgxmock.NewPool: %v", err)
		}
		defer mock.Close()

		rumpf := new(bytes.Buffer)
		schreiber := multipart.NewWriter(rumpf)
		teil, err := schreiber.CreateFormFile("file", "test.csv")
		if err != nil {
			t.Fatalf("schreiber.CreateFormFile: %v", err)
		}
		_, err = teil.Write([]byte("isbn,titel,autor,bestand\n9781234567890,Buch1,Autor1,5"))
		if err != nil {
			t.Fatalf("teil.Write: %v", err)
		}
		err = schreiber.Close()
		if err != nil {
			t.Fatalf("schreiber.Close: %v", err)
		}

		anfrage := httptest.NewRequest(http.MethodPost, "/api/books/import", rumpf)
		anfrage.Header.Set("Content-Type", schreiber.FormDataContentType())

		w := httptest.NewRecorder()

		tueren := &Tueren{Basis: &tuer.Basis{DB: &db.Verbindung{Pool: mock}}, Katalogdienste: offlineMetadatenClient()}

		mock.ExpectBegin()
		mock.ExpectExec(".*").WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg()).WillReturnError(errTest)
		mock.ExpectRollback()
		// Rückfall je Zeile: schreibeTitel schlägt kein Fach nach (das Fach ist leer statt „Unbekannt")
		// und öffnet direkt seine eigene Transaktion.
		mock.ExpectBegin()
		mock.ExpectQuery(".*").WillReturnError(errTest)
		mock.ExpectRollback()

		tueren.ExcelImport()(w, anfrage)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Status = %v, erwartet %v", w.Code, http.StatusBadRequest)
		}
		var antwort map[string]any
		err = json.NewDecoder(w.Body).Decode(&antwort)
		if err != nil {
			t.Fatalf("json.NewDecoder(w.Body).Decode: %v", err)
		}
		// Der Text des Fehlers aus der Datenbank geht nicht hinaus, die Meldung nennt den Lauf.
		if got := antwort["error"]; got != "Keine Zeile der Liste ließ sich importieren" {
			t.Errorf("antwort[\"error\"] = %v, erwartet die Meldung ohne den Text des Fehlers (%q)", got, errTest)
		}
	})

	t.Run("Partial Failure", func(t *testing.T) {
		mock, err := pgxmock.NewPool(pgxmock.QueryMatcherOption(pgxmock.QueryMatcherRegexp))
		if err != nil {
			t.Fatalf("pgxmock.NewPool: %v", err)
		}
		defer mock.Close()

		rumpf := new(bytes.Buffer)
		schreiber := multipart.NewWriter(rumpf)
		teil, err := schreiber.CreateFormFile("file", "test.csv")
		if err != nil {
			t.Fatalf("schreiber.CreateFormFile: %v", err)
		}
		_, err = teil.Write([]byte("isbn,titel,autor,bestand\n9781234567890,Buch1,Autor1,5\n9780987654321,Buch2,Autor2,5"))
		if err != nil {
			t.Fatalf("teil.Write: %v", err)
		}
		err = schreiber.Close()
		if err != nil {
			t.Fatalf("schreiber.Close: %v", err)
		}

		anfrage := httptest.NewRequest(http.MethodPost, "/api/books/import", rumpf)
		anfrage.Header.Set("Content-Type", schreiber.FormDataContentType())

		w := httptest.NewRecorder()

		tueren := &Tueren{Basis: &tuer.Basis{DB: &db.Verbindung{Pool: mock}}, Katalogdienste: offlineMetadatenClient()}

		mock.ExpectBegin()
		mock.ExpectExec(".*").WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg()).WillReturnError(errTest)
		mock.ExpectRollback()
		// Fallback Einzel-Upsert je Zeile (ohne Fach-Nachschlag, das Fach ist leer): erste
		// Zeile scheitert im Upsert (Begin→Rollback), zweite gelingt (Begin→…→Commit).
		mock.ExpectBegin()
		mock.ExpectQuery(".*").WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg()).WillReturnError(errTest)
		mock.ExpectRollback()
		mock.ExpectBegin()
		mock.ExpectQuery(".*").WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg()).WillReturnRows(pgxmock.NewRows([]string{"id"}).AddRow("test-id"))
		erwarteBarcodeVergabe(mock, 5) // bestand 5 der zweiten Zeile
		mock.ExpectExec("INSERT INTO buecher_exemplare").WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg()).WillReturnResult(pgxmock.NewResult("INSERT", 5))
		mock.ExpectCommit()

		tueren.ExcelImport()(w, anfrage)

		if w.Code != http.StatusOK {
			t.Errorf("Status = %v, erwartet %v", w.Code, http.StatusOK)
		}
		var antwort map[string]any
		err = json.NewDecoder(w.Body).Decode(&antwort)
		if err != nil {
			t.Fatalf("json.NewDecoder(w.Body).Decode: %v", err)
		}
		if antwort["message"] != "1 Titel importiert, 1 fehlgeschlagen" {
			t.Errorf("antwort[\"message\"] = %v, erwartet %v", antwort["message"], "1 Titel importiert, 1 fehlgeschlagen")
		}
		if antwort["imported"] != float64(1) {
			t.Errorf("antwort[\"imported\"] = %v, erwartet %v", antwort["imported"], float64(1))
		}
		if antwort["failed"] != float64(1) {
			t.Errorf("antwort[\"failed\"] = %v, erwartet %v", antwort["failed"], float64(1))
		}
	})
}

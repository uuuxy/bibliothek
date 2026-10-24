package inventur

import (
	"bibliothek/internal/tuertest"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/pashagolub/pgxmock/v5"
)

// mockTransport steht in metadaten_client_test.go im selben Paket und ist hier nutzbar.

// Pflicht ist beim Anlegen der Titel. Die ISBN darf fehlen und wird nur geprüft, wenn sie
// dasteht.
func TestPruefeTitelEingabe(t *testing.T) {
	faelle := []struct {
		name       string
		eingabe    titelEingabe
		wantResult bool
		wantError  string
	}{
		{name: "Titel und ISBN", eingabe: titelEingabe{Titel: "T", ISBN: "978-3-16-148410-0"}, wantResult: true},
		{name: "ohne ISBN", eingabe: titelEingabe{Titel: "T"}, wantResult: true},
		{name: "ohne Titel", eingabe: titelEingabe{ISBN: "978-3-16-148410-0"}, wantError: "Der Titel darf nicht leer sein"},
		{name: "Titel nur aus Leerzeichen", eingabe: titelEingabe{Titel: "  "}, wantError: "Der Titel darf nicht leer sein"},
		{name: "Nummer ohne ISBN-Form", eingabe: titelEingabe{Titel: "T", ISBN: "123"}, wantError: "Die ISBN ist ungültig"},
	}

	for _, f := range faelle {
		t.Run(f.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			ergebnis := pruefeTitelEingabe(recorder, f.eingabe)
			if ergebnis != f.wantResult {
				t.Errorf("pruefeTitelEingabe(…) = %v, erwartet %v", ergebnis, f.wantResult)
			}
			if f.wantResult {
				return
			}
			if recorder.Code != http.StatusBadRequest || !strings.Contains(recorder.Body.String(), f.wantError) {
				t.Errorf("pruefeTitelEingabe(…): Status %d, %s, erwartet 400 mit %q", recorder.Code, recorder.Body.String(), f.wantError)
			}
		})
	}
}

func TestTitelLoeschen(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("failed to create pgxmock: %v", err)
	}
	defer mock.Close()

	abfragen := NewTitelAbfragen(mock)
	tueren := &Tueren{abfragen: abfragen}

	t.Run("Invalid JSON", func(t *testing.T) {
		anfrage := httptest.NewRequest(http.MethodDelete, "/api/books", strings.NewReader("invalid json"))
		rec := httptest.NewRecorder()
		tueren.TitelLoeschen(rec, anfrage)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("rec.Code = %d, erwartet %d", rec.Code, http.StatusBadRequest)
		}
	})

	t.Run("Empty IDs", func(t *testing.T) {
		anfrage := httptest.NewRequest(http.MethodDelete, "/api/books", strings.NewReader(`{"ids": []}`))
		rec := httptest.NewRecorder()
		tueren.TitelLoeschen(rec, anfrage)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("rec.Code = %d, erwartet %d", rec.Code, http.StatusBadRequest)
		}
	})

	t.Run("Success", func(t *testing.T) {
		t.Chdir(t.TempDir()) // das Aufräumen der Cover-Dateien legt sonst inventur/uploads im Repo an

		// Die Erwartungen in der Reihenfolge von LoescheTitel (Begin zuerst, jedes Lesen in der
		// Transaktion, die Befehle des Löschens liefern ihre Spur über RETURNING). Die Reihenfolge selbst
		// prüft titel_loeschen_test.go; hier zählt, dass die Tür mit 200 antwortet.
		mock.ExpectBegin()
		erwarteAuflagenSperreTuer(mock, pgxmock.AnyArg())

		// Barcode-Snapshots vor den DELETEs (Theken-Auskunft; hier: keine Exemplare).
		mock.ExpectQuery(`FROM buecher_exemplare e`).
			WithArgs(pgxmock.AnyArg()).
			WillReturnRows(pgxmock.NewRows([]string{"id", "barcode_id", "titel", "war_im_bestand"}))

		// Die drei Kinder des Titels über CASCADE im Protokoll: Vormerkungen, Reservierungen und
		// Zuordnungen von Klassensätzen.
		mock.ExpectQuery(`FROM vormerkungen v`).
			WithArgs(pgxmock.AnyArg()).
			WillReturnRows(pgxmock.NewRows([]string{"id", "titel_id", "titel", "wer", "status", "seit", "schueler_id"}))
		mock.ExpectQuery(`FROM klassensatz_reservierungen r`).
			WithArgs(pgxmock.AnyArg()).
			WillReturnRows(pgxmock.NewRows([]string{"id", "titel_id", "titel", "klasse", "status", "seit"}))
		mock.ExpectQuery(`FROM class_books c`).
			WithArgs(pgxmock.AnyArg()).
			WillReturnRows(pgxmock.NewRows([]string{"id", "titel_id", "titel", "klasse"}))

		mock.ExpectQuery(`SELECT t\.cover_url`).
			WithArgs(pgxmock.AnyArg()).
			WillReturnRows(pgxmock.NewRows([]string{"cover_url"}).AddRow("/uploads/cover.jpg"))

		// Offene Forderungen und laufende Ausleihen kommen als Spur aus dem Löschbefehl
		// selbst (hier: keine).
		mock.ExpectQuery(`DELETE FROM schadensfaelle sf`).
			WithArgs(pgxmock.AnyArg()).
			WillReturnRows(pgxmock.NewRows([]string{"id", "exemplar_id", "barcode_id", "titel",
				"schuldner", "schueler_id", "betrag", "beschreibung", "seit"}))
		mock.ExpectQuery(`DELETE FROM ausleihen a`).
			WithArgs(pgxmock.AnyArg()).
			WillReturnRows(pgxmock.NewRows([]string{"id", "exemplar_id", "barcode_id", "titel", "entleiher", "schueler_id", "seit"}))

		mock.ExpectExec("DELETE FROM buecher_titel").
			WithArgs(pgxmock.AnyArg()).
			WillReturnResult(pgxmock.NewResult("DELETE", 1))

		mock.ExpectCommit()

		rumpf := `{"ids": ["11111111-1111-1111-1111-111111111111"]}`
		anfrage := httptest.NewRequest(http.MethodDelete, "/api/books", strings.NewReader(rumpf))
		rec := httptest.NewRecorder()
		tueren.TitelLoeschen(rec, anfrage)

		if rec.Code != http.StatusOK {
			t.Errorf("rec.Code = %d, erwartet %d", rec.Code, http.StatusOK)
		}
	})
}

func TestTitelAnlegen(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("failed to create pgxmock: %v", err)
	}
	defer mock.Close()

	abfragen := NewTitelAbfragen(mock)

	// Attrappe des HTTP-Clients für den MetadatenClient.
	mockHTTP := &http.Client{
		Transport: tuertest.Transport(func(anfrage *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       http.NoBody,
			}, nil
		}),
	}
	metadatenClient := metadatenUeber(mockHTTP)

	tueren := &Tueren{
		abfragen:  abfragen,
		metadaten: metadatenClient,
	}

	t.Run("Invalid JSON", func(t *testing.T) {
		anfrage := httptest.NewRequest(http.MethodPost, "/api/books", strings.NewReader("invalid json"))
		rec := httptest.NewRecorder()
		tueren.TitelAnlegen(rec, anfrage)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("rec.Code = %d, erwartet %d", rec.Code, http.StatusBadRequest)
		}
	})

	t.Run("ohne Titel wird nichts angelegt", func(t *testing.T) {
		rumpf := `{"isbn": "978-3-16-148410-0", "subject": "Math"}`
		anfrage := httptest.NewRequest(http.MethodPost, "/api/books", strings.NewReader(rumpf))
		rec := httptest.NewRecorder()
		tueren.TitelAnlegen(rec, anfrage)

		if rec.Code != http.StatusBadRequest || !strings.Contains(rec.Body.String(), "Der Titel darf nicht leer sein") {
			t.Errorf("TitelAnlegen: Status %d, %s, erwartet 400 mit „Der Titel darf nicht leer sein“", rec.Code, rec.Body.String())
		}
	})

	// chk_listenpreis_nonneg: Liefe der Wert bis in die Datenbank, hieße die Antwort nur „buch
	// konnte nicht erstellt werden".
	t.Run("Negativer Listenpreis nennt den erlaubten Bereich", func(t *testing.T) {
		rumpf := `{"isbn": "978-3-16-148410-0", "title": "T", "author": "A", "listenpreis": -5}`
		anfrage := httptest.NewRequest(http.MethodPost, "/api/books", strings.NewReader(rumpf))
		rec := httptest.NewRecorder()
		tueren.TitelAnlegen(rec, anfrage)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("rec.Code = %d, erwartet %d", rec.Code, http.StatusBadRequest)
		}
		var antwort struct {
			Error string `json:"error"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &antwort); err != nil {
			t.Fatalf("Antwort ist kein JSON: %v — %s", err, rec.Body.String())
		}
		if antwort.Error != "Der Listenpreis darf nicht negativ sein. Leer lassen, wenn er unbekannt ist." {
			t.Errorf("Meldung nennt den erlaubten Bereich nicht: %q", antwort.Error)
		}
	})

	t.Run("Success", func(t *testing.T) {
		// „gradeLevel" nennt ein Browser-Fenster von vor dem Umbau noch: Die Tür liest das Feld
		// nicht mehr und legt den Titel ohne es an.
		rumpf := `{
			"isbn": "978-3-16-148410-0",
			"subject": "Math",
			"gradeLevel": 5,
			"title": "Test Title",
			"author": "Test Author",
			"coverUrl": "test.jpg"
		}`

		erwarteFachBekanntTuer(mock, "Math") // auf dem Pool, vor der Tx
		// LegeTitelAn ist atomar (Tx): Begin, INSERT, Commit — kein Stock im Body, also
		// wird gleicheBestandAb nicht aufgerufen.
		mock.ExpectBegin()
		erwarteKeineDubletteTuer(mock)
		mock.ExpectQuery(`INSERT INTO buecher_titel`).
			WithArgs(
				"978-3-16-148410-0", // isbn
				"Test Title",        // title
				"Test Author",       // author
				"test.jpg",          // cover_url
				"Math",              // subject
				"",                  // track
				pgxmock.AnyArg(),    // last_counted
				"Buch",              // medientyp
				pgxmock.AnyArg(),    // erweiterte_eigenschaften
				0,                   // jahrgang_von
				0,                   // jahrgang_bis
				"",                  // untertitel
				"",                  // verlag
				0,                   // erscheinungsjahr
				"",                  // signatur
				false,               // ist_lernmittel
				"",                  // auflage
				// listenpreis: nil, weil dieser Test keine ISBN-Metadaten liefert. Bei
				// einem echten Nachschlagen füllt ihn der DNB-Ladenpreis.
				pgxmock.AnyArg(),
				false, // mehrjahresband
			).
			WillReturnRows(pgxmock.NewRows([]string{"id"}).AddRow("11111111-1111-1111-1111-111111111111"))
		mock.ExpectCommit()

		anfrage := httptest.NewRequest(http.MethodPost, "/api/books", strings.NewReader(rumpf))
		// Ein Wert im Kontext (etwa die Sitzung) ist für diese Türen nicht nötig.
		rec := httptest.NewRecorder()
		tueren.TitelAnlegen(rec, anfrage)

		if rec.Code != http.StatusCreated {
			t.Errorf("rec.Code = %d, erwartet %d", rec.Code, http.StatusCreated)
		}

		var httpAntwort map[string]any
		if err := json.Unmarshal(rec.Body.Bytes(), &httpAntwort); err != nil {
			t.Fatalf("failed to parse response body: %v", err)
		}
		if httpAntwort["message"] != "Titel angelegt" {
			t.Errorf("TitelAnlegen: Meldung %v, erwartet 'Titel angelegt'", httpAntwort["message"])
		}
	})
}

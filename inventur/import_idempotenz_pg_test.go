package inventur

import (
	"bytes"
	"context"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"bibliothek/internal/pgtest"

	"github.com/google/uuid"
)

// Der Listenimport legt je Zeile Exemplare an — additiv. Zweimal dieselbe Liste mit
// demselben Schlüssel darf den Bestand nicht verdoppeln, und der zweite Aufruf bekommt die
// Antwort des ersten. Am echten Postgres, weil der Schutz aus INSERT … ON CONFLICT und
// dem UPDATE danach besteht — pgxmock würde nur nachspielen, was ich erwarte.
func TestListenimport_SchluesselVerhindertDoppeltenBestand(t *testing.T) {
	pool := pgtest.Pool(t)
	ctx := context.Background()
	const isbn = "9783161484100"
	if _, err := pool.Exec(ctx, `DELETE FROM buecher_titel WHERE isbn = $1`, isbn); err != nil {
		t.Fatalf("Vorbereitung: %v", err)
	}
	t.Cleanup(func() {
		if _, err := pool.Exec(ctx, `DELETE FROM buecher_titel WHERE isbn = $1`, isbn); err != nil {
			t.Logf("Aufräumen: %v", err)
		}
	})

	handler := &APIHandler{repo: &BookRepository{db: pool}, metadaten: offlineMetadatenClient()}
	schluessel := uuid.NewString()
	sende := func(key string) *httptest.ResponseRecorder {
		body := new(bytes.Buffer)
		w := multipart.NewWriter(body)
		teil, err := w.CreateFormFile("file", "liste.csv")
		if err != nil {
			t.Fatalf("Formularteil: %v", err)
		}
		if _, err := teil.Write([]byte("isbn,titel,autor,bestand\n" + isbn + ",Idempotenz-Titel,Autor,3\n")); err != nil {
			t.Fatalf("CSV schreiben: %v", err)
		}
		if err := w.Close(); err != nil {
			t.Fatalf("Formular schließen: %v", err)
		}
		req := httptest.NewRequest(http.MethodPost, "/api/books/import", body)
		req.Header.Set("Content-Type", w.FormDataContentType())
		if key != "" {
			req.Header.Set(importSchluesselKopf, key)
		}
		rec := httptest.NewRecorder()
		handler.handleImportExcel(rec, req)
		return rec
	}
	zaehleExemplare := func() int {
		var n int
		if err := pool.QueryRow(ctx, `
			SELECT count(*) FROM buecher_exemplare e JOIN buecher_titel t ON t.id = e.titel_id
			 WHERE t.isbn = $1`, isbn).Scan(&n); err != nil {
			t.Fatalf("zählen: %v", err)
		}
		return n
	}

	erster := sende(schluessel)
	if erster.Code != http.StatusOK {
		t.Fatalf("erster Lauf: %d %s", erster.Code, erster.Body.String())
	}
	if n := zaehleExemplare(); n != 3 {
		t.Fatalf("nach erstem Lauf %d Exemplare, want 3", n)
	}

	zweiter := sende(schluessel)
	if zweiter.Code != http.StatusOK {
		t.Fatalf("zweiter Lauf: %d %s", zweiter.Code, zweiter.Body.String())
	}
	var a, b map[string]any
	if err := json.Unmarshal(erster.Body.Bytes(), &a); err != nil {
		t.Fatalf("erste Antwort: %v", err)
	}
	if err := json.Unmarshal(zweiter.Body.Bytes(), &b); err != nil {
		t.Fatalf("zweite Antwort: %v", err)
	}
	if a["imported"] != b["imported"] || a["message"] != b["message"] {
		t.Errorf("zweite Antwort weicht ab: %v vs %v", a, b)
	}
	if n := zaehleExemplare(); n != 3 {
		t.Errorf("nach Wiederholung %d Exemplare, want 3 — der Import lief doppelt", n)
	}

	// Ein reservierter, noch offener Lauf: 409, kein zweiter Import.
	offen := uuid.NewString()
	if frisch, _, err := handler.reserviereImportLauf(ctx, offen); err != nil || !frisch {
		t.Fatalf("reservieren: frisch=%v err=%v", frisch, err)
	}
	dritter := sende(offen)
	if dritter.Code != http.StatusConflict {
		t.Errorf("laufender Schlüssel: %d, want 409", dritter.Code)
	}
	if n := zaehleExemplare(); n != 3 {
		t.Errorf("laufender Schlüssel hat importiert: %d Exemplare", n)
	}

	// Ohne Schlüssel bleibt der Import, was er war: additiv.
	if vierter := sende(""); vierter.Code != http.StatusOK {
		t.Fatalf("ohne Schlüssel: %d %s", vierter.Code, vierter.Body.String())
	}
	if n := zaehleExemplare(); n != 6 {
		t.Errorf("ohne Schlüssel %d Exemplare, want 6", n)
	}
	if kaputt := sende("kein-uuid"); kaputt.Code != http.StatusBadRequest {
		t.Errorf("ungültiger Schlüssel: %d, want 400", kaputt.Code)
	}
}

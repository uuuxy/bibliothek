package katalog

import (
	"bytes"
	"context"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"bibliothek/api/tuer"
	"bibliothek/db"
	"bibliothek/internal/pgtest"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
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

	tueren := &Tueren{Basis: &tuer.Basis{DB: &db.Verbindung{Pool: pool}}, Katalogdienste: offlineMetadatenClient()}
	schluessel := uuid.NewString()
	sende := func(idempotenzSchluessel string) *httptest.ResponseRecorder {
		rec := httptest.NewRecorder()
		tueren.ExcelImport()(rec, listenAnfrage(t, "isbn,titel,autor,bestand\n"+isbn+",Idempotenz-Titel,Autor,3\n", idempotenzSchluessel))
		return rec
	}
	zaehleExemplare := func() int { return zaehleExemplareZurISBN(t, pool, isbn) }

	erster := sende(schluessel)
	if erster.Code != http.StatusOK {
		t.Fatalf("erster Lauf: %d %s", erster.Code, erster.Body.String())
	}
	if n := zaehleExemplare(); n != 3 {
		t.Fatalf("nach erstem Lauf %d Exemplare, erwartet 3", n)
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
		t.Errorf("nach Wiederholung %d Exemplare, erwartet 3 — der Import lief doppelt", n)
	}

	// Ein reservierter, noch offener Lauf: 409, kein zweiter Import.
	offen := uuid.NewString()
	if frisch, _, err := tueren.reserviereImportLauf(ctx, offen); err != nil || !frisch {
		t.Fatalf("reservieren: frisch=%v err=%v", frisch, err)
	}
	dritter := sende(offen)
	if dritter.Code != http.StatusConflict {
		t.Errorf("laufender Schlüssel: %d, erwartet 409", dritter.Code)
	}
	if n := zaehleExemplare(); n != 3 {
		t.Errorf("laufender Schlüssel hat importiert: %d Exemplare", n)
	}

	// Ohne Schlüssel bleibt der Import, was er war: additiv.
	if vierter := sende(""); vierter.Code != http.StatusOK {
		t.Fatalf("ohne Schlüssel: %d %s", vierter.Code, vierter.Body.String())
	}
	if n := zaehleExemplare(); n != 6 {
		t.Errorf("ohne Schlüssel %d Exemplare, erwartet 6", n)
	}
	if kaputt := sende("kein-uuid"); kaputt.Code != http.StatusBadRequest {
		t.Errorf("ungültiger Schlüssel: %d, erwartet 400", kaputt.Code)
	}
}

// Bricht die Verbindung ab, während der Server noch nachschlägt, scheitert das Schreiben, und
// nichts ist importiert. Derselbe Klick danach muss echt importieren: Der Lauf gibt den Schlüssel
// frei, auch mit abgebrochener Anfrage. Sonst antwortete der Schlüssel bis zum Aufräumen nach
// 24 Stunden mit „läuft noch" oder mit der Fehlmeldung des abgebrochenen Laufs.
func TestListenimport_AbgebrochenerLaufGibtDenSchluesselFrei(t *testing.T) {
	pool := pgtest.Pool(t)
	ctx := context.Background()
	const isbn = "9780306406157"
	raeume := func() {
		if _, err := pool.Exec(ctx, `DELETE FROM buecher_titel WHERE isbn = $1`, isbn); err != nil {
			t.Errorf("aufräumen: %v", err)
		}
	}
	raeume()
	t.Cleanup(raeume)
	schluessel := uuid.NewString()
	liste := "isbn,titel,autor,bestand\n" + isbn + ",Abbruch-Titel,Autor,2\n"

	// Der Browser geht, während der Server auf das Nachschlagen der ISBN wartet.
	abgebrochen, abbrechen := context.WithCancel(ctx)
	defer abbrechen()
	brichtAb := &Tueren{Basis: &tuer.Basis{DB: &db.Verbindung{Pool: pool}}, Katalogdienste: metadatenUeber(&http.Client{
		Transport: &mockTransportCover{roundTripFunc: func(*http.Request) (*http.Response, error) {
			abbrechen()
			return nil, context.Canceled
		}},
	})}
	erster := httptest.NewRecorder()
	brichtAb.ExcelImport()(erster, listenAnfrage(t, liste, schluessel).WithContext(abgebrochen))
	if erster.Code != http.StatusBadRequest {
		t.Fatalf("abgebrochener Lauf: %d %s, erwartet 400 (nichts geschrieben)", erster.Code, erster.Body.String())
	}
	if n := zaehleExemplareZurISBN(t, pool, isbn); n != 0 {
		t.Fatalf("der abgebrochene Lauf hat %d Exemplare geschrieben, erwartet 0", n)
	}

	zweiter := httptest.NewRecorder()
	tueren := &Tueren{Basis: &tuer.Basis{DB: &db.Verbindung{Pool: pool}}, Katalogdienste: offlineMetadatenClient()}
	tueren.ExcelImport()(zweiter, listenAnfrage(t, liste, schluessel))
	if zweiter.Code != http.StatusOK {
		t.Fatalf("derselbe Klick danach: %d %s, erwartet 200 — der Schlüssel blieb belegt", zweiter.Code, zweiter.Body.String())
	}
	if n := zaehleExemplareZurISBN(t, pool, isbn); n != 2 {
		t.Errorf("nach dem zweiten Klick %d Exemplare, erwartet 2", n)
	}
}

// listenAnfrage baut den Aufruf der Tür POST /api/books/import mit einer CSV-Liste; ohne
// Schlüssel fehlt die Kopfzeile.
// Eine Liste aus Excel in Windows-1252: Die Zeile mit dem Umlaut scheiterte unumgewandelt an der
// Datenbank, und die Antwort nannte nur „1 fehlgeschlagen".
func TestListenimport_CSVAusExcelInWindows1252(t *testing.T) {
	pool := pgtest.Pool(t)
	const isbn = "9783866801860"
	raeumeAuf := func() {
		if _, err := pool.Exec(context.Background(), `DELETE FROM buecher_titel WHERE isbn = $1`, isbn); err != nil {
			t.Logf("Aufräumen: %v", err)
		}
	}
	raeumeAuf()
	t.Cleanup(raeumeAuf)

	tueren := &Tueren{Basis: &tuer.Basis{DB: &db.Verbindung{Pool: pool}}, Katalogdienste: offlineMetadatenClient()}
	rec := httptest.NewRecorder()
	tueren.ExcelImport()(rec, listenAnfrage(t, "isbn;titel;autor;bestand\r\n"+isbn+";Mathe f\xfcr alle;M\xfcller;1\r\n", ""))
	if rec.Code != http.StatusOK {
		t.Fatalf("Import = %d: %s", rec.Code, rec.Body.String())
	}
	if n := pgtest.ZaehleZeilen(t, pool, `SELECT count(*) FROM buecher_titel
		WHERE isbn = $1 AND titel = 'Mathe für alle' AND autor = 'Müller'`, isbn); n != 1 {
		t.Errorf("%d Titel mit richtigem Titel und Autor, erwartet 1: %s", n, rec.Body.String())
	}
}

func listenAnfrage(t *testing.T, csv, schluessel string) *http.Request {
	t.Helper()
	rumpf := new(bytes.Buffer)
	w := multipart.NewWriter(rumpf)
	teil, err := w.CreateFormFile("file", "liste.csv")
	if err != nil {
		t.Fatalf("Formularteil: %v", err)
	}
	if _, err := teil.Write([]byte(csv)); err != nil {
		t.Fatalf("CSV schreiben: %v", err)
	}
	if err := w.Close(); err != nil {
		t.Fatalf("Formular schließen: %v", err)
	}
	anfrage := httptest.NewRequest(http.MethodPost, "/api/books/import", rumpf)
	anfrage.Header.Set("Content-Type", w.FormDataContentType())
	if schluessel != "" {
		anfrage.Header.Set(importSchluesselKopf, schluessel)
	}
	return anfrage
}

func zaehleExemplareZurISBN(t *testing.T, pool *pgxpool.Pool, isbn string) int {
	t.Helper()
	return pgtest.ZaehleZeilen(t, pool, `
		SELECT count(*) FROM buecher_exemplare e JOIN buecher_titel t ON t.id = e.titel_id
		 WHERE t.isbn = $1`, isbn)
}

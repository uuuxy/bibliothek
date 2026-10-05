package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"bibliothek/auth"
	"bibliothek/db"
)

// Die Annahme-Tür für Meldungen nimmt, was das Formular im Kollegiums-Portal schickt — Art,
// Text, Klasse, Beschreibung — und nichts darüber hinaus.
//
// Der Test schickt `titel_id` und `isbn` trotzdem mit, und zwar mit Werten, die eine Tür mit
// diesen Feldern annähme: eine echte Titel-Kennung (der Fremdschlüssel hielte) und eine ISBN.
// Landen beide in der Zeile, ist er rot.
func TestCreateAnliegen_NimmtNurDieFelderDesFormulars(t *testing.T) {
	pool := pgTestPool(t)
	ctx := context.Background()
	srv := &Server{DB: &db.Database{Pool: pool}}

	var lehrkraftID, titelID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO benutzer (vorname, nachname, email, rolle, aktiv)
		VALUES ('Tilda', 'Tuer', 'anliegen-tuer@test.invalid', 'kollegium', true)
		ON CONFLICT (email) DO UPDATE SET vorname = EXCLUDED.vorname
		RETURNING id`).Scan(&lehrkraftID); err != nil {
		t.Fatalf("Lehrkraft anlegen: %v", err)
	}
	if err := pool.QueryRow(ctx, `
		INSERT INTO buecher_titel (titel) VALUES ('Anliegen-Tür: vorhandener Titel')
		RETURNING id`).Scan(&titelID); err != nil {
		t.Fatalf("Titel anlegen: %v", err)
	}
	t.Cleanup(func() {
		aufraeumen := context.Background()
		if _, err := pool.Exec(aufraeumen, `DELETE FROM lehrer_anliegen WHERE angefordert_von = $1`, lehrkraftID); err != nil {
			t.Logf("Aufräumen Anliegen: %v", err)
		}
		if _, err := pool.Exec(aufraeumen, `DELETE FROM buecher_titel WHERE id = $1`, titelID); err != nil {
			t.Logf("Aufräumen Titel: %v", err)
		}
	})

	rumpf := `{"art":"meldung","titel_text":"Markl Biologie 2","klasse":"8G3",` +
		`"kommentar":"falsche Auflage bekommen","isbn":"978-3-12-150010-9","titel_id":"` + titelID + `"}`
	req := httptest.NewRequest(http.MethodPost, "/api/anliegen", strings.NewReader(rumpf))
	req = req.WithContext(context.WithValue(req.Context(), auth.ClaimsContextKey,
		&auth.Claims{UserID: lehrkraftID}))
	rec := httptest.NewRecorder()
	srv.CreateAnliegenHandler()(rec, req)

	// Unbekannte Felder sind kein Fehler: Wer sie schickt, bekommt seine Meldung trotzdem
	// angelegt. Verloren ginge sonst eine Meldung, nicht ein Feld.
	if rec.Code != http.StatusCreated {
		t.Fatalf("Status = %d, want 201 — %s", rec.Code, rec.Body.String())
	}
	var antwort map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &antwort); err != nil || antwort["id"] == "" {
		t.Fatalf("Antwort ohne id: %v — %s", err, rec.Body.String())
	}

	var ohneTitel bool
	var isbn, titelText, klasse, kommentar string
	if err := pool.QueryRow(ctx, `
		SELECT titel_id IS NULL, isbn, titel_text, klasse, kommentar
		FROM lehrer_anliegen WHERE id = $1`, antwort["id"]).
		Scan(&ohneTitel, &isbn, &titelText, &klasse, &kommentar); err != nil {
		t.Fatalf("Zeile lesen: %v", err)
	}
	if !ohneTitel {
		t.Errorf("titel_id wurde geschrieben — die Tür nimmt wieder eine Titel-Kennung an")
	}
	if isbn != "" {
		t.Errorf("isbn = %q, want leer — die Tür nimmt wieder eine ISBN an", isbn)
	}
	// Die Gegenprobe: Was das Formular wirklich schickt, kommt an, jedes in seiner Spalte.
	if titelText != "Markl Biologie 2" || klasse != "8G3" || kommentar != "falsche Auflage bekommen" {
		t.Errorf("Formularfelder falsch: titel_text=%q klasse=%q kommentar=%q", titelText, klasse, kommentar)
	}
}

// Die Tür nimmt nur Meldungen an, und nur mit einer Beschreibung. Einen Buchwunsch kennt
// das Portal nicht mehr: Ein Buch für eine Klasse wird dort reserviert. Und eine Meldung
// ohne den Satz, was nicht stimmt, nennt nur ein Buch — das Formular verlangt ihn, die Tür
// hält dieselbe Regel.
func TestCreateAnliegen_NimmtNurMeldungenMitBeschreibung(t *testing.T) {
	pool, router, token := portalWelt(t)
	ctx := context.Background()

	sende := func(rumpf string) *httptest.ResponseRecorder {
		req := mitCSRF(httptest.NewRequest(http.MethodPost, "/api/anliegen", strings.NewReader(rumpf)))
		req.Header.Set("Content-Type", "application/json")
		req.AddCookie(&http.Cookie{Name: "session_token", Value: token})
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		return rec
	}
	zeilen := func() int {
		t.Helper()
		var n int
		if err := pool.QueryRow(ctx, `SELECT count(*) FROM lehrer_anliegen`).Scan(&n); err != nil {
			t.Fatal(err)
		}
		return n
	}
	if _, err := pool.Exec(ctx, `DELETE FROM lehrer_anliegen`); err != nil {
		t.Fatal(err)
	}

	abgelehnt := map[string]string{
		"ein Wunsch":                       `{"art":"wunsch","titel_text":"Markl Biologie 2","klasse":"8G3","kommentar":"bitte zum Halbjahr"}`,
		"eine Meldung ohne Beschreibung":   `{"art":"meldung","titel_text":"Markl Biologie 2","klasse":"8G3"}`,
		"eine Meldung nur mit Leerzeichen": `{"art":"meldung","titel_text":"Markl Biologie 2","kommentar":"   "}`,
	}
	for name, rumpf := range abgelehnt {
		if rec := sende(rumpf); rec.Code != http.StatusBadRequest {
			t.Errorf("%s: HTTP %d statt 400 — %s", name, rec.Code, rec.Body.String())
		}
	}
	if n := zeilen(); n != 0 {
		t.Fatalf("%d Zeilen angelegt, obwohl jede Anfrage abgelehnt sein sollte", n)
	}

	rec := sende(`{"art":"meldung","titel_text":"Markl Biologie 2","klasse":"8G3","kommentar":"falsche Auflage bekommen"}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("Meldung mit Beschreibung: HTTP %d statt 201 — %s", rec.Code, rec.Body.String())
	}
	var art, kommentar string
	if err := pool.QueryRow(ctx, `SELECT art, kommentar FROM lehrer_anliegen`).Scan(&art, &kommentar); err != nil {
		t.Fatalf("Zeile lesen: %v", err)
	}
	if art != "meldung" || kommentar != "falsche Auflage bekommen" {
		t.Errorf("angelegt: art=%q kommentar=%q", art, kommentar)
	}
}

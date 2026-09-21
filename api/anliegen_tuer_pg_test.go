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

// Die Annahme-Tür für Wünsche und Meldungen nimmt, was das Formular im Kollegiums-Portal
// schickt — Art, Text, Klasse, Anmerkung — und nichts darüber hinaus.
//
// Bis zum 21.09.2026 nahm sie zusätzlich `titel_id` und `isbn` an und schrieb beide in die
// Zeile. Kein Formular hat sie je geschickt; entschieden ist, dass das Formular Freitext
// bleibt (docs/OFFEN.md 4.14). Der Test schickt beide Felder trotzdem mit, und zwar mit
// Werten, die die alte Tür angenommen hätte: eine echte Titel-Kennung (der Fremdschlüssel
// hielte) und eine ISBN. Am alten Stand landen beide in der Zeile — dann ist er rot.
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

	rumpf := `{"art":"wunsch","titel_text":"Markl Biologie 2","klasse":"8G3",` +
		`"kommentar":"bitte zum Halbjahr","isbn":"978-3-12-150010-9","titel_id":"` + titelID + `"}`
	req := httptest.NewRequest(http.MethodPost, "/api/anliegen", strings.NewReader(rumpf))
	req = req.WithContext(context.WithValue(req.Context(), auth.ClaimsContextKey,
		&auth.Claims{UserID: lehrkraftID}))
	rec := httptest.NewRecorder()
	srv.CreateAnliegenHandler()(rec, req)

	// Unbekannte Felder sind kein Fehler: Wer sie schickt, bekommt seinen Wunsch trotzdem
	// angelegt. Verloren ginge sonst ein Wunsch, nicht ein Feld.
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
	if titelText != "Markl Biologie 2" || klasse != "8G3" || kommentar != "bitte zum Halbjahr" {
		t.Errorf("Formularfelder falsch: titel_text=%q klasse=%q kommentar=%q", titelText, klasse, kommentar)
	}
}

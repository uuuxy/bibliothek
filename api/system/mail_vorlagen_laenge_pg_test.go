package system

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"bibliothek/api/tuer"
	"bibliothek/db"
	"bibliothek/internal/pgtest"
)

// Der Betreff einer Mail-Vorlage hat höchstens 255 Zeichen (VARCHAR(255)). Ein längerer kam als
// 500 zurück; jetzt sagt die Tür, was zu tun ist. Gezählt wird in Zeichen wie in PostgreSQL:
// 255 Umlaute passen, obwohl sie 510 Bytes sind.
func TestMailVorlageAendern_BetreffLaenge(t *testing.T) {
	pool := pgtest.Pool(t)
	ctx := context.Background()
	srv := &Tueren{Basis: &tuer.Basis{DB: &db.Verbindung{Pool: pool}}}

	var id string
	if err := pool.QueryRow(ctx, `INSERT INTO mail_vorlagen (typ, betreff, text_body)
		VALUES ('PROBE_BETREFF_LAENGE', 'Alt', 'Text') RETURNING id`).Scan(&id); err != nil {
		t.Fatalf("Vorlage anlegen: %v", err)
	}
	t.Cleanup(func() {
		if _, err := pool.Exec(context.Background(), `DELETE FROM mail_vorlagen WHERE id = $1`, id); err != nil {
			t.Errorf("Vorlage entfernen: %v", err)
		}
	})
	speichere := func(betreff string) *httptest.ResponseRecorder {
		t.Helper()
		rumpf := `{"betreff":"` + betreff + `","text_body":"Text"}`
		req := httptest.NewRequest(http.MethodPut, "/api/mail-templates/"+id, strings.NewReader(rumpf))
		req.SetPathValue("id", id)
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		srv.MailVorlageAendern().ServeHTTP(rec, req)
		return rec
	}

	if rec := speichere(strings.Repeat("ä", 255)); rec.Code != http.StatusOK {
		t.Errorf("255 Zeichen = %d %s, erwartet 200", rec.Code, rec.Body.String())
	}
	rec := speichere(strings.Repeat("x", 256))
	if rec.Code != http.StatusBadRequest || !strings.Contains(rec.Body.String(), "höchstens 255 Zeichen") {
		t.Errorf("256 Zeichen = %d %s, erwartet 400 mit „höchstens 255 Zeichen“", rec.Code, rec.Body.String())
	}
	var betreff string
	if err := pool.QueryRow(ctx, `SELECT betreff FROM mail_vorlagen WHERE id = $1`, id).Scan(&betreff); err != nil {
		t.Fatalf("Vorlage lesen: %v", err)
	}
	if betreff != strings.Repeat("ä", 255) {
		t.Errorf("gespeicherter Betreff hat %d Zeichen, erwartet die 255 Umlaute", len([]rune(betreff)))
	}
}

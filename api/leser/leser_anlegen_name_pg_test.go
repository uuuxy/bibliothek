package leser

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"bibliothek/api/tuer"
	"bibliothek/auth"
	"bibliothek/db"
	"bibliothek/internal/pgtest"
)

// Ein Name aus Leerzeichen besteht `validate:"required"`, und die Tür kürzt ihn erst danach. Sie
// weist ihn ab wie die Änderung, statt einen Leser ohne Namen anzulegen.
func TestLeserAnlegen_NameAusLeerzeichenAbgewiesen(t *testing.T) {
	pool := pgtest.Pool(t)
	ctx := context.Background()
	srv := &Tueren{Basis: &tuer.Basis{DB: &db.Verbindung{Pool: pool}}}
	claims := &auth.Claims{KontoID: "00000000-0000-0000-0000-00000000a128", Rolle: auth.RolleAdmin}
	const angelegt = `nachname = 'Leerzeichentest' OR (vorname = 'Leerzeichentest' AND nachname = '')`
	t.Cleanup(func() {
		if _, err := pool.Exec(ctx, `DELETE FROM leser WHERE `+angelegt); err != nil {
			t.Errorf("aufräumen: %v", err)
		}
	})

	for _, fall := range []struct{ feld, rumpf string }{
		{"Vorname", `{"art":"fachbereich","vorname":"   ","nachname":"Leerzeichentest"}`},
		{"Nachname", `{"art":"fachbereich","vorname":"Leerzeichentest","nachname":" \t "}`},
	} {
		req := httptest.NewRequest(http.MethodPost, "/api/schueler", strings.NewReader(fall.rumpf))
		req.Header.Set("Content-Type", "application/json")
		req = req.WithContext(context.WithValue(req.Context(), auth.ClaimsSchluessel, claims))
		rec := httptest.NewRecorder()
		srv.LeserAnlegen().ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest || !strings.Contains(rec.Body.String(), fall.feld) {
			t.Errorf("%s aus Leerzeichen: Status %d (%s), erwartet 400 mit dem Namen des Felds", fall.feld, rec.Code, rec.Body.String())
		}
	}

	var anzahl int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM leser WHERE `+angelegt).Scan(&anzahl); err != nil {
		t.Fatalf("Leserzeilen zählen: %v", err)
	}
	if anzahl != 0 {
		t.Errorf("%d Leser ohne Vor- oder Nachnamen angelegt", anzahl)
	}
}

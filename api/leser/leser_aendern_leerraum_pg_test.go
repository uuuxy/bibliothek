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
	"bibliothek/internal/testdaten"
	"bibliothek/repository"
)

// Die Änderung kürzt Leerraum am Rand wie die Anlage (LeserAnlegen), auch das geschützte
// Leerzeichen. Sonst wäre „09H1 " aus der Akte eine andere Klasse als „09H1" aus Anlage und
// LUSD-Import, und ein Leerzeichen vor dem Jahrgang läse repository.AbschlussklasseSQL anders als
// repository.AbschlussJahrgang.
func TestLeserAendern_KuerztLeerraumWieDieAnlage(t *testing.T) {
	pool := pgtest.Pool(t)
	testdaten.LeereBestandsdaten(t, pool)
	ctx := context.Background()
	srv := &Tueren{Basis: &tuer.Basis{DB: &db.Verbindung{Pool: pool}}}

	var id string
	if err := pool.QueryRow(ctx, `
		INSERT INTO schueler (vorname, nachname, klasse, barcode_id, abgaenger_jahr)
		VALUES ('Rand', 'Test', '07a', 'RAND-1', 2030)
		RETURNING id`).Scan(&id); err != nil {
		t.Fatalf("Schüler anlegen: %v", err)
	}

	// Das Formular schickt, was im Feld steht; hier mit Leerzeichen und geschütztem Leerzeichen
	// am Rand.
	rumpf := `{"vorname":" Anna ","nachname":"\u00a0Test\u00a0","klasse":" 09H1\u00a0","barcode_id":" RAND-2 "}`
	req := httptest.NewRequest(http.MethodPatch, "/api/schueler/"+id, strings.NewReader(rumpf))
	req.SetPathValue("id", id)
	req = req.WithContext(context.WithValue(req.Context(), auth.ClaimsSchluessel,
		&auth.Claims{KontoID: id, Rolle: auth.RolleAdmin}))
	rec := httptest.NewRecorder()
	srv.LeserAendern(repository.NewProtokollAbfragen(pool))(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("PATCH gab %d: %s", rec.Code, rec.Body.String())
	}

	var vorname, nachname, klasse, ausweis string
	if err := pool.QueryRow(ctx, `SELECT vorname, nachname, klasse, barcode_id FROM schueler WHERE id = $1`, id).
		Scan(&vorname, &nachname, &klasse, &ausweis); err != nil {
		t.Fatalf("zurücklesen: %v", err)
	}
	for _, f := range []struct{ feld, ist, soll string }{
		{"vorname", vorname, "Anna"}, {"nachname", nachname, "Test"},
		{"klasse", klasse, "09H1"}, {"barcode_id", ausweis, "RAND-2"},
	} {
		if f.ist != f.soll {
			t.Errorf("%s = %q, erwartet %q: Die Änderung speichert Leerraum am Rand, die Anlage nicht", f.feld, f.ist, f.soll)
		}
	}
}

package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"bibliothek/auth"
	"bibliothek/db"

	"github.com/jackc/pgx/v5/pgxpool"
)

// „Neuer Leser" legt bei einer Lehrkraft das Anmeldekonto gleich mit an — und genau das
// verhindert den Doppeleintrag.
//
// Der Fall, um den es geht (Peter, 16.09.2026): Eine Lehrkraft wird von Hand in die
// Leserdatei eingetragen und meldet sich später über „Mein Portal" selbst an. Bis heute
// entstand dabei ein zweiter Eintrag — `legeZugangsanfrageAn` schreibt das Konto, und der
// Wächter trg_benutzer_hat_leserzeile hängt eine FRISCHE Leserzeile daran, ohne zu prüfen,
// ob die Person schon dasteht. Ausweis und Ausleihen blieben am ersten Eintrag, die
// Anmeldung am zweiten, und niemand merkte es.
//
// Die Lösung ist Peters: Die Schul-E-Mail ist beim Anlegen Pflicht. Damit entsteht das
// Konto sofort — und weil der Anmeldeweg eine Zugangsanfrage NUR anlegt, wenn zu der
// Adresse gar kein Konto existiert, findet die spätere Selbstanmeldung das vorhandene und
// legt nichts Zweites an.
//
// Dass die Selbstanmeldung selbst keine Waisen hinterlässt, prüft
// auth/selbstanmeldung_pg_test.go am echten Anmeldeweg. Hier wird die andere Hälfte
// geprüft: dass die Handanlage überhaupt ein Konto erzeugt, an genau EINER Leserzeile.
func TestKollegiumAnlegen_KontoUndKeinDoppel(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	ctx := context.Background()
	srv := &Server{DB: &db.Database{Pool: pool}}

	anlegen := func(t *testing.T, rumpf string, rolle auth.Role) *httptest.ResponseRecorder {
		t.Helper()
		req := httptest.NewRequest(http.MethodPost, "/api/schueler", strings.NewReader(rumpf))
		req = req.WithContext(context.WithValue(req.Context(), auth.ClaimsContextKey,
			&auth.Claims{UserID: "00000000-0000-0000-0000-0000000000aa", Rolle: rolle}))
		rec := httptest.NewRecorder()
		srv.CreateStudentHandler()(rec, req)
		return rec
	}

	t.Run("Lehrkraft ohne E-Mail wird abgewiesen", func(t *testing.T) {
		rec := anlegen(t, `{"vorname":"Ohne","nachname":"Mail","art":"lehrkraft"}`, auth.RoleAdmin)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("Antwort %d (erwartet 400): %s", rec.Code, rec.Body.String())
		}
		if n := zaehleLeser(t, pool, "Mail"); n != 0 {
			t.Fatalf("trotz Ablehnung wurde eine Leserzeile angelegt (%d)", n)
		}
	})

	t.Run("Schüler bekommt kein Konto und keine Adresse", func(t *testing.T) {
		rec := anlegen(t, `{"vorname":"Mit","nachname":"Mail","art":"schueler","klasse":"07A",
			"geburtsdatum":"2012-05-04","email":"kind@schule.invalid"}`, auth.RoleAdmin)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("Antwort %d (erwartet 400): %s", rec.Code, rec.Body.String())
		}
	})

	t.Run("Admin legt eine Lehrkraft mit Konto an — aktiv", func(t *testing.T) {
		rec := anlegen(t, `{"vorname":"Katrin","nachname":"Wendlandt","art":"lehrkraft",
			"email":"katrin.wendlandt@schule.invalid"}`, auth.RoleAdmin)
		if rec.Code != http.StatusCreated {
			t.Fatalf("Antwort %d: %s", rec.Code, rec.Body.String())
		}

		var aktiv bool
		var rolleStr, leserID string
		if err := pool.QueryRow(ctx, `
			SELECT aktiv, rolle::text, leser_id::text FROM benutzer
			 WHERE lower(email) = 'katrin.wendlandt@schule.invalid'`).Scan(&aktiv, &rolleStr, &leserID); err != nil {
			t.Fatalf("Konto lesen: %v", err)
		}
		if !aktiv {
			t.Error("der Administrator darf freischalten — das Konto muss aktiv sein")
		}
		if rolleStr != "kollegium" {
			t.Errorf("Rolle ist %q, erwartet kollegium — eine Rolle vergibt der Admin eigens", rolleStr)
		}

		// Der Kern: EINE Leserzeile, und das Konto hängt an genau ihr. Liefe der Wächter
		// an, stünden hier zwei.
		if n := zaehleLeser(t, pool, "Wendlandt"); n != 1 {
			t.Fatalf("erwartet genau eine Leserzeile, gefunden: %d", n)
		}
		var art string
		if err := pool.QueryRow(ctx, `SELECT art FROM leser WHERE id = $1`, leserID).Scan(&art); err != nil {
			t.Fatalf("Leserzeile des Kontos: %v", err)
		}
		if art != "lehrkraft" {
			t.Errorf("das Konto hängt an einer Zeile der Art %q — der Wächter hat eine zweite angelegt", art)
		}
	})

	t.Run("belegte Adresse ist eine Auskunft, kein Fehler", func(t *testing.T) {
		rec := anlegen(t, `{"vorname":"Katrin","nachname":"Zweitname","art":"lehrkraft",
			"email":"katrin.wendlandt@schule.invalid"}`, auth.RoleAdmin)
		if rec.Code != http.StatusConflict {
			t.Fatalf("Antwort %d (erwartet 409): %s", rec.Code, rec.Body.String())
		}
		if !strings.Contains(rec.Body.String(), "bereits ein Zugang") {
			t.Errorf("die Meldung soll auf den vorhandenen Eintrag zeigen: %s", rec.Body.String())
		}
		if n := zaehleLeser(t, pool, "Zweitname"); n != 0 {
			t.Fatalf("die abgewiesene Anlage hat eine Leserzeile hinterlassen (%d)", n)
		}
	})
}

// zaehleLeser zählt die Leserzeilen zu einem Nachnamen. Gefragt wird die TABELLE `leser`
// und nicht die Sicht `schueler`: Die zeigt nur Schüler, ein Kollege stünde nicht darin —
// der Test könnte den Doppeleintrag gar nicht sehen, den er beweisen soll.
func zaehleLeser(t *testing.T, pool *pgxpool.Pool, nachname string) int {
	t.Helper()
	var n int
	if err := pool.QueryRow(context.Background(),
		`SELECT count(*) FROM leser WHERE nachname = $1 AND deleted_at IS NULL`, nachname).Scan(&n); err != nil {
		t.Fatalf("Leserzeilen zählen: %v", err)
	}
	return n
}

package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"bibliothek/auth"
	"bibliothek/db"
	"bibliothek/repository"
)

// Eine Ausweisnummer gehört genau einer Person — Schüler wie Kollegium. Bis Migration 118
// prüfte jeder Schreibweg nur seine eigene Tabelle, die Theke suchte unter beiden und lud
// still den Schüler. Seit Migration 125 stehen alle Leser in EINER Tabelle; die Zusage ist
// dieselbe geblieben, sie hängt jetzt am partiellen Index statt am Trigger.
//
// Hier über die echten Handler: Die Bibliothek bekommt eine Auskunft, keinen 500.
func TestAusweisnummer_UeberSchuelerUndKollegium(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	ctx := context.Background()
	srv := &Server{DB: &db.Database{Pool: pool}}
	userRepo := repository.NewUserRepository(pool)
	admin := &auth.Claims{UserID: "00000000-0000-0000-0000-00000000a118", Rolle: auth.RoleAdmin}
	t.Cleanup(func() {
		if _, err := pool.Exec(ctx, `DELETE FROM leser WHERE id IN
			(SELECT leser_id FROM benutzer WHERE email LIKE '%@ausweis118.invalid')`); err != nil {
			t.Errorf("Leserzeilen aufräumen: %v", err)
		}
		if _, err := pool.Exec(ctx, `DELETE FROM benutzer WHERE email LIKE '%@ausweis118.invalid'`); err != nil {
			t.Errorf("Lehrkräfte aufräumen: %v", err)
		}
	})

	// Der Ausweis einer Lehrkraft steht an ihrer LESERZEILE (Migration 125) — zwei
	// Anweisungen, weil eine schreibende CTE ihre eigene Zeile noch nicht sieht.
	lehrkraft := func(t *testing.T, barcode, email string) string {
		t.Helper()
		var id, leserID string
		if err := pool.QueryRow(ctx, `INSERT INTO benutzer (vorname, nachname, email, rolle, aktiv)
			VALUES ('Leh', 'Rer', $1, 'kollegium', true) RETURNING id, leser_id::text`, email).
			Scan(&id, &leserID); err != nil {
			t.Fatalf("Lehrkraft anlegen: %v", err)
		}
		if _, err := pool.Exec(ctx, `UPDATE leser SET barcode_id = $1 WHERE id = $2`, barcode, leserID); err != nil {
			t.Fatalf("Ausweis der Lehrkraft eintragen: %v", err)
		}
		return id
	}
	schueler := func(t *testing.T, barcode string) string {
		t.Helper()
		var id string
		if err := pool.QueryRow(ctx, `INSERT INTO schueler (barcode_id, vorname, nachname, klasse, abgaenger_jahr)
			VALUES ($1, 'Sch', 'Ueler', '7a', 2030) RETURNING id`, barcode).Scan(&id); err != nil {
			t.Fatalf("Schüler anlegen: %v", err)
		}
		return id
	}
	fahre := func(h http.Handler, methode, pfad, id, body string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(methode, pfad, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		if id != "" {
			req.SetPathValue("id", id)
		}
		req = req.WithContext(context.WithValue(req.Context(), auth.ClaimsContextKey, admin))
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		return rec
	}
	pruefe := func(t *testing.T, rec *httptest.ResponseRecorder, status int, teil string) {
		t.Helper()
		if rec.Code != status {
			t.Fatalf("Status %d, erwartet %d: %s", rec.Code, status, rec.Body.String())
		}
		if teil != "" && !strings.Contains(rec.Body.String(), teil) {
			t.Errorf("Meldung nennt den Grund nicht (%q fehlt): %s", teil, rec.Body.String())
		}
	}

	t.Run("Schüler anlegen mit der Nummer einer Lehrkraft", func(t *testing.T) {
		lehrkraft(t, "AW118-A", "a@ausweis118.invalid")
		code, body := createStudent(t, srv,
			`{"vorname":"Ada","nachname":"Anlage","klasse":"7a","geburtsdatum":"2012-01-02","barcode_id":"AW118-A"}`)
		if code != http.StatusBadRequest || !strings.Contains(body, "bereits verwendet") {
			t.Fatalf("erwartet 400 „bereits verwendet“, war %d: %s", code, body)
		}
	})

	t.Run("Schülerakte: Nummer einer Lehrkraft eintragen", func(t *testing.T) {
		lehrkraft(t, "AW118-B", "b@ausweis118.invalid")
		id := schueler(t, "AW118-B-S")
		rec := fahre(srv.PatchStudentHandler(repository.NewAuditRepository(pool)),
			http.MethodPatch, "/api/schueler/"+id, id, `{"barcode_id":"AW118-B"}`)
		pruefe(t, rec, http.StatusConflict, "Ausweis")
	})

	t.Run("Wiederherstellen, während eine Lehrkraft die Nummer trägt", func(t *testing.T) {
		id := schueler(t, "AW118-C")
		bearbeiter := lehrkraft(t, "AW118-C-L", "c-bearbeiter@ausweis118.invalid")
		if err := repository.NewAuditRepository(pool).DeleteStudent(ctx, id, bearbeiter, "Test"); err != nil {
			t.Fatalf("in den Papierkorb legen: %v", err)
		}
		lehrkraft(t, "AW118-C", "c@ausweis118.invalid")
		rec := fahre(srv.RestoreStudentHandler(), http.MethodPost, "/api/schueler/"+id+"/restore", id, "")
		pruefe(t, rec, http.StatusConflict, "Ausweis")
	})

	t.Run("Lehrkraft anlegen mit der Nummer eines Schülers", func(t *testing.T) {
		schueler(t, "AW118-D")
		rec := fahre(srv.CreateUserHandler(userRepo), http.MethodPost, "/api/benutzer", "",
			`{"barcode_id":"AW118-D","vorname":"Dora","nachname":"Doppel","email":"d@ausweis118.invalid","rolle":"kollegium"}`)
		pruefe(t, rec, http.StatusBadRequest, "bereits")
	})

	t.Run("Lehrkraft ändern auf die Nummer eines Schülers", func(t *testing.T) {
		schueler(t, "AW118-E")
		id := lehrkraft(t, "AW118-E-L", "e@ausweis118.invalid")
		rec := fahre(srv.UpdateUserHandler(userRepo), http.MethodPut, "/api/benutzer/"+id, id,
			`{"barcode_id":"AW118-E","vorname":"Leh","nachname":"Rer","email":"e@ausweis118.invalid","rolle":"kollegium","aktiv":true}`)
		pruefe(t, rec, http.StatusBadRequest, "bereits")
	})

	t.Run("Gegenprobe: die Nummer eines gelöschten Schülers ist frei", func(t *testing.T) {
		id := schueler(t, "AW118-F")
		bearbeiter := lehrkraft(t, "AW118-F-L", "f-bearbeiter@ausweis118.invalid")
		if err := repository.NewAuditRepository(pool).DeleteStudent(ctx, id, bearbeiter, "Test"); err != nil {
			t.Fatalf("in den Papierkorb legen: %v", err)
		}
		rec := fahre(srv.CreateUserHandler(userRepo), http.MethodPost, "/api/benutzer", "",
			`{"barcode_id":"AW118-F","vorname":"Fia","nachname":"Frei","email":"f@ausweis118.invalid","rolle":"kollegium"}`)
		pruefe(t, rec, http.StatusOK, "")
	})
}

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
	"bibliothek/repository"
)

// Wiederherstellen aus dem Papierkorb kann an einem aktiven Zwilling scheitern — und das
// ist eine Auskunft, kein Serverfehler.
//
// Drei Teilindizes gelten nur für AKTIVE Zeilen (WHERE deleted_at IS NULL): der
// Namensindex (seit Migration 108 in der Normalform suchnorm), die LUSD-ID und der
// Ausweis-Barcode. Solange die Zeile im Papierkorb liegt, ist ihr Platz frei — ein
// Import oder eine Handanlage kann ihn besetzen. Der Restore lief bis zum 12.09.2026
// blind hinein: 23505 → 500 „Ein interner Datenbankfehler ist aufgetreten" (Register,
// Bestands-Durchgang 10.09.2026). Für die Bibliothek war das eine Sackgasse — die
// Meldung nennt weder den Zwilling noch den Weg heraus (zusammenführen, umbenennen,
// Barcode neu vergeben).
func TestRestore_KollisionMitAktivemZwillingIstKonflikt(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	ctx := context.Background()
	srv := &Server{DB: &db.Database{Pool: pool}}
	auditRepo := repository.NewAuditRepository(pool)
	bearbeiter := seedPortalLehrkraft(t, pool, "restore-kollision@test.invalid")

	// Eine Zeile in den Papierkorb legen und danach ihren Platz besetzen.
	vorbereiten := func(t *testing.T, anlegen, zwilling string) string {
		t.Helper()
		var id string
		if err := pool.QueryRow(ctx, anlegen).Scan(&id); err != nil {
			t.Fatalf("Schüler anlegen: %v", err)
		}
		if err := auditRepo.DeleteStudent(ctx, id, bearbeiter, "Test"); err != nil {
			t.Fatalf("in den Papierkorb legen: %v", err)
		}
		if _, err := pool.Exec(ctx, zwilling); err != nil {
			t.Fatalf("Zwilling anlegen: %v", err)
		}
		return id
	}
	wiederherstellen := func(id string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodPost, "/api/schueler/"+id+"/restore", nil)
		req.SetPathValue("id", id)
		req = req.WithContext(context.WithValue(req.Context(), auth.ClaimsContextKey,
			&auth.Claims{UserID: bearbeiter}))
		rec := httptest.NewRecorder()
		srv.RestoreStudentHandler()(rec, req)
		return rec
	}
	pruefe := func(t *testing.T, rec *httptest.ResponseRecorder, erwarteterTeil string) {
		t.Helper()
		if rec.Code != http.StatusConflict {
			t.Fatalf("Status %d, erwartet 409 — der Zwilling ist eine Lage, kein Serverfehler: %s",
				rec.Code, rec.Body.String())
		}
		var body map[string]string
		if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
			t.Fatalf("Body ist kein JSON: %q", rec.Body.String())
		}
		if !strings.Contains(body["error"], erwarteterTeil) {
			t.Errorf("Meldung nennt den Grund nicht (%q fehlt): %q", erwarteterTeil, body["error"])
		}
	}

	t.Run("gleicher Name und gleiches Geburtsdatum", func(t *testing.T) {
		id := vorbereiten(t,
			`INSERT INTO schueler (barcode_id, vorname, nachname, klasse, abgaenger_jahr, geburtsdatum)
			 VALUES ('S-ZW-NAME-1', 'Anna', 'Müller', '7a', 2030, '2012-03-04') RETURNING id`,
			`INSERT INTO schueler (barcode_id, vorname, nachname, klasse, abgaenger_jahr, geburtsdatum)
			 VALUES ('S-ZW-NAME-2', 'anna', 'Mueller', '7a', 2030, '2012-03-04')`)
		pruefe(t, wiederherstellen(id), "Name")
	})

	t.Run("gleicher Ausweis-Barcode", func(t *testing.T) {
		id := vorbereiten(t,
			`INSERT INTO schueler (barcode_id, vorname, nachname, klasse, abgaenger_jahr)
			 VALUES ('S-ZW-BC', 'Ben', 'Bach', '7a', 2030) RETURNING id`,
			`INSERT INTO schueler (barcode_id, vorname, nachname, klasse, abgaenger_jahr)
			 VALUES ('S-ZW-BC', 'Bea', 'Bund', '7b', 2030)`)
		pruefe(t, wiederherstellen(id), "Barcode")
	})

	t.Run("ohne Zwilling gelingt die Wiederherstellung", func(t *testing.T) {
		var id string
		if err := pool.QueryRow(ctx,
			`INSERT INTO schueler (barcode_id, vorname, nachname, klasse, abgaenger_jahr)
			 VALUES ('S-ZW-FREI', 'Cem', 'Cord', '7a', 2030) RETURNING id`).Scan(&id); err != nil {
			t.Fatalf("Schüler anlegen: %v", err)
		}
		if err := auditRepo.DeleteStudent(ctx, id, bearbeiter, "Test"); err != nil {
			t.Fatalf("in den Papierkorb legen: %v", err)
		}
		if rec := wiederherstellen(id); rec.Code != http.StatusOK {
			t.Fatalf("Status %d, erwartet 200: %s", rec.Code, rec.Body.String())
		}
	})
}

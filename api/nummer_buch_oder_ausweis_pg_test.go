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

// Eine Nummer ist entweder ein Buch oder ein Ausweis (Migration 131). Die Regel liegt in
// der Datenbank (db/constraints_nummer_ueber_buch_und_ausweis_pg_test.go); hier geht es um
// die Türen, an denen ein Mensch eine Nummer tippt: Die Bibliothek bekommt eine Auskunft
// mit dem Grund (409), keinen 500 — und beim Wiederherstellen einen Satz mit dem Ausweg.
func TestNummerBuchOderAusweis_AnDenTueren(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	ctx := context.Background()
	srv := &Server{DB: &db.Database{Pool: pool}}
	admin := &auth.Claims{UserID: "00000000-0000-0000-0000-00000000a131", Rolle: auth.RoleAdmin}

	var titelID string
	if err := pool.QueryRow(ctx, `INSERT INTO buecher_titel (titel) VALUES ('Nummernprobe 131') RETURNING id`).Scan(&titelID); err != nil {
		t.Fatal(err)
	}
	exemplar := func(t *testing.T, barcode string) string {
		t.Helper()
		var id string
		if err := pool.QueryRow(ctx, `INSERT INTO buecher_exemplare (titel_id, barcode_id) VALUES ($1, $2) RETURNING id`, titelID, barcode).Scan(&id); err != nil {
			t.Fatalf("Exemplar %s anlegen: %v", barcode, err)
		}
		return id
	}
	schueler := func(t *testing.T, barcode string) string {
		t.Helper()
		var id string
		if err := pool.QueryRow(ctx, `INSERT INTO schueler (barcode_id, vorname, nachname, klasse, abgaenger_jahr)
			VALUES ($1, 'Num', 'Merprobe', '7a', 2030) RETURNING id`, barcode).Scan(&id); err != nil {
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
		if !strings.Contains(rec.Body.String(), teil) {
			t.Errorf("Meldung nennt den Grund nicht (%q fehlt): %s", teil, rec.Body.String())
		}
	}

	t.Run("Buch umetikettieren auf die Nummer eines Ausweises", func(t *testing.T) {
		schueler(t, "N131-A")
		ex := exemplar(t, "N131-A-B")
		rec := fahre(srv.UpdateCopyBarcodeHandler(repository.NewBookRepository(pool)),
			http.MethodPut, "/api/buecher/exemplare/"+ex+"/barcode", ex, `{"barcode":"N131-A"}`)
		pruefe(t, rec, http.StatusConflict, "Ausweis eines Lesers")
	})

	t.Run("Schülerakte: den Barcode eines Buchs als Ausweis eintragen", func(t *testing.T) {
		exemplar(t, "N131-B")
		id := schueler(t, "N131-B-S")
		rec := fahre(srv.PatchStudentHandler(repository.NewAuditRepository(pool)),
			http.MethodPatch, "/api/schueler/"+id, id, `{"barcode_id":"N131-B"}`)
		pruefe(t, rec, http.StatusConflict, "Barcode eines Buchs")
	})

	t.Run("Leser anlegen mit dem Barcode eines Buchs", func(t *testing.T) {
		exemplar(t, "N131-C")
		code, body := createStudent(t, srv,
			`{"vorname":"Ada","nachname":"Anlage","klasse":"7a","geburtsdatum":"2012-01-02","barcode_id":"N131-C"}`)
		if code != http.StatusConflict || !strings.Contains(body, "Barcode eines Buchs") {
			t.Fatalf("erwartet 409 „Barcode eines Buchs“, war %d: %s", code, body)
		}
	})

	t.Run("Wiederherstellen, während ein Buch die Nummer trägt", func(t *testing.T) {
		id := schueler(t, "N131-D")
		// Der Papierkorb protokolliert den Bearbeiter — ein echtes Konto, kein Platzhalter.
		var bearbeiter string
		if err := pool.QueryRow(ctx, `INSERT INTO benutzer (vorname, nachname, email, rolle, aktiv)
			VALUES ('Bea', 'Rbeiter', 'bearbeiter@nummer131.invalid', 'kollegium', true) RETURNING id`).Scan(&bearbeiter); err != nil {
			t.Fatalf("Bearbeiter anlegen: %v", err)
		}
		t.Cleanup(func() {
			if _, err := pool.Exec(ctx, `DELETE FROM leser WHERE id = (SELECT leser_id FROM benutzer WHERE id = $1)`, bearbeiter); err != nil {
				t.Errorf("Leserzeile des Bearbeiters aufräumen: %v", err)
			}
			if _, err := pool.Exec(ctx, `DELETE FROM benutzer WHERE id = $1`, bearbeiter); err != nil {
				t.Errorf("Bearbeiter aufräumen: %v", err)
			}
		})
		if err := repository.NewAuditRepository(pool).DeleteStudent(ctx, id, bearbeiter, "Test"); err != nil {
			t.Fatalf("in den Papierkorb legen: %v", err)
		}
		exemplar(t, "N131-D")
		rec := fahre(srv.RestoreStudentHandler(), http.MethodPost, "/api/schueler/"+id+"/restore", id, "")
		pruefe(t, rec, http.StatusConflict, "Barcode eines Buchs")
	})
}

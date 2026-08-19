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

// lusd_id ist der einzige Zuordnungsschlüssel des Landesabgleichs (kein Adress-/
// Geburtsdatum-Fallback im Import). Sie darf über den Schüler-PATCH nur KONTROLLIERT
// nachgetragen werden: nachtragbar nur wenn leer (Waise adoptieren), eindeutig, und
// jeder Nachtrag wird auditiert. Ein bestehender Wert wird weder überschrieben noch
// geleert. Betreiber-Entscheidung 18.08.2026; der automatische Gegenpart ist das
// Import-Auto-Matching über Name+Geburtsdatum.
func TestLusdIDKontrolliertNachtragbar(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	ctx := context.Background()

	var adminID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO benutzer (vorname, nachname, email, rolle, aktiv)
		VALUES ('Lusd', 'Admin', 'lusd-admin@test.invalid', 'admin', true)
		ON CONFLICT (email) DO UPDATE SET vorname = EXCLUDED.vorname
		RETURNING id`).Scan(&adminID); err != nil {
		t.Fatalf("Test-Admin: %v", err)
	}

	neueID := func(barcode string, lusd *string) string {
		var id string
		if err := pool.QueryRow(ctx, `
			INSERT INTO schueler (vorname, nachname, klasse, barcode_id, abgaenger_jahr, lusd_id)
			VALUES ('Wai','Se',$1,$2,2030,$3) RETURNING id`, "5a", barcode, lusd).Scan(&id); err != nil {
			t.Fatalf("Schüler anlegen: %v", err)
		}
		return id
	}

	auditRepo := repository.NewAuditRepository(pool)
	patch := func(id, body string) *httptest.ResponseRecorder {
		srv := &Server{DB: &db.Database{Pool: pool}}
		req := httptest.NewRequest(http.MethodPatch, "/x", strings.NewReader(body))
		req.SetPathValue("id", id)
		req = req.WithContext(context.WithValue(req.Context(), auth.ClaimsContextKey,
			&auth.Claims{UserID: adminID, Rolle: auth.RoleAdmin}))
		rec := httptest.NewRecorder()
		srv.PatchStudentHandler(auditRepo)(rec, req)
		return rec
	}
	lusdVon := func(id string) string {
		var s string
		if err := pool.QueryRow(ctx, "SELECT COALESCE(lusd_id,'') FROM schueler WHERE id=$1", id).Scan(&s); err != nil {
			t.Fatalf("lusd_id lesen: %v", err)
		}
		return s
	}
	auditZaehler := func() int {
		var n int
		if err := pool.QueryRow(ctx, "SELECT count(*) FROM audit_logs WHERE aktion='LUSD_ID_NACHGETRAGEN'").Scan(&n); err != nil {
			t.Fatalf("Audit zählen: %v", err)
		}
		return n
	}

	t.Run("Waise adoptieren: leer -> gesetzt, auditiert", func(t *testing.T) {
		id := neueID("W-1", nil)
		vorher := auditZaehler()
		rec := patch(id, `{"lusd_id":"LUSD-NEU-1"}`)
		if rec.Code != http.StatusOK {
			t.Fatalf("erwartet 200, war %d: %s", rec.Code, rec.Body.String())
		}
		if lusdVon(id) != "LUSD-NEU-1" {
			t.Errorf("lusd_id nicht nachgetragen: %q", lusdVon(id))
		}
		if auditZaehler() != vorher+1 {
			t.Error("Nachtrag muss genau 1 Audit-Eintrag erzeugen")
		}
	})

	t.Run("bestehende lusd_id kann nicht geaendert werden", func(t *testing.T) {
		alt := "LUSD-ALT"
		id := neueID("W-2", &alt)
		rec := patch(id, `{"lusd_id":"LUSD-ANDERS"}`)
		if rec.Code != http.StatusForbidden {
			t.Fatalf("erwartet 403, war %d: %s", rec.Code, rec.Body.String())
		}
		if lusdVon(id) != "LUSD-ALT" {
			t.Errorf("lusd_id wurde trotz 403 verändert: %q", lusdVon(id))
		}
	})

	t.Run("bestehende lusd_id kann nicht geleert werden", func(t *testing.T) {
		alt := "LUSD-BLEIBT"
		id := neueID("W-3", &alt)
		rec := patch(id, `{"lusd_id":""}`)
		if rec.Code != http.StatusForbidden {
			t.Fatalf("erwartet 403, war %d: %s", rec.Code, rec.Body.String())
		}
		if lusdVon(id) != "LUSD-BLEIBT" {
			t.Errorf("lusd_id wurde geleert: %q", lusdVon(id))
		}
	})

	t.Run("Dublette wird als 409 abgewiesen", func(t *testing.T) {
		belegt := "LUSD-BELEGT"
		_ = neueID("W-4a", &belegt)
		waise := neueID("W-4b", nil)
		rec := patch(waise, `{"lusd_id":"LUSD-BELEGT"}`)
		if rec.Code != http.StatusConflict {
			t.Fatalf("erwartet 409, war %d: %s", rec.Code, rec.Body.String())
		}
		if lusdVon(waise) != "" {
			t.Errorf("Waise bekam trotz Dublette eine lusd_id: %q", lusdVon(waise))
		}
	})

	t.Run("unveraenderte lusd_id im Formular ist ein No-op (kein Audit)", func(t *testing.T) {
		gleich := "LUSD-GLEICH"
		id := neueID("W-5", &gleich)
		vorher := auditZaehler()
		// Formular schickt die bestehende lusd_id UND eine echte Änderung mit.
		rec := patch(id, `{"lusd_id":"LUSD-GLEICH","klasse":"6b"}`)
		if rec.Code != http.StatusOK {
			t.Fatalf("erwartet 200, war %d: %s", rec.Code, rec.Body.String())
		}
		if auditZaehler() != vorher {
			t.Error("ein No-op auf lusd_id darf keinen Audit-Eintrag erzeugen")
		}
	})
}

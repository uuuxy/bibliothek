package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"bibliothek/auth"
	"bibliothek/db"
	"bibliothek/sse"
)

// „Mahnbriefe drucken" zählt die Mahnung. Die Route verlangt deshalb das Recht des
// Mahnversands (create_orders) und nicht das Leserecht, an dem die Seite hängt
// (view_students). Am echten Router: Eine Rolle mit view_students und ohne create_orders
// bekommt 403 vom Rechte-Wächter, und an der Ausleihe ändert sich nichts.
//
// Gegenprobe mit derselben Sitzung und derselben Anfrage: Mit create_orders kommt der Brief,
// und die Mahnung ist gezählt. Ohne sie bewiese das 403 nur, dass die Anfrage scheitert.
func TestMahnbriefeDrucken_VerlangtDasRechtDesMahnversands(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	t.Setenv("RATE_LIMIT", "100000")
	ctx := context.Background()

	if err := (&db.Database{Pool: pool}).InitPermissions(ctx); err != nil {
		t.Fatalf("InitPermissions: %v", err)
	}
	authenticator, err := auth.NewAuthenticator(
		"mahnbrief-recht-testgeheimnis-mind-32-bytes!", pool, time.Hour)
	if err != nil {
		t.Fatalf("Authenticator: %v", err)
	}
	router := NewServer(&db.Database{Pool: pool}, authenticator, sse.NewBroker(), false).Routes()

	var kontoID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO benutzer (vorname, nachname, email, rolle, aktiv)
		VALUES ('Mara', 'Mahnrecht', 'mara@mahnbrief-recht.invalid', 'mitarbeiter', true)
		RETURNING id`).Scan(&kontoID); err != nil {
		t.Fatalf("Konto anlegen: %v", err)
	}
	token, err := authenticator.GenerateToken(kontoID, "MAHNRECHT-1", auth.RoleMitarbeiter, "")
	if err != nil {
		t.Fatalf("Sitzung: %v", err)
	}
	t.Cleanup(func() {
		// Die Rechte der Rolle stehen danach wieder auf der Vorgabe.
		aufraeumen(t, pool, `DELETE FROM role_permissions WHERE role = 'MITARBEITER'`)
		for _, e := range db.RechteVorgabe {
			if e.Role == "MITARBEITER" {
				aufraeumen(t, pool, `INSERT INTO role_permissions (role, permission, allowed)
					VALUES ($1, $2, $3) ON CONFLICT (role, permission) DO NOTHING`, e.Role, e.Permission, e.Allowed)
			}
		}
		aufraeumen(t, pool, `DELETE FROM benutzer WHERE id = $1`, kontoID)
		InvalidatePermissionCache()
	})

	leser := seedSchueler(t, pool, "MR-1", "Mona", "7a")
	ausleihe := seedAusleihe(t, pool, leser, "Buch zum Mahnrecht", time.Now().AddDate(0, 0, -30))

	drucke := func() *httptest.ResponseRecorder {
		req := jsonPost("/api/admin/mahnungen/bulk-print", `{"ausleih_ids":["`+ausleihe+`"]}`)
		req.AddCookie(&http.Cookie{Name: "session_token", Value: token})
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		return rec
	}

	setzeGenauEinRecht(t, pool, "view_students")
	rec := drucke()
	if rec.Code != http.StatusForbidden {
		t.Fatalf("mit view_students allein: HTTP %d statt 403 — %q", rec.Code, firstBytes(rec.Body.Bytes(), 120))
	}
	if rumpf := rec.Body.String(); !strings.Contains(rumpf, "keine Berechtigung") {
		t.Fatalf("das 403 kam nicht vom Rechte-Wächter: %s", rumpf)
	}
	if stufe, datum := mahnState(t, pool, ausleihe); stufe != 0 || datum != nil {
		t.Errorf("abgewiesener Druck hat gezählt: Mahnstufe %d, Mahndatum %v", stufe, datum)
	}

	setzeGenauEinRecht(t, pool, "create_orders")
	rec = drucke()
	if rec.Code != http.StatusOK {
		t.Fatalf("mit create_orders: HTTP %d statt 200 — %s", rec.Code, rec.Body.String())
	}
	if kopf := rec.Body.Bytes(); len(kopf) < 4 || string(kopf[:4]) != "%PDF" {
		t.Fatalf("Antwort ist kein PDF (Kopf: %q)", firstBytes(rec.Body.Bytes(), 8))
	}
	if stufe, datum := mahnState(t, pool, ausleihe); stufe != 1 || datum == nil {
		t.Errorf("gedruckter Brief: Mahnstufe %d (erwartet 1), Mahndatum %v", stufe, datum)
	}
}

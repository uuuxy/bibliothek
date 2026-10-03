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
	"bibliothek/repository"
)

// Was das Löschen eines Zugangskontos ablehnt, über die Tür: das eigene Konto, das Konto
// eines Administrators durch jemanden ohne Adminrechte, ein Konto mit offenen Ausleihen und
// eine unbekannte Kennung. Jede Ablehnung lässt das Konto stehen; erst der letzte Fall löscht.
func TestBenutzerLoeschen_AblehnungenUeberDieTuer(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	ctx := context.Background()
	srv := &Server{DB: &db.Database{Pool: pool}}
	auditRepo := repository.NewAuditRepository(pool)
	userRepo := repository.NewUserRepository(pool)

	konto := func(vorname, rolle string) string {
		t.Helper()
		var id string
		if err := pool.QueryRow(ctx, `
			INSERT INTO benutzer (vorname, nachname, email, rolle, aktiv)
			VALUES ($1, 'Loeschtuer', $2, $3::benutzer_rolle, true)
			RETURNING id::text`, vorname, strings.ToLower(vorname)+"@loeschtuer.invalid", rolle).Scan(&id); err != nil {
			t.Fatalf("Konto %s anlegen: %v", vorname, err)
		}
		return id
	}
	admin, zweiterAdmin := konto("Ada", "admin"), konto("Alex", "admin")
	mitarbeiter := konto("Mia", "mitarbeiter")
	mitBuch, ohneBuch := konto("Bert", "kollegium"), konto("Kai", "kollegium")

	var leserMitBuch string
	if err := pool.QueryRow(ctx, `SELECT leser_id::text FROM benutzer WHERE id = $1`, mitBuch).Scan(&leserMitBuch); err != nil {
		t.Fatalf("Leserzeile des Kontos lesen: %v", err)
	}
	seedAusleihe(t, pool, leserMitBuch, "Handapparat Loeschtuer", time.Now().Add(30*24*time.Hour))

	loesche := func(ziel string, claims *auth.Claims) *httptest.ResponseRecorder {
		t.Helper()
		req := httptest.NewRequest(http.MethodDelete, "/api/benutzer/"+ziel, nil)
		req.SetPathValue("id", ziel)
		if claims != nil {
			req = req.WithContext(context.WithValue(req.Context(), auth.ClaimsContextKey, claims))
		}
		rec := httptest.NewRecorder()
		srv.DeleteUserHandler(auditRepo, userRepo)(rec, req)
		return rec
	}
	alsAdmin := &auth.Claims{UserID: admin, Rolle: auth.RoleAdmin}
	alsMitarbeiter := &auth.Claims{UserID: mitarbeiter, Rolle: auth.RoleMitarbeiter}

	for _, f := range []struct {
		name   string
		ziel   string
		claims *auth.Claims
		status int
		stueck string
	}{
		{"ohne Sitzung", ohneBuch, nil, http.StatusUnauthorized, "session"},
		{"das eigene Konto", admin, alsAdmin, http.StatusForbidden, "eigenes Konto"},
		{"ein Administrator, gelöscht ohne Adminrechte", zweiterAdmin, alsMitarbeiter, http.StatusForbidden, "Administrator"},
		{"ein Konto mit offener Ausleihe", mitBuch, alsAdmin, http.StatusConflict, "1 offen"},
		{"eine unbekannte Kennung", "00000000-0000-0000-0000-00000000dead", alsAdmin, http.StatusNotFound, "error"},
	} {
		rec := loesche(f.ziel, f.claims)
		if rec.Code != f.status {
			t.Errorf("%s: Status %d, erwartet %d: %s", f.name, rec.Code, f.status, rec.Body.String())
		}
		if !strings.Contains(rec.Body.String(), f.stueck) {
			t.Errorf("%s: die Antwort nennt %q nicht: %s", f.name, f.stueck, rec.Body.String())
		}
	}
	if n := zaehleZeilen(t, pool, `SELECT count(*) FROM benutzer WHERE email LIKE '%@loeschtuer.invalid'`); n != 5 {
		t.Fatalf("nach fünf Ablehnungen stehen %d von 5 Konten", n)
	}

	if rec := loesche(ohneBuch, alsMitarbeiter); rec.Code != http.StatusOK {
		t.Fatalf("ein Kollegiumskonto ohne Ausleihen: Status %d, erwartet 200: %s", rec.Code, rec.Body.String())
	}
	if n := zaehleZeilen(t, pool, `SELECT count(*) FROM benutzer WHERE id = $1`, ohneBuch); n != 0 {
		t.Errorf("das gelöschte Konto steht noch da")
	}
}

// Ein Konto zu ändern, das es nicht gibt, ist ein 404 und hinterlässt keinen Eintrag im
// Protokoll: Der Eintrag beschriebe eine Änderung, die nie stattfand.
func TestBenutzerAendern_UnbekannteKennungOhneProtokoll(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	ctx := context.Background()
	srv := &Server{DB: &db.Database{Pool: pool}}

	var admin string
	if err := pool.QueryRow(ctx, `
		INSERT INTO benutzer (vorname, nachname, email, rolle, aktiv)
		VALUES ('Ada', 'Aendertuer', 'ada@aendertuer.invalid', 'admin', true) RETURNING id::text`).Scan(&admin); err != nil {
		t.Fatalf("Konto anlegen: %v", err)
	}
	const unbekannt = "00000000-0000-0000-0000-00000000dead"
	req := httptest.NewRequest(http.MethodPut, "/api/benutzer/"+unbekannt,
		strings.NewReader(`{"vorname":"Nie","nachname":"Da","email":"nie.da@aendertuer.invalid","rolle":"helfer","aktiv":true}`))
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("id", unbekannt)
	req = req.WithContext(context.WithValue(req.Context(), auth.ClaimsContextKey,
		&auth.Claims{UserID: admin, Rolle: auth.RoleAdmin}))
	rec := httptest.NewRecorder()
	srv.UpdateUserHandler(repository.NewUserRepository(pool))(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("Status %d, erwartet 404: %s", rec.Code, rec.Body.String())
	}
	if n := zaehleZeilen(t, pool, `SELECT count(*) FROM audit_logs WHERE aktion = 'USER_UPDATE'`); n != 0 {
		t.Errorf("%d Einträge USER_UPDATE zu einer Änderung, die es nicht gab", n)
	}
}

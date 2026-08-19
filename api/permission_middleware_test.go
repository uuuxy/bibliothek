package api

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"bibliothek/auth"
	"bibliothek/db"

	"github.com/jackc/pgx/v5"
	"github.com/pashagolub/pgxmock/v4"
)

const testJWTSecret = "0123456789abcdef0123456789abcdef" // 32 Byte

func setupRBAC(t *testing.T) (*Server, pgxmock.PgxPoolIface) {
	t.Helper()
	InvalidatePermissionCache() // globaler Cache darf nicht zwischen Tests lecken

	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock init: %v", err)
	}
	a, err := auth.NewAuthenticator(testJWTSecret, mock, time.Hour)
	if err != nil {
		t.Fatalf("authenticator init: %v", err)
	}
	return &Server{DB: &db.Database{Pool: mock}, Auth: a}, mock
}

// expectBlacklistPass erwartet die beiden Prüfungen in VerifyToken bei gültigem Token:
// die Blacklist ("nicht widerrufen") und direkt danach Kontostatus samt AKTUELLER Rolle.
// Die Rolle muss mitgeliefert werden, weil VerifyToken sie über die im Token signierte
// schreibt — die Autorisierung folgt der Datenbank, nicht dem Token.
func expectBlacklistPass(mock pgxmock.PgxPoolIface, rolle auth.Role) {
	mock.ExpectQuery("revoked_tokens").
		WithArgs(pgxmock.AnyArg()).
		WillReturnRows(pgxmock.NewRows([]string{"exists"}).AddRow(false))
	mock.ExpectQuery("SELECT aktiv, rolle FROM benutzer").
		WithArgs(pgxmock.AnyArg()).
		WillReturnRows(pgxmock.NewRows([]string{"aktiv", "rolle"}).AddRow(true, string(rolle)))
}

func reqWithToken(t *testing.T, s *Server, role auth.Role) *http.Request {
	t.Helper()
	token, err := s.Auth.GenerateToken("u1", "BC1", role)
	if err != nil {
		t.Fatalf("token generation: %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.AddCookie(&http.Cookie{Name: "session_token", Value: token})
	return req
}

// serve führt die Middleware aus und meldet zurück, ob der geschützte Handler erreicht wurde.
func serve(s *Server, permission string, req *http.Request) (*httptest.ResponseRecorder, bool) {
	reached := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reached = true
		w.WriteHeader(http.StatusOK)
	})
	rr := httptest.NewRecorder()
	s.RequirePermission(permission)(next).ServeHTTP(rr, req)
	return rr, reached
}

func TestRequirePermission_NoCookieUnauthorized(t *testing.T) {
	s, mock := setupRBAC(t)
	defer mock.Close()

	req := httptest.NewRequest(http.MethodGet, "/protected", nil) // ohne Cookie
	rr, reached := serve(s, "buch.loeschen", req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("ohne Cookie erwartet 401, bekam %d", rr.Code)
	}
	if reached {
		t.Error("geschützter Handler darf ohne Auth nicht erreicht werden")
	}
}

func TestRequirePermission_AdminBypassesCheck(t *testing.T) {
	s, mock := setupRBAC(t)
	defer mock.Close()

	expectBlacklistPass(mock, auth.RoleAdmin)
	// KEINE role_permissions-Abfrage erwartet — Admin hat implizit alle Rechte.

	req := reqWithToken(t, s, auth.RoleAdmin)
	rr, reached := serve(s, "buch.loeschen", req)

	if rr.Code != http.StatusOK || !reached {
		t.Errorf("Admin soll durchgelassen werden: code %d, reached %v", rr.Code, reached)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unerfüllte Erwartungen: %v", err)
	}
}

func TestRequirePermission_GrantedAllowsAccess(t *testing.T) {
	s, mock := setupRBAC(t)
	defer mock.Close()

	expectBlacklistPass(mock, auth.RoleKollegium)
	mock.ExpectQuery("role_permissions").
		WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg()).
		WillReturnRows(pgxmock.NewRows([]string{"allowed"}).AddRow(true))

	req := reqWithToken(t, s, auth.RoleKollegium)
	rr, reached := serve(s, "buch.ausleihen", req)

	if rr.Code != http.StatusOK || !reached {
		t.Errorf("gewährte Berechtigung soll durchlassen: code %d, reached %v", rr.Code, reached)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unerfüllte Erwartungen: %v", err)
	}
}

func TestRequirePermission_DeactivatedUserUnauthorized(t *testing.T) {
	s, mock := setupRBAC(t)
	defer mock.Close()

	// Blacklist ok, aber das Konto wurde nach Token-Ausstellung deaktiviert → VerifyToken
	// muss ablehnen (Echtzeit-Widerruf), bevor überhaupt Rechte geprüft werden.
	mock.ExpectQuery("revoked_tokens").
		WithArgs(pgxmock.AnyArg()).
		WillReturnRows(pgxmock.NewRows([]string{"exists"}).AddRow(false))
	mock.ExpectQuery("SELECT aktiv, rolle FROM benutzer").
		WithArgs(pgxmock.AnyArg()).
		WillReturnRows(pgxmock.NewRows([]string{"aktiv", "rolle"}).AddRow(false, string(auth.RoleKollegium)))

	req := reqWithToken(t, s, auth.RoleKollegium)
	rr, reached := serve(s, "buch.ausleihen", req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("deaktiviertes Konto erwartet 401, bekam %d", rr.Code)
	}
	if reached {
		t.Error("deaktiviertes Konto darf den geschützten Handler nicht erreichen")
	}
}

func TestRequirePermission_ExplicitlyDeniedForbidden(t *testing.T) {
	s, mock := setupRBAC(t)
	defer mock.Close()

	expectBlacklistPass(mock, auth.RoleKollegium)
	mock.ExpectQuery("role_permissions").
		WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg()).
		WillReturnRows(pgxmock.NewRows([]string{"allowed"}).AddRow(false))

	req := reqWithToken(t, s, auth.RoleKollegium)
	rr, reached := serve(s, "buch.loeschen", req)

	if rr.Code != http.StatusForbidden {
		t.Errorf("explizit verweigert erwartet 403, bekam %d", rr.Code)
	}
	if reached {
		t.Error("verweigerter Zugriff darf Handler nicht erreichen")
	}
}

func TestRequirePermission_NoRowForbidden(t *testing.T) {
	s, mock := setupRBAC(t)
	defer mock.Close()

	expectBlacklistPass(mock, auth.RoleKollegium)
	mock.ExpectQuery("role_permissions").
		WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg()).
		WillReturnError(pgx.ErrNoRows)

	req := reqWithToken(t, s, auth.RoleKollegium)
	rr, reached := serve(s, "unbekanntes.recht", req)

	if rr.Code != http.StatusForbidden {
		t.Errorf("fehlender Eintrag (kein Recht) erwartet 403, bekam %d", rr.Code)
	}
	if reached {
		t.Error("fehlendes Recht darf Handler nicht erreichen")
	}
}

func TestRequirePermission_TransientDBErrorIsServerError(t *testing.T) {
	s, mock := setupRBAC(t)
	defer mock.Close()

	expectBlacklistPass(mock, auth.RoleKollegium)
	mock.ExpectQuery("role_permissions").
		WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg()).
		WillReturnError(errors.New("connection reset"))

	req := reqWithToken(t, s, auth.RoleKollegium)
	rr, reached := serve(s, "buch.ausleihen", req)

	// Transiente DB-Fehler sind 500, nicht 403 — und dürfen NICHT als Negativ-Entscheidung gecacht werden.
	if rr.Code != http.StatusInternalServerError {
		t.Errorf("transienter DB-Fehler erwartet 500, bekam %d", rr.Code)
	}
	if reached {
		t.Error("bei DB-Fehler darf Handler nicht erreicht werden")
	}
}

func TestRequirePermission_DenyDecisionIsCached(t *testing.T) {
	s, mock := setupRBAC(t)
	defer mock.Close()

	// Erster Request: Blacklist + role_permissions (false) → 403, Entscheidung wird gecacht.
	expectBlacklistPass(mock, auth.RoleKollegium)
	mock.ExpectQuery("role_permissions").
		WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg()).
		WillReturnRows(pgxmock.NewRows([]string{"allowed"}).AddRow(false))

	// Zweiter Request: NUR Blacklist erwartet — role_permissions kommt aus dem Cache.
	expectBlacklistPass(mock, auth.RoleKollegium)

	for i := 0; i < 2; i++ {
		req := reqWithToken(t, s, auth.RoleKollegium)
		rr, reached := serve(s, "buch.loeschen", req)
		if rr.Code != http.StatusForbidden || reached {
			t.Errorf("Request %d: erwartet 403 ohne Handler-Zugriff, bekam %d reached=%v", i+1, rr.Code, reached)
		}
	}

	// Schlägt fehl, falls der zweite Request doch eine role_permissions-Abfrage ausgelöst hätte.
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Cache hat zweite DB-Abfrage nicht verhindert: %v", err)
	}
}

// invalidiereWaehrendQuery stellt das Rennen deterministisch nach: WÄHREND eine
// Berechtigungs-Entscheidung aus der DB unterwegs ist (nach dem Lesen, vor dem
// Cache-Schreiben), schlägt die Invalidierung zu — z. B. weil ein Admin genau jetzt
// die Rechte-Matrix geändert hat.
type invalidiereWaehrendQuery struct {
	pgxmock.PgxPoolIface
	imFlug func()
}

func (p *invalidiereWaehrendQuery) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	row := p.PgxPoolIface.QueryRow(ctx, sql, args...)
	if strings.Contains(sql, "role_permissions") {
		p.imFlug()
	}
	return row
}

// TestBerechtigungsCache_UeberholterLeserCachtNicht schließt das 60-Sekunden-Fenster
// aus dem Nebenläufigkeits-Audit (19.08.): Die Invalidierung nach einer Rechteänderung
// leert zwar den Cache — aber ein Request, der den ALTEN Stand kurz vorher aus der DB
// gelesen hatte, schrieb ihn DANACH wieder hinein. Das entzogene Recht wirkte dann bis
// zu 60 s weiter. Eine überholte Entscheidung darf den Cache nicht mehr erreichen.
func TestBerechtigungsCache_UeberholterLeserCachtNicht(t *testing.T) {
	s, mock := setupRBAC(t)
	defer mock.Close()

	// Die DB liefert den alten Stand: Recht (noch) gewährt.
	mock.ExpectQuery("role_permissions").
		WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg()).
		WillReturnRows(pgxmock.NewRows([]string{"allowed"}).AddRow(true))
	s.DB.Pool = &invalidiereWaehrendQuery{PgxPoolIface: mock, imFlug: InvalidatePermissionCache}

	const cacheKey = "KOLLEGIUM:buch.ausleihen"
	if _, err := s.ermittleUndCacheBerechtigung(t.Context(), "KOLLEGIUM", "buch.ausleihen", cacheKey); err != nil {
		t.Fatalf("Berechtigung ermitteln: %v", err)
	}

	// Der laufende Request durfte mit dem gelesenen Stand antworten — aber der Cache
	// darf die überholte Entscheidung nicht halten: Der nächste Request muss frisch lesen.
	if _, found := leseBerechtigungCache(cacheKey); found {
		t.Fatal("überholte Entscheidung wurde NACH der Invalidierung gecacht — das entzogene Recht wirkt bis zu 60 s weiter")
	}
}

func TestRequirePermission_BlacklistDBDownFailsClosed(t *testing.T) {
	s, mock := setupRBAC(t)
	defer mock.Close()

	// Blacklist-Prüfung schlägt fehl → IsBlacklisted gibt fail-closed true → Token gilt als widerrufen.
	mock.ExpectQuery("revoked_tokens").
		WithArgs(pgxmock.AnyArg()).
		WillReturnError(errors.New("db down"))

	req := reqWithToken(t, s, auth.RoleKollegium)
	rr, reached := serve(s, "buch.ausleihen", req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("fail-closed Blacklist erwartet 401, bekam %d", rr.Code)
	}
	if reached {
		t.Error("bei nicht verifizierbarem Token darf Handler nicht erreicht werden")
	}
}

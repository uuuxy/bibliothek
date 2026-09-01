package auth

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/pashagolub/pgxmock/v4"
)

// Ein DB-Ausfall ist kein falsches Passwort.
//
// Bis zum 31.08.2026 behandelte der Login JEDEN Fehler der benutzer-Abfrage wie „Zeile
// nicht vorhanden" und lief in den Selbstanmelde-Pfad; dessen Ablehnung endete als 401
// „invalid email or password" MIT gezähltem Fehlversuch. Bei einem DB-Aussetzer sperrte
// sich eine Lehrkraft mit korrektem Passwort nach fünf Versuchen selbst für 15 Minuten —
// dieselbe Massen-Selbstsperre wie beim Mailserver-Ausfall (Ausfallmatrix 20.08.2026),
// nur mit der Datenbank als Auslöser. Erwartet: 503 ohne recordFailure, wie beim IMAP-Fall.
func TestLogin_DBFehlerIstKeinFalschesPasswort(t *testing.T) {
	t.Setenv("IMAP_HOST", "mock")
	t.Setenv("APP_ENV", "test")

	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock: %v", err)
	}
	defer mock.Close()
	authn, err := NewAuthenticator("test-secret-mit-mindestens-32-zeichen!!", mock, time.Hour)
	if err != nil {
		t.Fatalf("Authenticator: %v", err)
	}

	for versuch := 1; versuch <= 6; versuch++ {
		mock.ExpectQuery(`SELECT id, coalesce\(barcode_id`).
			WithArgs(pgxmock.AnyArg()).
			WillReturnError(errors.New("dial tcp: connection refused"))

		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/login",
			strings.NewReader(`{"email":"dbfehler@test.invalid","password":"richtig"}`))
		req.Header.Set("Content-Type", "application/json")
		req.RemoteAddr = "203.0.113.7:4711"
		LoginHandler(mock, authn, false)(rec, req)

		if rec.Code == http.StatusTooManyRequests {
			t.Fatalf("Versuch %d: 429 — der DB-Ausfall wurde als Fehlversuch gezählt und hat das Konto gesperrt", versuch)
		}
		if rec.Code != http.StatusServiceUnavailable {
			t.Errorf("Versuch %d: HTTP %d, erwartet 503 (Anmeldedienst gestört, kein Passwortfehler): %s",
				versuch, rec.Code, rec.Body.String())
		}
	}
}

// Ein BESTEHENDES Konto wird nicht zur „Zugangsanfrage".
//
// legeZugangsanfrageAn setzte neuAngelegt bedingungslos — auch wenn der INSERT per
// ON CONFLICT gar nichts geschrieben hatte. Landete ein bestehendes Konto durch einen
// transienten Fehler der ersten Abfrage in diesem Pfad, bekam eine längst
// freigeschaltete Lehrkraft „Zugang beantragt — die Bibliothek muss ihn noch
// freischalten", und der Audit-Trail erhielt eine SELBSTANMELDUNG-Zeile für ein Konto,
// das sich nie selbst angemeldet hat.
func TestSelbstanmeldung_BestehendesKontoWirdKeineAnfrage(t *testing.T) {
	pool := pgPoolFuerSelbstanmeldung(t)
	t.Setenv("SELBSTANMELDUNG_DOMAIN", "test.invalid")
	ctx := t.Context()

	const mail = "bestandskonto@test.invalid"
	raeumeKontoAb(t, pool, mail)
	if _, err := pool.Exec(ctx, `
		INSERT INTO benutzer (vorname, nachname, email, rolle, aktiv)
		VALUES ('Bestands', 'Konto', $1, 'kollegium', true)`, mail); err != nil {
		t.Fatalf("Bestandskonto anlegen: %v", err)
	}

	u, err := legeZugangsanfrageAn(ctx, pool, mail)
	if err != nil {
		t.Fatalf("legeZugangsanfrageAn: %v", err)
	}
	if u.neuAngelegt {
		t.Error("bestehendes, aktives Konto wurde als „neu angelegt“ markiert — die Lehrkraft bekäme „Zugang beantragt“")
	}
	if !u.aktiv {
		t.Error("aktives Konto kam als inaktiv zurück")
	}
	var auditZeilen int
	if err := pool.QueryRow(ctx, `
		SELECT count(*) FROM audit_logs
		WHERE aktion = 'SELBSTANMELDUNG' AND details->>'email' = $1`, mail).Scan(&auditZeilen); err != nil {
		t.Fatalf("Audit zählen: %v", err)
	}
	if auditZeilen != 0 {
		t.Errorf("%d SELBSTANMELDUNG-Audit-Zeilen für ein Konto, das sich nie selbst angemeldet hat", auditZeilen)
	}
}

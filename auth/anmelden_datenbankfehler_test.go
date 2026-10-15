package auth

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/pashagolub/pgxmock/v5"
)

// Ein Ausfall der Datenbank ist kein falsches Passwort.
//
// Ein Fehler der Abfrage auf benutzer ist nicht „Zeile nicht vorhanden": Sonst liefe die
// Anmeldung in die Selbstanmeldung, deren Ablehnung als 401 „invalid email or password" mit
// gezähltem Fehlversuch endet, und bei einem Aussetzer der Datenbank sperrte sich eine Lehrkraft
// mit richtigem Passwort nach fünf Versuchen für 15 Minuten aus. Erwartet: 503 ohne
// Fehlversuch, wie beim Ausfall des Mailservers.
func TestAnmelden_DatenbankfehlerIstKeinFalschesPasswort(t *testing.T) {
	t.Setenv("IMAP_HOST", "mock")
	t.Setenv("APP_ENV", "test")

	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock: %v", err)
	}
	defer mock.Close()
	anmeldung, err := NewAnmeldung("test-secret-mit-mindestens-32-zeichen!!", mock, time.Hour)
	if err != nil {
		t.Fatalf("NewAnmeldung: %v", err)
	}

	for versuch := 1; versuch <= 6; versuch++ {
		mock.ExpectQuery(`SELECT id, coalesce\(barcode_id`).
			WithArgs(pgxmock.AnyArg()).
			WillReturnError(errors.New("dial tcp: connection refused"))

		rec := httptest.NewRecorder()
		anfrage := httptest.NewRequest(http.MethodPost, "/login",
			strings.NewReader(`{"email":"dbfehler@test.invalid","password":"richtig"}`))
		anfrage.Header.Set("Content-Type", "application/json")
		anfrage.RemoteAddr = "203.0.113.7:4711"
		Anmelden(mock, anmeldung, false)(rec, anfrage)

		if rec.Code == http.StatusTooManyRequests {
			t.Fatalf("Versuch %d: 429 — der DB-Ausfall wurde als Fehlversuch gezählt und hat das Konto gesperrt", versuch)
		}
		if rec.Code != http.StatusServiceUnavailable {
			t.Errorf("Versuch %d: HTTP %d, erwartet 503 (Anmeldedienst gestört, kein Passwortfehler): %s",
				versuch, rec.Code, rec.Body.String())
		}
	}
}

// Ein bestehendes Konto wird nicht zur Zugangsanfrage.
//
// neuAngelegt gilt nur, wenn der INSERT wirklich eine Zeile geschrieben hat, nicht schon, wenn
// ON CONFLICT nichts tat. Sonst bekäme eine längst freigeschaltete Lehrkraft, deren Konto nach
// einem vorübergehenden Fehler der ersten Abfrage hier landet, „Zugang beantragt — die
// Bibliothek muss ihn noch freischalten", und das Protokoll erhielte eine Zeile SELBSTANMELDUNG
// für ein Konto, das sich nie selbst angemeldet hat.
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

	konto, err := legeZugangsanfrageAn(ctx, pool, mail)
	if err != nil {
		t.Fatalf("legeZugangsanfrageAn: %v", err)
	}
	if konto.neuAngelegt {
		t.Error("bestehendes, aktives Konto wurde als „neu angelegt“ markiert — die Lehrkraft bekäme „Zugang beantragt“")
	}
	if !konto.aktiv {
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

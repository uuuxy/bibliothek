package auth

import (
	"bibliothek/internal/testdaten"
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
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
		mock.ExpectQuery(`SELECT b\.id, b\.rolle`).
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
	// Sonst endete die Anmeldung an einer unerwarteten Abfrage, nicht am Ausfall, den der Test
	// nachstellt.
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Abfrage des Kontos nicht gestellt: %v", err)
	}
}

// Bricht der Browser die Anmeldung ab, während das Konto gelesen wird, ist die Anmeldung nicht
// gestört: 499 statt 503, und kein Fehlversuch. Mit 503 stünde der Abbruch als ERROR im Log und
// löste die tägliche Alarm-Mail aus.
func TestAnmelden_AbbruchDesBrowsersIstKeinAusfall(t *testing.T) {
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
	mock.ExpectQuery(`SELECT b\.id, b\.rolle`).
		WithArgs(pgxmock.AnyArg()).
		WillReturnError(fmt.Errorf("zeile lesen: %w", context.Canceled))

	rec := httptest.NewRecorder()
	anfrage := httptest.NewRequest(http.MethodPost, "/login",
		strings.NewReader(`{"email":"abbruch@test.invalid","password":"richtig"}`))
	anfrage.Header.Set("Content-Type", "application/json")
	anfrage.RemoteAddr = "203.0.113.8:4711"
	Anmelden(mock, anmeldung, false)(rec, anfrage)

	// 499 ist der Status für „Browser hat abgebrochen" (apierrors.statusAbgebrochen).
	if rec.Code != 499 {
		t.Errorf("Anmelden bei abgebrochener Anfrage = %d, erwartet 499: %s", rec.Code, rec.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Abfrage des Kontos nicht gestellt: %v", err)
	}
}

// Ein Konto, das ohne Administrator entstand, und kein Eintrag im Protokoll: Das ist ein Fehler,
// den jemand ansehen muss (docs/STIL.md, Abschnitt 11), wie in tuer.Basis.ProtokolliereVerwaltung.
// Als Warnung stünde er nicht auf der Seite „Fehler im Log“.
func TestSelbstanmeldung_ProtokollFehltStehtAlsFehler(t *testing.T) {
	t.Setenv(selbstanmeldeDomainVariable, "test.invalid")
	var protokoll bytes.Buffer
	vorher := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&protokoll, nil)))
	t.Cleanup(func() { slog.SetDefault(vorher) })

	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock: %v", err)
	}
	defer mock.Close()
	const mail = "ohne.protokoll@test.invalid"
	mock.ExpectExec(`INSERT INTO benutzer`).
		WithArgs("Ohne", "Protokoll", mail).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))
	mock.ExpectQuery(`SELECT b\.id, b\.rolle`).
		WithArgs(mail).
		WillReturnRows(pgxmock.NewRows([]string{"id", "rolle", "vorname", "nachname", "aktiv", "email", "beantragt"}).
			AddRow("00000000-0000-0000-0000-0000000000a1", "kollegium", "Ohne", "Protokoll", false, mail, true))
	mock.ExpectExec(`INSERT INTO audit_logs`).
		WithArgs(pgxmock.AnyArg(), "SELBSTANMELDUNG", pgxmock.AnyArg(), pgxmock.AnyArg()).
		WillReturnError(errors.New("audit_logs nicht beschreibbar"))

	konto, err := legeZugangsanfrageAn(t.Context(), mock, mail)
	if err != nil {
		t.Fatalf("legeZugangsanfrageAn: %v — das fehlende Protokoll darf die Anmeldung nicht aufhalten", err)
	}
	if !konto.neuAngelegt {
		t.Error("das neue Konto kam nicht als neu angelegt zurück")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Schreiben des Protokolls nicht versucht: %v", err)
	}
	zeile := `level=ERROR msg="Selbstanmeldung: Audit-Zeile konnte nicht geschrieben werden" fehler="audit_logs nicht beschreibbar"`
	if !strings.Contains(protokoll.String(), zeile) {
		t.Errorf("im Log fehlt %s:\n%s", zeile, protokoll.String())
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
	testdaten.LegeKontoAn(t, pool, "Bestands", "Konto", mail, "kollegium", true)

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

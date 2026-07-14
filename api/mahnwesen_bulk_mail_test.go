package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"bibliothek/db"
	"bibliothek/repository"

	"github.com/pashagolub/pgxmock/v4"
)

// Während einer Ferien-/Schließzeit MUSS der Massenversand mit 403 abbrechen und
// nichts senden — sonst gingen Mahnungen in den Ferien raus.
func TestSendBulkOverdueHandler_FerienGesperrt(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock: %v", err)
	}
	defer mock.Close()

	// CheckFerienAktiv findet einen aktiven Zeitraum → gesperrt.
	mock.ExpectQuery("ferien_schliesszeiten").
		WillReturnRows(pgxmock.NewRows([]string{"bezeichnung"}).AddRow("Sommerferien"))

	server := &Server{DB: &db.Database{Pool: mock}}
	mahnRepo := repository.NewMahnwesenRepository(mock)

	req := httptest.NewRequest(http.MethodPost, "/api/mail/send-bulk-overdue", nil)
	rec := httptest.NewRecorder()
	server.SendBulkOverdueHandler(mahnRepo)(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("erwartet 403 während Ferien, bekam %d: %s", rec.Code, rec.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unerfüllte Mock-Erwartungen: %v", err)
	}
}

// Ohne konfigurierten Mailserver (SMTP_HOST leer) → 503, kein Versand. Der Check
// greift NACH der Ferien-Prüfung und VOR jeder Klassen-Query.
func TestSendBulkOverdueHandler_SmtpFehlt(t *testing.T) {
	t.Setenv("SMTP_HOST", "")

	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock: %v", err)
	}
	defer mock.Close()

	// Keine Ferien: leeres Ergebnis → CheckFerienAktiv liefert (false, "", nil).
	mock.ExpectQuery("ferien_schliesszeiten").
		WillReturnRows(pgxmock.NewRows([]string{"bezeichnung"}))

	server := &Server{DB: &db.Database{Pool: mock}}
	mahnRepo := repository.NewMahnwesenRepository(mock)

	req := httptest.NewRequest(http.MethodPost, "/api/mail/send-bulk-overdue", nil)
	rec := httptest.NewRecorder()
	server.SendBulkOverdueHandler(mahnRepo)(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("erwartet 503 ohne SMTP, bekam %d: %s", rec.Code, rec.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unerfüllte Mock-Erwartungen: %v", err)
	}
}

// Kern der Datenschutz-Garantie: Klassen OHNE Lehrer-E-Mail oder OHNE Schüler
// werden übersprungen und erhalten KEINE Mail; gültige Klassen bekommen genau EINE
// Mail an genau die hinterlegte Lehrer-Adresse (keine klassenübergreifenden Empfänger).
func TestVersendeKlassenMahnungen_SkipLogik(t *testing.T) {
	var empfaenger []string
	fakeSend := func(m MailRequest) error {
		empfaenger = append(empfaenger, m.To)
		return nil
	}
	fakePDF := func(_ []repository.MahnwesenKlasse) ([]byte, error) {
		return []byte("%PDF-fake"), nil
	}

	klassen := []repository.MahnwesenKlasse{
		{
			Klasse:      "5a",
			LehrerEmail: "lehrer5a@schule.de",
			Schueler: []repository.UeberfaelligerSchueler{
				{SchuelerID: "s1", Name: "Max", Klasse: "5a", Medien: []repository.UeberfaelligesMedium{{Titel: "Buch", TageUeberfaellig: 5}}},
			},
		},
		{ // keine Lehrer-Mail → übersprungen, DARF NICHT gesendet werden
			Klasse:      "6b",
			LehrerEmail: "",
			Schueler: []repository.UeberfaelligerSchueler{
				{SchuelerID: "s2", Name: "Erika", Klasse: "6b", Medien: []repository.UeberfaelligesMedium{{Titel: "Buch2"}}},
			},
		},
		{ // keine Schüler → übersprungen
			Klasse:      "7c",
			LehrerEmail: "lehrer7c@schule.de",
			Schueler:    nil,
		},
	}

	sent, skipped := versendeKlassenMahnungen(klassen, fakePDF, fakeSend)

	if sent != 1 || skipped != 2 {
		t.Fatalf("sent=%d skipped=%d, want sent=1 skipped=2", sent, skipped)
	}
	if len(empfaenger) != 1 || empfaenger[0] != "lehrer5a@schule.de" {
		t.Fatalf("Empfänger = %v, want genau [lehrer5a@schule.de] — keine Mail an Klassen ohne Adresse!", empfaenger)
	}
}

// Schlägt der Versand einer Klasse fehl, läuft der Rest weiter (Best-Effort) und
// die betroffene Klasse zählt als übersprungen.
func TestVersendeKlassenMahnungen_VersandfehlerZaehltAlsSkip(t *testing.T) {
	fakePDF := func(_ []repository.MahnwesenKlasse) ([]byte, error) {
		return []byte("%PDF-fake"), nil
	}
	fakeSend := func(m MailRequest) error {
		if m.To == "kaputt@schule.de" {
			return http.ErrHandlerTimeout // beliebiger Versandfehler
		}
		return nil
	}

	schueler := []repository.UeberfaelligerSchueler{
		{SchuelerID: "s1", Name: "Max", Klasse: "x", Medien: []repository.UeberfaelligesMedium{{Titel: "Buch"}}},
	}
	klassen := []repository.MahnwesenKlasse{
		{Klasse: "ok", LehrerEmail: "ok@schule.de", Schueler: schueler},
		{Klasse: "err", LehrerEmail: "kaputt@schule.de", Schueler: schueler},
	}

	sent, skipped := versendeKlassenMahnungen(klassen, fakePDF, fakeSend)
	if sent != 1 || skipped != 1 {
		t.Fatalf("sent=%d skipped=%d, want sent=1 skipped=1 (Versandfehler = skip, Lauf bricht nicht ab)", sent, skipped)
	}
}

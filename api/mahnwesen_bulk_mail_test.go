package api

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"bibliothek/auth"
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

// ── Auswahl & Override ───────────────────────────────────────────────────────

// argFaenger reicht ein Exec-Argument durch und schreibt es mit. pgxmock.AnyArg()
// würde den Payload zwar passieren lassen, aber nicht herausgeben — und geprüft
// werden soll genau sein Inhalt.
type argFaenger struct{ ziel *string }

func (a *argFaenger) Match(v interface{}) bool {
	s, ok := v.(string)
	if ok {
		*a.ziel = s
	}
	return ok
}

// Der Body ist optional, aber nicht beliebig: Ein FEHLENDES klassen-Feld heisst
// „alle" (alter Vertrag), ein LEERES Array heisst „niemand" und muss abgewiesen
// werden. Würden beide gleich behandelt, löste eine leere Auswahl den grössten
// denkbaren Fehlversand aus.
func TestParseBulkOverdueRequest(t *testing.T) {
	tests := []struct {
		name        string
		body        string
		wantErr     bool
		wantAlle    bool     // Klassen == nil → alle Klassen
		wantKlassen []string // nur geprüft, wenn wantAlle == false
		wantEmail   string
	}{
		{name: "kein Body → alle Klassen", body: "", wantAlle: true},
		{name: "leeres Objekt → alle Klassen", body: `{}`, wantAlle: true},
		{name: "leeres Array → Fehler", body: `{"klassen":[]}`, wantErr: true},
		{name: "nur Leerzeichen-Klassen → Fehler", body: `{"klassen":["  ",""]}`, wantErr: true},
		{name: "kaputtes JSON → Fehler", body: `{"klassen":`, wantErr: true},
		{
			name: "Auswahl wird getrimmt", body: `{"klassen":[" 5a ","6b"]}`,
			wantKlassen: []string{"5a", "6b"},
		},
		{
			name: "Override wird getrimmt", body: `{"klassen":["5a"],"override_email":"  sek@schule.de "}`,
			wantKlassen: []string{"5a"}, wantEmail: "sek@schule.de",
		},
		{name: "ungültige Override-Adresse → Fehler", body: `{"klassen":["5a"],"override_email":"sek@"}`, wantErr: true},
		{name: "Header-Injektion in der Adresse → Fehler", body: `{"klassen":["5a"],"override_email":"a@b.de\nBcc: c@d.de"}`, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var body io.Reader
			if tt.body != "" {
				body = strings.NewReader(tt.body)
			}
			req := httptest.NewRequest(http.MethodPost, "/api/mail/send-bulk-overdue", body)

			got, err := parseBulkOverdueRequest(req)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("Fehler erwartet, bekam keinen (%+v)", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unerwarteter Fehler: %v", err)
			}
			if tt.wantAlle {
				if got.Klassen != nil {
					t.Fatalf("Klassen = %v, want nil (= alle Klassen)", *got.Klassen)
				}
			} else if !reflect.DeepEqual(*got.Klassen, tt.wantKlassen) {
				t.Fatalf("Klassen = %v, want %v", *got.Klassen, tt.wantKlassen)
			}
			if got.OverrideEmail != tt.wantEmail {
				t.Fatalf("OverrideEmail = %q, want %q", got.OverrideEmail, tt.wantEmail)
			}
		})
	}
}

// waehleKlassen darf nur die angehakten Klassen durchlassen — und muss melden, was
// es nicht (mehr) gibt, damit ein Format-Versatz („5a" vs. „05A") nicht als stiller
// Nullversand endet.
func TestWaehleKlassen(t *testing.T) {
	alle := []repository.MahnwesenKlasse{
		{Klasse: "5a", LehrerEmail: "a@schule.de"},
		{Klasse: "6b", LehrerEmail: "b@schule.de"},
		{Klasse: "7c", LehrerEmail: "c@schule.de"},
	}

	t.Run("nil = alle Klassen (alter Vertrag)", func(t *testing.T) {
		gewaehlt, unbekannt := waehleKlassen(alle, nil)
		if len(gewaehlt) != 3 || unbekannt != nil {
			t.Fatalf("gewaehlt=%d unbekannt=%v, want 3 und nil", len(gewaehlt), unbekannt)
		}
	})

	t.Run("schneidet auf die Auswahl zu", func(t *testing.T) {
		auswahl := []string{"6b"}
		gewaehlt, unbekannt := waehleKlassen(alle, &auswahl)
		if len(gewaehlt) != 1 || gewaehlt[0].Klasse != "6b" {
			t.Fatalf("gewaehlt = %+v, want genau [6b]", gewaehlt)
		}
		if unbekannt != nil {
			t.Fatalf("unbekannt = %v, want nil", unbekannt)
		}
	})

	t.Run("meldet unbekannte Klassen, statt sie zu verschlucken", func(t *testing.T) {
		auswahl := []string{"5a", "05A"}
		gewaehlt, unbekannt := waehleKlassen(alle, &auswahl)
		if len(gewaehlt) != 1 || gewaehlt[0].Klasse != "5a" {
			t.Fatalf("gewaehlt = %+v, want genau [5a]", gewaehlt)
		}
		if !reflect.DeepEqual(unbekannt, []string{"05A"}) {
			t.Fatalf("unbekannt = %v, want [05A]", unbekannt)
		}
	})

	t.Run("doppelte Auswahl mahnt nicht doppelt", func(t *testing.T) {
		auswahl := []string{"5a", "5a", "5a"}
		gewaehlt, _ := waehleKlassen(alle, &auswahl)
		if len(gewaehlt) != 1 {
			t.Fatalf("gewaehlt = %d Klassen, want 1 — sonst bekommt die Klasse drei Mahnungen", len(gewaehlt))
		}
	})
}

// Mit Override-Adresse gehen die Listen an genau diese eine Adresse — auch für
// Klassen ohne hinterlegte Klassenleitung (das ist der Vertretungsfall, für den
// das Feld existiert). Klassen OHNE überfällige Fälle bleiben trotzdem aussen vor.
func TestZieleAufOverride_LenktAlleKlassenUm(t *testing.T) {
	schueler := []repository.UeberfaelligerSchueler{
		{SchuelerID: "s1", Name: "Max", Klasse: "x", Medien: []repository.UeberfaelligesMedium{{Titel: "Buch"}}},
	}
	original := []repository.MahnwesenKlasse{
		{Klasse: "5a", LehrerEmail: "lehrer5a@schule.de", Schueler: schueler},
		{Klasse: "6b", LehrerEmail: "", Schueler: schueler}, // ohne Mapping — sonst übersprungen
		{Klasse: "7c", LehrerEmail: "lehrer7c@schule.de", Schueler: nil},
	}

	umgelenkt := zieleAufOverride(original, "sekretariat@schule.de")

	var empfaenger []string
	sent, skipped := versendeKlassenMahnungen(
		umgelenkt,
		func(_ []repository.MahnwesenKlasse) ([]byte, error) { return []byte("%PDF-fake"), nil },
		func(m MailRequest) error { empfaenger = append(empfaenger, m.To); return nil },
	)

	if sent != 2 || skipped != 1 {
		t.Fatalf("sent=%d skipped=%d, want sent=2 skipped=1 (7c hat keine Fälle)", sent, skipped)
	}
	for _, e := range empfaenger {
		if e != "sekretariat@schule.de" {
			t.Fatalf("Empfänger = %v, want ausschliesslich sekretariat@schule.de", empfaenger)
		}
	}
	if original[0].LehrerEmail != "lehrer5a@schule.de" {
		t.Fatalf("Original wurde mutiert (LehrerEmail = %q) — die geladenen Daten müssen unangetastet bleiben", original[0].LehrerEmail)
	}
}

// Eine leere Auswahl darf nicht erst in der DB auffallen: Der Handler weist sie ab,
// bevor er auch nur eine Query absetzt.
func TestSendBulkOverdueHandler_LeereAuswahlOhneDBZugriff(t *testing.T) {
	for _, body := range []string{`{"klassen":[]}`, `{"klassen":["5a"],"override_email":"kein-mail"}`} {
		mock, err := pgxmock.NewPool()
		if err != nil {
			t.Fatalf("pgxmock: %v", err)
		}

		server := &Server{DB: &db.Database{Pool: mock}}
		req := httptest.NewRequest(http.MethodPost, "/api/mail/send-bulk-overdue", strings.NewReader(body))
		rec := httptest.NewRecorder()
		server.SendBulkOverdueHandler(repository.NewMahnwesenRepository(mock))(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("Body %s: erwartet 400, bekam %d: %s", body, rec.Code, rec.Body.String())
		}
		// Keine einzige erwartete Query registriert → es gab keinen DB-Zugriff.
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("Body %s: unerwarteter DB-Zugriff: %v", body, err)
		}
		mock.Close()
	}
}

// Revisionssicherheit: Der Audit-Eintrag muss die Override-Adresse im Klartext
// führen und gültiges JSON sein — auch wenn die Adresse Zeichen enthält, die ein
// handgebautes JSON-Literal zerrissen hätten (früher: fmt.Sprintf).
func TestLogBulkOverdueAudit_SchreibtGueltigesJSONMitEmpfaenger(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock: %v", err)
	}
	defer mock.Close()

	var details string
	mock.ExpectExec(`INSERT INTO audit_logs`).
		WithArgs("admin-1", "BULK_OVERDUE_MAIL", &argFaenger{ziel: &details}, pgxmock.AnyArg()).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	server := &Server{DB: &db.Database{Pool: mock}}
	req := httptest.NewRequest(http.MethodPost, "/api/mail/send-bulk-overdue", nil)
	req = req.WithContext(context.WithValue(req.Context(), auth.ClaimsContextKey, &auth.Claims{UserID: "admin-1"}))

	server.logBulkOverdueAudit(req, bulkOverdueAudit{
		Phase:         "ende",
		Klassen:       []string{"5a", "6b"},
		OverrideEmail: `"seltsam"@schule.de`,
		Sent:          2,
		Skipped:       0,
	})

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("Audit-INSERT fehlt: %v", err)
	}

	// Der geschriebene Payload muss als JSON parsebar sein (Spalte ist jsonb).
	var geparst bulkOverdueAudit
	if err := json.Unmarshal([]byte(details), &geparst); err != nil {
		t.Fatalf("details ist kein gültiges JSON: %v (%s)", err, details)
	}
	if geparst.OverrideEmail != `"seltsam"@schule.de` {
		t.Fatalf("OverrideEmail = %q — die Adresse muss unverfälscht im Log stehen", geparst.OverrideEmail)
	}
}

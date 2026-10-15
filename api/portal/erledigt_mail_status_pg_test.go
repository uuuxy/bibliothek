package portal

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"bibliothek/api/tuer"
	"bibliothek/db"
	"bibliothek/internal/pgtest"
	"bibliothek/internal/testdaten"
	"bibliothek/mailservice"
)

// Die Türen zum Erledigen (Klassensatz bereit, Anliegen abgehakt) verschicken ihre
// Benachrichtigung nach dem Commit und nennen in der Antwort, was aus der Mail wurde: Ein
// Ausfall des Versands stünde sonst nur im Log, und die Theke hielte die Lehrkraft für
// benachrichtigt. Bei "fehlgeschlagen" warnt die Oberfläche.
//
// An der echten Datenbank, weil beide Türen das Erledigen und den Empfänger der Mail in einer
// Abfrage lesen; pgxmock sähe nicht, wenn das Schema davon abwiche.

// mailAttrappe nimmt jede Mail an, statt sie zu verschicken; sende entscheidet die Antwort.
type mailAttrappe struct {
	sende func(mailservice.Nachricht) error
}

func (a mailAttrappe) Sende(m mailservice.Nachricht) error { return a.sende(m) }

func (mailAttrappe) IstKonfiguriert() bool { return true }

func (mailAttrappe) Konfig() (mailservice.SMTPKonfig, error) { return mailservice.SMTPKonfig{}, nil }

// versandAntwortet lässt jeden Versand mit fehler antworten (nil heißt: verschickt).
func versandAntwortet(fehler error) mailAttrappe {
	return mailAttrappe{sende: func(mailservice.Nachricht) error { return fehler }}
}

// mailStatusAusAntwort erwartet 200 und liest das "mail"-Feld.
func mailStatusAusAntwort(t *testing.T, rec *httptest.ResponseRecorder) string {
	t.Helper()
	if rec.Code != http.StatusOK {
		t.Fatalf("Status %d, erwartet 200 — Antwort: %s", rec.Code, rec.Body.String())
	}
	var antwort map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &antwort); err != nil {
		t.Fatalf("Antwort kein JSON: %v — %s", err, rec.Body.String())
	}
	return antwort["mail"]
}

func TestAnliegenErledigen_MailAusfallStehtInDerAntwort(t *testing.T) {
	pool := pgtest.Pool(t)
	testdaten.LeereBestandsdaten(t, pool)
	ctx := context.Background()
	srv := &Tueren{Basis: &tuer.Basis{DB: &db.Verbindung{Pool: pool}}}

	var lehrerID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO benutzer (vorname, nachname, email, rolle, aktiv)
		VALUES ('Lisa', 'Lehrkraft', 'anliegen-mailstatus@test.invalid', 'mitarbeiter', true)
		ON CONFLICT (email) DO UPDATE SET vorname = EXCLUDED.vorname
		RETURNING id`).Scan(&lehrerID); err != nil {
		t.Fatalf("Lehrkraft anlegen: %v", err)
	}

	erledige := func(t *testing.T, anliegenID string) *httptest.ResponseRecorder {
		t.Helper()
		req := httptest.NewRequest(http.MethodPut, "/api/anliegen/"+anliegenID+"/erledigen",
			strings.NewReader(`{"notiz":"besorgt"}`))
		req.SetPathValue("id", anliegenID)
		rec := httptest.NewRecorder()
		srv.AnliegenErledigen()(rec, req)
		return rec
	}

	neuesAnliegen := func(t *testing.T) string {
		t.Helper()
		var id string
		if err := pool.QueryRow(ctx, `
			INSERT INTO lehrer_anliegen (art, titel_text, angefordert_von)
			VALUES ('wunsch', 'Faust II', $1) RETURNING id`, lehrerID).Scan(&id); err != nil {
			t.Fatalf("Anliegen anlegen: %v", err)
		}
		return id
	}

	t.Run("Versand kaputt → fehlgeschlagen", func(t *testing.T) {
		srv.Mail = versandAntwortet(errors.New("smtp down"))
		if got := mailStatusAusAntwort(t, erledige(t, neuesAnliegen(t))); got != "fehlgeschlagen" {
			t.Fatalf("mail=%q, erwartet fehlgeschlagen", got)
		}
	})

	t.Run("Versand ok → versendet", func(t *testing.T) {
		srv.Mail = versandAntwortet(nil)
		if got := mailStatusAusAntwort(t, erledige(t, neuesAnliegen(t))); got != "versendet" {
			t.Fatalf("mail=%q, erwartet versendet", got)
		}
	})

	t.Run("ohne Konto → keine_adresse, Versand wird nicht versucht", func(t *testing.T) {
		srv.Mail = versandAntwortet(errors.New("darf nicht aufgerufen werden"))
		var id string
		if err := pool.QueryRow(ctx, `
			INSERT INTO lehrer_anliegen (art, titel_text)
			VALUES ('meldung', 'Rücken kaputt') RETURNING id`).Scan(&id); err != nil {
			t.Fatalf("Anliegen ohne Konto anlegen: %v", err)
		}
		if got := mailStatusAusAntwort(t, erledige(t, id)); got != "keine_adresse" {
			t.Fatalf("mail=%q, erwartet keine_adresse", got)
		}
	})

	t.Run("die Mail nennt Art, Klasse und Notiz, ein zweites Abhaken schickt keine", func(t *testing.T) {
		var mails []mailservice.Nachricht
		srv.Mail = mailAttrappe{sende: func(m mailservice.Nachricht) error { mails = append(mails, m); return nil }}

		var meldung string
		if err := pool.QueryRow(ctx, `
			INSERT INTO lehrer_anliegen (art, titel_text, klasse, angefordert_von)
			VALUES ('meldung', 'Atlas, Seite 12 fehlt', '7b', $1) RETURNING id`, lehrerID).Scan(&meldung); err != nil {
			t.Fatalf("Meldung anlegen: %v", err)
		}
		if got := mailStatusAusAntwort(t, erledige(t, meldung)); got != "versendet" {
			t.Fatalf("mail=%q, erwartet versendet", got)
		}
		if got := mailStatusAusAntwort(t, erledige(t, neuesAnliegen(t))); got != "versendet" {
			t.Fatalf("mail=%q, erwartet versendet", got)
		}
		if len(mails) != 2 {
			t.Fatalf("%d Mails nach zwei erledigten Anliegen, erwartet 2", len(mails))
		}
		m, wunsch := mails[0], mails[1]
		if m.An != "anliegen-mailstatus@test.invalid" || m.Betreff != "Ihre Meldung ist erledigt: Atlas, Seite 12 fehlt" {
			t.Errorf("Meldung: an %q mit Betreff %q", m.An, m.Betreff)
		}
		for _, stueck := range []string{"Betreff: Atlas, Seite 12 fehlt", "Klasse:  7b", "Notiz:   besorgt"} {
			if !strings.Contains(m.Text, stueck) {
				t.Errorf("Meldung: der Text nennt %q nicht: %s", stueck, m.Text)
			}
		}
		if wunsch.Betreff != "Ihr Wunsch ist erledigt: Faust II" || strings.Contains(wunsch.Text, "Klasse:") {
			t.Errorf("Wunsch ohne Klasse: Betreff %q, Text %s", wunsch.Betreff, wunsch.Text)
		}

		if rec := erledige(t, meldung); rec.Code != http.StatusNotFound {
			t.Errorf("zweites Abhaken: Status %d, erwartet 404: %s", rec.Code, rec.Body.String())
		}
		if len(mails) != 2 {
			t.Errorf("das zweite Abhaken hat eine Mail geschickt (%d statt 2)", len(mails))
		}
	})
}

func TestKlassensatzReservierungErledigen_MailAusfallStehtInDerAntwort(t *testing.T) {
	pool := pgtest.Pool(t)
	testdaten.LeereBestandsdaten(t, pool)
	ctx := context.Background()
	srv := &Tueren{Basis: &tuer.Basis{DB: &db.Verbindung{Pool: pool}}}

	var lehrerID, titelID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO benutzer (vorname, nachname, email, rolle, aktiv)
		VALUES ('Karl', 'Klassensatz', 'klassensatz-mailstatus@test.invalid', 'mitarbeiter', true)
		ON CONFLICT (email) DO UPDATE SET vorname = EXCLUDED.vorname
		RETURNING id`).Scan(&lehrerID); err != nil {
		t.Fatalf("Lehrkraft anlegen: %v", err)
	}
	if err := pool.QueryRow(ctx, `
		INSERT INTO buecher_titel (titel) VALUES ('Der Besuch der alten Dame') RETURNING id`).Scan(&titelID); err != nil {
		t.Fatalf("Titel anlegen: %v", err)
	}

	// body nil: ein Browser mit altem Stand der Oberfläche schickt keinen Rumpf und muss
	// weiter abschließen können.
	erledigeMit := func(t *testing.T, body *string) (*httptest.ResponseRecorder, string) {
		t.Helper()
		var resID string
		if err := pool.QueryRow(ctx, `
			INSERT INTO klassensatz_reservierungen (titel_id, klasse, anzahl, notiz, angefordert_von)
			VALUES ($1, '9c', 25, 'ab 15. September', $2) RETURNING id`, titelID, lehrerID).Scan(&resID); err != nil {
			t.Fatalf("Reservierung anlegen: %v", err)
		}
		var r io.Reader
		if body != nil {
			r = strings.NewReader(*body)
		}
		req := httptest.NewRequest(http.MethodPut, "/api/reservierungen/klassensatz/"+resID+"/erledigen", r)
		req.SetPathValue("id", resID)
		rec := httptest.NewRecorder()
		srv.KlassensatzReservierungErledigen()(rec, req)
		return rec, resID
	}
	erledige := func(t *testing.T) *httptest.ResponseRecorder {
		rec, _ := erledigeMit(t, nil)
		return rec
	}

	t.Run("Versand kaputt → fehlgeschlagen", func(t *testing.T) {
		srv.Mail = versandAntwortet(errors.New("smtp down"))
		if got := mailStatusAusAntwort(t, erledige(t)); got != "fehlgeschlagen" {
			t.Fatalf("mail=%q, erwartet fehlgeschlagen", got)
		}
	})

	t.Run("Versand ok → versendet", func(t *testing.T) {
		srv.Mail = versandAntwortet(nil)
		if got := mailStatusAusAntwort(t, erledige(t)); got != "versendet" {
			t.Fatalf("mail=%q, erwartet versendet", got)
		}
	})

	// Die Antwort der Bibliothek steht in der Mail und in der Zeile; die Notiz der Lehrkraft
	// steht in der Mail, damit die Antwort einen Bezug hat. Ohne Rumpf geht die Mail ohne
	// Notiz der Bibliothek, aber sie geht.
	t.Run("Notiz der Bibliothek steht in Mail und Zeile", func(t *testing.T) {
		var gesendet mailservice.Nachricht
		srv.Mail = mailAttrappe{sende: func(m mailservice.Nachricht) error { gesendet = m; return nil }}

		body := `{"notiz":"24 von 30, der Rest ist bei der 8a"}`
		rec, resID := erledigeMit(t, &body)
		if got := mailStatusAusAntwort(t, rec); got != "versendet" {
			t.Fatalf("mail=%q, erwartet versendet", got)
		}
		for _, muss := range []string{"24 von 30, der Rest ist bei der 8a", "Ihre Notiz: ab 15. September", "Der Besuch der alten Dame"} {
			if !strings.Contains(gesendet.Text, muss) {
				t.Errorf("Mail ohne %q:\n%s", muss, gesendet.Text)
			}
		}
		var gespeichert string
		if err := pool.QueryRow(ctx, `SELECT erledigt_notiz FROM klassensatz_reservierungen WHERE id = $1`, resID).Scan(&gespeichert); err != nil {
			t.Fatalf("erledigt_notiz lesen: %v", err)
		}
		if gespeichert != "24 von 30, der Rest ist bei der 8a" {
			t.Fatalf("erledigt_notiz=%q", gespeichert)
		}

		gesendet = mailservice.Nachricht{}
		rec, _ = erledigeMit(t, nil)
		if got := mailStatusAusAntwort(t, rec); got != "versendet" {
			t.Fatalf("ohne Body: mail=%q, erwartet versendet", got)
		}
		if strings.Contains(gesendet.Text, "Notiz der Bibliothek") {
			t.Errorf("ohne Body darf keine Bibliotheks-Notiz in der Mail stehen:\n%s", gesendet.Text)
		}
	})
}

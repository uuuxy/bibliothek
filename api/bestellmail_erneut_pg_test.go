package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"bibliothek/auth"
	"bibliothek/db"
	"bibliothek/internal/smtptest"
	"bibliothek/mailservice"
	"bibliothek/sse"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Scheitert die Mail an den Lieferanten, stand es danach nirgends an der Bestellung: Die
// Meldung blieb fünf Sekunden stehen, die Bestellhistorie zeigte die Bestellung wie jede
// andere. Jetzt steht der gescheiterte Versand an der Bestellung, und sie lässt sich erneut
// senden. Geprüft über die echten Routen, mit der fertigen Nachricht auf dem Draht.

// toterPort liefert einen Port, an dem niemand zuhört.
func toterPort(t *testing.T) string {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Port belegen: %v", err)
	}
	_, port, err := net.SplitHostPort(ln.Addr().String())
	if err != nil {
		t.Fatalf("Port lesen: %v", err)
	}
	if err := ln.Close(); err != nil {
		t.Fatalf("Port freigeben: %v", err)
	}
	return port
}

// mailserverNichtErreichbar biegt den Versand auf einen Port um, an dem niemand zuhört.
// beimVersand läuft, sobald der Versand selbst die Zugangsdaten liest (nil = nichts).
func mailserverNichtErreichbar(t *testing.T, beimVersand func()) {
	t.Helper()
	t.Setenv("SMTP_ALLOW_PLAINTEXT", "true")
	port := toterPort(t)
	aufrufe := 0
	alterLader := smtpKonfigLader
	smtpKonfigLader = func() (mailservice.SMTPKonfig, error) {
		// Der erste Aufruf fragt nur, ob ein Mailserver eingerichtet ist; ab dem zweiten
		// läuft der Versand.
		aufrufe++
		if aufrufe >= 2 && beimVersand != nil {
			beimVersand()
		}
		return mailservice.SMTPKonfig{Host: "127.0.0.1", Port: port, Absender: "bibliothek@schule.invalid"}, nil
	}
	t.Cleanup(func() { smtpKonfigLader = alterLader })
}

// bestellmailLage: der Server über seine Routen, mit einer angemeldeten Person, die bestellen darf.
type bestellmailLage struct {
	pool    *pgxpool.Pool
	router  http.Handler
	sitzung string
	adminID string
}

func bestellmailLageAnlegen(t *testing.T, kennung string) *bestellmailLage {
	t.Helper()
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	authenticator, err := auth.NewAuthenticator("bestellmail-erneut-testgeheimnis-32-by!!", pool, time.Hour)
	if err != nil {
		t.Fatalf("Authenticator: %v", err)
	}
	l := &bestellmailLage{pool: pool}
	if err := pool.QueryRow(t.Context(), `
		INSERT INTO benutzer (vorname, nachname, email, rolle, aktiv)
		VALUES ('Berta', 'Bestellung', $1, 'admin', true) RETURNING id`,
		"bestellmail-"+kennung+"@example.org").Scan(&l.adminID); err != nil {
		t.Fatalf("Konto anlegen: %v", err)
	}
	if l.sitzung, err = authenticator.GenerateToken(l.adminID, "BMAIL-"+kennung, auth.RoleAdmin, ""); err != nil {
		t.Fatalf("Sitzung: %v", err)
	}
	l.router = NewServer(&db.Database{Pool: pool}, authenticator, sse.NewBroker(), false).Routes()
	t.Cleanup(func() {
		aufraeumen(t, pool, `DELETE FROM audit_logs WHERE aktion = 'BESTELLMAIL_ERNEUT_GESENDET'`)
	})
	return l
}

func (l *bestellmailLage) rufe(t *testing.T, methode, pfad, rumpf string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(methode, pfad, strings.NewReader(rumpf))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: "session_token", Value: l.sitzung})
	rec := httptest.NewRecorder()
	l.router.ServeHTTP(rec, mitCSRF(req))
	return rec
}

// bestelle löst eine Bestellung über die Route aus und liefert ihre Kennung.
func (l *bestellmailLage) bestelle(t *testing.T, lieferantID, titelID string) (string, *httptest.ResponseRecorder) {
	t.Helper()
	rec := l.rufe(t, http.MethodPost, "/api/bestellungen", fmt.Sprintf(
		`{"supplier_id":%q,"mittel":"land","items":[{"titel_id":%q,"menge":2,"preis":9.5,"generate_barcodes":true}]}`,
		lieferantID, titelID))
	if rec.Code != http.StatusOK {
		t.Fatalf("Bestellung: Status %d: %s", rec.Code, rec.Body.String())
	}
	var id string
	if err := l.pool.QueryRow(context.Background(),
		`SELECT id FROM bestellungen_verlauf WHERE lieferant_id = $1`, lieferantID).Scan(&id); err != nil {
		t.Fatalf("Bestellung lesen: %v", err)
	}
	return id, rec
}

// gescheitertAm liest über die Detail-Route, ob die Bestellung einen gescheiterten Versand trägt.
func (l *bestellmailLage) gescheitert(t *testing.T, bestellungID string) bool {
	t.Helper()
	rec := l.rufe(t, http.MethodGet, "/api/bestellhistorie/"+bestellungID, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("Detail: Status %d: %s", rec.Code, rec.Body.String())
	}
	var kopf struct {
		MailGescheitertAm *time.Time `json:"mail_gescheitert_am"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &kopf); err != nil {
		t.Fatalf("Detail unlesbar: %v", err)
	}
	return kopf.MailGescheitertAm != nil
}

func TestBestellmail_GescheitertStehtAnDerBestellungUndGehtErneutRaus(t *testing.T) {
	l := bestellmailLageAnlegen(t, "erneut")
	setzeOeffentlicheAdresse(t, l.pool, "https://bib.example.invalid")
	lieferant := haendler(t, l.pool, "Naacher-Erneut", true)
	titel := titelMitMeldebestand(t, l.pool, "LMF-Mathe-Erneut", 0)

	// 1. Der Mailserver ist nicht erreichbar: Die Bestellung ist gespeichert und trägt den Vermerk.
	mailserverNichtErreichbar(t, nil)
	bestellung, rec := l.bestelle(t, lieferant, titel)
	if antwort := rec.Body.String(); !strings.Contains(antwort, `"warning"`) {
		t.Errorf("Die Antwort warnt nicht, obwohl die Mail nicht rausging: %s", antwort)
	}
	if !l.gescheitert(t, bestellung) {
		t.Fatal("Die Bestellung trägt keinen gescheiterten Versand, obwohl die Mail nicht rausging")
	}
	liste := l.rufe(t, http.MethodGet, "/api/bestellhistorie", "")
	if !strings.Contains(liste.Body.String(), `"mail_gescheitert_am"`) {
		t.Errorf("Die Liste der Bestellhistorie nennt den gescheiterten Versand nicht: %s", liste.Body.String())
	}

	// 2. Erneut senden bei weiter totem Mailserver: Die Antwort sagt es, der Vermerk bleibt.
	rec = l.rufe(t, http.MethodPost, "/api/bestellungen/"+bestellung+"/mail", "")
	if rec.Code != http.StatusBadGateway {
		t.Fatalf("Erneut senden ohne Mailserver: Status %d, erwartet 502: %s", rec.Code, rec.Body.String())
	}
	// Die Meldung erreicht den Bildschirm im Wortlaut; eine 500 ersetzte der Sanitizer.
	if !strings.Contains(rec.Body.String(), "erneut gescheitert") {
		t.Errorf("Die Antwort nennt den gescheiterten Versand nicht: %s", rec.Body.String())
	}
	if !l.gescheitert(t, bestellung) {
		t.Error("Der Vermerk ist nach einem zweiten gescheiterten Versuch verschwunden")
	}

	// 3. Der Mailserver ist wieder da, die Adresse des Lieferanten inzwischen berichtigt.
	if _, err := l.pool.Exec(t.Context(),
		`UPDATE lieferanten SET email = 'berichtigt@example.invalid' WHERE id = $1`, lieferant); err != nil {
		t.Fatalf("Adresse berichtigen: %v", err)
	}
	sitzungen := mailAbfangen(t)
	if rec := l.rufe(t, http.MethodPost, "/api/bestellungen/"+bestellung+"/mail", ""); rec.Code != http.StatusOK {
		t.Fatalf("Erneut senden: Status %d: %s", rec.Code, rec.Body.String())
	}
	var mail smtptest.Sitzung
	select {
	case mail = <-sitzungen:
	case <-time.After(10 * time.Second):
		t.Fatal("es wurde keine Mail verschickt")
	}
	if len(mail.Empfaenger) != 1 || mail.Empfaenger[0] != "berichtigt@example.invalid" {
		t.Errorf("Empfänger %v, erwartet die heutige Adresse des Lieferanten", mail.Empfaenger)
	}
	for _, teil := range []string{
		"https://bib.example.invalid/bestellung/", "bestellanschreiben", "barcode_mapping",
		"Kundennummer: K-Naacher-Erneut", "Bestellte Titel: 1", "Gesamtanzahl Exemplare: 2",
	} {
		if !strings.Contains(mail.Nachricht, teil) {
			t.Errorf("%q fehlt in der erneut gesendeten Mail:\n%s", teil, kopf(mail.Nachricht))
		}
	}
	// Die Etiketten liegen hinter dem Link, wie bei der ersten Mail.
	if strings.Contains(mail.Nachricht, "etiketten_klein") {
		t.Error("Der Etikettenbogen hängt an der Mail, obwohl ein Link mitgeht")
	}
	// Der Link aus der Mail führt zur Bestellung.
	token := zwischen(mail.Nachricht, "https://bib.example.invalid/bestellung/")
	if oeffentlich := l.rufe(t, http.MethodGet, "/api/public/bestellung/"+token, ""); oeffentlich.Code != http.StatusOK {
		t.Errorf("Der Link aus der Mail öffnet die Bestellung nicht: Status %d", oeffentlich.Code)
	}
	if l.gescheitert(t, bestellung) {
		t.Error("Der Vermerk steht nach dem gelungenen Versand noch an der Bestellung")
	}
	var adresseAmBeleg string
	if err := l.pool.QueryRow(t.Context(),
		`SELECT lieferant_email FROM bestellungen_verlauf WHERE id = $1`, bestellung).Scan(&adresseAmBeleg); err != nil {
		t.Fatal(err)
	}
	if adresseAmBeleg != "berichtigt@example.invalid" {
		t.Errorf("Am Beleg steht %q, die Mail ging an die berichtigte Adresse", adresseAmBeleg)
	}

	// 4. Ein zweiter Klick schickt nichts doppelt.
	if rec := l.rufe(t, http.MethodPost, "/api/bestellungen/"+bestellung+"/mail", ""); rec.Code != http.StatusConflict {
		t.Errorf("Zweites Senden: Status %d, erwartet 409: %s", rec.Code, rec.Body.String())
	}
	select {
	case zweite := <-sitzungen:
		t.Errorf("Eine zweite Mail ging raus, an %v", zweite.Empfaenger)
	case <-time.After(300 * time.Millisecond):
	}

	// 5. Das Protokoll nennt den Versand: Bearbeiter und Bestellung, keine Adresse.
	var wer, protokollierte string
	var details []byte
	if err := l.pool.QueryRow(t.Context(), `
		SELECT coalesce(admin_id::text, ''), details->>'bestellung_id', details
		FROM audit_logs WHERE aktion = 'BESTELLMAIL_ERNEUT_GESENDET'`).Scan(&wer, &protokollierte, &details); err != nil {
		t.Fatalf("Protokolleintrag lesen (genau einer erwartet): %v", err)
	}
	if wer != l.adminID || protokollierte != bestellung {
		t.Errorf("Eintrag nennt Bearbeiter %q und Bestellung %q", wer, protokollierte)
	}
	if strings.Contains(string(details), "@") {
		t.Errorf("Der Eintrag trägt eine Adresse: %s", details)
	}
}

// Ohne eingerichteten Mailserver ist die Bestellung gespeichert und trägt den Vermerk; das
// erneute Senden nennt den Grund, statt zu scheitern wie ein Serverfehler.
func TestBestellmail_OhneMailserverStehtEsAnDerBestellung(t *testing.T) {
	l := bestellmailLageAnlegen(t, "ohne")
	alterLader := smtpKonfigLader
	smtpKonfigLader = func() (mailservice.SMTPKonfig, error) { return mailservice.SMTPKonfig{}, nil }
	t.Cleanup(func() { smtpKonfigLader = alterLader })

	lieferant := haendler(t, l.pool, "Buchhandlung-Ohne", false)
	bestellung, _ := l.bestelle(t, lieferant, titelMitMeldebestand(t, l.pool, "LMF-Mathe-Ohne", 0))
	if !l.gescheitert(t, bestellung) {
		t.Fatal("Die Bestellung trägt keinen Vermerk, obwohl kein Mailserver eingerichtet ist")
	}
	rec := l.rufe(t, http.MethodPost, "/api/bestellungen/"+bestellung+"/mail", "")
	if rec.Code != http.StatusBadRequest || !strings.Contains(rec.Body.String(), "Mailserver") {
		t.Errorf("Erneut senden ohne Mailserver: Status %d, erwartet 400 mit dem Grund: %s", rec.Code, rec.Body.String())
	}
	if !l.gescheitert(t, bestellung) {
		t.Error("Der Vermerk ist verschwunden, obwohl nichts versendet wurde")
	}
}

// Eine Bestellung, deren Mail rausging, und eine unbekannte lassen sich nicht erneut senden.
func TestBestellmail_ErneutNurNachGescheitertemVersand(t *testing.T) {
	l := bestellmailLageAnlegen(t, "nur")
	sitzungen := mailAbfangen(t)
	lieferant := haendler(t, l.pool, "Buchhandlung-Nur", false)
	bestellung, _ := l.bestelle(t, lieferant, titelMitMeldebestand(t, l.pool, "LMF-Mathe-Nur", 0))
	warteAufMail(t, sitzungen)
	if l.gescheitert(t, bestellung) {
		t.Fatal("Die Bestellung trägt einen Vermerk, obwohl die Mail rausging")
	}
	if rec := l.rufe(t, http.MethodPost, "/api/bestellungen/"+bestellung+"/mail", ""); rec.Code != http.StatusConflict {
		t.Errorf("Erneut senden nach gelungenem Versand: Status %d, erwartet 409", rec.Code)
	}
	if rec := l.rufe(t, http.MethodPost, "/api/bestellungen/00000000-0000-4000-8000-000000000000/mail", ""); rec.Code != http.StatusNotFound {
		t.Errorf("Unbekannte Bestellung: Status %d, erwartet 404", rec.Code)
	}
	if rec := l.rufe(t, http.MethodPost, "/api/bestellungen/keine-kennung/mail", ""); rec.Code != http.StatusBadRequest {
		t.Errorf("Keine Kennung: Status %d, erwartet 400", rec.Code)
	}
	select {
	case zweite := <-sitzungen:
		t.Errorf("Eine weitere Mail ging raus, an %v", zweite.Empfaenger)
	case <-time.After(300 * time.Millisecond):
	}
}

// Gibt der Browser das Warten auf, während der Mailserver nicht antwortet, steht der Vermerk
// trotzdem an der Bestellung: Gerade dann hat auch niemand die Meldung gesehen. Dasselbe beim
// erneuten Senden, das der Bestellung den Vermerk für die Dauer des Versuchs nimmt.
func TestBestellmail_VermerkAuchWennDieAnfrageAbgebrochenIst(t *testing.T) {
	l := bestellmailLageAnlegen(t, "abbruch")
	lieferant := haendler(t, l.pool, "Buchhandlung-Abbruch", false)
	titel := titelMitMeldebestand(t, l.pool, "LMF-Mathe-Abbruch", 0)

	// abgebrochen schickt die Anfrage und bricht sie ab, sobald der Versand begonnen hat.
	abgebrochen := func(pfad, rumpf string) {
		t.Helper()
		ctx, abbrechen := context.WithCancel(t.Context())
		defer abbrechen()
		mailserverNichtErreichbar(t, abbrechen)
		req := httptest.NewRequest(http.MethodPost, pfad, strings.NewReader(rumpf)).WithContext(ctx)
		req.Header.Set("Content-Type", "application/json")
		req.AddCookie(&http.Cookie{Name: "session_token", Value: l.sitzung})
		l.router.ServeHTTP(httptest.NewRecorder(), mitCSRF(req))
		if ctx.Err() == nil {
			t.Fatal("Die Anfrage ist nicht abgebrochen worden; der Versand hat nicht begonnen")
		}
	}
	vermerkt := func() bool {
		t.Helper()
		var gescheitert bool
		if err := l.pool.QueryRow(t.Context(), `
			SELECT mail_gescheitert_am IS NOT NULL FROM bestellungen_verlauf WHERE lieferant_id = $1`,
			lieferant).Scan(&gescheitert); err != nil {
			t.Fatalf("Bestellung lesen: %v", err)
		}
		return gescheitert
	}

	abgebrochen("/api/bestellungen", fmt.Sprintf(
		`{"supplier_id":%q,"mittel":"land","items":[{"titel_id":%q,"menge":1,"preis":9.5,"generate_barcodes":true}]}`,
		lieferant, titel))
	if !vermerkt() {
		t.Fatal("Die Bestellung trägt keinen Vermerk, nachdem die Anfrage während des Versands abbrach")
	}

	var bestellung string
	if err := l.pool.QueryRow(t.Context(),
		`SELECT id FROM bestellungen_verlauf WHERE lieferant_id = $1`, lieferant).Scan(&bestellung); err != nil {
		t.Fatal(err)
	}
	abgebrochen("/api/bestellungen/"+bestellung+"/mail", "")
	if !vermerkt() {
		t.Error("Nach einem abgebrochenen erneuten Versuch steht die Bestellung als versendet da")
	}
}

// zwischen liefert die Zeichen hinter dem Anfang bis zum nächsten Leerraum oder Anführungszeichen.
func zwischen(text, anfang string) string {
	_, rest, gefunden := strings.Cut(text, anfang)
	if !gefunden {
		return ""
	}
	ende := strings.IndexAny(rest, " \r\n\"<>")
	if ende < 0 {
		return rest
	}
	return rest[:ende]
}

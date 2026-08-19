package api

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"bibliothek/db"
	"bibliothek/internal/smtptest"
	"bibliothek/mailservice"
	"bibliothek/repository"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Was kommt beim Händler WIRKLICH an?
//
// Am 06.08.2026 ging eine Bestellung an den Hauptlieferanten raus, die Oberfläche meldete
// "erfolgreich gesendet", und in der Mail lagen vier PDFs — aber kein Bestätigungs-Link.
// Kein einziger Test war rot: Die Link-Erzeugung, die öffentliche Seite und die Bestätigung
// waren alle für sich in Ordnung. Es fehlte eine EINSTELLUNG, und ihr Fehlen fiel auf dem
// Weg von der Einstellung bis in die Mail nirgends auf.
//
// Deshalb steht dieses Gate am Ende der Kette, an der fertigen Nachricht auf dem Draht:
// echter Handler, echte Transaktion, echter SMTP-Versand — nur der Mailserver ist ein
// Testserver, damit niemals eine Mail an den Schulserver rausgeht (siehe internal/smtptest).

// mailAbfangen biegt den Versand auf einen Testserver um und liefert die abgefangene
// Nachricht. Der Kanal nimmt genau eine Sitzung an.
func mailAbfangen(t *testing.T) <-chan smtptest.Sitzung {
	t.Helper()
	// Der Testserver kündigt kein STARTTLS an; sichereVerbindung bräche den Versand
	// sonst ab. Dass die Erzwingung wirkt, prüft mailservice.
	t.Setenv("SMTP_ALLOW_PLAINTEXT", "true")
	host, port, sitzungen := smtptest.Starte(t, smtptest.Normal)

	alterLader := smtpKonfigLader
	smtpKonfigLader = func() (mailservice.SMTPKonfig, error) {
		return mailservice.SMTPKonfig{Host: host, Port: port, Absender: "bibliothek@schule.invalid"}, nil
	}
	t.Cleanup(func() { smtpKonfigLader = alterLader })

	return sitzungen
}

// setzeOeffentlicheAdresse hinterlegt (oder entfernt) die Adresse, aus der der
// Bestätigungs-Link entsteht.
func setzeOeffentlicheAdresse(t *testing.T, pool *pgxpool.Pool, adresse string) {
	t.Helper()
	ctx := context.Background()
	var err error
	if adresse == "" {
		_, err = pool.Exec(ctx, `DELETE FROM system_einstellungen WHERE schluessel = 'oeffentliche_adresse'`)
	} else {
		_, err = pool.Exec(ctx, `
			INSERT INTO system_einstellungen (schluessel, wert) VALUES ('oeffentliche_adresse', $1)
			ON CONFLICT (schluessel) DO UPDATE SET wert = EXCLUDED.wert`, adresse)
	}
	if err != nil {
		t.Fatalf("öffentliche Adresse setzen: %v", err)
	}
}

// bestelleUeberHandler löst eine Bestellung über den echten HTTP-Handler aus und liefert
// die Antwort.
func bestelleUeberHandler(t *testing.T, srv *Server, lieferantID, titelID string) *httptest.ResponseRecorder {
	t.Helper()
	rumpf := fmt.Sprintf(
		`{"supplier_id":%q,"items":[{"titel_id":%q,"menge":2,"preis":9.5,"generate_barcodes":true}]}`,
		lieferantID, titelID)

	req := httptest.NewRequest(http.MethodPost, "/api/bestellungen", strings.NewReader(rumpf))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler := srv.SubmitOrderHandler(NewOrderService(srv.DB, repository.NewBookRepository(srv.DB.Pool)), NewPDFService())
	handler(rec, req)
	return rec
}

// warteAufMail holt die abgefangene Nachricht — mit Frist, damit ein ausbleibender
// Versand als klarer Fehlschlag endet und nicht als hängender Test.
func warteAufMail(t *testing.T, sitzungen <-chan smtptest.Sitzung) string {
	t.Helper()
	select {
	case s := <-sitzungen:
		return s.Nachricht
	case <-time.After(10 * time.Second):
		t.Fatal("es wurde gar keine Mail verschickt")
		return ""
	}
}

func TestBestellversand_MitAdresseGehtDerLinkRausUndKeineEtikettenboegen(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)

	setzeOeffentlicheAdresse(t, pool, "https://bib.example.invalid")
	sitzungen := mailAbfangen(t)

	srv := &Server{DB: &db.Database{Pool: pool}}
	lieferant := haendler(t, pool, "Naacher-Link", true)
	titel := titelMitMeldebestand(t, pool, "LMF-Mathe-Link", 0)

	rec := bestelleUeberHandler(t, srv, lieferant, titel)
	if rec.Code != http.StatusOK {
		t.Fatalf("Status = %d, want 200: %s", rec.Code, rec.Body.String())
	}
	if antwort := rec.Body.String(); !strings.Contains(antwort, `"success"`) {
		t.Errorf("Antwort meldet keinen Erfolg, obwohl der Link mitging: %s", antwort)
	}

	nachricht := warteAufMail(t, sitzungen)

	// Der Link IST der Ablauf: Ohne ihn kann der Händler nicht bestätigen, und genau das
	// war der gemeldete Fehler.
	if !strings.Contains(nachricht, "https://bib.example.invalid/bestellung/") {
		t.Errorf("kein Bestätigungs-Link in der Mail:\n%s", kopf(nachricht))
	}
	// Und die Etiketten liegen NICHT daneben im Postfach: Wer sie von dort druckt,
	// klickt den Link nie — die Schule wartete dann auf eine Bestätigung, die nie kommt.
	for _, name := range []string{"etiketten_klein", "etiketten_gross"} {
		if strings.Contains(nachricht, name) {
			t.Errorf("%s hängt an der Mail, obwohl die Etiketten hinter dem Link liegen", name)
		}
	}
	for _, name := range []string{"bestellanschreiben", "barcode_mapping"} {
		if !strings.Contains(nachricht, name) {
			t.Errorf("%s fehlt in der Mail", name)
		}
	}
}

// Die Gegenrichtung — und der eigentliche Vorfall: keine öffentliche Adresse hinterlegt.
// Die Mail geht raus (die Bestellung ist ja gebucht), aber sie sagt es, statt "erfolgreich"
// zu melden. Und der Etikettenbogen MUSS dann beiliegen: Der Händler beklebt trotzdem
// selbst, seine Exemplare gelten bereits als etikettiert.
func TestBestellversand_OhneAdresseWarntUndLegtDieBoegenBei(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)

	setzeOeffentlicheAdresse(t, pool, "")
	sitzungen := mailAbfangen(t)

	srv := &Server{DB: &db.Database{Pool: pool}}
	lieferant := haendler(t, pool, "Naacher-OhneAdresse", true)
	titel := titelMitMeldebestand(t, pool, "LMF-Mathe-OhneAdresse", 0)

	rec := bestelleUeberHandler(t, srv, lieferant, titel)
	if rec.Code != http.StatusOK {
		t.Fatalf("Status = %d, want 200: %s", rec.Code, rec.Body.String())
	}
	antwort := rec.Body.String()
	if !strings.Contains(antwort, `"warning"`) || !strings.Contains(antwort, "öffentliche Adresse") {
		t.Errorf("die Antwort verschweigt den fehlenden Link:\n%s", antwort)
	}

	nachricht := warteAufMail(t, sitzungen)
	if strings.Contains(nachricht, "/bestellung/") {
		t.Error("Link in der Mail, obwohl keine öffentliche Adresse hinterlegt ist")
	}
	for _, name := range []string{"etiketten_klein", "etiketten_gross"} {
		if !strings.Contains(nachricht, name) {
			t.Errorf("%s fehlt — der Händler hätte nichts zum Bekleben", name)
		}
	}
}

// kopf kürzt die Nachricht auf den Textteil; die base64-kodierten Anhänge machen jede
// Fehlermeldung sonst unlesbar.
func kopf(nachricht string) string {
	if len(nachricht) > 1200 {
		return nachricht[:1200] + "\n… (gekürzt)"
	}
	return nachricht
}

// bestelleMitSchluessel löst eine Bestellung über den echten Handler MIT Idempotenz-
// Schlüssel aus.
func bestelleMitSchluessel(t *testing.T, srv *Server, lieferantID, titelID, key string) *httptest.ResponseRecorder {
	t.Helper()
	rumpf := fmt.Sprintf(
		`{"supplier_id":%q,"idempotency_key":%q,"items":[{"titel_id":%q,"menge":2,"preis":9.5,"generate_barcodes":true}]}`,
		lieferantID, key, titelID)
	req := httptest.NewRequest(http.MethodPost, "/api/bestellungen", strings.NewReader(rumpf))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler := srv.SubmitOrderHandler(NewOrderService(srv.DB, repository.NewBookRepository(srv.DB.Pool)), NewPDFService())
	handler(rec, req)
	return rec
}

// TestBestellungIdempotenz belegt den Doppelklick-Schutz (Migration 077): Zwei
// Bestellungen mit DEMSELBEN Idempotenz-Schlüssel erzeugen nur EINE Bestellung und
// GENAU EINE Lieferanten-Mail; die zweite meldet „bereits erfasst".
func TestBestellungIdempotenz(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	ctx := context.Background()
	setzeOeffentlicheAdresse(t, pool, "")
	sitzungen := mailAbfangen(t)

	srv := &Server{DB: &db.Database{Pool: pool}}
	lieferant := haendler(t, pool, "IdemLieferant", false)
	titel := titelMitMeldebestand(t, pool, "LMF-Idem", 0)
	key := "33333333-3333-3333-3333-333333333333"

	rec1 := bestelleMitSchluessel(t, srv, lieferant, titel, key)
	if rec1.Code != http.StatusOK {
		t.Fatalf("erste Bestellung Status %d: %s", rec1.Code, rec1.Body.String())
	}
	// Erste Bestellung löst genau eine Mail aus.
	_ = warteAufMail(t, sitzungen)

	// Doppelklick: gleicher Schlüssel → keine zweite Bestellung, keine zweite Mail.
	rec2 := bestelleMitSchluessel(t, srv, lieferant, titel, key)
	if rec2.Code != http.StatusOK {
		t.Fatalf("zweite Bestellung Status %d: %s", rec2.Code, rec2.Body.String())
	}
	if !strings.Contains(rec2.Body.String(), "bereits erfasst") {
		t.Errorf("zweiter Klick muss „bereits erfasst\" melden: %s", rec2.Body.String())
	}

	// Genau eine Bestellung in der DB.
	var anzahl int
	if err := pool.QueryRow(ctx,
		`SELECT count(*) FROM bestellungen_verlauf WHERE idempotenz_schluessel = $1`, key).Scan(&anzahl); err != nil {
		t.Fatal(err)
	}
	if anzahl != 1 {
		t.Fatalf("genau 1 Bestellung erwartet, waren %d", anzahl)
	}

	// KEINE zweite Mail: Der Kanal darf jetzt leer sein.
	select {
	case s := <-sitzungen:
		t.Fatalf("es ging eine ZWEITE Mail raus (Doppelklick nicht abgefangen): %s", kopf(s.Nachricht))
	case <-time.After(1 * time.Second):
		// gut — keine zweite Mail
	}
}

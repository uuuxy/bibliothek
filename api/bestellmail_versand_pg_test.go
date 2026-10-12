package api

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"bibliothek/api/tuer"
	"bibliothek/db"
	"bibliothek/internal/pgtest"
	"bibliothek/internal/service"
	"bibliothek/internal/smtptest"
	"bibliothek/internal/testdaten"
	"bibliothek/internal/tuertest"
	"bibliothek/pkg/schulzeit"
	"bibliothek/repository"
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

// bestelleUeberHandler löst eine Bestellung über den echten HTTP-Handler aus und liefert
// die Antwort.
func bestelleUeberHandler(t *testing.T, srv *Server, lieferantID, titelID string) *httptest.ResponseRecorder {
	t.Helper()
	rumpf := fmt.Sprintf(
		`{"supplier_id":%q,"mittel":"land","items":[{"titel_id":%q,"menge":2,"preis":9.5,"generate_barcodes":true}]}`,
		lieferantID, titelID)

	req := httptest.NewRequest(http.MethodPost, "/api/bestellungen", strings.NewReader(rumpf))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler := srv.SubmitOrderHandler(service.NewOrderService(srv.DB, repository.NewBookRepository(srv.DB.Pool)))
	handler(rec, req)
	return rec
}

func TestBestellversand_MitAdresseGehtDerLinkRausUndKeineEtikettenboegen(t *testing.T) {
	pool := pgtest.Pool(t)
	testdaten.ResetBestandsdaten(t, pool)

	testdaten.SetzeOeffentlicheAdresse(t, pool, "https://bib.example.invalid")
	sitzungen := tuertest.MailAbfangen(t)

	srv := &Server{Basis: tuer.Basis{DB: &db.Database{Pool: pool}}}
	lieferant := testdaten.Haendler(t, pool, "Naacher-Link", true)
	titel := testdaten.TitelMitMeldebestand(t, pool, "LMF-Mathe-Link", 0)

	rec := bestelleUeberHandler(t, srv, lieferant, titel)
	if rec.Code != http.StatusOK {
		t.Fatalf("Status = %d, want 200: %s", rec.Code, rec.Body.String())
	}
	if antwort := rec.Body.String(); !strings.Contains(antwort, `"success"`) {
		t.Errorf("Antwort meldet keinen Erfolg, obwohl der Link mitging: %s", antwort)
	}

	nachricht := smtptest.WarteAufMail(t, sitzungen)

	// Der Link IST der Ablauf: Ohne ihn kann der Händler nicht bestätigen, und genau das
	// war der gemeldete Fehler.
	if !strings.Contains(nachricht, "https://bib.example.invalid/bestellung/") {
		t.Errorf("kein Bestätigungs-Link in der Mail:\n%s", smtptest.Kopf(nachricht))
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
	pool := pgtest.Pool(t)
	testdaten.ResetBestandsdaten(t, pool)

	testdaten.SetzeOeffentlicheAdresse(t, pool, "")
	sitzungen := tuertest.MailAbfangen(t)

	srv := &Server{Basis: tuer.Basis{DB: &db.Database{Pool: pool}}}
	lieferant := testdaten.Haendler(t, pool, "Naacher-OhneAdresse", true)
	titel := testdaten.TitelMitMeldebestand(t, pool, "LMF-Mathe-OhneAdresse", 0)

	rec := bestelleUeberHandler(t, srv, lieferant, titel)
	if rec.Code != http.StatusOK {
		t.Fatalf("Status = %d, want 200: %s", rec.Code, rec.Body.String())
	}
	antwort := rec.Body.String()
	if !strings.Contains(antwort, `"warning"`) || !strings.Contains(antwort, "öffentliche Adresse") {
		t.Errorf("die Antwort verschweigt den fehlenden Link:\n%s", antwort)
	}

	nachricht := smtptest.WarteAufMail(t, sitzungen)
	if strings.Contains(nachricht, "/bestellung/") {
		t.Error("Link in der Mail, obwohl keine öffentliche Adresse hinterlegt ist")
	}
	for _, name := range []string{"etiketten_klein", "etiketten_gross"} {
		if !strings.Contains(nachricht, name) {
			t.Errorf("%s fehlt — der Händler hätte nichts zum Bekleben", name)
		}
	}
}

// Die Mail nennt die Kundennummer und zählt Titel und Exemplare: Ein Titel in zwei Exemplaren
// kommt beim Händler nicht als zwei Titel in einem Exemplar an.
func TestBestellversand_MailNenntKundennummerTitelUndExemplare(t *testing.T) {
	pool := pgtest.Pool(t)
	testdaten.ResetBestandsdaten(t, pool)

	testdaten.SetzeOeffentlicheAdresse(t, pool, "")
	sitzungen := tuertest.MailAbfangen(t)

	srv := &Server{Basis: tuer.Basis{DB: &db.Database{Pool: pool}}}
	lieferant := testdaten.Haendler(t, pool, "Naacher-Zahlen", false)
	titel := testdaten.TitelMitMeldebestand(t, pool, "LMF-Mathe-Zahlen", 0)

	if rec := bestelleUeberHandler(t, srv, lieferant, titel); rec.Code != http.StatusOK {
		t.Fatalf("Status = %d, want 200: %s", rec.Code, rec.Body.String())
	}

	nachricht := smtptest.WarteAufMail(t, sitzungen)
	for _, zeile := range []string{"Kundennummer: K-Naacher-Zahlen", "Bestellte Titel: 1", "Gesamtanzahl Exemplare: 2"} {
		if !strings.Contains(nachricht, zeile) {
			t.Errorf("%q fehlt in der Mail:\n%s", zeile, smtptest.Kopf(nachricht))
		}
	}
}

// Die Mail nennt den Topf der Bestellung, im Betreff und im Text, und den Ablauf des Links als
// Kalendertag der Schule. Beides reicht die Tür an den Mailtext; fehlt eines, sieht die Mail
// trotzdem vollständig aus.
func TestBestellversand_MailNenntTopfUndAblaufDesLinks(t *testing.T) {
	pool := pgtest.Pool(t)
	testdaten.ResetBestandsdaten(t, pool)

	testdaten.SetzeOeffentlicheAdresse(t, pool, "https://bib.example.invalid")
	sitzungen := tuertest.MailAbfangen(t)

	srv := &Server{Basis: tuer.Basis{DB: &db.Database{Pool: pool}}}
	lieferant := testdaten.Haendler(t, pool, "Naacher-Topf", true)
	titel := testdaten.TitelMitMeldebestand(t, pool, "LMF-Mathe-Topf", 0)

	if rec := bestelleUeberHandler(t, srv, lieferant, titel); rec.Code != http.StatusOK {
		t.Fatalf("Status = %d, want 200: %s", rec.Code, rec.Body.String())
	}
	var gueltigBis time.Time
	if err := pool.QueryRow(context.Background(),
		`SELECT token_gueltig_bis FROM bestellungen_verlauf WHERE lieferant_id = $1`, lieferant).Scan(&gueltigBis); err != nil {
		t.Fatalf("Ablauf des Links lesen: %v", err)
	}

	nachricht := smtptest.WarteAufMail(t, sitzungen)
	for _, zeile := range []string{
		"Subject: Buchbestellung Lernmittelfreiheit - ",
		"Diese Bestellung: Lernmittelfreiheit",
		"bis zum " + gueltigBis.In(schulzeit.Zone()).Format(tuer.DateFormatDE),
	} {
		if !strings.Contains(nachricht, zeile) {
			t.Errorf("%q fehlt in der Mail:\n%s", zeile, smtptest.Kopf(nachricht))
		}
	}
}

// Bestellt ohne Vorab-Barcode zählt die Mail die Exemplare trotzdem: Die Zahl kam aus den
// Etiketten des Barcodebogens, und eine Position ohne Vorab-Barcode hat keine.
func TestBestellversand_MailZaehltExemplareOhneVorabBarcode(t *testing.T) {
	pool := pgtest.Pool(t)
	testdaten.ResetBestandsdaten(t, pool)

	testdaten.SetzeOeffentlicheAdresse(t, pool, "")
	sitzungen := tuertest.MailAbfangen(t)

	srv := &Server{Basis: tuer.Basis{DB: &db.Database{Pool: pool}}}
	lieferant := testdaten.Haendler(t, pool, "Naacher-OhneBarcode", false)
	mitBarcode := testdaten.TitelMitMeldebestand(t, pool, "LMF-Mathe-MitBarcode", 0)
	ohneBarcode := testdaten.TitelMitMeldebestand(t, pool, "LMF-Mathe-OhneBarcode", 0)

	rumpf := fmt.Sprintf(`{"supplier_id":%q,"mittel":"land","items":[
		{"titel_id":%q,"menge":2,"preis":9.5,"generate_barcodes":true},
		{"titel_id":%q,"menge":3,"preis":9.5,"generate_barcodes":false}]}`, lieferant, mitBarcode, ohneBarcode)
	req := httptest.NewRequest(http.MethodPost, "/api/bestellungen", strings.NewReader(rumpf))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	srv.SubmitOrderHandler(service.NewOrderService(srv.DB, repository.NewBookRepository(srv.DB.Pool)))(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("Status = %d, want 200: %s", rec.Code, rec.Body.String())
	}

	nachricht := smtptest.WarteAufMail(t, sitzungen)
	for _, zeile := range []string{"Bestellte Titel: 2", "Gesamtanzahl Exemplare: 5"} {
		if !strings.Contains(nachricht, zeile) {
			t.Errorf("%q fehlt in der Mail:\n%s", zeile, smtptest.Kopf(nachricht))
		}
	}
}

// bestelleMitSchluessel löst eine Bestellung über den echten Handler MIT Idempotenz-
// Schlüssel aus.
func bestelleMitSchluessel(t *testing.T, srv *Server, lieferantID, titelID, key string) *httptest.ResponseRecorder {
	t.Helper()
	rumpf := fmt.Sprintf(
		`{"supplier_id":%q,"idempotency_key":%q,"mittel":"land","items":[{"titel_id":%q,"menge":2,"preis":9.5,"generate_barcodes":true}]}`,
		lieferantID, key, titelID)
	req := httptest.NewRequest(http.MethodPost, "/api/bestellungen", strings.NewReader(rumpf))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler := srv.SubmitOrderHandler(service.NewOrderService(srv.DB, repository.NewBookRepository(srv.DB.Pool)))
	handler(rec, req)
	return rec
}

// TestBestellungIdempotenz belegt den Doppelklick-Schutz (Migration 077): Zwei
// Bestellungen mit DEMSELBEN Idempotenz-Schlüssel erzeugen nur EINE Bestellung und
// GENAU EINE Lieferanten-Mail; die zweite meldet „bereits erfasst".
func TestBestellungIdempotenz(t *testing.T) {
	pool := pgtest.Pool(t)
	testdaten.ResetBestandsdaten(t, pool)
	ctx := context.Background()
	testdaten.SetzeOeffentlicheAdresse(t, pool, "")
	sitzungen := tuertest.MailAbfangen(t)

	srv := &Server{Basis: tuer.Basis{DB: &db.Database{Pool: pool}}}
	lieferant := testdaten.Haendler(t, pool, "IdemLieferant", false)
	titel := testdaten.TitelMitMeldebestand(t, pool, "LMF-Idem", 0)
	key := "33333333-3333-3333-3333-333333333333"

	rec1 := bestelleMitSchluessel(t, srv, lieferant, titel, key)
	if rec1.Code != http.StatusOK {
		t.Fatalf("erste Bestellung Status %d: %s", rec1.Code, rec1.Body.String())
	}
	// Erste Bestellung löst genau eine Mail aus.
	_ = smtptest.WarteAufMail(t, sitzungen)

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
		t.Fatalf("es ging eine ZWEITE Mail raus (Doppelklick nicht abgefangen): %s", smtptest.Kopf(s.Nachricht))
	case <-time.After(1 * time.Second):
		// gut — keine zweite Mail
	}
}

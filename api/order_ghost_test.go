package api

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"bibliothek/db"
)

// TestSendOrderMail_SmtpFehlerRolltZurueck sichert #3 (Ghost-Orders) ab: Schlägt der
// Mailversand fehl, dürfen KEINE Bestell-Platzhalter in der DB zurückbleiben. Vorher
// wurde erst committet und dann gemailt — bei SMTP-Ausfall waren die Bücher „bestellt",
// der Händler hatte aber nie eine Mail.
func TestSendOrderMail_SmtpFehlerRolltZurueck(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	ctx := context.Background()

	// Ein Titel mit echtem Bestellbedarf (Meldebestand 3, nichts vorhanden).
	titelMitMeldebestand(t, pool, "Fehlt komplett", 3)

	// SendEmail für diesen Test scheitern lassen und danach wiederherstellen.
	orig := SendEmail
	SendEmail = func(MailRequest) error { return errors.New("SMTP down") }
	defer func() { SendEmail = orig }()

	srv := &Server{DB: &db.Database{Pool: pool}}
	req := httptest.NewRequest(http.MethodPost, "/api/bestellung/mail",
		strings.NewReader(`{"email":"lieferant@example.org"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	srv.SendOrderMailHandler().ServeHTTP(rec, req)

	// Der Handler meldet den Mail-Fehler (502).
	if rec.Code != http.StatusBadGateway {
		t.Errorf("erwartet 502 bei SMTP-Fehler, war %d: %s", rec.Code, rec.Body.String())
	}

	// Entscheidend: KEINE Bestell-Platzhalter in der DB (Transaktion zurückgerollt).
	var platzhalter int
	if err := pool.QueryRow(ctx,
		`SELECT count(*) FROM buecher_exemplare WHERE zustand_notiz = 'bestellt'`).Scan(&platzhalter); err != nil {
		t.Fatal(err)
	}
	if platzhalter != 0 {
		t.Errorf("nach SMTP-Fehler blieben %d Bestell-Platzhalter zurück (Ghost-Order)", platzhalter)
	}
}

// TestSendOrderMail_ErfolgCommittet: Der Positivfall — geht die Mail raus, werden die
// Platzhalter dauerhaft.
func TestSendOrderMail_ErfolgCommittet(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	ctx := context.Background()

	titelMitMeldebestand(t, pool, "Fehlt auch", 2)

	orig := SendEmail
	var gesendetAn string
	SendEmail = func(r MailRequest) error { gesendetAn = r.To; return nil }
	defer func() { SendEmail = orig }()

	srv := &Server{DB: &db.Database{Pool: pool}}
	req := httptest.NewRequest(http.MethodPost, "/api/bestellung/mail",
		strings.NewReader(`{"email":"lieferant@example.org"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	srv.SendOrderMailHandler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("erwartet 200, war %d: %s", rec.Code, rec.Body.String())
	}
	if gesendetAn != "lieferant@example.org" {
		t.Errorf("Mail-Empfänger: erwartet lieferant@example.org, war %q", gesendetAn)
	}
	var platzhalter int
	if err := pool.QueryRow(ctx,
		`SELECT count(*) FROM buecher_exemplare WHERE zustand_notiz = 'bestellt'`).Scan(&platzhalter); err != nil {
		t.Fatal(err)
	}
	if platzhalter != 2 {
		t.Errorf("nach erfolgreicher Bestellung: erwartet 2 Platzhalter, waren %d", platzhalter)
	}
}

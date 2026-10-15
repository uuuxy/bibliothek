package bestellung

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"bibliothek/api/tuer"
	"bibliothek/db"
	"bibliothek/internal/pgtest"
	"bibliothek/internal/smtptest"
	"bibliothek/internal/testdaten"
	"bibliothek/internal/tuertest"
	"bibliothek/pkg/mitteltopf"
)

// Lassen sich die Einstellungen nicht lesen, liefert das Lesen die Vorgaben des Programms: kein
// Name der Schule, keine öffentliche Adresse. Ein Blatt oder eine Mail daraus sähe vollständig
// aus. Beide Türen müssen den Lesefehler deshalb als Fehler behandeln.

func TestBestellbericht_EinstellungenUnlesbar_KeinBlatt(t *testing.T) {
	pool := pgtest.Pool(t)
	testdaten.LeereBestandsdaten(t, pool)

	rufe := func(datenbank db.Pool) *httptest.ResponseRecorder {
		srv := &Tueren{Basis: &tuer.Basis{DB: &db.Verbindung{Pool: datenbank}}}
		req := httptest.NewRequest(http.MethodGet, "/api/bestellungen/bericht.pdf?von=2019-09-01&bis=2019-09-30", nil)
		rec := httptest.NewRecorder()
		srv.BestellberichtPDF().ServeHTTP(rec, req)
		return rec
	}

	rec := rufe(testdaten.PoolMitLesefehler{Pool: pool, Muster: testdaten.SQLEinstellungen})
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("Status %d, erwartet 500: Der Bericht entsteht mit den Vorgaben statt mit den Angaben der Schule", rec.Code)
	}
	if bytes.HasPrefix(rec.Body.Bytes(), []byte("%PDF-")) {
		t.Error("trotz Lesefehler kommt ein Blatt zurück")
	}

	// Gegenprobe: Am ungestörten Pool entsteht das Blatt. Ohne sie bestünde der Test auch an
	// einer Tür, die immer 500 antwortet.
	if rec := rufe(pool); rec.Code != http.StatusOK || !bytes.HasPrefix(rec.Body.Bytes(), []byte("%PDF-")) {
		t.Errorf("ungestört: Status %d, erwartet 200 mit einem Blatt", rec.Code)
	}
}

func TestBestellmail_EinstellungenUnlesbar_KeineMail(t *testing.T) {
	pool := pgtest.Pool(t)
	testdaten.LeereBestandsdaten(t, pool)
	testdaten.SetzeOeffentlicheAdresse(t, pool, "https://bibliothek.example.invalid")
	t.Cleanup(func() { testdaten.SetzeOeffentlicheAdresse(t, pool, "") })

	daten := bestellmailDaten{
		Empfaenger:        "haendler@example.invalid",
		Kundennummer:      "K-7",
		Mittel:            mitteltopf.Land,
		Exemplare:         1,
		IstHauptlieferant: true,
		Token:             "token-lesefehler",
		Positionen:        []bestellterTitel{{Titel: "Titel 1", Autor: "Autor 1", ISBN: "isbn-1", Verlag: "Verlag 1", Menge: 1}},
	}

	t.Run("gestört", func(t *testing.T) {
		post, sitzungen := tuertest.FangeMailAb(t)
		gestoert := testdaten.PoolMitLesefehler{Pool: pool, Muster: testdaten.SQLEinstellungen}
		srv := &Tueren{Basis: &tuer.Basis{DB: &db.Verbindung{Pool: gestoert}, Mail: post}}

		_, err := srv.sendeBestellmail(context.Background(), daten)
		if err == nil {
			t.Error("der Versand meldet Erfolg, obwohl die Einstellungen nicht lesbar waren")
		}
		select {
		case s := <-sitzungen:
			t.Errorf("an den Lieferanten ging eine Mail ohne die Angaben der Schule und ohne Link: %s", smtptest.Kopf(s.Nachricht))
		case <-time.After(500 * time.Millisecond):
		}
	})

	// Gegenprobe: Am ungestörten Pool geht die Mail hinaus und trägt den Link.
	t.Run("ungestört", func(t *testing.T) {
		post, sitzungen := tuertest.FangeMailAb(t)
		srv := &Tueren{Basis: &tuer.Basis{DB: &db.Verbindung{Pool: pool}, Mail: post}}

		mitLink, err := srv.sendeBestellmail(context.Background(), daten)
		if err != nil {
			t.Fatalf("Versand: %v", err)
		}
		if !mitLink {
			t.Error("die Mail ging ohne Link hinaus, obwohl eine öffentliche Adresse hinterlegt ist")
		}
		if nachricht := smtptest.WarteAufMail(t, sitzungen); !strings.Contains(nachricht, "haendler@example.invalid") {
			t.Errorf("die Mail nennt den Lieferanten nicht: %s", smtptest.Kopf(nachricht))
		}
	})
}

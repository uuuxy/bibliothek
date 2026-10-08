package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"bibliothek/db"
	"bibliothek/internal/service"
	"bibliothek/inventur"
)

// POST /api/bestellungen/suche sucht einen Titel zum Bestellen im eigenen Katalog und in der
// DNB. Gemessen am 08.10.2026 führte kein Go-Test die Tür aus (13,3 %, OFFEN.md 5.10); die
// Suche dahinter prüft order_service_test.go mit nachgespielten Datenbank-Antworten.
//
// Hier mit echter Datenbank und nachgestellter DNB: was die Person im Bestellwesen an einem
// Treffer abliest, bevor sie bestellt — ob der Titel schon im Haus ist und wie viele
// Exemplare davon im Bestand stehen.

func dnbSatz(isbn, titel string) string {
	return fmt.Sprintf(`<record><recordData><record xmlns="http://www.loc.gov/MARC21/slim">
		<datafield tag="020" ind1=" " ind2=" "><subfield code="a">%s</subfield></datafield>
		<datafield tag="245" ind1="1" ind2="0"><subfield code="a">%s</subfield></datafield>
	</record></recordData></record>`, isbn, titel)
}

func TestBestellsuche_UeberDieTuer(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	srv := &Server{DB: &db.Database{Pool: pool}}

	// Der Katalog führt die ISBN dreizehnstellig; die zehnstellige Form desselben Buchs endet
	// auf ein anderes Prüfzeichen, ein Teilstring-Vergleich fände sie nicht.
	const isbn13, isbn10 = "9789991940021", "9-99194-002-2"
	var imHaus string
	if err := pool.QueryRow(t.Context(), `
		INSERT INTO buecher_titel (titel, autor, isbn, ist_lernmittel)
		VALUES ('Bestellsuche Probeband', 'Muster, Erika', $1, true) RETURNING id`, isbn13).Scan(&imHaus); err != nil {
		t.Fatal(err)
	}
	exemplar(t, pool, imHaus, "BSU-1", true, "")
	exemplar(t, pool, imHaus, "BSU-2", true, "")
	if _, err := pool.Exec(t.Context(), `
		UPDATE buecher_exemplare SET ist_ausgesondert = true, aussonderung_grund = 'AUSSORTIERT', ist_ausleihbar = false
		WHERE barcode_id = 'BSU-2'`); err != nil {
		t.Fatal(err)
	}

	dnbAntwort := `<searchRetrieveResponse xmlns="http://www.loc.gov/zing/srw/"><records>` +
		dnbSatz(isbn10, "Bestellsuche Probeband") + dnbSatz("9789991940045", "Ein anderes Buch") +
		`</records></searchRetrieveResponse>`

	// ausfall ist der Kopf der jüngsten Antwort, mit dem die Tür einen Ausfall der DNB nennt.
	var ausfall string
	suche := func(t *testing.T, dnb http.RoundTripper, rumpf string) (int, []service.OrderSearchItem, string) {
		t.Helper()
		client := inventur.NeuerMetadatenClient()
		client.SetzeHTTPClientFuerTest(&http.Client{Transport: dnb})
		rec := httptest.NewRecorder()
		srv.sucheBestellung(client)(rec, httptest.NewRequest(http.MethodPost, "/api/bestellungen/suche", strings.NewReader(rumpf)))
		ausfall = rec.Header().Get(dnbAusfallKopf)
		var treffer []service.OrderSearchItem
		if rec.Code == http.StatusOK {
			if err := json.Unmarshal(rec.Body.Bytes(), &treffer); err != nil {
				t.Fatalf("Antwort unlesbar (%s): %v", rec.Body.String(), err)
			}
		}
		return rec.Code, treffer, rec.Body.String()
	}
	aus := func(treffer []service.OrderSearchItem, quelle string) []service.OrderSearchItem {
		var gefiltert []service.OrderSearchItem
		for _, tr := range treffer {
			if tr.Source == quelle {
				gefiltert = append(gefiltert, tr)
			}
		}
		return gefiltert
	}

	t.Run("Treffer im Katalog: mit Bestand ohne Ausgesondertes und als Lernmittel", func(t *testing.T) {
		code, treffer, rumpf := suche(t, dnbAttrappe{satz: dnbAntwort}, `{"query":"Probeband"}`)
		if code != http.StatusOK {
			t.Fatalf("Status %d, erwartet 200: %s", code, rumpf)
		}
		lokal := aus(treffer, "local")
		if len(lokal) != 1 || lokal[0].ID != imHaus {
			t.Fatalf("Treffer aus dem Katalog: %+v — erwartet genau den Probeband", lokal)
		}
		if lokal[0].CurrentStock != 1 {
			t.Errorf("Bestand %d, erwartet 1 (zwei Exemplare, eines ausgesondert)", lokal[0].CurrentStock)
		}
		if !lokal[0].IstLernmittel {
			t.Error("der Treffer trägt das Merkmal Lernmittel nicht; daran hängt der Vorschlag für den Topf")
		}
		if ausfall != "" {
			t.Errorf("die DNB hat geantwortet, die Antwort nennt einen Ausfall (%s: %q)", dnbAusfallKopf, ausfall)
		}
	})

	// Die DNB nennt ältere Bücher zehnstellig und mit Bindestrichen. Fehlt die Markierung,
	// legt die Person einen Titel an, den es schon gibt.
	t.Run("Treffer der DNB zu einem Titel im Haus ist als vorhanden markiert", func(t *testing.T) {
		_, treffer, rumpf := suche(t, dnbAttrappe{satz: dnbAntwort}, `{"query":"Probeband"}`)
		dnb := aus(treffer, "dnb")
		if len(dnb) != 2 {
			t.Fatalf("%d Treffer der DNB, erwartet 2: %s", len(dnb), rumpf)
		}
		for _, tr := range dnb {
			// Die Suche gibt die ISBN der DNB ohne Bindestriche aus.
			if erwartet := tr.ISBN == strings.ReplaceAll(isbn10, "-", ""); tr.IsDuplicate != erwartet {
				t.Errorf("%q (ISBN %s): vorhanden = %v, erwartet %v", tr.Titel, tr.ISBN, tr.IsDuplicate, erwartet)
			}
		}
	})

	t.Run("die zehnstellige ISBN findet den Titel im Katalog", func(t *testing.T) {
		_, treffer, rumpf := suche(t, dnbAttrappe{satz: dnbAntwort}, `{"query":"`+isbn10+`"}`)
		if lokal := aus(treffer, "local"); len(lokal) != 1 || lokal[0].ID != imHaus {
			t.Errorf("Treffer aus dem Katalog: %+v — erwartet den Probeband: %s", lokal, rumpf)
		}
	})

	// Ein Ausfall der DNB nimmt der Suche nicht die Treffer aus dem eigenen Katalog, und die
	// Antwort nennt ihn: Ein Buch, das die DNB kennt, sähe sonst aus wie eines, das sie nicht
	// kennt.
	t.Run("DNB nicht erreichbar: die Treffer aus dem Katalog kommen trotzdem, die Antwort nennt den Ausfall", func(t *testing.T) {
		code, treffer, rumpf := suche(t, dnbAttrappe{status: http.StatusServiceUnavailable}, `{"query":"Probeband"}`)
		if code != http.StatusOK {
			t.Fatalf("Status %d, erwartet 200: %s", code, rumpf)
		}
		if lokal := aus(treffer, "local"); len(lokal) != 1 {
			t.Errorf("%d Treffer aus dem Katalog, erwartet 1: %s", len(lokal), rumpf)
		}
		if dnb := aus(treffer, "dnb"); len(dnb) != 0 {
			t.Errorf("%d Treffer der DNB bei einem Ausfall", len(dnb))
		}
		if ausfall != "1" {
			t.Errorf("%s = %q, erwartet 1", dnbAusfallKopf, ausfall)
		}
	})

	// Ohne Treffer im Katalog bliebe von einem Ausfall sonst nur eine leere Liste.
	t.Run("DNB nicht erreichbar, kein Treffer im Katalog: leere Liste mit dem Ausfall im Kopf", func(t *testing.T) {
		code, treffer, rumpf := suche(t, dnbAttrappe{status: http.StatusBadGateway}, `{"query":"gibtesnichtimkatalog"}`)
		if code != http.StatusOK || len(treffer) != 0 {
			t.Fatalf("Status %d mit %d Treffern, erwartet 200 und keinen: %s", code, len(treffer), rumpf)
		}
		if ausfall != "1" {
			t.Errorf("%s = %q, erwartet 1", dnbAusfallKopf, ausfall)
		}
	})

	t.Run("die DNB antwortet ohne Treffer: kein Ausfall", func(t *testing.T) {
		leer := `<searchRetrieveResponse xmlns="http://www.loc.gov/zing/srw/"><numberOfRecords>0</numberOfRecords></searchRetrieveResponse>`
		code, treffer, rumpf := suche(t, dnbAttrappe{satz: leer}, `{"query":"gibtesnichtimkatalog"}`)
		if code != http.StatusOK || len(treffer) != 0 {
			t.Fatalf("Status %d mit %d Treffern, erwartet 200 und keinen: %s", code, len(treffer), rumpf)
		}
		if ausfall != "" {
			t.Errorf("%s = %q bei einer Antwort ohne Treffer", dnbAusfallKopf, ausfall)
		}
	})

	t.Run("leere Suche: leere Liste, die DNB wird nicht gefragt", func(t *testing.T) {
		var anfragen atomic.Int32
		code, _, rumpf := suche(t, zaehlendeDNB{dnbAttrappe{satz: dnbAntwort}, &anfragen}, `{"query":"   "}`)
		if code != http.StatusOK || strings.TrimSpace(rumpf) != "[]" {
			t.Errorf("Status %d, Antwort %s — erwartet 200 und []", code, rumpf)
		}
		if n := anfragen.Load(); n != 0 {
			t.Errorf("%d Anfragen an die DNB für eine leere Suche", n)
		}
		// Gegenprobe: Eine Suche mit Text fragt sie.
		suche(t, zaehlendeDNB{dnbAttrappe{satz: dnbAntwort}, &anfragen}, `{"query":"Probeband"}`)
		if n := anfragen.Load(); n == 0 {
			t.Error("Gegenprobe: Auch die Suche mit Text hat die DNB nicht gefragt — die Zählung misst nichts")
		}
	})

	t.Run("kein JSON: 400", func(t *testing.T) {
		if code, _, rumpf := suche(t, dnbAttrappe{satz: dnbAntwort}, `{"query":`); code != http.StatusBadRequest {
			t.Errorf("Status %d, erwartet 400: %s", code, rumpf)
		}
	})
}

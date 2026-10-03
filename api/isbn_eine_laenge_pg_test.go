package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"bibliothek/db"
	"bibliothek/inventur"
)

// zaehlendeDNB zählt, wie oft die Tür die DNB fragt, und antwortet wie dnbAttrappe.
type zaehlendeDNB struct {
	dnbAttrappe
	anfragen *atomic.Int32
}

func (z zaehlendeDNB) RoundTrip(r *http.Request) (*http.Response, error) {
	if r.URL.Host == "services.dnb.de" {
		z.anfragen.Add(1)
	}
	return z.dnbAttrappe.RoundTrip(r)
}

// Die zehn- und die dreizehnstellige Form einer ISBN sind dieselbe Nummer (Migration 157):
// Steht das Buch im Katalog, findet die Bestelltür es über die zehnstellige vom
// Titelblatt wie über die dreizehnstellige vom Strichcode, legt nichts an und fragt die DNB
// nicht. Ein neuer Titel entsteht dreizehnstellig. Die Nummern sind die Beispielpaare der
// ISBN-Norm (3-16-148410-X ↔ 978-3-16-148410-0, 0-306-40615-2 ↔ 978-0-306-40615-7).
//
// Eine zehnstellige Nummer mit falschem Prüfzeichen ist eine andere Nummer: Am Testserver
// steht unter 3499500252 ein anderes Buch als unter 9783499500251.
func TestISBNZuTitel_BeideLaengenSindDieselbeISBN(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	ctx := context.Background()
	lege := func(titel, isbn string) (id string) {
		t.Helper()
		if err := pool.QueryRow(ctx, `INSERT INTO buecher_titel (titel, isbn, verlag)
			VALUES ($1, $2, 'Klett') RETURNING id::text`, titel, isbn).Scan(&id); err != nil {
			t.Fatal(err)
		}
		return id
	}
	chemie := lege("Elemente Chemie 1", "316148410X")
	roman := lege("Ein Roman", "9783499500251")

	var anfragen atomic.Int32
	client := inventur.NeuerMetadatenClient()
	client.SetzeHTTPClientFuerTest(&http.Client{Transport: zaehlendeDNB{anfragen: &anfragen, dnbAttrappe: dnbAttrappe{satz: `
		<searchRetrieveResponse xmlns="http://www.loc.gov/zing/srw/"><records><record><recordData>
		  <record xmlns="http://www.loc.gov/MARC21/slim">
			<datafield tag="245" ind1="1" ind2="0"><subfield code="a">Aus der DNB</subfield></datafield>
		  </record>
		</recordData></record></records></searchRetrieveResponse>`}}})
	tuer := (&Server{DB: &db.Database{Pool: pool}}).isbnZuTitel(client)

	frage := func(isbn string) ISBNLookupResponse {
		t.Helper()
		rec := httptest.NewRecorder()
		tuer(rec, httptest.NewRequest(http.MethodPost, "/api/buecher/aus-isbn", strings.NewReader(`{"isbn":"`+isbn+`"}`)))
		if rec.Code != http.StatusOK {
			t.Fatalf("aus-isbn %s: %d %s", isbn, rec.Code, rec.Body.String())
		}
		if strings.Contains(rec.Body.String(), "andere_form") {
			t.Errorf("aus-isbn %s: die Antwort schlägt einen Titel unter der anderen Form vor: %s", isbn, rec.Body.String())
		}
		var antwort ISBNLookupResponse
		if err := json.Unmarshal(rec.Body.Bytes(), &antwort); err != nil {
			t.Fatal(err)
		}
		return antwort
	}
	stand := func() (titel int, dnb int32) {
		t.Helper()
		if err := pool.QueryRow(ctx, `SELECT count(*) FROM buecher_titel`).Scan(&titel); err != nil {
			t.Fatal(err)
		}
		return titel, anfragen.Load()
	}

	// Im Katalog, in jeder Schreibweise und Länge: der vorhandene Titel mit der ISBN, wie
	// der Katalog sie trägt.
	for _, isbn := range []string{"978-3-16-148410-0", "9783161484100", "3-16-148410-x", "316148410X"} {
		a := frage(isbn)
		if !a.Exists || a.TitelID != chemie || a.ISBN != "9783161484100" {
			t.Errorf("%s: %+v — erwartet den Titel %s unter 9783161484100", isbn, a, chemie)
		}
	}
	if titel, dnb := stand(); titel != 2 || dnb != 0 {
		t.Errorf("nach vier Fragen zu einem vorhandenen Titel: %d Titel, %d DNB-Anfragen — erwartet 2 und 0", titel, dnb)
	}

	// Nicht im Katalog, zehnstellig eingegeben: Der Titel aus der DNB entsteht dreizehnstellig,
	// und die dreizehnstellige findet ihn danach.
	neu := frage("0-306-40615-2")
	if neu.Exists || neu.TitelID == "" || neu.Titel != "Aus der DNB" || neu.ISBN != "9780306406157" {
		t.Fatalf("neuer Titel: %+v — erwartet „Aus der DNB“ unter 9780306406157", neu)
	}
	if wieder := frage("9780306406157"); !wieder.Exists || wieder.TitelID != neu.TitelID {
		t.Errorf("die dreizehnstellige danach: %+v — erwartet den Titel %s", wieder, neu.TitelID)
	}
	if titel, dnb := stand(); titel != 3 || dnb != 1 {
		t.Errorf("nach dem Anlegen: %d Titel, %d DNB-Anfragen — erwartet 3 und 1", titel, dnb)
	}

	// Falsches Prüfzeichen: gerechnet führte 3499500252 auf 9783499500251. Die Tür nimmt den
	// Roman nicht dafür, sie legt die Nummer als eigenen Titel an.
	falsch := frage("3499500252")
	if falsch.Exists || falsch.TitelID == "" || falsch.TitelID == roman || falsch.ISBN != "3499500252" {
		t.Errorf("falsches Prüfzeichen: %+v — erwartet einen neuen Titel unter 3499500252, nicht %s", falsch, roman)
	}
	if a := frage("9783499500251"); !a.Exists || a.TitelID != roman {
		t.Errorf("die dreizehnstellige daneben: %+v — erwartet den Titel %s", a, roman)
	}
}

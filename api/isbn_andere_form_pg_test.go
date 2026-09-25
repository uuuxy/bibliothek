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

// Steht die ISBN in der anderen Länge im Katalog (ISBN-10 ↔ ISBN-13), schlägt die Tür diesen
// Titel vor und legt nichts an (docs/OFFEN.md 4.18 Stufe 4, 5.5). Erst „Neu anlegen"
// (neu_anlegen) holt den Titel aus der DNB. Die Nummern sind das zweite Beispielpaar der
// ISBN-Norm, mit Prüfzeichen X (3-16-148410-X ↔ 978-3-16-148410-0).
func TestISBNZuTitel_SchlaegtDieAndereFormVor(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	ctx := context.Background()
	var zehn string
	if err := pool.QueryRow(ctx, `INSERT INTO buecher_titel (titel, isbn, verlag)
		VALUES ('Elemente Chemie 1', '316148410X', 'Klett') RETURNING id::text`).Scan(&zehn); err != nil {
		t.Fatal(err)
	}

	var anfragen atomic.Int32
	client := inventur.NeuerMetadatenClient()
	client.SetzeHTTPClientFuerTest(&http.Client{Transport: zaehlendeDNB{anfragen: &anfragen, dnbAttrappe: dnbAttrappe{satz: `
		<searchRetrieveResponse xmlns="http://www.loc.gov/zing/srw/"><records><record><recordData>
		  <record xmlns="http://www.loc.gov/MARC21/slim">
			<datafield tag="245" ind1="1" ind2="0"><subfield code="a">Aus der DNB</subfield></datafield>
		  </record>
		</recordData></record></records></searchRetrieveResponse>`}}})
	tuer := (&Server{DB: &db.Database{Pool: pool}}).isbnZuTitel(client)

	frage := func(rumpf string) ISBNLookupResponse {
		t.Helper()
		rec := httptest.NewRecorder()
		tuer(rec, httptest.NewRequest(http.MethodPost, "/api/buecher/aus-isbn", strings.NewReader(rumpf)))
		if rec.Code != http.StatusOK {
			t.Fatalf("aus-isbn %s: %d %s", rumpf, rec.Code, rec.Body.String())
		}
		var antwort ISBNLookupResponse
		if err := json.Unmarshal(rec.Body.Bytes(), &antwort); err != nil {
			t.Fatal(err)
		}
		return antwort
	}
	titel := func() (n int) {
		t.Helper()
		if err := pool.QueryRow(ctx, `SELECT count(*) FROM buecher_titel`).Scan(&n); err != nil {
			t.Fatal(err)
		}
		return n
	}

	// Gescannt wird die EAN-13: Vorschlag, nichts angelegt, die DNB nicht gefragt.
	vorschlag := frage(`{"isbn":"978-3-16-148410-0"}`)
	if vorschlag.TitelID != "" || vorschlag.AndereForm == nil || vorschlag.AndereForm.TitelID != zehn ||
		vorschlag.AndereForm.ISBN != "316148410X" || !vorschlag.AndereForm.Exists {
		t.Fatalf("Vorschlag: %+v (andere_form %+v) — erwartet keine titel_id und den Titel %s unter 316148410X", vorschlag, vorschlag.AndereForm, zehn)
	}
	if n := titel(); n != 1 || anfragen.Load() != 0 {
		t.Errorf("nach dem Vorschlag: %d Titel, %d DNB-Anfragen — erwartet 1 und 0", n, anfragen.Load())
	}

	// „Neu anlegen": der Titel aus der DNB, ein zweiter neben dem vorgeschlagenen.
	neu := frage(`{"isbn":"9783161484100","neu_anlegen":true}`)
	if neu.Exists || neu.TitelID == "" || neu.TitelID == zehn || neu.Titel != "Aus der DNB" || neu.AndereForm != nil {
		t.Fatalf("Neu anlegen: %+v — erwartet einen neuen Titel „Aus der DNB“", neu)
	}
	if n := titel(); n != 2 {
		t.Errorf("nach „Neu anlegen“: %d Titel — erwartet 2", n)
	}

	// Steht die Schreibweise selbst im Katalog, gewinnt sie — in beiden Richtungen.
	if wieder := frage(`{"isbn":"9783161484100"}`); !wieder.Exists || wieder.TitelID != neu.TitelID || wieder.AndereForm != nil {
		t.Errorf("EAN-13 im Katalog: %+v — erwartet den Titel %s ohne Vorschlag", wieder, neu.TitelID)
	}
	if alt := frage(`{"isbn":"3-16-148410-x"}`); !alt.Exists || alt.TitelID != zehn || alt.AndereForm != nil {
		t.Errorf("ISBN-10 im Katalog: %+v — erwartet den Titel %s ohne Vorschlag", alt, zehn)
	}
}

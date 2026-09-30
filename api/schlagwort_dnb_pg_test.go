package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"

	"bibliothek/db"
	"bibliothek/inventur"
	"bibliothek/repository"
)

// GET /api/schlagworte/dnb-vorschlag (entschieden am 30.09.2026): der Vorschlag aus der DNB
// für das Buchformular und das Nachbestellen, nach derselben Regel wie beim Anlegen über
// aus-isbn — Wörter der Liste auch über Gattung und Verlagswort, Werbewörter nur, wenn die Liste
// sie kennt, Normdatei-Wörter außerhalb der Liste getrennt als neu. Geschrieben wird nichts.
func TestDnbSchlagwortVorschlag_VorschlagOhneEintrag(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	ctx := context.Background()
	if _, err := pool.Exec(ctx, `TRUNCATE schlagworte CASCADE`); err != nil {
		t.Fatal(err)
	}
	vorhanden := titelMitSignatur(t, pool, "Die Welle", "", 0)
	if _, err := repository.SetzeSchlagworte(ctx, pool, vorhanden, []string{"Krieg", "Fantasy"}); err != nil {
		t.Fatal(err)
	}
	zaehle := func() (woerter, zuordnungen int) {
		t.Helper()
		if err := pool.QueryRow(ctx, `SELECT (SELECT count(*) FROM schlagworte), (SELECT count(*) FROM titel_schlagworte)`).
			Scan(&woerter, &zuordnungen); err != nil {
			t.Fatal(err)
		}
		return woerter, zuordnungen
	}
	vorherW, vorherZ := zaehle()

	const satz = `<searchRetrieveResponse xmlns="http://www.loc.gov/zing/srw/"><records><record><recordData>
		<record xmlns="http://www.loc.gov/MARC21/slim">
		  <datafield tag="245" ind1="1" ind2="0"><subfield code="a">Dunkelnacht</subfield></datafield>
		  <datafield tag="655" ind1=" " ind2="7"><subfield code="a">fantasy</subfield><subfield code="2">gatbeg</subfield></datafield>
		  <datafield tag="653" ind1=" " ind2=" "><subfield code="a">Krieg</subfield></datafield>
		  <datafield tag="653" ind1=" " ind2=" "><subfield code="a">TikTok</subfield></datafield>
		  <datafield tag="650" ind1=" " ind2="7"><subfield code="a">Schulstress</subfield><subfield code="2">gnd</subfield></datafield>
		</record>
	</recordData></record></records></searchRetrieveResponse>`
	const leer = `<searchRetrieveResponse xmlns="http://www.loc.gov/zing/srw/"><numberOfRecords>0</numberOfRecords></searchRetrieveResponse>`

	faelle := []struct {
		name   string
		isbn   string
		dnb    dnbAttrappe
		status int
		rumpf  string // wörtlich in der Antwort
	}{
		{"Satz mit Wörtern", "978-3-7512-0053-0", dnbAttrappe{satz: satz}, http.StatusOK,
			`{"dnb_satz":true,"schlagwort_vorschlaege":["Fantasy","Krieg"],"schlagwort_vorschlaege_neu":["Schulstress"]}`},
		{"DNB kennt die ISBN nicht", "9783751200530", dnbAttrappe{satz: leer}, http.StatusOK,
			`{"dnb_satz":false,"schlagwort_vorschlaege":[],"schlagwort_vorschlaege_neu":[]}`},
		{"DNB nicht erreichbar", "9783751200530", dnbAttrappe{status: http.StatusServiceUnavailable}, http.StatusBadGateway,
			`DNB nicht erreichbar`},
		{"keine ISBN", "Dunkelnacht", dnbAttrappe{satz: satz}, http.StatusBadRequest,
			`keine gültige ISBN`},
	}
	for _, f := range faelle {
		t.Run(f.name, func(t *testing.T) {
			client := inventur.NeuerMetadatenClient()
			client.SetzeHTTPClientFuerTest(&http.Client{Transport: f.dnb})
			tuer := (&Server{DB: &db.Database{Pool: pool}}).dnbSchlagwortVorschlag(client)

			rec := httptest.NewRecorder()
			tuer(rec, httptest.NewRequest(http.MethodGet, "/api/schlagworte/dnb-vorschlag?isbn="+f.isbn, nil))

			if rec.Code != f.status {
				t.Fatalf("Status %d, erwartet %d: %s", rec.Code, f.status, rec.Body.String())
			}
			if !strings.Contains(rec.Body.String(), f.rumpf) {
				t.Errorf("Antwort %s, erwartet darin %s", rec.Body.String(), f.rumpf)
			}
			if w, z := zaehle(); w != vorherW || z != vorherZ {
				t.Errorf("Liste %d → %d Wörter, %d → %d Zuordnungen — der Vorschlag darf nichts schreiben", vorherW, w, vorherZ, z)
			}
		})
	}

	// Dieselbe Regel wie beim Anlegen: Die Tür des Bestellens schlägt zum selben Satz dieselben
	// Wörter vor. Hier über titelAusNachschlagen, das einen neuen Titel anlegt — deshalb zuletzt.
	var antwort DnbSchlagwortVorschlag
	client := inventur.NeuerMetadatenClient()
	client.SetzeHTTPClientFuerTest(&http.Client{Transport: dnbAttrappe{satz: satz}})
	rec := httptest.NewRecorder()
	(&Server{DB: &db.Database{Pool: pool}}).dnbSchlagwortVorschlag(client)(rec,
		httptest.NewRequest(http.MethodGet, "/api/schlagworte/dnb-vorschlag?isbn=9783751200530", nil))
	if err := json.Unmarshal(rec.Body.Bytes(), &antwort); err != nil {
		t.Fatal(err)
	}
	meta, err := client.SucheDNBNachISBN(ctx, "9783751200530")
	if err != nil {
		t.Fatal(err)
	}
	angelegt, err := (&Server{DB: &db.Database{Pool: pool}}).titelAusNachschlagen(ctx, "9783751200530", meta)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(angelegt.SchlagwortVorschlaege, antwort.SchlagwortVorschlaege) ||
		!slices.Equal(angelegt.SchlagwortVorschlaegeNeu, antwort.SchlagwortVorschlaegeNeu) {
		t.Errorf("Bestellen schlägt %q / %q vor, der Vorschlag hier %q / %q — zwei Regeln",
			angelegt.SchlagwortVorschlaege, angelegt.SchlagwortVorschlaegeNeu,
			antwort.SchlagwortVorschlaege, antwort.SchlagwortVorschlaegeNeu)
	}
}

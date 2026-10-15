package katalog

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"

	"bibliothek/api/tuer"
	"bibliothek/db"
	"bibliothek/internal/pgtest"
	"bibliothek/internal/testdaten"
	"bibliothek/internal/tuertest"
	"bibliothek/inventur"
	"bibliothek/repository"
)

// GET /api/schlagworte/dnb-vorschlag: der Vorschlag aus der DNB
// für das Buchformular und das Nachbestellen, nach derselben Regel wie beim Anlegen über
// aus-isbn — Wörter der Liste auch über Gattung und Verlagswort, Werbewörter nur, wenn die Liste
// sie kennt, Normdatei-Wörter außerhalb der Liste getrennt als neu. Geschrieben wird nichts.
func TestDNBSchlagwortVorschlag_VorschlagOhneEintrag(t *testing.T) {
	pool := pgtest.Pool(t)
	testdaten.LeereBestandsdaten(t, pool)
	ctx := context.Background()
	if _, err := pool.Exec(ctx, `TRUNCATE schlagworte CASCADE`); err != nil {
		t.Fatal(err)
	}
	vorhanden := testdaten.LegeTitelMitSignaturAn(t, pool, "Die Welle", "", 0)
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
		dnb    tuertest.DNBAttrappe
		status int
		rumpf  string // wörtlich in der Antwort
	}{
		{"Satz mit Wörtern", "978-3-7512-0053-0", tuertest.DNBAttrappe{Satz: satz}, http.StatusOK,
			`{"dnb_satz":true,"schlagwort_vorschlaege":["Fantasy","Krieg"],"schlagwort_vorschlaege_neu":["Schulstress"]}`},
		{"DNB kennt die ISBN nicht", "9783751200530", tuertest.DNBAttrappe{Satz: leer}, http.StatusOK,
			`{"dnb_satz":false,"schlagwort_vorschlaege":[],"schlagwort_vorschlaege_neu":[]}`},
		{"DNB nicht erreichbar", "9783751200530", tuertest.DNBAttrappe{Status: http.StatusServiceUnavailable}, http.StatusBadGateway,
			`DNB nicht erreichbar`},
		{"keine ISBN", "Dunkelnacht", tuertest.DNBAttrappe{Satz: satz}, http.StatusBadRequest,
			`Keine gültige ISBN`},
	}
	for _, f := range faelle {
		t.Run(f.name, func(t *testing.T) {
			client := inventur.NewMetadatenClient()
			client.SetzeHTTPClientFuerTest(&http.Client{Transport: f.dnb})
			tuer := (&Tueren{Basis: &tuer.Basis{DB: &db.Verbindung{Pool: pool}}}).dnbSchlagwortVorschlagTuer(client)

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
	// Wörter vor. Hier über legeTitelAusTrefferAn, das einen neuen Titel anlegt — deshalb zuletzt.
	var antwort dnbSchlagwortVorschlagAntwort
	client := inventur.NewMetadatenClient()
	client.SetzeHTTPClientFuerTest(&http.Client{Transport: tuertest.DNBAttrappe{Satz: satz}})
	rec := httptest.NewRecorder()
	(&Tueren{Basis: &tuer.Basis{DB: &db.Verbindung{Pool: pool}}}).dnbSchlagwortVorschlagTuer(client)(rec,
		httptest.NewRequest(http.MethodGet, "/api/schlagworte/dnb-vorschlag?isbn=9783751200530", nil))
	if err := json.Unmarshal(rec.Body.Bytes(), &antwort); err != nil {
		t.Fatal(err)
	}
	meta, err := client.SucheDNBNachISBN(ctx, "9783751200530")
	if err != nil {
		t.Fatal(err)
	}
	angelegt, err := (&Tueren{Basis: &tuer.Basis{DB: &db.Verbindung{Pool: pool}}}).legeTitelAusTrefferAn(ctx, "9783751200530", meta)
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

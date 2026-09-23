package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"

	"bibliothek/db"
	"bibliothek/inventur"
	"bibliothek/repository"
)

// Bestellen per ISBN legt einen neuen Titel aus der DNB an und schlägt dabei Schlagworte vor
// (docs/OFFEN.md 4.20): die Wörter der eigenen Liste, die der Satz nennt. Ein Vorschlag, kein
// Eintrag — am neuen Titel darf danach KEIN Schlagwort stehen, bis im Bestellkorb jemand
// eines übernimmt.
func TestTitelAusNachschlagen_SchlaegtSchlagworteVorOhneSieZuSetzen(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	ctx := context.Background()
	if _, err := pool.Exec(ctx, `TRUNCATE schlagworte CASCADE`); err != nil {
		t.Fatal(err)
	}
	vorhanden := titelMitSignatur(t, pool, "Die Welle", "", 0)
	if _, err := repository.SetzeSchlagworte(ctx, pool, vorhanden, []string{"Krieg", "Erste Liebe"}); err != nil {
		t.Fatal(err)
	}

	srv := &Server{DB: &db.Database{Pool: pool}}
	resp, err := srv.titelAusNachschlagen(ctx, "9783751200530", &inventur.MetadatenErgebnis{
		Titel: "Dunkelnacht", Autor: "Boie, Kirsten",
		Stichwoerter: []string{"Jugendbuch", "Krieg", "Erste Liebe", "TikTok"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"Erste Liebe", "Krieg"}; !slices.Equal(resp.SchlagwortVorschlaege, want) {
		t.Errorf("Vorschlag %q, erwartet %q", resp.SchlagwortVorschlaege, want)
	}
	if resp.Exists || resp.TitelID == "" {
		t.Errorf("neuer Titel erwartet: %+v", resp)
	}
	var amTitel int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM titel_schlagworte WHERE titel_id = $1`, resp.TitelID).Scan(&amTitel); err != nil {
		t.Fatal(err)
	}
	if amTitel != 0 {
		t.Errorf("am neuen Titel stehen %d Schlagworte — der Vorschlag darf nichts eintragen", amTitel)
	}
	daten, err := json.Marshal(resp)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(daten), `"schlagwort_vorschlaege":["Erste Liebe","Krieg"]`) {
		t.Errorf("Antwort trägt den Vorschlag nicht unter schlagwort_vorschlaege: %s", daten)
	}
}

// dnbAttrappe beantwortet die Anfrage an die DNB mit einem festen Satz; alles andere
// (Cover-Quellen) gibt es nicht.
type dnbAttrappe struct{ satz string }

func (d dnbAttrappe) RoundTrip(r *http.Request) (*http.Response, error) {
	if r.URL.Host == "services.dnb.de" {
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(d.satz)), Header: http.Header{}, Request: r}, nil
	}
	return &http.Response{StatusCode: http.StatusNotFound, Body: http.NoBody, Header: http.Header{}, Request: r}, nil
}

// Über die Tür: POST /api/buecher/aus-isbn gibt den Vorschlag weiter — beim ersten Mal, wenn
// der Titel aus der DNB entsteht. Beim zweiten Mal steht der Titel schon da; dann fragt die
// Tür die DNB nicht, und einen Vorschlag gibt es nicht.
func TestISBNZuTitel_TuerGibtDenVorschlagWeiter(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	ctx := context.Background()
	if _, err := pool.Exec(ctx, `TRUNCATE schlagworte CASCADE`); err != nil {
		t.Fatal(err)
	}
	vorhanden := titelMitSignatur(t, pool, "Die Welle", "", 0)
	if _, err := repository.SetzeSchlagworte(ctx, pool, vorhanden, []string{"Krieg"}); err != nil {
		t.Fatal(err)
	}

	client := inventur.NeuerMetadatenClient()
	client.SetzeHTTPClientFuerTest(&http.Client{Transport: dnbAttrappe{satz: `
		<searchRetrieveResponse xmlns="http://www.loc.gov/zing/srw/"><records><record><recordData>
		  <record xmlns="http://www.loc.gov/MARC21/slim">
			<datafield tag="245" ind1="1" ind2="0"><subfield code="a">Dunkelnacht</subfield></datafield>
			<datafield tag="100" ind1="1" ind2=" "><subfield code="a">Boie, Kirsten</subfield></datafield>
			<datafield tag="653" ind1=" " ind2=" "><subfield code="a">Krieg</subfield></datafield>
			<datafield tag="653" ind1=" " ind2=" "><subfield code="a">TikTok</subfield></datafield>
		  </record>
		</recordData></record></records></searchRetrieveResponse>`}})
	tuer := (&Server{DB: &db.Database{Pool: pool}}).isbnZuTitel(client)

	frage := func() map[string]any {
		t.Helper()
		rec := httptest.NewRecorder()
		tuer(rec, httptest.NewRequest(http.MethodPost, "/api/buecher/aus-isbn", strings.NewReader(`{"isbn":"978-3-7512-0053-0"}`)))
		if rec.Code != http.StatusOK {
			t.Fatalf("aus-isbn: %d %s", rec.Code, rec.Body.String())
		}
		var antwort map[string]any
		if err := json.Unmarshal(rec.Body.Bytes(), &antwort); err != nil {
			t.Fatal(err)
		}
		return antwort
	}

	erste := frage()
	if erste["exists"] != false || fmt.Sprint(erste["schlagwort_vorschlaege"]) != "[Krieg]" {
		t.Errorf("neu aus der DNB: exists=%v schlagwort_vorschlaege=%v, erwartet false und [Krieg]", erste["exists"], erste["schlagwort_vorschlaege"])
	}
	zweite := frage()
	if zweite["exists"] != true || zweite["schlagwort_vorschlaege"] != nil {
		t.Errorf("schon im Katalog: exists=%v schlagwort_vorschlaege=%v, erwartet true und keinen Vorschlag", zweite["exists"], zweite["schlagwort_vorschlaege"])
	}
}

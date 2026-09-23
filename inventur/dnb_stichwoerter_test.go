package inventur

import (
	"context"
	"io"
	"net/http"
	"slices"
	"strings"
	"testing"
)

// Die Kandidaten des Schlagwort-Vorschlags (docs/OFFEN.md 4.20): Gattungsbegriffe aus 655 $a
// und die freien Verlagswörter aus 653 $a — aber nicht die 653-Einträge mit Vorsatz, die
// Angaben anderer Art sind. Der Satz ist einem echten DNB-Satz nachgebaut (Kirsten Boie,
// „Dunkelnacht", abgerufen am 23.09.2026), gekürzt.
func TestSucheDNB_StichwoerterOhneVorsatzEintraege(t *testing.T) {
	const satz = `
		<searchRetrieveResponse xmlns="http://www.loc.gov/zing/srw/">
		  <records><record><recordData>
			<record xmlns="http://www.loc.gov/MARC21/slim">
			  <datafield tag="245" ind1="1" ind2="0"><subfield code="a">Dunkelnacht</subfield></datafield>
			  <datafield tag="100" ind1="1" ind2=" "><subfield code="a">Boie, Kirsten</subfield></datafield>
			  <datafield tag="653" ind1=" " ind2=" "><subfield code="a">(Produktform)Hardback</subfield></datafield>
			  <datafield tag="653" ind1=" " ind2=" "><subfield code="a">(Zielgruppe)Für Jugendliche und Erwachsene,ab 15 Jahren</subfield></datafield>
			  <datafield tag="653" ind1=" " ind2=" "><subfield code="a">(Lesealter)ab 15 Jahre</subfield></datafield>
			  <datafield tag="653" ind1=" " ind2=" "><subfield code="a">Burgen</subfield></datafield>
			  <datafield tag="653" ind1=" " ind2=" "><subfield code="a">Erste Liebe</subfield></datafield>
			  <datafield tag="653" ind1=" " ind2=" "><subfield code="a">(VLB-WN)1113: Hardcover, Softcover / Belletristik/Historische Romane, Erzählungen</subfield></datafield>
			  <datafield tag="653" ind1=" " ind2=" "><subfield code="a">(BISAC Subject Heading)YAF024070</subfield></datafield>
			  <datafield tag="653" ind1=" " ind2=" "><subfield code="a">Krieg</subfield></datafield>
			  <datafield tag="655" ind1=" " ind2="7"><subfield code="0">(DE-588)4306252-0</subfield><subfield code="a">Jugendbuch</subfield><subfield code="2">gnd-content</subfield></datafield>
			  <datafield tag="655" ind1=" " ind2="7"><subfield code="0">(DE-101)101071841X</subfield><subfield code="a">Jugendbücher ab 12 Jahre</subfield><subfield code="2">gatbeg</subfield></datafield>
			</record>
		  </recordData></record></records>
		</searchRetrieveResponse>`
	client := &MetadatenClient{httpClient: &http.Client{Transport: &mockTransport{
		roundTripFunc: func(*http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(satz))}, nil
		},
	}}}

	res, err := client.sucheDNB(context.Background(), "9783751200530")
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"Jugendbuch", "Jugendbücher ab 12 Jahre", "Burgen", "Erste Liebe", "Krieg"}
	if !slices.Equal(res.Stichwoerter, want) {
		t.Errorf("Stichwoerter = %q, erwartet %q — Einträge mit Vorsatz sind keine Verlagswörter", res.Stichwoerter, want)
	}
	if res.Zielgruppe != "Für Jugendliche und Erwachsene,ab 15 Jahren" {
		t.Errorf("Zielgruppe = %q — der Vorsatz-Eintrag wird weiter gelesen", res.Zielgruppe)
	}
}

// Die Freitextsuche der Bestellung schickt „jedes Wort irgendwo im Satz" (any all "…").
// Bis zum 23.09.2026 ging any=Dunkelnacht+Boie hinaus, und die DNB lieferte mit zwei Wörtern
// keinen Satz. Anführungszeichen und Backslash der Eingabe sind maskiert, sonst endete die
// Zeichenkette mitten in der Eingabe; „and" ist ein Wort, kein Operator.
func TestCqlAlleWoerter(t *testing.T) {
	faelle := map[string]string{
		"Dunkelnacht":      `any all "Dunkelnacht"`,
		"Dunkelnacht Boie": `any all "Dunkelnacht Boie"`,
		`Der "Hobbit"`:     `any all "Der \"Hobbit\""`,
		`a\b`:              `any all "a\\b"`,
		"Harry and Potter": `any all "Harry and Potter"`,
	}
	for eingabe, erwartet := range faelle {
		if got := cqlAlleWoerter(eingabe); got != erwartet {
			t.Errorf("cqlAlleWoerter(%q) = %s, erwartet %s", eingabe, got, erwartet)
		}
	}
}

// Was wirklich an die DNB geht: die Abfrage als Parameter query, nach dem Dekodieren genau
// any all "…".
func TestSucheTextDNB_SchicktAlleWoerter(t *testing.T) {
	var gesendet string
	client := &MetadatenClient{httpClient: &http.Client{Transport: &mockTransport{
		roundTripFunc: func(r *http.Request) (*http.Response, error) {
			gesendet = r.URL.Query().Get("query")
			return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`<searchRetrieveResponse/>`))}, nil
		},
	}}}
	if _, err := client.SucheTextDNB(context.Background(), "  Dunkelnacht Boie "); err != nil {
		t.Fatal(err)
	}
	if gesendet != `any all "Dunkelnacht Boie"` {
		t.Errorf("an die DNB ging query=%s, erwartet any all \"Dunkelnacht Boie\"", gesendet)
	}
}

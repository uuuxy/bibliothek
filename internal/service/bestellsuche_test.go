package service

import (
	"bibliothek/internal/katalogdienste"
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"testing"

	"github.com/pashagolub/pgxmock/v5"
)

type dnbTransportAttrappe struct {
	antwort func(req *http.Request) (*http.Response, error)
}

func (t *dnbTransportAttrappe) RoundTrip(req *http.Request) (*http.Response, error) {
	return t.antwort(req)
}

func TestFindeZumBestellen_KatalogUndDNBZusammen(t *testing.T) {
	mock, err := pgxmock.NewPool(pgxmock.QueryMatcherOption(pgxmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("pgxmock: %v", err)
	}
	defer mock.Close()

	ctx := context.Background()

	localQuery := `WITH matched_titels AS`
	mock.ExpectQuery(localQuery).
		WithArgs("TestBook").
		WillReturnRows(pgxmock.NewRows([]string{"id", "titel", "autor", "isbn", "verlag", "cover_url", "signatur", "ist_lernmittel", "current_stock"}).
			AddRow("id-1", "Lokales Buch", "Autor A", "9781234567890", "Verlag X", "cover-lokal", "SIG-1", true, 3))

	mockTransport := &dnbTransportAttrappe{
		antwort: func(req *http.Request) (*http.Response, error) {
			if req.URL.Host == "services.dnb.de" {
				xml := `<searchRetrieveResponse xmlns="http://www.loc.gov/zing/srw/">
  <numberOfRecords>1</numberOfRecords>
  <records>
    <record>
      <recordData>
        <record xmlns="http://www.loc.gov/MARC21/slim">
          <datafield tag="020" ind1=" " ind2=" ">
            <subfield code="a">9780987654321</subfield>
            <subfield code="c">19,90 EUR</subfield>
          </datafield>
          <datafield tag="245" ind1="1" ind2="0">
            <subfield code="a">DNB Buch</subfield>
          </datafield>
          <datafield tag="100" ind1="1" ind2=" ">
            <subfield code="a">Autor B</subfield>
          </datafield>
          <datafield tag="264" ind1=" " ind2="1">
            <subfield code="b">Verlag Y</subfield>
          </datafield>
        </record>
      </recordData>
    </record>
  </records>
</searchRetrieveResponse>`
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewBufferString(xml)),
					Header:     make(http.Header),
				}, nil
			}
			return nil, errors.New("unexpected host")
		},
	}

	metaClient := katalogdienste.NewMetadatenClient()
	metaClient.SetzeHTTPClientFuerTest(&http.Client{Transport: mockTransport})

	mock.ExpectQuery(`SELECT isbn_normalform\(isbn\) FROM buecher_titel`).
		WithArgs(pgxmock.AnyArg()).
		WillReturnRows(pgxmock.NewRows([]string{"isbn_normalform"}).
			AddRow("9780987654321"))

	results, dnbAusfall, err := FindeZumBestellen(ctx, mock, metaClient, "TestBook")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if dnbAusfall {
		t.Error("die DNB hat geantwortet, die Suche meldet einen Ausfall")
	}

	if len(results) != 2 {
		t.Fatalf("FindeZumBestellen(…) = %d Treffer, erwartet 2", len(results))
	}

	// Jedes Feld des eigenen Treffers kommt aus seiner Spalte; jede trägt einen eigenen Wert.
	lokal := BestellTreffer{
		ID: "id-1", Titel: "Lokales Buch", Autor: "Autor A", ISBN: "9781234567890", Verlag: "Verlag X",
		CoverURL: "cover-lokal", Signatur: "SIG-1", Quelle: "local", Bestand: 3, IstLernmittel: true,
	}
	if results[0] != lokal {
		t.Errorf("eigener Treffer:\n  ist  %+v\n  soll %+v", results[0], lokal)
	}

	if results[1].Quelle != "dnb" || results[1].Titel != "DNB Buch" {
		t.Errorf("unexpected DNB result: %+v", results[1])
	}

	if !results[1].IstDublette {
		t.Errorf("FindeZumBestellen(…)[1].IstDublette = false, erwartet true")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestFindeZumBestellen_NurKatalogWennDNBScheitert(t *testing.T) {
	mock, err := pgxmock.NewPool(pgxmock.QueryMatcherOption(pgxmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("pgxmock: %v", err)
	}
	defer mock.Close()

	ctx := context.Background()

	localQuery := `WITH matched_titels AS`
	mock.ExpectQuery(localQuery).
		WithArgs("TestBook").
		WillReturnRows(pgxmock.NewRows([]string{"id", "titel", "autor", "isbn", "verlag", "cover_url", "signatur", "ist_lernmittel", "current_stock"}).
			AddRow("id-1", "Lokales Buch", "Autor A", "9781234567890", "Verlag X", "", "SIG-1", false, 3))

	mockTransport := &dnbTransportAttrappe{
		antwort: func(req *http.Request) (*http.Response, error) {
			return nil, errors.New("network error")
		},
	}

	metaClient := katalogdienste.NewMetadatenClient()
	metaClient.SetzeHTTPClientFuerTest(&http.Client{Transport: mockTransport})

	results, dnbAusfall, err := FindeZumBestellen(ctx, mock, metaClient, "TestBook")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Ohne das Merkmal sähe ein Buch, das die DNB kennt, aus wie eines, das sie nicht kennt.
	if !dnbAusfall {
		t.Error("die DNB hat nicht geantwortet, die Suche meldet keinen Ausfall")
	}

	if len(results) != 1 {
		t.Fatalf("FindeZumBestellen(…) = %d Treffer, erwartet 1", len(results))
	}

	if results[0].Quelle != "local" {
		t.Errorf("FindeZumBestellen(…)[0].Quelle = %s, erwartet local", results[0].Quelle)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestFindeZumBestellen_NurDNBWennKatalogScheitert(t *testing.T) {
	mock, err := pgxmock.NewPool(pgxmock.QueryMatcherOption(pgxmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("pgxmock: %v", err)
	}
	defer mock.Close()

	ctx := context.Background()

	localQuery := `WITH matched_titels AS`
	mock.ExpectQuery(localQuery).
		WithArgs("TestBook").
		WillReturnError(errors.New("db error"))

	mockTransport := &dnbTransportAttrappe{
		antwort: func(req *http.Request) (*http.Response, error) {
			if req.URL.Host == "services.dnb.de" {
				xml := `<searchRetrieveResponse xmlns="http://www.loc.gov/zing/srw/">
  <numberOfRecords>1</numberOfRecords>
  <records>
    <record>
      <recordData>
        <record xmlns="http://www.loc.gov/MARC21/slim">
          <datafield tag="020" ind1=" " ind2=" ">
            <subfield code="a">9780987654321</subfield>
          </datafield>
          <datafield tag="245" ind1="1" ind2="0">
            <subfield code="a">DNB Buch</subfield>
          </datafield>
        </record>
      </recordData>
    </record>
  </records>
</searchRetrieveResponse>`
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewBufferString(xml)),
					Header:     make(http.Header),
				}, nil
			}
			return nil, errors.New("unexpected host")
		},
	}

	metaClient := katalogdienste.NewMetadatenClient()
	metaClient.SetzeHTTPClientFuerTest(&http.Client{Transport: mockTransport})

	mock.ExpectQuery(`SELECT isbn_normalform\(isbn\) FROM buecher_titel`).
		WithArgs(pgxmock.AnyArg()).
		WillReturnRows(pgxmock.NewRows([]string{"isbn_normalform"}))

	results, dnbAusfall, err := FindeZumBestellen(ctx, mock, metaClient, "TestBook")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if dnbAusfall {
		t.Error("die DNB hat geantwortet, die Suche meldet einen Ausfall")
	}

	if len(results) != 1 {
		t.Fatalf("FindeZumBestellen(…) = %d Treffer, erwartet 1", len(results))
	}

	if results[0].Quelle != "dnb" {
		t.Errorf("FindeZumBestellen(…)[0].Quelle = %s, erwartet dnb", results[0].Quelle)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestBaueDNBTreffer(t *testing.T) {
	ergebnis := katalogdienste.MetadatenErgebnis{
		ISBN:     "978-3-16-148410-0",
		Titel:    "Test Titel",
		Autor:    "Test Autor",
		Verlag:   "Test Verlag",
		Preis:    1999,
		CoverURL: "http://example.com/cover.jpg",
	}

	existing := map[string]struct{}{
		"9783161484100": {},
	}

	item := baueDNBTreffer(ergebnis, existing)

	if item.Titel != "Test Titel" {
		t.Errorf("item.Titel = %q, erwartet Titel 'Test Titel'", item.Titel)
	}
	if item.CoverURL != "http://example.com/cover.jpg" {
		t.Errorf("item.CoverURL = %q, erwartet CoverURL 'http://example.com/cover.jpg'", item.CoverURL)
	}
	if !item.IstDublette {
		t.Errorf("baueDNBTreffer(…).IstDublette = false, erwartet true")
	}

	ergebnisOhneCover := ergebnis
	ergebnisOhneCover.CoverURL = ""

	existingLeer := map[string]struct{}{}
	item2 := baueDNBTreffer(ergebnisOhneCover, existingLeer)

	if item2.CoverURL != "https://portal.dnb.de/opac/mvb/cover?isbn=978-3-16-148410-0" {
		t.Errorf("baueDNBTreffer(ohne Cover).CoverURL = %q, erwartet den Cover-Fallback der DNB", item2.CoverURL)
	}
	if item2.IstDublette {
		t.Errorf("baueDNBTreffer(ohne Cover).IstDublette = true, erwartet false")
	}
}

// Die DNB nennt eine ISBN mit Bindestrichen und bei älteren Sätzen zehnstellig, der Katalog
// führt sie in der Normalform: Der Treffer heißt „Vorhanden", nicht „Neu". Eine zehnstellige
// Nummer mit falschem Prüfzeichen ist keine andere Schreibweise der dreizehnstelligen.
func TestBaueDNBTreffer_VorhandenInJederSchreibweise(t *testing.T) {
	for name, f := range map[string]struct {
		dnb, imKatalog string
		vorhanden      bool
	}{
		"dreizehnstellig mit Bindestrichen": {"978-3-551-55167-2", "9783551551672", true},
		"zehnstellig aus der DNB":           {"3-551-55167-7", "9783551551672", true},
		"ein anderes Buch":                  {"978-3-551-55167-2", "9783551551665", false},
		"falsches Prüfzeichen im Katalog":   {"978-3-499-50025-1", "3499500252", false},
		"ohne ISBN":                         {"", "9783551551672", false},
	} {
		treffer := katalogdienste.MetadatenErgebnis{ISBN: f.dnb, Titel: "Probe"}
		item := baueDNBTreffer(treffer, map[string]struct{}{f.imKatalog: {}})
		if item.IstDublette != f.vorhanden {
			t.Errorf("%s: IstDublette = %v, erwartet %v", name, item.IstDublette, f.vorhanden)
		}
	}
}

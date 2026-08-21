package service

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"testing"

	"bibliothek/inventur"
	"github.com/pashagolub/pgxmock/v4"
)

type orderMockTransport struct {
	roundTripFunc func(req *http.Request) (*http.Response, error)
}

func (t *orderMockTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	return t.roundTripFunc(req)
}

func TestSearchOrders_ReturnsCombinedResults(t *testing.T) {
	mock, err := pgxmock.NewPool(pgxmock.QueryMatcherOption(pgxmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("pgxmock: %v", err)
	}
	defer mock.Close()

	ctx := context.Background()

	localQuery := `WITH matched_titels AS`
	mock.ExpectQuery(localQuery).
		WithArgs("TestBook").
		WillReturnRows(pgxmock.NewRows([]string{"id", "titel", "autor", "isbn", "verlag", "cover_url", "signatur", "current_stock"}).
			AddRow("id-1", "Lokales Buch", "Autor A", "9781234567890", "Verlag X", "", "SIG-1", 3))

	mockTransport := &orderMockTransport{
		roundTripFunc: func(req *http.Request) (*http.Response, error) {
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

	metaClient := inventur.NeuerMetadatenClient()
	metaClient.SetzeHTTPClientFuerTest(&http.Client{Transport: mockTransport})

	mock.ExpectQuery(`SELECT replace\(isbn, '-', ''\) FROM buecher_titel`).
		WithArgs(pgxmock.AnyArg()).
		WillReturnRows(pgxmock.NewRows([]string{"replace"}).
			AddRow("9780987654321"))

	results, err := SearchOrders(ctx, mock, metaClient, "TestBook")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}

	if results[0].Source != "local" || results[0].Titel != "Lokales Buch" {
		t.Errorf("unexpected local result: %+v", results[0])
	}

	if results[1].Source != "dnb" || results[1].Titel != "DNB Buch" {
		t.Errorf("unexpected DNB result: %+v", results[1])
	}

	if !results[1].IsDuplicate {
		t.Errorf("expected DNB result to be marked as duplicate")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestSearchOrders_LocalOnlyWhenDNBFails(t *testing.T) {
	mock, err := pgxmock.NewPool(pgxmock.QueryMatcherOption(pgxmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("pgxmock: %v", err)
	}
	defer mock.Close()

	ctx := context.Background()

	localQuery := `WITH matched_titels AS`
	mock.ExpectQuery(localQuery).
		WithArgs("TestBook").
		WillReturnRows(pgxmock.NewRows([]string{"id", "titel", "autor", "isbn", "verlag", "cover_url", "signatur", "current_stock"}).
			AddRow("id-1", "Lokales Buch", "Autor A", "9781234567890", "Verlag X", "", "SIG-1", 3))

	mockTransport := &orderMockTransport{
		roundTripFunc: func(req *http.Request) (*http.Response, error) {
			return nil, errors.New("network error")
		},
	}

	metaClient := inventur.NeuerMetadatenClient()
	metaClient.SetzeHTTPClientFuerTest(&http.Client{Transport: mockTransport})

	results, err := SearchOrders(ctx, mock, metaClient, "TestBook")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	if results[0].Source != "local" {
		t.Errorf("expected local result, got %s", results[0].Source)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestSearchOrders_DNBOnlyWhenLocalFails(t *testing.T) {
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

	mockTransport := &orderMockTransport{
		roundTripFunc: func(req *http.Request) (*http.Response, error) {
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

	metaClient := inventur.NeuerMetadatenClient()
	metaClient.SetzeHTTPClientFuerTest(&http.Client{Transport: mockTransport})

	mock.ExpectQuery(`SELECT replace\(isbn, '-', ''\) FROM buecher_titel`).
		WithArgs(pgxmock.AnyArg()).
		WillReturnRows(pgxmock.NewRows([]string{"replace"}))

	results, err := SearchOrders(ctx, mock, metaClient, "TestBook")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	if results[0].Source != "dnb" {
		t.Errorf("expected DNB result, got %s", results[0].Source)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestBaueDNBSuchItem(t *testing.T) {
	ergebnis := inventur.MetadatenErgebnis{
		ISBN:     "978-3-16-148410-0",
		Titel:    "Test Titel",
		Autor:    "Test Autor",
		Verlag:   "Test Verlag",
		Preis:    19.99,
		CoverURL: "http://example.com/cover.jpg",
	}

	existing := map[string]struct{}{
		"9783161484100": {},
	}

	item := baueDNBSuchItem(ergebnis, existing)

	if item.Titel != "Test Titel" {
		t.Errorf("erwartet Titel 'Test Titel', bekam %q", item.Titel)
	}
	if item.CoverURL != "http://example.com/cover.jpg" {
		t.Errorf("erwartet CoverURL 'http://example.com/cover.jpg', bekam %q", item.CoverURL)
	}
	if !item.IsDuplicate {
		t.Errorf("erwartet, dass IsDuplicate true ist")
	}

	ergebnisOhneCover := ergebnis
	ergebnisOhneCover.CoverURL = ""

	existingLeer := map[string]struct{}{}
	item2 := baueDNBSuchItem(ergebnisOhneCover, existingLeer)

	if item2.CoverURL != "https://portal.dnb.de/opac/mvb/cover?isbn=978-3-16-148410-0" {
		t.Errorf("erwartet DNB Cover-Fallback, bekam %q", item2.CoverURL)
	}
	if item2.IsDuplicate {
		t.Errorf("erwartet, dass IsDuplicate false ist")
	}
}

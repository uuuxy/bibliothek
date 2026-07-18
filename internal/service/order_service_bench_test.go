package service

import (
	"context"
	"testing"
    "fmt"

	"github.com/pashagolub/pgxmock/v4"
	"bibliothek/inventur"
    "net/http"
    "bytes"
    "io"
)

type MockTransport struct{}

func (t *MockTransport) RoundTrip(req *http.Request) (*http.Response, error) {
    xml := `<searchRetrieveResponse xmlns="http://www.loc.gov/zing/srw/">
  <numberOfRecords>50</numberOfRecords>
  <records>`
    for i := 0; i < 50; i++ {
        xml += fmt.Sprintf(`
    <record>
      <recordData>
        <record xmlns="http://www.loc.gov/MARC21/slim">
          <datafield tag="020" ind1=" " ind2=" ">
            <subfield code="a">9781234567%03d</subfield>
          </datafield>
          <datafield tag="245" ind1="1" ind2="0">
            <subfield code="a">Test Buch %d</subfield>
          </datafield>
        </record>
      </recordData>
    </record>`, i, i)
    }
    xml += `
  </records>
</searchRetrieveResponse>`

	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(bytes.NewBufferString(xml)),
		Header:     make(http.Header),
	}, nil
}

func BenchmarkSearchDNBOrders(b *testing.B) {
	ctx := context.Background()
	pool, err := pgxmock.NewPool(pgxmock.QueryMatcherOption(pgxmock.QueryMatcherRegexp))
	if err != nil {
		b.Fatalf("failed to create mock pool: %v", err)
	}
	defer pool.Close()

    original := http.DefaultTransport
	http.DefaultTransport = &MockTransport{}
	defer func() { http.DefaultTransport = original }()

    metaClient := inventur.NeuerMetadatenClient()

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        // We set up expectation for the bulk query
        pool.ExpectQuery("SELECT replace\\(isbn, '-', ''\\) FROM buecher_titel").
             WithArgs(pgxmock.AnyArg()).
             WillReturnRows(pgxmock.NewRows([]string{"replace"}))
        _ = searchDNBOrders(ctx, pool, metaClient, "test")
    }
}

package inventur

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"io"
	"net/http"
	"os"
	"testing"

	"github.com/pashagolub/pgxmock/v4"
)

type mockTransport struct {
	RoundTripFunc func(req *http.Request) (*http.Response, error)
}

func (m *mockTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	return m.RoundTripFunc(req)
}

func TestRunCoverMigration(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer mock.Close()

	// Mocking http transport
	originalTransport := http.DefaultTransport
	defer func() { http.DefaultTransport = originalTransport }()

	// create 10x10 image
	img := image.NewRGBA(image.Rect(0, 0, 10, 10))
	for x := 0; x < 10; x++ {
		for y := 0; y < 10; y++ {
			img.Set(x, y, color.RGBA{255, 0, 0, 255})
		}
	}
	var buf bytes.Buffer
	err = png.Encode(&buf, img)
	if err != nil {
		t.Fatalf("failed to encode image: %v", err)
	}

	http.DefaultTransport = &mockTransport{
		RoundTripFunc: func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(bytes.NewReader(buf.Bytes())),
				Header:     make(http.Header),
			}, nil
		},
	}

	// db query
	mock.ExpectQuery("^SELECT id, isbn, cover_url, titel AS title FROM buecher_titel WHERE cover_url LIKE 'http%'$").
		WillReturnRows(pgxmock.NewRows([]string{"id", "isbn", "cover_url", "title"}).
			AddRow("1", "9783161484100", "http://openlibrary.org/cover.jpg", "Test Book"))

	// UPDATE
	mock.ExpectExec("^UPDATE buecher_titel SET cover_url = \\$1 WHERE id = \\$2$").
		WithArgs(pgxmock.AnyArg(), "1").
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	// Clean uploads folder after test
	defer os.RemoveAll("uploads")

	RunCoverMigration(mock)

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}

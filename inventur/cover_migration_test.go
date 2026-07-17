package inventur

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"testing"

	"github.com/pashagolub/pgxmock/v4"
)

type mockCoverTransport struct {
	roundTripFunc func(req *http.Request) (*http.Response, error)
}

func (m *mockCoverTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	return m.roundTripFunc(req)
}

func TestLadeExterneCoverBuecher(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer mock.Close()

	t.Run("Success", func(t *testing.T) {
		mock.ExpectQuery("SELECT id, isbn, cover_url, titel AS title FROM buecher_titel WHERE cover_url LIKE 'http%'").
			WillReturnRows(pgxmock.NewRows([]string{"id", "isbn", "cover_url", "title"}).
				AddRow("1", "1234567890", "http://openlibrary.org/cover.jpg", "Test Buch"))

		books, ok := ladeExterneCoverBuecher(context.Background(), mock)
		if !ok {
			t.Errorf("expected ok = true")
		}
		if len(books) != 1 {
			t.Errorf("expected 1 book, got %d", len(books))
		}
		if books[0].ID != "1" {
			t.Errorf("expected ID 1, got %s", books[0].ID)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("there were unfulfilled expectations: %s", err)
		}
	})
}

func TestMigriereEinzelnesCover(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer mock.Close()

	// Create uploads directory if it doesn't exist, and defer removal of test files if any.
	os.MkdirAll("uploads", 0750)
	defer os.RemoveAll("uploads")

	t.Run("Empty Cover URL", func(t *testing.T) {
		client := &http.Client{}
		b := coverMigrationBuch{ID: "1", ISBN: "123", CoverURL: "", Title: "Test"}
		erfolgreich, fehlerhaft := migriereEinzelnesCover(context.Background(), client, mock, b)
		if erfolgreich || fehlerhaft {
			t.Errorf("expected false, false, got %v, %v", erfolgreich, fehlerhaft)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("there were unfulfilled expectations: %s", err)
		}
	})

	t.Run("Failed Download", func(t *testing.T) {
		originalTransport := http.DefaultTransport
		http.DefaultTransport = &mockCoverTransport{
			roundTripFunc: func(req *http.Request) (*http.Response, error) {
				return nil, fmt.Errorf("network error")
			},
		}
		defer func() { http.DefaultTransport = originalTransport }()

		client := &http.Client{Transport: &mockCoverTransport{
			roundTripFunc: func(req *http.Request) (*http.Response, error) {
				return nil, fmt.Errorf("network error")
			},
		}}
		b := coverMigrationBuch{ID: "4", ISBN: "1234567890", CoverURL: "http://covers.openlibrary.org/b/isbn/123-L.jpg", Title: "Test"}

		erfolgreich, fehlerhaft := migriereEinzelnesCover(context.Background(), client, mock, b)
		if erfolgreich || !fehlerhaft {
			t.Errorf("expected false, true, got %v, %v", erfolgreich, fehlerhaft)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("there were unfulfilled expectations: %s", err)
		}
	})

	t.Run("Successful Download and Update", func(t *testing.T) {
		originalTransport := http.DefaultTransport
		http.DefaultTransport = &mockCoverTransport{
			roundTripFunc: func(req *http.Request) (*http.Response, error) {
				if req.Method == http.MethodHead {
					return &http.Response{
						StatusCode: http.StatusOK,
						Body:       io.NopCloser(bytes.NewBufferString("")),
					}, nil
				}

				tinyPNG := []byte{
					0x52, 0x49, 0x46, 0x46, 0x40, 0x00, 0x00, 0x00,
					0x57, 0x45, 0x42, 0x50, 0x56, 0x50, 0x38, 0x20,
					0x34, 0x00, 0x00, 0x00, 0x30, 0x02, 0x00, 0x9d,
					0x01, 0x2a, 0x0b, 0x00, 0x0b, 0x00, 0x00, 0xc0,
					0x12, 0x25, 0xa0, 0x02, 0x74, 0xba, 0x01, 0xf8,
					0x01, 0xf8, 0x00, 0x04, 0x68, 0x00, 0x00, 0xfe,
					0xfa, 0x21, 0x97, 0xff, 0x77, 0x9a, 0xb0, 0xd3,
					0x77, 0xf5, 0xad, 0xff, 0xf5, 0xa3, 0x9f, 0xae,
					0x89, 0xfe, 0xb4, 0x73, 0xff, 0x59, 0x58, 0x00,
				}

				h := make(http.Header)
				h.Set("Content-Type", "image/webp")
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewBuffer(tinyPNG)),
					Header:     h,
				}, nil
			},
		}
		defer func() { http.DefaultTransport = originalTransport }()

		client := &http.Client{Transport: &mockCoverTransport{
			roundTripFunc: func(req *http.Request) (*http.Response, error) {
				if req.Method == http.MethodHead {
					return &http.Response{
						StatusCode: http.StatusOK,
						Body:       io.NopCloser(bytes.NewBufferString("")),
					}, nil
				}

				tinyPNG := []byte{
					0x52, 0x49, 0x46, 0x46, 0x40, 0x00, 0x00, 0x00,
					0x57, 0x45, 0x42, 0x50, 0x56, 0x50, 0x38, 0x20,
					0x34, 0x00, 0x00, 0x00, 0x30, 0x02, 0x00, 0x9d,
					0x01, 0x2a, 0x0b, 0x00, 0x0b, 0x00, 0x00, 0xc0,
					0x12, 0x25, 0xa0, 0x02, 0x74, 0xba, 0x01, 0xf8,
					0x01, 0xf8, 0x00, 0x04, 0x68, 0x00, 0x00, 0xfe,
					0xfa, 0x21, 0x97, 0xff, 0x77, 0x9a, 0xb0, 0xd3,
					0x77, 0xf5, 0xad, 0xff, 0xf5, 0xa3, 0x9f, 0xae,
					0x89, 0xfe, 0xb4, 0x73, 0xff, 0x59, 0x58, 0x00,
				}

				h := make(http.Header)
				h.Set("Content-Type", "image/webp")
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewBuffer(tinyPNG)),
					Header:     h,
				}, nil
			},
		}}
		b := coverMigrationBuch{ID: "2", ISBN: "1234567890", CoverURL: "http://covers.openlibrary.org/b/isbn/123-L.jpg", Title: "Test"}

		mock.ExpectExec("UPDATE buecher_titel SET cover_url = \\$1 WHERE id = \\$2").
			WithArgs(pgxmock.AnyArg(), "2").
			WillReturnResult(pgxmock.NewResult("UPDATE", 1))

		erfolgreich, fehlerhaft := migriereEinzelnesCover(context.Background(), client, mock, b)
		if !erfolgreich || fehlerhaft {
			t.Errorf("expected true, false, got %v, %v", erfolgreich, fehlerhaft)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("there were unfulfilled expectations: %s", err)
		}
	})

	t.Run("Failed Update", func(t *testing.T) {
		originalTransport := http.DefaultTransport
		http.DefaultTransport = &mockCoverTransport{
			roundTripFunc: func(req *http.Request) (*http.Response, error) {
				if req.Method == http.MethodHead {
					return &http.Response{
						StatusCode: http.StatusOK,
						Body:       io.NopCloser(bytes.NewBufferString("")),
					}, nil
				}

				tinyPNG := []byte{
					0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a,
					0x00, 0x00, 0x00, 0x0d, 0x49, 0x48, 0x44, 0x52,
					0x00, 0x00, 0x00, 0x0a, 0x00, 0x00, 0x00, 0x0a,
					0x08, 0x06, 0x00, 0x00, 0x00, 0x8d, 0x32, 0xcf, 0xbd,
					0x00, 0x00, 0x00, 0x0c, 0x49, 0x44, 0x41, 0x54,
					0x18, 0x57, 0x63, 0xf8, 0xff, 0xff, 0x3f, 0x00,
					0x05, 0xfe, 0x02, 0xfe, 0xa7, 0x35, 0x81, 0x84,
					0x00, 0x00, 0x00, 0x00, 0x49, 0x45, 0x4e, 0x44,
					0xae, 0x42, 0x60, 0x82,
				}

				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewBuffer(tinyPNG)),
					Header:     make(http.Header),
				}, nil
			},
		}
		defer func() { http.DefaultTransport = originalTransport }()

		client := &http.Client{Transport: &mockCoverTransport{
			roundTripFunc: func(req *http.Request) (*http.Response, error) {
				if req.Method == http.MethodHead {
					return &http.Response{
						StatusCode: http.StatusOK,
						Body:       io.NopCloser(bytes.NewBufferString("")),
					}, nil
				}

				tinyPNG := []byte{
					0x52, 0x49, 0x46, 0x46, 0x40, 0x00, 0x00, 0x00,
					0x57, 0x45, 0x42, 0x50, 0x56, 0x50, 0x38, 0x20,
					0x34, 0x00, 0x00, 0x00, 0x30, 0x02, 0x00, 0x9d,
					0x01, 0x2a, 0x0b, 0x00, 0x0b, 0x00, 0x00, 0xc0,
					0x12, 0x25, 0xa0, 0x02, 0x74, 0xba, 0x01, 0xf8,
					0x01, 0xf8, 0x00, 0x04, 0x68, 0x00, 0x00, 0xfe,
					0xfa, 0x21, 0x97, 0xff, 0x77, 0x9a, 0xb0, 0xd3,
					0x77, 0xf5, 0xad, 0xff, 0xf5, 0xa3, 0x9f, 0xae,
					0x89, 0xfe, 0xb4, 0x73, 0xff, 0x59, 0x58, 0x00,
				}

				h := make(http.Header)
				h.Set("Content-Type", "image/webp")
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewBuffer(tinyPNG)),
					Header:     h,
				}, nil
			},
		}}
		b := coverMigrationBuch{ID: "3", ISBN: "1234567890", CoverURL: "http://covers.openlibrary.org/b/isbn/123-L.jpg", Title: "Test"}

		mock.ExpectExec("UPDATE buecher_titel SET cover_url = \\$1 WHERE id = \\$2").
			WithArgs(pgxmock.AnyArg(), "3").
			WillReturnError(fmt.Errorf("update error"))

		erfolgreich, fehlerhaft := migriereEinzelnesCover(context.Background(), client, mock, b)
		if erfolgreich || !fehlerhaft {
			t.Errorf("expected false, true, got %v, %v", erfolgreich, fehlerhaft)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("there were unfulfilled expectations: %s", err)
		}
	})
}

func TestRunCoverMigration(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer mock.Close()

	os.MkdirAll("uploads", 0750)
	defer os.RemoveAll("uploads")

	// Replace http.DefaultTransport
	originalTransport := http.DefaultTransport
	http.DefaultTransport = &mockCoverTransport{
		roundTripFunc: func(req *http.Request) (*http.Response, error) {
			if req.Method == http.MethodHead {
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewBufferString("")),
				}, nil
			}

			tinyPNG := []byte{
				0x52, 0x49, 0x46, 0x46, 0x40, 0x00, 0x00, 0x00,
				0x57, 0x45, 0x42, 0x50, 0x56, 0x50, 0x38, 0x20,
				0x34, 0x00, 0x00, 0x00, 0x30, 0x02, 0x00, 0x9d,
				0x01, 0x2a, 0x0b, 0x00, 0x0b, 0x00, 0x00, 0xc0,
				0x12, 0x25, 0xa0, 0x02, 0x74, 0xba, 0x01, 0xf8,
				0x01, 0xf8, 0x00, 0x04, 0x68, 0x00, 0x00, 0xfe,
				0xfa, 0x21, 0x97, 0xff, 0x77, 0x9a, 0xb0, 0xd3,
				0x77, 0xf5, 0xad, 0xff, 0xf5, 0xa3, 0x9f, 0xae,
				0x89, 0xfe, 0xb4, 0x73, 0xff, 0x59, 0x58, 0x00,
			}
			h := make(http.Header)
			h.Set("Content-Type", "image/webp")
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewBuffer(tinyPNG)),
				Header:     h,
			}, nil
		},
	}
	defer func() { http.DefaultTransport = originalTransport }()

	mock.ExpectQuery("SELECT id, isbn, cover_url, titel AS title FROM buecher_titel WHERE cover_url LIKE 'http%'").
		WillReturnRows(pgxmock.NewRows([]string{"id", "isbn", "cover_url", "title"}).
			AddRow("1", "1234567890", "http://covers.openlibrary.org/b/isbn/123-L.jpg", "Test Buch").
			AddRow("2", "0987654321", "", "Empty Buch"))

	mock.ExpectExec("UPDATE buecher_titel SET cover_url = \\$1 WHERE id = \\$2").
		WithArgs(pgxmock.AnyArg(), "1").
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	RunCoverMigration(mock)

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}

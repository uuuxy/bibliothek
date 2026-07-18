package inventur

import (
	"context"
	"net/http"
	"sync"
	"testing"

	"github.com/pashagolub/pgxmock/v4"
)

// MockTransport is a custom http.RoundTripper for testing MetadatenClient.
type MockTransport struct {
	ResponseFunc func(req *http.Request) (*http.Response, error)
}

func (m *MockTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if m.ResponseFunc != nil {
		return m.ResponseFunc(req)
	}
	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       http.NoBody,
	}, nil
}

func TestIstPlatzhalterCover(t *testing.T) {
	tests := []struct {
		name     string
		coverURL string
		isbn     string
		want     bool
	}{
		{"Empty URL", "", "123", true},
		{"Generic OpenLibrary", openLibraryLeeresCover, "123", true},
		{"ISBN specific placeholder", "https://covers.openlibrary.org/b/isbn/123-L.jpg", "123", true},
		{"Valid cover URL", "https://example.com/cover.jpg", "123", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := istPlatzhalterCover(tt.coverURL, tt.isbn); got != tt.want {
				t.Errorf("istPlatzhalterCover() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBrauchtAktualisierung(t *testing.T) {
	tests := []struct {
		name string
		book Book
		want bool
	}{
		{
			name: "Needs update - empty cover",
			book: Book{CoverURL: "", ISBN: "123", Subject: "Math", GradeLevel: 5},
			want: true,
		},
		{
			name: "Needs update - empty subject",
			book: Book{CoverURL: "https://example.com/cover.jpg", ISBN: "123", Subject: "", GradeLevel: 5},
			want: true,
		},
		{
			name: "Needs update - Kein Fach",
			book: Book{CoverURL: "https://example.com/cover.jpg", ISBN: "123", Subject: "Kein Fach", GradeLevel: 5},
			want: true,
		},
		{
			name: "Needs update - zero grade level",
			book: Book{CoverURL: "https://example.com/cover.jpg", ISBN: "123", Subject: "Math", GradeLevel: 0},
			want: true,
		},
		{
			name: "No update needed",
			book: Book{CoverURL: "https://example.com/cover.jpg", ISBN: "123", Subject: "Math", GradeLevel: 5},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := brauchtAktualisierung(tt.book); got != tt.want {
				t.Errorf("brauchtAktualisierung() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAktualisiereKategorie(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer mock.Close()

	repo := NewBookRepository(mock)
	ctx := context.Background()

	t.Run("No new info", func(t *testing.T) {
		b := Book{Subject: "Math", GradeLevel: 5}
		nachschlagen := &MetadatenErgebnis{Fach: "", KlassenStufe: ""}
		if got := aktualisiereKategorie(ctx, repo, b, nachschlagen); got != 0 {
			t.Errorf("aktualisiereKategorie() = %v, want 0", got)
		}
	})

	t.Run("Update subject and grade", func(t *testing.T) {
		b := Book{ID: "00000000-0000-0000-0000-000000000001", Subject: "", GradeLevel: 0, ISBN: "123"}
		nachschlagen := &MetadatenErgebnis{Fach: "Science", KlassenStufe: "7"}

		mock.ExpectExec("UPDATE buecher_titel").
			WithArgs("Science", int16(7), "00000000-0000-0000-0000-000000000001").
			WillReturnResult(pgxmock.NewResult("UPDATE", 1))

		if got := aktualisiereKategorie(ctx, repo, b, nachschlagen); got != 1 {
			t.Errorf("aktualisiereKategorie() = %v, want 1", got)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("there were unfulfilled expectations: %s", err)
		}
	})

	t.Run("Parse invalid grade level", func(t *testing.T) {
		b := Book{ID: "00000000-0000-0000-0000-000000000001", Subject: "", GradeLevel: 0, ISBN: "123"}
		nachschlagen := &MetadatenErgebnis{Fach: "Science", KlassenStufe: "abc"}

		mock.ExpectExec("UPDATE buecher_titel").
			WithArgs("Science", int16(0), "00000000-0000-0000-0000-000000000001").
			WillReturnResult(pgxmock.NewResult("UPDATE", 1))

		if got := aktualisiereKategorie(ctx, repo, b, nachschlagen); got != 1 {
			t.Errorf("aktualisiereKategorie() = %v, want 1", got)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("there were unfulfilled expectations: %s", err)
		}
	})

	t.Run("No change to category", func(t *testing.T) {
		b := Book{ID: "00000000-0000-0000-0000-000000000001", Subject: "Science", GradeLevel: 7, ISBN: "123"}
		nachschlagen := &MetadatenErgebnis{Fach: "Science", KlassenStufe: "7"}

		if got := aktualisiereKategorie(ctx, repo, b, nachschlagen); got != 0 {
			t.Errorf("aktualisiereKategorie() = %v, want 0", got)
		}
	})
}

func TestAktualisiereCover(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer mock.Close()

	repo := NewBookRepository(mock)
	ctx := context.Background()

	t.Run("No new cover", func(t *testing.T) {
		b := Book{CoverURL: "", ISBN: "123"}
		nachschlagen := &MetadatenErgebnis{CoverURL: ""}
		if got := aktualisiereCover(ctx, repo, b, nachschlagen); got != 0 {
			t.Errorf("aktualisiereCover() = %v, want 0", got)
		}
	})

	t.Run("Existing cover is not placeholder", func(t *testing.T) {
		b := Book{CoverURL: "https://example.com/cover.jpg", ISBN: "123"}
		nachschlagen := &MetadatenErgebnis{CoverURL: "https://example.com/newcover.jpg"}
		if got := aktualisiereCover(ctx, repo, b, nachschlagen); got != 0 {
			t.Errorf("aktualisiereCover() = %v, want 0", got)
		}
	})

	t.Run("Update cover", func(t *testing.T) {
		b := Book{ID: "00000000-0000-0000-0000-000000000001", Title: "T", Author: "A", CoverURL: "", ISBN: "123"}
		nachschlagen := &MetadatenErgebnis{CoverURL: "https://example.com/newcover.jpg"}

		mock.ExpectExec("UPDATE buecher_titel").
			WithArgs("T", "A", "https://example.com/newcover.jpg", "00000000-0000-0000-0000-000000000001").
			WillReturnResult(pgxmock.NewResult("UPDATE", 1))

		if got := aktualisiereCover(ctx, repo, b, nachschlagen); got != 1 {
			t.Errorf("aktualisiereCover() = %v, want 1", got)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("there were unfulfilled expectations: %s", err)
		}
	})
}

func TestLadeMetadaten(t *testing.T) {
	ctx := context.Background()
	var cache sync.Map

	// Create client with mocked transport
	client := &MetadatenClient{
		httpClient: &http.Client{
			Transport: &MockTransport{
				ResponseFunc: func(req *http.Request) (*http.Response, error) {
					// Simulate empty response or no hits
					return &http.Response{
						StatusCode: http.StatusOK,
						Body:       http.NoBody,
					}, nil
				},
			},
		},
	}

	t.Run("From cache", func(t *testing.T) {
		expected := &MetadatenErgebnis{CoverURL: "cache"}
		cache.Store("123", expected)

		got := ladeMetadaten(ctx, client, "123", &cache)
		if got != expected {
			t.Errorf("ladeMetadaten() = %v, want %v", got, expected)
		}
	})
}

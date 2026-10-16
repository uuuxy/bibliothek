package inventur

import (
	"bibliothek/repository"
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/jackc/pgx/v5"
	"github.com/pashagolub/pgxmock/v5"
)

func TestLadeTitel(t *testing.T) {
	ctx := context.Background()
	abfrageMuster := `SELECT id, COALESCE\(isbn, ''\) AS isbn`

	t.Run("success", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		if err != nil {
			t.Fatalf("pgxmock.NewPool: %v", err)
		}
		defer mock.Close()
		abfragen := NewTitelAbfragen(mock)

		zuletztGezaehlt := "2023-01-01"
		mock.ExpectQuery(abfrageMuster).
			WithArgs("valid-id").
			WillReturnRows(pgxmock.NewRows([]string{
				"id", "isbn", "title", "author", "signatur", "cover_url",
				"subject", "track", "stock", "last_counted",
				"sort_order", "medientyp", "jahrgang_von", "jahrgang_bis",
				"erweiterte_eigenschaften", "auflage",
			}).AddRow(
				"valid-id", "1234567890", "Test Title", "Test Author", "SIG", "http://cover",
				"Math", "A", 10, &zuletztGezaehlt,
				1, "Buch", 5, 10,
				map[string]any{"key": "value"}, "4. Aufl. 2023",
			))

		eintrag, err := abfragen.ladeTitel(ctx, "valid-id")
		if err != nil {
			t.Fatalf("abfragen.LadeTitel: %v", err)
		}
		if eintrag == nil {
			t.Error("eintrag = nil, erwartet einen Wert")
		}
		if eintrag.ID != "valid-id" {
			t.Errorf("eintrag.ID = %q, erwartet %q", eintrag.ID, "valid-id")
		}
		if eintrag.Titel != "Test Title" {
			t.Errorf("eintrag.Titel = %q, erwartet %q", eintrag.Titel, "Test Title")
		}
		if diff := cmp.Diff(map[string]any{"key": "value"}, eintrag.ErweiterteEigenschaften); diff != "" {
			t.Errorf("eintrag.ErweiterteEigenschaften weicht ab (-erwartet +ist):\n%s", diff)
		}
	})

	t.Run("not found", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		if err != nil {
			t.Fatalf("pgxmock.NewPool: %v", err)
		}
		defer mock.Close()
		abfragen := NewTitelAbfragen(mock)

		mock.ExpectQuery(abfrageMuster).
			WithArgs("invalid-id").
			WillReturnError(pgx.ErrNoRows)

		eintrag, err := abfragen.ladeTitel(ctx, "invalid-id")
		if !errors.Is(err, repository.ErrTitelNichtGefunden) {
			t.Errorf("abfragen.LadeTitel: Fehler %v, erwartet %v", err, repository.ErrTitelNichtGefunden)
		}
		if eintrag != nil {
			t.Errorf("eintrag = %v, erwartet nil", eintrag)
		}
	})

	// Ein Fehler der Datenbank ist kein „nicht gefunden": Ein Fehler beim Scan (NULL in string)
	// endete sonst als 404.
	t.Run("db error ist kein not-found", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		if err != nil {
			t.Fatalf("pgxmock.NewPool: %v", err)
		}
		defer mock.Close()
		abfragen := NewTitelAbfragen(mock)

		attrappenFehler := errors.New("db error")
		mock.ExpectQuery(abfrageMuster).
			WithArgs("error-id").
			WillReturnError(attrappenFehler)

		eintrag, err := abfragen.ladeTitel(ctx, "error-id")
		if !errors.Is(err, attrappenFehler) {
			t.Errorf("abfragen.LadeTitel: Fehler %v, erwartet %v", err, attrappenFehler)
		}
		if errors.Is(err, repository.ErrTitelNichtGefunden) {
			t.Errorf("abfragen.LadeTitel: Fehler %v, erwartet einen anderen als %v", err, repository.ErrTitelNichtGefunden)
		}
		if eintrag != nil {
			t.Errorf("eintrag = %v, erwartet nil", eintrag)
		}
	})
}

func TestAendereTitelMetadaten(t *testing.T) {
	ctx := context.Background()
	abfrageMuster := `UPDATE buecher_titel`

	t.Run("success", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		if err != nil {
			t.Fatalf("pgxmock.NewPool: %v", err)
		}
		defer mock.Close()
		abfragen := NewTitelAbfragen(mock)

		mock.ExpectExec(abfrageMuster).
			WithArgs("New Title", "New Author", "http://new-cover", "valid-id").
			WillReturnResult(pgxmock.NewResult("UPDATE", 1))

		err = abfragen.aendereTitelMetadaten(ctx, "valid-id", "New Title", "New Author", "http://new-cover")
		if err != nil {
			t.Fatalf("abfragen.AendereTitelMetadaten: %v", err)
		}
	})

	t.Run("not found", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		if err != nil {
			t.Fatalf("pgxmock.NewPool: %v", err)
		}
		defer mock.Close()
		abfragen := NewTitelAbfragen(mock)

		mock.ExpectExec(abfrageMuster).
			WithArgs("New Title", "New Author", "http://new-cover", "invalid-id").
			WillReturnResult(pgxmock.NewResult("UPDATE", 0))

		err = abfragen.aendereTitelMetadaten(ctx, "invalid-id", "New Title", "New Author", "http://new-cover")
		if !errors.Is(err, repository.ErrTitelNichtGefunden) {
			t.Errorf("abfragen.AendereTitelMetadaten: Fehler %v, erwartet %v", err, repository.ErrTitelNichtGefunden)
		}
	})

	t.Run("error", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		if err != nil {
			t.Fatalf("pgxmock.NewPool: %v", err)
		}
		defer mock.Close()
		abfragen := NewTitelAbfragen(mock)

		attrappenFehler := errors.New("db error")
		mock.ExpectExec(abfrageMuster).
			WithArgs("New Title", "New Author", "http://new-cover", "error-id").
			WillReturnError(attrappenFehler)

		err = abfragen.aendereTitelMetadaten(ctx, "error-id", "New Title", "New Author", "http://new-cover")
		if err == nil {
			t.Error("abfragen.AendereTitelMetadaten: kein Fehler, erwartet einen")
		}
		if got := err.Error(); !strings.Contains(got, "metadaten konnten nicht aktualisiert werden: db error") {
			t.Errorf("err.Error() = %q, erwartet darin %q", got, "metadaten konnten nicht aktualisiert werden: db error")
		}
	})
}

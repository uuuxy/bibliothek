package inventur

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"bibliothek/repository"

	"github.com/google/go-cmp/cmp"
	"github.com/pashagolub/pgxmock/v5"
)

func TestTitelAbfragen_LadeTitelListe(t *testing.T) {
	ctx := context.Background()
	zuletztGezaehlt := "2023-01-01"

	t.Run("success", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		if err != nil {
			t.Fatalf("pgxmock.NewPool: %v", err)
		}
		defer mock.Close()
		abfragen := NewTitelAbfragen(mock)

		mock.ExpectQuery(`SELECT.+FROM buecher_titel bt LEFT JOIN buecher_exemplare e.+`).
			WithArgs("Math", "algebra", 50000).
			WillReturnRows(pgxmock.NewRows([]string{
				"id", "isbn", "title", "author", "signatur", "cover_url", "subject", "track", "ist_lernmittel",
				"verfuegbar", "gesamt", "im_zulauf", "last_counted", "sort_order", "medientyp", "jahrgang_von", "jahrgang_bis",
				"untertitel", "verlag", "erscheinungsjahr", "erweiterte_eigenschaften", "auflage",
				"listenpreis", "mehrjahresband", "werk_id", "werk_rang",
			}).AddRow(
				"book-1", "123", "Algebra", "Smith", "SIG-1", "url", "Math", "A", false,
				2, 3, 0, &zuletztGezaehlt, 1, "Buch", 5, 6,
				"", "", 2020, map[string]any{}, "4. Aufl. 2023", nil, false, "werk-1", 2,
			))
		mock.ExpectQuery(`SELECT tsw.titel_id.+FROM schlagworte sw`).
			WithArgs([]string{"book-1"}).
			WillReturnRows(pgxmock.NewRows([]string{"titel_id", "woerter"}))
		mock.ExpectQuery(`SELECT e.titel_id::text, e.standort.+FROM buecher_exemplare e`).
			WithArgs([]string{"book-1"}).
			WillReturnRows(pgxmock.NewRows([]string{"titel_id", "standort", "anzahl"}).
				AddRow("book-1", "Lehrerschrank", 2))

		eintraege, err := abfragen.LadeTitelListe(ctx, "Math", "algebra", false)
		if err != nil {
			t.Fatalf("abfragen.LadeTitelListe: %v", err)
		}
		if len(eintraege) != 1 {
			t.Errorf("len(eintraege) = %d, erwartet %d", len(eintraege), 1)
		}
		if len(eintraege) > 0 {
			if eintraege[0].ID != "book-1" {
				t.Errorf("eintraege[0].ID = %q, erwartet %q", eintraege[0].ID, "book-1")
			}
			if eintraege[0].Bestand != 3 {
				t.Errorf("eintraege[0].Bestand = %v, erwartet %v", eintraege[0].Bestand, 3)
			}
			// Die Standorte der Exemplare kommen mit der Liste.
			if diff := cmp.Diff([]repository.StandortZahl{{Standort: "Lehrerschrank", Anzahl: 2}}, eintraege[0].Standorte); diff != "" {
				t.Errorf("eintraege[0].Standorte weicht ab (-erwartet +ist):\n%s", diff)
			}
			// Buch und Rang der Auflage stehen am Ende der Zeile.
			if eintraege[0].WerkID != "werk-1" {
				t.Errorf("eintraege[0].WerkID = %q, erwartet %q", eintraege[0].WerkID, "werk-1")
			}
			if eintraege[0].WerkRang != 2 {
				t.Errorf("eintraege[0].WerkRang = %v, erwartet %v", eintraege[0].WerkRang, 2)
			}
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("offene Erwartungen: %v", err)
		}
	})

	t.Run("db error", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		if err != nil {
			t.Fatalf("pgxmock.NewPool: %v", err)
		}
		defer mock.Close()
		abfragen := NewTitelAbfragen(mock)

		mock.ExpectQuery(`SELECT.+FROM buecher_titel bt LEFT JOIN buecher_exemplare e.+`).
			WithArgs("", "", 50000).
			WillReturnError(fmt.Errorf("db connection failed"))

		eintraege, err := abfragen.LadeTitelListe(ctx, "", "", false)
		if err == nil || !strings.Contains(err.Error(), "bücher konnten nicht geladen werden") {
			t.Errorf("abfragen.LadeTitelListe: Fehler %v, erwartet einen mit %q", err, "bücher konnten nicht geladen werden")
		}
		if eintraege != nil {
			t.Errorf("eintraege = %v, erwartet nil", eintraege)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("offene Erwartungen: %v", err)
		}
	})
}

func TestTitelAbfragen_LadeTitelMitFremdemCover(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		if err != nil {
			t.Fatalf("pgxmock.NewPool: %v", err)
		}
		defer mock.Close()
		abfragen := NewTitelAbfragen(mock)

		mock.ExpectQuery(`SELECT id, COALESCE\(isbn, ''\) AS isbn, titel AS title, COALESCE\(cover_url, ''\) AS cover_url FROM buecher_titel WHERE cover_url LIKE 'http%' ORDER BY id ASC LIMIT \$1`).
			WithArgs(10).
			WillReturnRows(pgxmock.NewRows([]string{"id", "isbn", "title", "cover_url"}).
				AddRow("book-1", "123", "Title 1", "http://example.com/1.jpg"))

		eintraege, err := abfragen.ladeTitelMitFremdemCover(ctx, 10)
		if err != nil {
			t.Fatalf("abfragen.LadeTitelMitFremdemCover: %v", err)
		}
		if len(eintraege) != 1 {
			t.Errorf("len(eintraege) = %d, erwartet %d", len(eintraege), 1)
		}
		if len(eintraege) > 0 {
			if eintraege[0].ID != "book-1" {
				t.Errorf("eintraege[0].ID = %q, erwartet %q", eintraege[0].ID, "book-1")
			}
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("offene Erwartungen: %v", err)
		}
	})

	t.Run("success with default limit", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		if err != nil {
			t.Fatalf("pgxmock.NewPool: %v", err)
		}
		defer mock.Close()
		abfragen := NewTitelAbfragen(mock)

		mock.ExpectQuery(`SELECT id, COALESCE\(isbn, ''\) AS isbn, titel AS title, COALESCE\(cover_url, ''\) AS cover_url FROM buecher_titel WHERE cover_url LIKE 'http%' ORDER BY id ASC LIMIT \$1`).
			WithArgs(100).
			WillReturnRows(pgxmock.NewRows([]string{"id", "isbn", "title", "cover_url"}).
				AddRow("book-1", "123", "Title 1", "http://example.com/1.jpg"))

		eintraege, err := abfragen.ladeTitelMitFremdemCover(ctx, 0)
		if err != nil {
			t.Fatalf("abfragen.LadeTitelMitFremdemCover: %v", err)
		}
		if len(eintraege) != 1 {
			t.Errorf("len(eintraege) = %d, erwartet %d", len(eintraege), 1)
		}
		if len(eintraege) > 0 {
			if eintraege[0].ID != "book-1" {
				t.Errorf("eintraege[0].ID = %q, erwartet %q", eintraege[0].ID, "book-1")
			}
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("offene Erwartungen: %v", err)
		}
	})

	t.Run("db error", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		if err != nil {
			t.Fatalf("pgxmock.NewPool: %v", err)
		}
		defer mock.Close()
		abfragen := NewTitelAbfragen(mock)

		mock.ExpectQuery(`SELECT id, COALESCE\(isbn, ''\) AS isbn, titel AS title, COALESCE\(cover_url, ''\) AS cover_url FROM buecher_titel WHERE cover_url LIKE 'http%' ORDER BY id ASC LIMIT \$1`).
			WithArgs(100).
			WillReturnError(fmt.Errorf("db failure"))

		eintraege, err := abfragen.ladeTitelMitFremdemCover(ctx, -1)
		if err == nil || !strings.Contains(err.Error(), "bücher mit externen covern konnten nicht geladen werden") {
			t.Errorf("abfragen.LadeTitelMitFremdemCover: Fehler %v, erwartet einen mit %q", err, "bücher mit externen covern konnten nicht geladen werden")
		}
		if eintraege != nil {
			t.Errorf("eintraege = %v, erwartet nil", eintraege)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("offene Erwartungen: %v", err)
		}
	})

	t.Run("scan error", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		if err != nil {
			t.Fatalf("pgxmock.NewPool: %v", err)
		}
		defer mock.Close()
		abfragen := NewTitelAbfragen(mock)

		mock.ExpectQuery(`SELECT id, COALESCE\(isbn, ''\) AS isbn, titel AS title, COALESCE\(cover_url, ''\) AS cover_url FROM buecher_titel WHERE cover_url LIKE 'http%' ORDER BY id ASC LIMIT \$1`).
			WithArgs(10).
			WillReturnRows(pgxmock.NewRows([]string{"id", "isbn", "title"}). // Missing column to trigger scan error
												AddRow("book-1", "123", "Title 1"))

		eintraege, err := abfragen.ladeTitelMitFremdemCover(ctx, 10)
		if err == nil || !strings.Contains(err.Error(), "bücher mit externen covern konnten nicht gelesen werden") {
			t.Errorf("abfragen.LadeTitelMitFremdemCover: Fehler %v, erwartet einen mit %q", err, "bücher mit externen covern konnten nicht gelesen werden")
		}
		if eintraege != nil {
			t.Errorf("eintraege = %v, erwartet nil", eintraege)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("offene Erwartungen: %v", err)
		}
	})

	t.Run("iteration error", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		if err != nil {
			t.Fatalf("pgxmock.NewPool: %v", err)
		}
		defer mock.Close()
		abfragen := NewTitelAbfragen(mock)

		mock.ExpectQuery(`SELECT id, COALESCE\(isbn, ''\) AS isbn, titel AS title, COALESCE\(cover_url, ''\) AS cover_url FROM buecher_titel WHERE cover_url LIKE 'http%' ORDER BY id ASC LIMIT \$1`).
			WithArgs(10).
			WillReturnRows(pgxmock.NewRows([]string{"id", "isbn", "title", "cover_url"}).
				AddRow("book-1", "123", "Title 1", "http://example.com/1.jpg").
				RowError(1, fmt.Errorf("row error")))

		eintraege, err := abfragen.ladeTitelMitFremdemCover(ctx, 10)
		if err == nil || !strings.Contains(err.Error(), "fehler beim iterieren externer cover-bücher") {
			t.Errorf("abfragen.LadeTitelMitFremdemCover: Fehler %v, erwartet einen mit %q", err, "fehler beim iterieren externer cover-bücher")
		}
		if eintraege != nil {
			t.Errorf("eintraege = %v, erwartet nil", eintraege)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("offene Erwartungen: %v", err)
		}
	})
}

func TestTitelAbfragen_LadeTitelNachIDs(t *testing.T) {
	ctx := context.Background()
	zuletztGezaehlt := "2023-01-01"

	t.Run("empty ids", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		if err != nil {
			t.Fatalf("pgxmock.NewPool: %v", err)
		}
		defer mock.Close()
		abfragen := NewTitelAbfragen(mock)

		eintraege, err := abfragen.LadeTitelNachIDs(ctx, []string{})
		if err != nil {
			t.Fatalf("abfragen.LadeTitelNachIDs: %v", err)
		}
		if len(eintraege) != 0 {
			t.Errorf("eintraege = %v, erwartet leer", eintraege)
		}
		// No query expected
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("offene Erwartungen: %v", err)
		}
	})

	t.Run("success", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		if err != nil {
			t.Fatalf("pgxmock.NewPool: %v", err)
		}
		defer mock.Close()
		abfragen := NewTitelAbfragen(mock)

		ids := []string{"book-1"}
		mock.ExpectQuery(`SELECT.+FROM buecher_titel bt LEFT JOIN buecher_exemplare e.+`).
			WithArgs(ids).
			WillReturnRows(pgxmock.NewRows([]string{
				"id", "isbn", "title", "author", "signatur", "cover_url", "subject", "track", "ist_lernmittel",
				"verfuegbar", "gesamt", "im_zulauf", "last_counted", "sort_order", "medientyp", "jahrgang_von", "jahrgang_bis",
				"untertitel", "verlag", "erscheinungsjahr", "erweiterte_eigenschaften", "auflage",
				"listenpreis", "mehrjahresband", "werk_id", "werk_rang",
			}).AddRow(
				"book-1", "123", "Algebra", "Smith", "SIG-1", "url", "Math", "A", false,
				2, 3, 0, &zuletztGezaehlt, 1, "Buch", 5, 6,
				"", "", 2020, map[string]any{}, "4. Aufl. 2023", nil, false, "", 0,
			))

		eintraege, err := abfragen.LadeTitelNachIDs(ctx, ids)
		if err != nil {
			t.Fatalf("abfragen.LadeTitelNachIDs: %v", err)
		}
		if len(eintraege) != 1 {
			t.Errorf("len(eintraege) = %d, erwartet %d", len(eintraege), 1)
		}
		if len(eintraege) > 0 {
			if eintraege[0].ID != "book-1" {
				t.Errorf("eintraege[0].ID = %q, erwartet %q", eintraege[0].ID, "book-1")
			}
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("offene Erwartungen: %v", err)
		}
	})

	t.Run("db error", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		if err != nil {
			t.Fatalf("pgxmock.NewPool: %v", err)
		}
		defer mock.Close()
		abfragen := NewTitelAbfragen(mock)

		ids := []string{"book-1"}
		mock.ExpectQuery(`SELECT.+FROM buecher_titel bt LEFT JOIN buecher_exemplare e.+`).
			WithArgs(ids).
			WillReturnError(fmt.Errorf("db connection failed"))

		eintraege, err := abfragen.LadeTitelNachIDs(ctx, ids)
		if err == nil || !strings.Contains(err.Error(), "bücher nach ids konnten nicht geladen werden") {
			t.Errorf("abfragen.LadeTitelNachIDs: Fehler %v, erwartet einen mit %q", err, "bücher nach ids konnten nicht geladen werden")
		}
		if eintraege != nil {
			t.Errorf("eintraege = %v, erwartet nil", eintraege)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("offene Erwartungen: %v", err)
		}
	})
}

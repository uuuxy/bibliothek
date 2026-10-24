package inventur

import (
	"context"
	"errors"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/pashagolub/pgxmock/v5"
)

func TestLadeTitelFuerCSVExport(t *testing.T) {
	t.Run("successful stream", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		if err != nil {
			t.Fatalf("pgxmock.NewPool: %v", err)
		}
		defer mock.Close()

		abfragen := NewTitelAbfragen(mock)
		ctx := context.Background()

		zeilen := pgxmock.NewRows(exportSpaltenAbfrage).
			AddRow("Buch 1", "Autor 1", "Verlag 1", "1234567890", 2021, "Kategorie 1", "BC1", "Gut", "Ma 8", "Algebra | Geometrie", "land", "Regal 3B", "LMF-9783060158423-001").
			AddRow("Buch 2", "", "", "", 0, "", "", "", "", "", "", "", "")

		mock.ExpectQuery("SELECT.+FROM buecher_titel.+LEFT JOIN buecher_exemplare").
			WithArgs(exportSchlagwortTrenner).
			WillReturnRows(zeilen)

		kopfGerufen := false
		kopf := func() error {
			kopfGerufen = true
			return nil
		}

		var ergebnisZeilen [][]string
		schreibe := func(zeile []string) error {
			ergebnisZeilen = append(ergebnisZeilen, zeile)
			return nil
		}

		err = abfragen.ladeTitelFuerCSVExport(ctx, kopf, schreibe)
		if err != nil {
			t.Fatalf("abfragen.LadeTitelFuerCSVExport: %v", err)
		}
		if !kopfGerufen {
			t.Error("kopf nicht gerufen")
		}
		if len(ergebnisZeilen) != 2 {
			t.Fatalf("len(ergebnisZeilen) = %d, erwartet %d", len(ergebnisZeilen), 2)
		}
		if diff := cmp.Diff([]string{"Buch 1", "Autor 1", "Verlag 1", "'1234567890", "2021", "Kategorie 1", "BC1", "Gut", "Ma 8", "Algebra | Geometrie", "Land", "Regal 3B", "LMF-9783060158423-001"}, ergebnisZeilen[0]); diff != "" {
			t.Errorf("ergebnisZeilen[0] weicht ab (-erwartet +ist):\n%s", diff)
		}
		if diff := cmp.Diff([]string{"Buch 2", "", "", "", "", "", "", "", "", "", "", "", ""}, ergebnisZeilen[1]); diff != "" {
			t.Errorf("ergebnisZeilen[1] weicht ab (-erwartet +ist):\n%s", diff)
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("offene Erwartungen: %v", err)
		}
	})

	t.Run("database error on query", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		if err != nil {
			t.Fatalf("pgxmock.NewPool: %v", err)
		}
		defer mock.Close()

		abfragen := NewTitelAbfragen(mock)
		ctx := context.Background()

		mock.ExpectQuery("SELECT.+FROM buecher_titel.+LEFT JOIN buecher_exemplare").
			WithArgs(exportSchlagwortTrenner).
			WillReturnError(errors.New("db error"))

		kopfGerufen := false
		kopf := func() error {
			kopfGerufen = true
			return nil
		}
		schreibe := func(zeile []string) error { return nil }

		err = abfragen.ladeTitelFuerCSVExport(ctx, kopf, schreibe)
		if err == nil {
			t.Fatal("abfragen.LadeTitelFuerCSVExport: kein Fehler, erwartet einen")
		}
		if got := err.Error(); got != "db error" {
			t.Errorf("err.Error() = %q, erwartet %q", got, "db error")
		}
		if kopfGerufen {
			t.Error("kopf gerufen, obwohl die Abfrage scheiterte")
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("offene Erwartungen: %v", err)
		}
	})

	t.Run("error in kopf", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		if err != nil {
			t.Fatalf("pgxmock.NewPool: %v", err)
		}
		defer mock.Close()

		abfragen := NewTitelAbfragen(mock)
		ctx := context.Background()

		zeilen := pgxmock.NewRows(exportSpaltenAbfrage)
		mock.ExpectQuery("SELECT.+FROM buecher_titel.+LEFT JOIN buecher_exemplare").
			WithArgs(exportSchlagwortTrenner).
			WillReturnRows(zeilen)

		kopf := func() error {
			return errors.New("kopf error")
		}
		schreibe := func(zeile []string) error { return nil }

		err = abfragen.ladeTitelFuerCSVExport(ctx, kopf, schreibe)
		if err == nil {
			t.Fatal("abfragen.LadeTitelFuerCSVExport: kein Fehler, erwartet einen")
		}
		if got := err.Error(); got != "kopf error" {
			t.Errorf("err.Error() = %q, erwartet %q", got, "kopf error")
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("offene Erwartungen: %v", err)
		}
	})

	t.Run("error in schreibe breaks stream", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		if err != nil {
			t.Fatalf("pgxmock.NewPool: %v", err)
		}
		defer mock.Close()

		abfragen := NewTitelAbfragen(mock)
		ctx := context.Background()

		zeilen := pgxmock.NewRows(exportSpaltenAbfrage).
			AddRow("Buch 1", "Autor 1", "Verlag 1", "1234567890", 2021, "Kategorie 1", "BC1", "Gut", "", "", "schultraeger", "", "")

		mock.ExpectQuery("SELECT.+FROM buecher_titel.+LEFT JOIN buecher_exemplare").
			WithArgs(exportSchlagwortTrenner).
			WillReturnRows(zeilen)

		kopf := func() error { return nil }
		schreibe := func(zeile []string) error {
			return errors.New("schreibe error")
		}

		err = abfragen.ladeTitelFuerCSVExport(ctx, kopf, schreibe)
		if err == nil {
			t.Fatal("abfragen.LadeTitelFuerCSVExport: kein Fehler, erwartet einen")
		}
		if got := err.Error(); got != "schreibe error" {
			t.Errorf("err.Error() = %q, erwartet %q", got, "schreibe error")
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("offene Erwartungen: %v", err)
		}
	})
}

// exportSpaltenAbfrage sind die Spalten der Export-Abfrage in ihrer Reihenfolge.
var exportSpaltenAbfrage = []string{"titel", "autor", "verlag", "isbn", "jahr", "subject", "barcode", "zustand", "signatur", "schlagworte", "topf", "standort", "lmf_barcode"}

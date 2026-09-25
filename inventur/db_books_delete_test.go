package inventur

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/pashagolub/pgxmock/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Die Reihenfolge ist hier der Prüfgegenstand — pgxmock verlangt sie genau so, wie sie
// erwartet wird: Begin ZUERST, dann jeder Leser in der Transaktion, dann die beiden
// Löschbefehle, die ihre Spur per RETURNING liefern. Bis zum 21.09.2026 standen die
// Leser vor dem Begin (OFFEN.md 5.5): Ein Rückbau dorthin lässt diesen Test rot werden,
// weil dann eine Abfrage VOR dem erwarteten Begin ankommt.
func TestDeleteBooks(t *testing.T) {
	ctx := context.Background()
	ids := []string{"id-1", "id-2"}

	// Die Leser in der Transaktion, in ihrer Reihenfolge — den Erfolgs- und den
	// Nicht-gefunden-Fall unterscheidet erst der Titel-DELETE danach.
	erwarteLeser := func(mock pgxmock.PgxPoolIface, cover string) {
		mock.ExpectBegin()
		erwarteAuflagenSperre(mock, ids)
		// Barcode-Snapshots ALLER Exemplare vor den DELETEs — die Tresen-Auskunft
		// findet gelöschte Exemplare nur über diese Spur (Befund 01.09.2026).
		mock.ExpectQuery(`FROM buecher_exemplare e`).
			WithArgs(ids).
			WillReturnRows(pgxmock.NewRows([]string{"id", "barcode_id", "titel"}).
				AddRow("ex-1", "BC-1", "Titel Eins"))
		// Die drei CASCADE-Kinder des Titels, die niemand nannte, bis Frage 12
		// („Gegenrichtung Schema", 06.09.2026) die DDL gelesen hat: Vormerkungen,
		// Klassensatz-Reservierungen, Klassensatz-Zuordnungen.
		mock.ExpectQuery(`FROM vormerkungen v`).
			WithArgs(ids).
			WillReturnRows(pgxmock.NewRows([]string{"id", "titel_id", "titel", "wer", "status", "seit", "schueler_id"}))
		mock.ExpectQuery(`FROM klassensatz_reservierungen r`).
			WithArgs(ids).
			WillReturnRows(pgxmock.NewRows([]string{"id", "titel_id", "titel", "klasse", "status", "seit"}))
		mock.ExpectQuery(`FROM class_books c`).
			WithArgs(ids).
			WillReturnRows(pgxmock.NewRows([]string{"id", "titel_id", "titel", "klasse"}))
		zeilen := pgxmock.NewRows([]string{"cover_url"})
		if cover != "" {
			zeilen.AddRow(cover)
		}
		mock.ExpectQuery(`SELECT cover_url FROM buecher_titel WHERE id = ANY\(\$1::uuid\[\]\) AND cover_url LIKE '/uploads/%'`).
			WithArgs(ids).
			WillReturnRows(zeilen)
	}
	schadenSpalten := []string{"id", "exemplar_id", "barcode_id", "titel", "schuldner", "schueler_id", "betrag", "beschreibung", "seit"}
	ausleiheSpalten := []string{"id", "exemplar_id", "barcode_id", "titel", "entleiher", "schueler_id", "seit"}

	t.Run("empty ids", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()
		repo := NewBookRepository(mock)

		err = repo.DeleteBooks(ctx, []string{})
		assert.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	// Scheitert das Löschen der Ausleihen, bleibt NICHTS stehen: kein Titel-DELETE,
	// keine Spur, Rollback. Ohne die Spur verschwände ein verliehenes Buch lautlos.
	t.Run("Ausleihen nicht löschbar", func(t *testing.T) {
		imTestVerzeichnis(t) // DeleteBooks legt sonst inventur/uploads/ im Repo an
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()
		repo := NewBookRepository(mock)

		erwarteLeser(mock, "")
		mock.ExpectQuery(`DELETE FROM schadensfaelle sf`).
			WithArgs(ids).
			WillReturnRows(pgxmock.NewRows(schadenSpalten))
		mock.ExpectQuery(`DELETE FROM ausleihen a`).
			WithArgs(ids).
			WillReturnError(fmt.Errorf("db error"))
		mock.ExpectRollback()

		err = repo.DeleteBooks(ctx, ids)
		assert.ErrorContains(t, err, "ausleihen konnten nicht gelöscht werden")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("success", func(t *testing.T) {
		imTestVerzeichnis(t) // DeleteBooks legt sonst inventur/uploads/ im Repo an
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()
		repo := NewBookRepository(mock)

		erwarteLeser(mock, "/uploads/cover1.jpg")
		// Offene Forderungen und laufende Ausleihen kommen als Spur aus dem Löschbefehl
		// selbst (RETURNING) — eine unbezahlte Forderung ist Geld, das ein Schüler
		// schuldet, und sie verschwand bis zum 06.09.2026 spurlos mit dem Titel.
		mock.ExpectQuery(`DELETE FROM schadensfaelle sf`).
			WithArgs(ids).
			WillReturnRows(pgxmock.NewRows(schadenSpalten))
		mock.ExpectQuery(`DELETE FROM ausleihen a`).
			WithArgs(ids).
			WillReturnRows(pgxmock.NewRows(ausleiheSpalten))
		mock.ExpectExec(`DELETE FROM buecher_titel WHERE id = ANY\(\$1::uuid\[\]\)`).
			WithArgs(ids).
			WillReturnResult(pgxmock.NewResult("DELETE", 2))
		mock.ExpectExec(`INSERT INTO audit_log`).
			WithArgs("ex-1", pgxmock.AnyArg(), pgxmock.AnyArg()).
			WillReturnResult(pgxmock.NewResult("INSERT", 1))
		mock.ExpectCommit()

		err = repo.DeleteBooks(ctx, ids)
		assert.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("book not found", func(t *testing.T) {
		imTestVerzeichnis(t) // DeleteBooks legt sonst inventur/uploads/ im Repo an
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()
		repo := NewBookRepository(mock)

		erwarteLeser(mock, "/uploads/cover1.jpg")
		mock.ExpectQuery(`DELETE FROM schadensfaelle sf`).
			WithArgs(ids).
			WillReturnRows(pgxmock.NewRows(schadenSpalten))
		mock.ExpectQuery(`DELETE FROM ausleihen a`).
			WithArgs(ids).
			WillReturnRows(pgxmock.NewRows(ausleiheSpalten))
		mock.ExpectExec(`DELETE FROM buecher_titel`).
			WithArgs(ids).
			WillReturnResult(pgxmock.NewResult("DELETE", 0))
		mock.ExpectRollback()

		err = repo.DeleteBooks(ctx, ids)
		assert.ErrorIs(t, err, ErrBookNotFound)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestSammleLokaleCoverPfade(t *testing.T) {
	ctx := context.Background()
	ids := []string{"id-1"}

	t.Run("success", func(t *testing.T) {
		imTestVerzeichnis(t) // DeleteBooks legt sonst inventur/uploads/ im Repo an
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		mock.ExpectQuery(`SELECT cover_url FROM buecher_titel`).
			WithArgs(ids).
			WillReturnRows(pgxmock.NewRows([]string{"cover_url"}).AddRow("/uploads/test1.jpg").AddRow("/uploads/test2.png"))

		paths, err := sammleLokaleCoverPfade(ctx, mock, ids)
		assert.NoError(t, err)
		assert.Equal(t, []string{"/uploads/test1.jpg", "/uploads/test2.png"}, paths)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("db error", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		mock.ExpectQuery(`SELECT cover_url FROM buecher_titel`).
			WithArgs(ids).
			WillReturnError(fmt.Errorf("db error"))

		paths, err := sammleLokaleCoverPfade(ctx, mock, ids)
		assert.ErrorContains(t, err, "cover-dateien konnten nicht ermittelt werden")
		assert.Nil(t, paths)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestLoescheLokaleCoverDateien(t *testing.T) {
	imTestVerzeichnis(t)
	err := os.MkdirAll("uploads", 0750)
	require.NoError(t, err)

	testFile := filepath.Join("uploads", "test_cover.jpg")
	err = os.WriteFile(testFile, []byte("data"), 0600)
	require.NoError(t, err)

	outsideFile := "outside.jpg"
	err = os.WriteFile(outsideFile, []byte("data"), 0600)
	require.NoError(t, err)

	loescheLokaleCoverDateien([]string{
		"/uploads/test_cover.jpg",
		"http://example.com/cover.jpg",
		"../outside.jpg",
		"/uploads/.",
		"/uploads//",
	})

	_, err = os.Stat(testFile)
	assert.True(t, os.IsNotExist(err), "testFile should have been deleted")

	_, err = os.Stat(outsideFile)
	assert.NoError(t, err, "outside file should NOT have been deleted")
}

// erwarteAuflagenSperre: DeleteBooks nimmt als ERSTES die Sperre der Auflagen und liest die
// Werke der Titel (repository.WerkeDerTitel, docs/OFFEN.md 4.18) — hier gehört keiner zu
// einem, also räumt repository.RaeumeWerkeAuf danach nichts auf.
func erwarteAuflagenSperre(mock pgxmock.PgxPoolIface, ids any) {
	mock.ExpectExec(`SELECT pg_advisory_xact_lock`).WithArgs(pgxmock.AnyArg()).
		WillReturnResult(pgxmock.NewResult("SELECT", 1))
	mock.ExpectQuery(`SELECT DISTINCT werk_id::text FROM buecher_titel`).WithArgs(ids).
		WillReturnRows(pgxmock.NewRows([]string{"werk_id"}))
}

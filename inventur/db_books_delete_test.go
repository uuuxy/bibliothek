package inventur

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/pashagolub/pgxmock/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDeleteBooks(t *testing.T) {
	ctx := context.Background()
	ids := []string{"id-1", "id-2"}

	t.Run("empty ids", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()
		repo := NewBookRepository(mock)

		err = repo.DeleteBooks(ctx, []string{})
		assert.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("active loans exist", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()
		repo := NewBookRepository(mock)

		mock.ExpectQuery(`SELECT EXISTS \( SELECT 1 FROM ausleihen a JOIN buecher_exemplare e ON a.exemplar_id = e.id WHERE e.titel_id = ANY\(\$1::uuid\[\]\) AND a.rueckgabe_am IS NULL \)`).
			WithArgs(ids).
			WillReturnRows(pgxmock.NewRows([]string{"exists"}).AddRow(true))

		err = repo.DeleteBooks(ctx, ids)
		assert.ErrorContains(t, err, "löschen abgebrochen")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("success", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()
		repo := NewBookRepository(mock)

		mock.ExpectQuery(`SELECT EXISTS`).
			WithArgs(ids).
			WillReturnRows(pgxmock.NewRows([]string{"exists"}).AddRow(false))

		mock.ExpectQuery(`SELECT cover_url FROM buecher_titel WHERE id = ANY\(\$1::uuid\[\]\) AND cover_url LIKE '/uploads/%'`).
			WithArgs(ids).
			WillReturnRows(pgxmock.NewRows([]string{"cover_url"}).AddRow("/uploads/cover1.jpg"))

		mock.ExpectBegin()
		mock.ExpectExec(`DELETE FROM schadensfaelle WHERE exemplar_id IN \(SELECT id FROM buecher_exemplare WHERE titel_id = ANY\(\$1::uuid\[\]\)\)`).
			WithArgs(ids).
			WillReturnResult(pgxmock.NewResult("DELETE", 1))
		mock.ExpectExec(`DELETE FROM ausleihen WHERE exemplar_id IN \(SELECT id FROM buecher_exemplare WHERE titel_id = ANY\(\$1::uuid\[\]\)\) AND rueckgabe_am IS NOT NULL`).
			WithArgs(ids).
			WillReturnResult(pgxmock.NewResult("DELETE", 1))
		mock.ExpectExec(`DELETE FROM buecher_titel WHERE id = ANY\(\$1::uuid\[\]\)`).
			WithArgs(ids).
			WillReturnResult(pgxmock.NewResult("DELETE", 2))
		mock.ExpectCommit()

		err = repo.DeleteBooks(ctx, ids)
		assert.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("book not found", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()
		repo := NewBookRepository(mock)

		mock.ExpectQuery(`SELECT EXISTS`).
			WithArgs(ids).
			WillReturnRows(pgxmock.NewRows([]string{"exists"}).AddRow(false))

		mock.ExpectQuery(`SELECT cover_url FROM buecher_titel`).
			WithArgs(ids).
			WillReturnRows(pgxmock.NewRows([]string{"cover_url"}).AddRow("/uploads/cover1.jpg"))

		mock.ExpectBegin()
		mock.ExpectExec(`DELETE FROM schadensfaelle`).
			WithArgs(ids).
			WillReturnResult(pgxmock.NewResult("DELETE", 0))
		mock.ExpectExec(`DELETE FROM ausleihen`).
			WithArgs(ids).
			WillReturnResult(pgxmock.NewResult("DELETE", 0))
		mock.ExpectExec(`DELETE FROM buecher_titel`).
			WithArgs(ids).
			WillReturnResult(pgxmock.NewResult("DELETE", 0))
		mock.ExpectRollback()

		err = repo.DeleteBooks(ctx, ids)
		assert.ErrorIs(t, err, ErrBookNotFound)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestPruefeKeineAktivenAusleihen(t *testing.T) {
	ctx := context.Background()
	ids := []string{"id-1"}

	t.Run("has active loans", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()
		repo := NewBookRepository(mock)

		mock.ExpectQuery(`SELECT EXISTS`).
			WithArgs(ids).
			WillReturnRows(pgxmock.NewRows([]string{"exists"}).AddRow(true))

		err = repo.pruefeKeineAktivenAusleihen(ctx, ids)
		assert.ErrorContains(t, err, "Mindestens ein Exemplar")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("no active loans", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()
		repo := NewBookRepository(mock)

		mock.ExpectQuery(`SELECT EXISTS`).
			WithArgs(ids).
			WillReturnRows(pgxmock.NewRows([]string{"exists"}).AddRow(false))

		err = repo.pruefeKeineAktivenAusleihen(ctx, ids)
		assert.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("db error", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()
		repo := NewBookRepository(mock)

		mock.ExpectQuery(`SELECT EXISTS`).
			WithArgs(ids).
			WillReturnError(fmt.Errorf("db error"))

		err = repo.pruefeKeineAktivenAusleihen(ctx, ids)
		assert.ErrorContains(t, err, "fehler bei der prüfung auf aktive ausleihen")
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestSammleLokaleCoverPfade(t *testing.T) {
	ctx := context.Background()
	ids := []string{"id-1"}

	t.Run("success", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()
		repo := NewBookRepository(mock)

		mock.ExpectQuery(`SELECT cover_url FROM buecher_titel`).
			WithArgs(ids).
			WillReturnRows(pgxmock.NewRows([]string{"cover_url"}).AddRow("/uploads/test1.jpg").AddRow("/uploads/test2.png"))

		paths, err := repo.sammleLokaleCoverPfade(ctx, ids)
		assert.NoError(t, err)
		assert.Equal(t, []string{"/uploads/test1.jpg", "/uploads/test2.png"}, paths)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("db error", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()
		repo := NewBookRepository(mock)

		mock.ExpectQuery(`SELECT cover_url FROM buecher_titel`).
			WithArgs(ids).
			WillReturnError(fmt.Errorf("db error"))

		paths, err := repo.sammleLokaleCoverPfade(ctx, ids)
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

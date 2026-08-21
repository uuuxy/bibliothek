package main

import (
	"os"
	"testing"

	"github.com/pashagolub/pgxmock/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMigriereAlleFotos_Empty(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	tempDir := t.TempDir()
	root, err := os.OpenRoot(tempDir)
	require.NoError(t, err)
	defer root.Close()

	processed, migrated := migriereAlleFotos(mock, root, []os.DirEntry{})
	assert.Equal(t, 0, processed)
	assert.Equal(t, 0, migrated)
}

func TestMigriereAlleFotos_MitDateien(t *testing.T) {
    os.Setenv("APP_ENCRYPTION_KEY", "01234567890123456789012345678901")
    defer os.Unsetenv("APP_ENCRYPTION_KEY")

	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	tempDir := t.TempDir()

	// Create test files
	file1, err := os.Create(tempDir + "/12345.jpg")
	require.NoError(t, err)
	file1.WriteString("fake image data")
	file1.Close()

	file2, err := os.Create(tempDir + "/67890.jpg")
	require.NoError(t, err)
	file2.WriteString("fake image data 2")
	file2.Close()

	// not a jpg
	file3, err := os.Create(tempDir + "/notajpg.txt")
	require.NoError(t, err)
	file3.WriteString("text")
	file3.Close()

	root, err := os.OpenRoot(tempDir)
	require.NoError(t, err)
	defer root.Close()

	dir, err := root.Open(".")
	require.NoError(t, err)
	entries, err := dir.ReadDir(-1)
	require.NoError(t, err)
	dir.Close()

	// Mock student IDs loading
	mock.ExpectQuery("SELECT barcode_id, id FROM schueler WHERE barcode_id = ANY").
		WithArgs(pgxmock.AnyArg()).
		WillReturnRows(pgxmock.NewRows([]string{"barcode_id", "id"}).
			AddRow("12345", "uuid-12345").
			AddRow("67890", "uuid-67890"))

    mock.ExpectExec("INSERT INTO schueler_fotos").
        WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg()).
        WillReturnResult(pgxmock.NewResult("INSERT", 1))

    mock.ExpectExec("INSERT INTO schueler_fotos").
        WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg()).
        WillReturnResult(pgxmock.NewResult("INSERT", 1))

	processed, migrated := migriereAlleFotos(mock, root, entries)
	assert.Equal(t, 2, processed)
	assert.Equal(t, 2, migrated)

	require.NoError(t, mock.ExpectationsWereMet())
}

func TestMigriereAlleFotos_KeinSchueler(t *testing.T) {
    os.Setenv("APP_ENCRYPTION_KEY", "01234567890123456789012345678901")
    defer os.Unsetenv("APP_ENCRYPTION_KEY")

	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	tempDir := t.TempDir()

	file1, err := os.Create(tempDir + "/99999.jpg")
	require.NoError(t, err)
	file1.WriteString("fake image data")
	file1.Close()

	root, err := os.OpenRoot(tempDir)
	require.NoError(t, err)
	defer root.Close()

	dir, err := root.Open(".")
	require.NoError(t, err)
	entries, err := dir.ReadDir(-1)
	require.NoError(t, err)
	dir.Close()

	// Mock student IDs loading returning NO ROWS
	mock.ExpectQuery("SELECT barcode_id, id FROM schueler WHERE barcode_id = ANY").
		WithArgs(pgxmock.AnyArg()).
		WillReturnRows(pgxmock.NewRows([]string{"barcode_id", "id"}))

	processed, migrated := migriereAlleFotos(mock, root, entries)
	assert.Equal(t, 1, processed)
	assert.Equal(t, 0, migrated)

	require.NoError(t, mock.ExpectationsWereMet())
}

func TestMigriereAlleFotos_DBFehlerLaden(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	tempDir := t.TempDir()

	file1, err := os.Create(tempDir + "/12345.jpg")
	require.NoError(t, err)
	file1.WriteString("fake")
	file1.Close()

	root, err := os.OpenRoot(tempDir)
	require.NoError(t, err)
	defer root.Close()

	dir, err := root.Open(".")
	require.NoError(t, err)
	entries, err := dir.ReadDir(-1)
	require.NoError(t, err)
	dir.Close()

	mock.ExpectQuery("SELECT barcode_id, id FROM schueler WHERE barcode_id = ANY").
		WithArgs(pgxmock.AnyArg()).
		WillReturnError(os.ErrPermission)

	processed, migrated := migriereAlleFotos(mock, root, entries)
	assert.Equal(t, 0, processed)
	assert.Equal(t, 0, migrated)

	require.NoError(t, mock.ExpectationsWereMet())
}

func TestMigriereFoto_KeinLesen(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	tempDir := t.TempDir()

	root, err := os.OpenRoot(tempDir)
	require.NoError(t, err)
	defer root.Close()

	ok := migriereFoto(mock, root, "nicht-da.jpg", "123", "uuid")
	assert.False(t, ok)
}

func TestMigriereFoto_VerschluesselungFehlgeschlagen(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	tempDir := t.TempDir()
	file1, err := os.Create(tempDir + "/123.jpg")
	require.NoError(t, err)
	file1.WriteString("fake data")
	file1.Close()

	root, err := os.OpenRoot(tempDir)
	require.NoError(t, err)
	defer root.Close()

    // No encryption key set, so it should fail
	ok := migriereFoto(mock, root, "123.jpg", "123", "uuid")
	assert.False(t, ok)
}

func TestMigriereFoto_DBFehler(t *testing.T) {
    os.Setenv("APP_ENCRYPTION_KEY", "01234567890123456789012345678901")
    defer os.Unsetenv("APP_ENCRYPTION_KEY")

	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	tempDir := t.TempDir()
	file1, err := os.Create(tempDir + "/123.jpg")
	require.NoError(t, err)
	file1.WriteString("fake data")
	file1.Close()

	root, err := os.OpenRoot(tempDir)
	require.NoError(t, err)
	defer root.Close()

    mock.ExpectExec("INSERT INTO schueler_fotos").
        WithArgs("uuid", pgxmock.AnyArg()).
        WillReturnError(os.ErrPermission)

	ok := migriereFoto(mock, root, "123.jpg", "123", "uuid")
	assert.False(t, ok)
    require.NoError(t, mock.ExpectationsWereMet())
}

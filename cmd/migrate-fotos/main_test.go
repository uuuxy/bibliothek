package main

import (
	"os"
	"testing"

	"bibliothek/internal/crypto"

	"github.com/pashagolub/pgxmock/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// erwarteGegenprobe spielt das Zurücklesen nach: Das Werkzeug löscht die Quelldatei
// erst, wenn das gespeicherte Foto wieder entschlüsselt und mit dem Original verglichen
// werden konnte. Der Mock liefert deshalb echten Chiffretext desselben Inhalts.
func erwarteGegenprobe(t *testing.T, mock pgxmock.PgxPoolIface, inhalt string) {
	t.Helper()
	chiffre, err := crypto.Encrypt([]byte(inhalt))
	require.NoError(t, err)
	mock.ExpectQuery("SELECT foto_encrypted FROM schueler_fotos").
		WithArgs(pgxmock.AnyArg()).
		WillReturnRows(pgxmock.NewRows([]string{"foto_encrypted"}).AddRow(chiffre))
}

func TestMigriereAlleFotos_Empty(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	tempDir := t.TempDir()
	root, err := os.OpenRoot(tempDir)
	require.NoError(t, err)
	defer root.Close() //nolint:errcheck

	processed, migrated, _ := migriereAlleFotos(mock, root, []os.DirEntry{}, true)
	assert.Equal(t, 0, processed)
	assert.Equal(t, 0, migrated)
}

func TestMigriereAlleFotos_MitDateien(t *testing.T) {
	t.Setenv("APP_ENCRYPTION_KEY", "01234567890123456789012345678901")

	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	tempDir := t.TempDir()

	// Create test files
	file1, err := os.Create(tempDir + "/12345.jpg")
	require.NoError(t, err)
	_, err = file1.WriteString("fake image data")
	require.NoError(t, err)
	require.NoError(t, file1.Close())

	file2, err := os.Create(tempDir + "/67890.jpg")
	require.NoError(t, err)
	_, err = file2.WriteString("fake image data 2")
	require.NoError(t, err)
	require.NoError(t, file2.Close())

	// not a jpg
	file3, err := os.Create(tempDir + "/notajpg.txt")
	require.NoError(t, err)
	_, err = file3.WriteString("text")
	require.NoError(t, err)
	require.NoError(t, file3.Close())

	root, err := os.OpenRoot(tempDir)
	require.NoError(t, err)
	defer root.Close() //nolint:errcheck

	dir, err := root.Open(".")
	require.NoError(t, err)
	entries, err := dir.ReadDir(-1)
	require.NoError(t, err)
	require.NoError(t, dir.Close())

	// Mock student IDs loading
	mock.ExpectQuery("SELECT barcode_id, id FROM schueler WHERE barcode_id = ANY").
		WithArgs(pgxmock.AnyArg()).
		WillReturnRows(pgxmock.NewRows([]string{"barcode_id", "id"}).
			AddRow("12345", "uuid-12345").
			AddRow("67890", "uuid-67890"))

	mock.ExpectExec("INSERT INTO schueler_fotos").
		WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg()).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))
	erwarteGegenprobe(t, mock, "fake image data")

	mock.ExpectExec("INSERT INTO schueler_fotos").
		WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg()).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))
	erwarteGegenprobe(t, mock, "fake image data 2")

	processed, migrated, _ := migriereAlleFotos(mock, root, entries, true)
	assert.Equal(t, 2, processed)
	assert.Equal(t, 2, migrated)

	require.NoError(t, mock.ExpectationsWereMet())
}

func TestMigriereAlleFotos_KeinSchueler(t *testing.T) {
	t.Setenv("APP_ENCRYPTION_KEY", "01234567890123456789012345678901")

	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	tempDir := t.TempDir()

	file1, err := os.Create(tempDir + "/99999.jpg")
	require.NoError(t, err)
	_, err = file1.WriteString("fake image data")
	require.NoError(t, err)
	require.NoError(t, file1.Close())

	root, err := os.OpenRoot(tempDir)
	require.NoError(t, err)
	defer root.Close() //nolint:errcheck

	dir, err := root.Open(".")
	require.NoError(t, err)
	entries, err := dir.ReadDir(-1)
	require.NoError(t, err)
	require.NoError(t, dir.Close())

	// Mock student IDs loading returning NO ROWS
	mock.ExpectQuery("SELECT barcode_id, id FROM schueler WHERE barcode_id = ANY").
		WithArgs(pgxmock.AnyArg()).
		WillReturnRows(pgxmock.NewRows([]string{"barcode_id", "id"}))

	processed, migrated, _ := migriereAlleFotos(mock, root, entries, true)
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
	_, err = file1.WriteString("fake")
	require.NoError(t, err)
	require.NoError(t, file1.Close())

	root, err := os.OpenRoot(tempDir)
	require.NoError(t, err)
	defer root.Close() //nolint:errcheck

	dir, err := root.Open(".")
	require.NoError(t, err)
	entries, err := dir.ReadDir(-1)
	require.NoError(t, err)
	require.NoError(t, dir.Close())

	mock.ExpectQuery("SELECT barcode_id, id FROM schueler WHERE barcode_id = ANY").
		WithArgs(pgxmock.AnyArg()).
		WillReturnError(os.ErrPermission)

	processed, migrated, _ := migriereAlleFotos(mock, root, entries, true)
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
	defer root.Close() //nolint:errcheck

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
	_, err = file1.WriteString("fake data")
	require.NoError(t, err)
	require.NoError(t, file1.Close())

	root, err := os.OpenRoot(tempDir)
	require.NoError(t, err)
	defer root.Close() //nolint:errcheck

	// Schlüssel AKTIV leeren — sonst hinge der Test an der Umgebung: Mit gesetztem
	// APP_ENCRYPTION_KEY verschlüsselte er erfolgreich, der Mock lehnte das unerwartete
	// Exec ab, und das false war nur ein Mock-Artefakt (Prüfung 22.08.2026). Würde die
	// Verschlüsselung trotzdem gelingen, liefe der INSERT durch und der Test würde rot.
	t.Setenv("APP_ENCRYPTION_KEY", "")
	mock.ExpectExec("INSERT INTO schueler_fotos").WillReturnResult(pgxmock.NewResult("INSERT", 1))
	ok := migriereFoto(mock, root, "123.jpg", "123", "uuid")
	assert.False(t, ok, "ohne Schlüssel darf keine Verschlüsselung gelingen")
}

func TestMigriereFoto_DBFehler(t *testing.T) {
	t.Setenv("APP_ENCRYPTION_KEY", "01234567890123456789012345678901")

	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	tempDir := t.TempDir()
	file1, err := os.Create(tempDir + "/123.jpg")
	require.NoError(t, err)
	_, err = file1.WriteString("fake data")
	require.NoError(t, err)
	require.NoError(t, file1.Close())

	root, err := os.OpenRoot(tempDir)
	require.NoError(t, err)
	defer root.Close() //nolint:errcheck

	mock.ExpectExec("INSERT INTO schueler_fotos").
		WithArgs("uuid", pgxmock.AnyArg()).
		WillReturnError(os.ErrPermission)

	ok := migriereFoto(mock, root, "123.jpg", "123", "uuid")
	assert.False(t, ok)
	require.NoError(t, mock.ExpectationsWereMet())
}

// TestMigriereAlleFotos_LoeschtQuelldateienNachGegenprobe ist das Gate zum Fund vom
// 23.08.2026 (Raster-Frage 9, Ausleitung): Das Werkzeug SAGTE nur "Du kannst das
// Verzeichnis jetzt sicher löschen".
//
// Was liegen blieb, sind unverschlüsselte Schülerfotos unter `/uploads/` — einem Pfad,
// der bewusst ohne Anmeldung lesbar ist —, und ihre Dateinamen sind die Barcode-IDs vom
// Schülerausweis, also vollständig aufzählbar. Ein Hinweis auf der Konsole ist für
// diesen Zustand die falsche Sicherung.
func TestMigriereAlleFotos_LoeschtQuelldateienNachGegenprobe(t *testing.T) {
	t.Setenv("APP_ENCRYPTION_KEY", "01234567890123456789012345678901")

	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	tempDir := t.TempDir()
	pfad := tempDir + "/12345.jpg"
	require.NoError(t, os.WriteFile(pfad, []byte("fake image data"), 0o600))

	root, err := os.OpenRoot(tempDir)
	require.NoError(t, err)
	defer root.Close() //nolint:errcheck

	dir, err := root.Open(".")
	require.NoError(t, err)
	entries, err := dir.ReadDir(-1)
	require.NoError(t, err)
	require.NoError(t, dir.Close())

	mock.ExpectQuery("SELECT barcode_id, id FROM schueler WHERE barcode_id = ANY").
		WithArgs(pgxmock.AnyArg()).
		WillReturnRows(pgxmock.NewRows([]string{"barcode_id", "id"}).AddRow("12345", "uuid-12345"))
	mock.ExpectExec("INSERT INTO schueler_fotos").
		WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg()).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))
	erwarteGegenprobe(t, mock, "fake image data")

	_, migriert, geloescht := migriereAlleFotos(mock, root, entries, false)

	assert.Equal(t, 1, migriert)
	assert.Equal(t, 1, geloescht)
	assert.NoFileExists(t, pfad, "das unverschlüsselte Foto liegt noch da")
	require.NoError(t, mock.ExpectationsWereMet())
}

// TestMigriereAlleFotos_MisslungeneGegenprobeBehaeltDieDatei: Bis zur bestandenen
// Gegenprobe ist die Datei die EINZIGE Kopie des Bildes. Ein "INSERT ohne Fehler" heißt
// noch nicht, dass sich das Foto je wieder anzeigen lässt — ein falsch abgeleiteter
// Schlüssel fällt erst beim Entschlüsseln auf, und dann wäre die Quelle weg.
func TestMigriereAlleFotos_MisslungeneGegenprobeBehaeltDieDatei(t *testing.T) {
	t.Setenv("APP_ENCRYPTION_KEY", "01234567890123456789012345678901")

	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	tempDir := t.TempDir()
	pfad := tempDir + "/12345.jpg"
	require.NoError(t, os.WriteFile(pfad, []byte("fake image data"), 0o600))

	root, err := os.OpenRoot(tempDir)
	require.NoError(t, err)
	defer root.Close() //nolint:errcheck

	dir, err := root.Open(".")
	require.NoError(t, err)
	entries, err := dir.ReadDir(-1)
	require.NoError(t, err)
	require.NoError(t, dir.Close())

	mock.ExpectQuery("SELECT barcode_id, id FROM schueler WHERE barcode_id = ANY").
		WithArgs(pgxmock.AnyArg()).
		WillReturnRows(pgxmock.NewRows([]string{"barcode_id", "id"}).AddRow("12345", "uuid-12345"))
	mock.ExpectExec("INSERT INTO schueler_fotos").
		WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg()).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))
	// Zurückgelesen wird ein ANDERES Bild — die Gegenprobe muss scheitern.
	erwarteGegenprobe(t, mock, "ein anderes bild")

	_, migriert, geloescht := migriereAlleFotos(mock, root, entries, false)

	assert.Equal(t, 0, migriert, "eine misslungene Gegenprobe darf nicht als Erfolg zählen")
	assert.Equal(t, 0, geloescht)
	assert.FileExists(t, pfad, "die einzige Kopie des Bildes wurde gelöscht")
}

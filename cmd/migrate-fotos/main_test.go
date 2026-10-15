package main

import (
	"errors"
	"os"
	"testing"

	"bibliothek/internal/crypto"

	"github.com/pashagolub/pgxmock/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// erwarteGegenprobe spielt das Zurücklesen nach: Das Werkzeug löscht die Quelldatei erst, wenn
// das gespeicherte Foto wieder entschlüsselt und mit dem Original verglichen werden konnte. Die
// Attrappe liefert deshalb echten verschlüsselten Text desselben Inhalts. Die Erwartung hängt am
// Schüler, nicht an der Reihenfolge der Aufrufe: ReadDir liefert die Dateien so, wie das
// Dateisystem sie hergibt (os.ReadDir sortiert, diese Methode nicht). Ein Test, der die
// Reihenfolge voraussetzt, ist lokal grün und in der CI rot.
func erwarteGegenprobe(t *testing.T, mock pgxmock.PgxPoolIface, schuelerID, inhalt string) {
	t.Helper()
	chiffre, err := crypto.Verschluessele([]byte(inhalt))
	require.NoError(t, err)
	mock.ExpectQuery("SELECT foto_encrypted FROM schueler_fotos").
		WithArgs(schuelerID).
		WillReturnRows(pgxmock.NewRows([]string{"foto_encrypted"}).AddRow(chiffre))
}

func TestMigriereAlleFotos_LeeresVerzeichnis(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	testVerzeichnis := t.TempDir()
	wurzel, err := os.OpenRoot(testVerzeichnis)
	require.NoError(t, err)
	defer wurzel.Close() //nolint:errcheck

	gefunden, uebernommen, _ := migriereAlleFotos(mock, wurzel, []os.DirEntry{}, true)
	assert.Equal(t, 0, gefunden)
	assert.Equal(t, 0, uebernommen)
}

func TestMigriereAlleFotos_MitDateien(t *testing.T) {
	t.Setenv("APP_ENCRYPTION_KEY", "01234567890123456789012345678901")

	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	testVerzeichnis := t.TempDir()

	// Testdateien anlegen.
	datei1, err := os.Create(testVerzeichnis + "/12345.jpg")
	require.NoError(t, err)
	_, err = datei1.WriteString("fake image data")
	require.NoError(t, err)
	require.NoError(t, datei1.Close())

	datei2, err := os.Create(testVerzeichnis + "/67890.jpg")
	require.NoError(t, err)
	_, err = datei2.WriteString("fake image data 2")
	require.NoError(t, err)
	require.NoError(t, datei2.Close())

	// kein JPEG
	datei3, err := os.Create(testVerzeichnis + "/notajpg.txt")
	require.NoError(t, err)
	_, err = datei3.WriteString("text")
	require.NoError(t, err)
	require.NoError(t, datei3.Close())

	wurzel, err := os.OpenRoot(testVerzeichnis)
	require.NoError(t, err)
	defer wurzel.Close() //nolint:errcheck

	ordner, err := wurzel.Open(".")
	require.NoError(t, err)
	eintraege, err := ordner.ReadDir(-1)
	require.NoError(t, err)
	require.NoError(t, ordner.Close())

	// Die Kennungen der Schüler laden. Ungeordnet prüfen: Die Erwartungen unterscheiden sich am
	// Schüler, nicht an ihrer Position. Sonst wäre der Test nur deshalb grün, weil das Werkzeug
	// gerade alphabetisch sortiert, und ein Test, der von einer Eigenschaft des Prüflings abhängt,
	// misst sie nicht mehr.
	mock.MatchExpectationsInOrder(false)

	mock.ExpectQuery("SELECT barcode_id, id FROM schueler WHERE barcode_id = ANY").
		WithArgs(pgxmock.AnyArg()).
		WillReturnRows(pgxmock.NewRows([]string{"barcode_id", "id"}).
			AddRow("12345", "uuid-12345").
			AddRow("67890", "uuid-67890"))

	mock.ExpectExec("INSERT INTO schueler_fotos").
		WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg()).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))
	erwarteGegenprobe(t, mock, "uuid-12345", "fake image data")

	mock.ExpectExec("INSERT INTO schueler_fotos").
		WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg()).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))
	erwarteGegenprobe(t, mock, "uuid-67890", "fake image data 2")

	gefunden, uebernommen, _ := migriereAlleFotos(mock, wurzel, eintraege, true)
	assert.Equal(t, 2, gefunden)
	assert.Equal(t, 2, uebernommen)

	require.NoError(t, mock.ExpectationsWereMet())
}

func TestMigriereAlleFotos_KeinSchueler(t *testing.T) {
	t.Setenv("APP_ENCRYPTION_KEY", "01234567890123456789012345678901")

	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	testVerzeichnis := t.TempDir()

	datei1, err := os.Create(testVerzeichnis + "/99999.jpg")
	require.NoError(t, err)
	_, err = datei1.WriteString("fake image data")
	require.NoError(t, err)
	require.NoError(t, datei1.Close())

	wurzel, err := os.OpenRoot(testVerzeichnis)
	require.NoError(t, err)
	defer wurzel.Close() //nolint:errcheck

	ordner, err := wurzel.Open(".")
	require.NoError(t, err)
	eintraege, err := ordner.ReadDir(-1)
	require.NoError(t, err)
	require.NoError(t, ordner.Close())

	// Die Kennungen der Schüler laden, ohne Zeile.
	mock.ExpectQuery("SELECT barcode_id, id FROM schueler WHERE barcode_id = ANY").
		WithArgs(pgxmock.AnyArg()).
		WillReturnRows(pgxmock.NewRows([]string{"barcode_id", "id"}))

	gefunden, uebernommen, _ := migriereAlleFotos(mock, wurzel, eintraege, true)
	assert.Equal(t, 1, gefunden)
	assert.Equal(t, 0, uebernommen)

	require.NoError(t, mock.ExpectationsWereMet())
}

func TestMigriereAlleFotos_DatenbankfehlerBeimLaden(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	testVerzeichnis := t.TempDir()

	datei1, err := os.Create(testVerzeichnis + "/12345.jpg")
	require.NoError(t, err)
	_, err = datei1.WriteString("fake")
	require.NoError(t, err)
	require.NoError(t, datei1.Close())

	wurzel, err := os.OpenRoot(testVerzeichnis)
	require.NoError(t, err)
	defer wurzel.Close() //nolint:errcheck

	ordner, err := wurzel.Open(".")
	require.NoError(t, err)
	eintraege, err := ordner.ReadDir(-1)
	require.NoError(t, err)
	require.NoError(t, ordner.Close())

	mock.ExpectQuery("SELECT barcode_id, id FROM schueler WHERE barcode_id = ANY").
		WithArgs(pgxmock.AnyArg()).
		WillReturnError(os.ErrPermission)

	gefunden, uebernommen, _ := migriereAlleFotos(mock, wurzel, eintraege, true)
	assert.Equal(t, 0, gefunden)
	assert.Equal(t, 0, uebernommen)

	require.NoError(t, mock.ExpectationsWereMet())
}

func TestMigriereFoto_KeinLesen(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	testVerzeichnis := t.TempDir()

	wurzel, err := os.OpenRoot(testVerzeichnis)
	require.NoError(t, err)
	defer wurzel.Close() //nolint:errcheck

	ok := migriereFoto(mock, wurzel, "nicht-da.jpg", "123", "uuid")
	assert.False(t, ok)
}

func TestMigriereFoto_VerschluesselungFehlgeschlagen(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	testVerzeichnis := t.TempDir()
	datei1, err := os.Create(testVerzeichnis + "/123.jpg")
	require.NoError(t, err)
	_, err = datei1.WriteString("fake data")
	require.NoError(t, err)
	require.NoError(t, datei1.Close())

	wurzel, err := os.OpenRoot(testVerzeichnis)
	require.NoError(t, err)
	defer wurzel.Close() //nolint:errcheck

	// Den Schlüssel ausdrücklich leeren, sonst hinge der Test an der Umgebung: Mit gesetztem
	// APP_ENCRYPTION_KEY verschlüsselte er erfolgreich, die Attrappe lehnte das unerwartete Exec ab,
	// und das false käme nur von der Attrappe. Gelänge die Verschlüsselung trotzdem, liefe der
	// INSERT durch und der Test würde rot.
	t.Setenv("APP_ENCRYPTION_KEY", "")
	mock.ExpectExec("INSERT INTO schueler_fotos").WillReturnResult(pgxmock.NewResult("INSERT", 1))
	ok := migriereFoto(mock, wurzel, "123.jpg", "123", "uuid")
	assert.False(t, ok, "ohne Schlüssel darf keine Verschlüsselung gelingen")
}

func TestMigriereFoto_Datenbankfehler(t *testing.T) {
	t.Setenv("APP_ENCRYPTION_KEY", "01234567890123456789012345678901")

	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	testVerzeichnis := t.TempDir()
	datei1, err := os.Create(testVerzeichnis + "/123.jpg")
	require.NoError(t, err)
	_, err = datei1.WriteString("fake data")
	require.NoError(t, err)
	require.NoError(t, datei1.Close())

	wurzel, err := os.OpenRoot(testVerzeichnis)
	require.NoError(t, err)
	defer wurzel.Close() //nolint:errcheck

	mock.ExpectExec("INSERT INTO schueler_fotos").
		WithArgs("uuid", pgxmock.AnyArg()).
		WillReturnError(os.ErrPermission)

	ok := migriereFoto(mock, wurzel, "123.jpg", "123", "uuid")
	assert.False(t, ok)
	require.NoError(t, mock.ExpectationsWereMet())
}

// Die Quelldateien sind unverschlüsselte Schülerfotos: Das Werkzeug löscht sie selbst,
// sobald die Gegenprobe bestanden ist, statt das Löschen nur zu empfehlen.
func TestMigriereAlleFotos_LoeschtQuelldateienNachGegenprobe(t *testing.T) {
	t.Setenv("APP_ENCRYPTION_KEY", "01234567890123456789012345678901")

	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	testVerzeichnis := t.TempDir()
	pfad := testVerzeichnis + "/12345.jpg"
	require.NoError(t, os.WriteFile(pfad, []byte("fake image data"), 0o600))

	wurzel, err := os.OpenRoot(testVerzeichnis)
	require.NoError(t, err)
	defer wurzel.Close() //nolint:errcheck

	ordner, err := wurzel.Open(".")
	require.NoError(t, err)
	eintraege, err := ordner.ReadDir(-1)
	require.NoError(t, err)
	require.NoError(t, ordner.Close())

	mock.ExpectQuery("SELECT barcode_id, id FROM schueler WHERE barcode_id = ANY").
		WithArgs(pgxmock.AnyArg()).
		WillReturnRows(pgxmock.NewRows([]string{"barcode_id", "id"}).AddRow("12345", "uuid-12345"))
	mock.ExpectExec("INSERT INTO schueler_fotos").
		WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg()).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))
	erwarteGegenprobe(t, mock, "uuid-12345", "fake image data")

	_, migriert, geloescht := migriereAlleFotos(mock, wurzel, eintraege, false)

	assert.Equal(t, 1, migriert)
	assert.Equal(t, 1, geloescht)
	assert.NoFileExists(t, pfad, "das unverschlüsselte Foto liegt noch da")
	require.NoError(t, mock.ExpectationsWereMet())
}

// Bis zur bestandenen Gegenprobe ist die Datei die einzige Kopie des Bildes. Ein "INSERT ohne
// Fehler" heißt noch nicht, dass sich das Foto je wieder anzeigen lässt; ein falsch abgeleiteter
// Schlüssel fällt erst beim Entschlüsseln auf, und dann wäre die Quelle weg.
func TestMigriereAlleFotos_MisslungeneGegenprobeBehaeltDieDatei(t *testing.T) {
	t.Setenv("APP_ENCRYPTION_KEY", "01234567890123456789012345678901")

	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	testVerzeichnis := t.TempDir()
	pfad := testVerzeichnis + "/12345.jpg"
	require.NoError(t, os.WriteFile(pfad, []byte("fake image data"), 0o600))

	wurzel, err := os.OpenRoot(testVerzeichnis)
	require.NoError(t, err)
	defer wurzel.Close() //nolint:errcheck

	ordner, err := wurzel.Open(".")
	require.NoError(t, err)
	eintraege, err := ordner.ReadDir(-1)
	require.NoError(t, err)
	require.NoError(t, ordner.Close())

	mock.ExpectQuery("SELECT barcode_id, id FROM schueler WHERE barcode_id = ANY").
		WithArgs(pgxmock.AnyArg()).
		WillReturnRows(pgxmock.NewRows([]string{"barcode_id", "id"}).AddRow("12345", "uuid-12345"))
	mock.ExpectExec("INSERT INTO schueler_fotos").
		WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg()).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))
	// Zurückgelesen wird ein anderes Bild; die Gegenprobe muss scheitern.
	erwarteGegenprobe(t, mock, "uuid-12345", "ein anderes bild")

	_, migriert, geloescht := migriereAlleFotos(mock, wurzel, eintraege, false)

	assert.Equal(t, 0, migriert, "eine misslungene Gegenprobe darf nicht als Erfolg zählen")
	assert.Equal(t, 0, geloescht)
	assert.FileExists(t, pfad, "die einzige Kopie des Bildes wurde gelöscht")
}

// Die Abfrage nennt genau die Barcodes der Fotos, in der Reihenfolge der Dateinamen: Die
// Endung zählt in jeder Schreibweise, ein Verzeichnis und eine andere Datei zählen nicht.
// Stimmt die Liste nicht, findet die Datenbank keinen Schüler, und jedes Foto bliebe liegen.
func TestMigriereAlleFotos_FragtNachDenBarcodesDerFotos(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	testVerzeichnis := t.TempDir()
	for _, name := range []string{"B-2.JPG", "a-1.jpg", "notiz.txt"} {
		require.NoError(t, os.WriteFile(testVerzeichnis+"/"+name, []byte("fake"), 0o600))
	}
	require.NoError(t, os.Mkdir(testVerzeichnis+"/ordner.jpg", 0o700))

	wurzel, err := os.OpenRoot(testVerzeichnis)
	require.NoError(t, err)
	defer wurzel.Close() //nolint:errcheck

	ordner, err := wurzel.Open(".")
	require.NoError(t, err)
	eintraege, err := ordner.ReadDir(-1)
	require.NoError(t, err)
	require.NoError(t, ordner.Close())

	mock.ExpectQuery("SELECT barcode_id, id FROM schueler WHERE barcode_id = ANY").
		WithArgs([]string{"B-2", "a-1"}).
		WillReturnRows(pgxmock.NewRows([]string{"barcode_id", "id"}))

	gefunden, uebernommen, geloescht := migriereAlleFotos(mock, wurzel, eintraege, false)
	assert.Equal(t, 2, gefunden, "zwei Fotos liegen im Verzeichnis")
	assert.Equal(t, 0, uebernommen)
	assert.Equal(t, 0, geloescht)
	require.NoError(t, mock.ExpectationsWereMet())
}

// Bricht die Liste der Schüler beim Lesen ab, ist sie unvollständig: Das Werkzeug migriert
// dann nichts, statt die fehlenden Schüler als „kein Schüler für Barcode" zu melden.
func TestMigriereAlleFotos_AbgebrocheneSchuelerlisteMigriertNichts(t *testing.T) {
	t.Setenv("APP_ENCRYPTION_KEY", "01234567890123456789012345678901")

	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	testVerzeichnis := t.TempDir()
	pfad := testVerzeichnis + "/12345.jpg"
	require.NoError(t, os.WriteFile(pfad, []byte("fake image data"), 0o600))

	wurzel, err := os.OpenRoot(testVerzeichnis)
	require.NoError(t, err)
	defer wurzel.Close() //nolint:errcheck

	ordner, err := wurzel.Open(".")
	require.NoError(t, err)
	eintraege, err := ordner.ReadDir(-1)
	require.NoError(t, err)
	require.NoError(t, ordner.Close())

	// Die erste Zeile kommt an, danach reißt die Antwort ab: Next liefert false, Err den Fehler.
	mock.ExpectQuery("SELECT barcode_id, id FROM schueler WHERE barcode_id = ANY").
		WithArgs(pgxmock.AnyArg()).
		WillReturnRows(pgxmock.NewRows([]string{"barcode_id", "id"}).
			AddRow("12345", "uuid-12345").
			RowError(1, errors.New("verbindung abgerissen")))

	gefunden, uebernommen, geloescht := migriereAlleFotos(mock, wurzel, eintraege, false)
	assert.Equal(t, 0, gefunden, "nach einer abgebrochenen Liste wird kein Foto angefasst")
	assert.Equal(t, 0, uebernommen)
	assert.Equal(t, 0, geloescht)
	assert.FileExists(t, pfad)
	require.NoError(t, mock.ExpectationsWereMet())
}

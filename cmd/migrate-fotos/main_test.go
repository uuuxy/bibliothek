package main

import (
	"errors"
	"os"
	"testing"

	"bibliothek/internal/crypto"

	"github.com/pashagolub/pgxmock/v5"
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
	if err != nil {
		t.Fatalf("crypto.Verschluessele: %v", err)
	}
	mock.ExpectQuery("SELECT foto_encrypted FROM schueler_fotos").
		WithArgs(schuelerID).
		WillReturnRows(pgxmock.NewRows([]string{"foto_encrypted"}).AddRow(chiffre))
}

func TestMigriereAlleFotos_LeeresVerzeichnis(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool: %v", err)
	}
	defer mock.Close()

	testVerzeichnis := t.TempDir()
	wurzel, err := os.OpenRoot(testVerzeichnis)
	if err != nil {
		t.Fatalf("os.OpenRoot: %v", err)
	}
	defer wurzel.Close() //nolint:errcheck

	gefunden, uebernommen, _ := migriereAlleFotos(mock, wurzel, []os.DirEntry{}, true)
	if gefunden != 0 {
		t.Errorf("gefunden = %v, erwartet %v", gefunden, 0)
	}
	if uebernommen != 0 {
		t.Errorf("uebernommen = %v, erwartet %v", uebernommen, 0)
	}
}

func TestMigriereAlleFotos_MitDateien(t *testing.T) {
	t.Setenv("APP_ENCRYPTION_KEY", "01234567890123456789012345678901")

	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool: %v", err)
	}
	defer mock.Close()

	testVerzeichnis := t.TempDir()

	// Testdateien anlegen.
	datei1, err := os.Create(testVerzeichnis + "/12345.jpg")
	if err != nil {
		t.Fatalf("os.Create: %v", err)
	}
	_, err = datei1.WriteString("fake image data")
	if err != nil {
		t.Fatalf("datei1.WriteString: %v", err)
	}
	if err := datei1.Close(); err != nil {
		t.Fatalf("datei1.Close: %v", err)
	}

	datei2, err := os.Create(testVerzeichnis + "/67890.jpg")
	if err != nil {
		t.Fatalf("os.Create: %v", err)
	}
	_, err = datei2.WriteString("fake image data 2")
	if err != nil {
		t.Fatalf("datei2.WriteString: %v", err)
	}
	if err := datei2.Close(); err != nil {
		t.Fatalf("datei2.Close: %v", err)
	}

	// kein JPEG
	datei3, err := os.Create(testVerzeichnis + "/notajpg.txt")
	if err != nil {
		t.Fatalf("os.Create: %v", err)
	}
	_, err = datei3.WriteString("text")
	if err != nil {
		t.Fatalf("datei3.WriteString: %v", err)
	}
	if err := datei3.Close(); err != nil {
		t.Fatalf("datei3.Close: %v", err)
	}

	wurzel, err := os.OpenRoot(testVerzeichnis)
	if err != nil {
		t.Fatalf("os.OpenRoot: %v", err)
	}
	defer wurzel.Close() //nolint:errcheck

	ordner, err := wurzel.Open(".")
	if err != nil {
		t.Fatalf("wurzel.Open: %v", err)
	}
	eintraege, err := ordner.ReadDir(-1)
	if err != nil {
		t.Fatalf("ordner.ReadDir: %v", err)
	}
	if err := ordner.Close(); err != nil {
		t.Fatalf("ordner.Close: %v", err)
	}

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
	if gefunden != 2 {
		t.Errorf("gefunden = %v, erwartet %v", gefunden, 2)
	}
	if uebernommen != 2 {
		t.Errorf("uebernommen = %v, erwartet %v", uebernommen, 2)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("offene Erwartungen: %v", err)
	}
}

func TestMigriereAlleFotos_KeinSchueler(t *testing.T) {
	t.Setenv("APP_ENCRYPTION_KEY", "01234567890123456789012345678901")

	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool: %v", err)
	}
	defer mock.Close()

	testVerzeichnis := t.TempDir()

	datei1, err := os.Create(testVerzeichnis + "/99999.jpg")
	if err != nil {
		t.Fatalf("os.Create: %v", err)
	}
	_, err = datei1.WriteString("fake image data")
	if err != nil {
		t.Fatalf("datei1.WriteString: %v", err)
	}
	if err := datei1.Close(); err != nil {
		t.Fatalf("datei1.Close: %v", err)
	}

	wurzel, err := os.OpenRoot(testVerzeichnis)
	if err != nil {
		t.Fatalf("os.OpenRoot: %v", err)
	}
	defer wurzel.Close() //nolint:errcheck

	ordner, err := wurzel.Open(".")
	if err != nil {
		t.Fatalf("wurzel.Open: %v", err)
	}
	eintraege, err := ordner.ReadDir(-1)
	if err != nil {
		t.Fatalf("ordner.ReadDir: %v", err)
	}
	if err := ordner.Close(); err != nil {
		t.Fatalf("ordner.Close: %v", err)
	}

	// Die Kennungen der Schüler laden, ohne Zeile.
	mock.ExpectQuery("SELECT barcode_id, id FROM schueler WHERE barcode_id = ANY").
		WithArgs(pgxmock.AnyArg()).
		WillReturnRows(pgxmock.NewRows([]string{"barcode_id", "id"}))

	gefunden, uebernommen, _ := migriereAlleFotos(mock, wurzel, eintraege, true)
	if gefunden != 1 {
		t.Errorf("gefunden = %v, erwartet %v", gefunden, 1)
	}
	if uebernommen != 0 {
		t.Errorf("uebernommen = %v, erwartet %v", uebernommen, 0)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("offene Erwartungen: %v", err)
	}
}

func TestMigriereAlleFotos_DatenbankfehlerBeimLaden(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool: %v", err)
	}
	defer mock.Close()

	testVerzeichnis := t.TempDir()

	datei1, err := os.Create(testVerzeichnis + "/12345.jpg")
	if err != nil {
		t.Fatalf("os.Create: %v", err)
	}
	_, err = datei1.WriteString("fake")
	if err != nil {
		t.Fatalf("datei1.WriteString: %v", err)
	}
	if err := datei1.Close(); err != nil {
		t.Fatalf("datei1.Close: %v", err)
	}

	wurzel, err := os.OpenRoot(testVerzeichnis)
	if err != nil {
		t.Fatalf("os.OpenRoot: %v", err)
	}
	defer wurzel.Close() //nolint:errcheck

	ordner, err := wurzel.Open(".")
	if err != nil {
		t.Fatalf("wurzel.Open: %v", err)
	}
	eintraege, err := ordner.ReadDir(-1)
	if err != nil {
		t.Fatalf("ordner.ReadDir: %v", err)
	}
	if err := ordner.Close(); err != nil {
		t.Fatalf("ordner.Close: %v", err)
	}

	mock.ExpectQuery("SELECT barcode_id, id FROM schueler WHERE barcode_id = ANY").
		WithArgs(pgxmock.AnyArg()).
		WillReturnError(os.ErrPermission)

	gefunden, uebernommen, _ := migriereAlleFotos(mock, wurzel, eintraege, true)
	if gefunden != 0 {
		t.Errorf("gefunden = %v, erwartet %v", gefunden, 0)
	}
	if uebernommen != 0 {
		t.Errorf("uebernommen = %v, erwartet %v", uebernommen, 0)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("offene Erwartungen: %v", err)
	}
}

func TestMigriereFoto_KeinLesen(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool: %v", err)
	}
	defer mock.Close()

	testVerzeichnis := t.TempDir()

	wurzel, err := os.OpenRoot(testVerzeichnis)
	if err != nil {
		t.Fatalf("os.OpenRoot: %v", err)
	}
	defer wurzel.Close() //nolint:errcheck

	ok := migriereFoto(mock, wurzel, "nicht-da.jpg", "123", "uuid")
	if ok {
		t.Error("migriereFoto(mock, wurzel, \"nicht-da.jpg\", \"123\", \"uuid\") = true, erwartet false")
	}
}

func TestMigriereFoto_VerschluesselungFehlgeschlagen(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool: %v", err)
	}
	defer mock.Close()

	testVerzeichnis := t.TempDir()
	datei1, err := os.Create(testVerzeichnis + "/123.jpg")
	if err != nil {
		t.Fatalf("os.Create: %v", err)
	}
	_, err = datei1.WriteString("fake data")
	if err != nil {
		t.Fatalf("datei1.WriteString: %v", err)
	}
	if err := datei1.Close(); err != nil {
		t.Fatalf("datei1.Close: %v", err)
	}

	wurzel, err := os.OpenRoot(testVerzeichnis)
	if err != nil {
		t.Fatalf("os.OpenRoot: %v", err)
	}
	defer wurzel.Close() //nolint:errcheck

	// Den Schlüssel ausdrücklich leeren, sonst hinge der Test an der Umgebung: Mit gesetztem
	// APP_ENCRYPTION_KEY verschlüsselte er erfolgreich, die Attrappe lehnte das unerwartete Exec ab,
	// und das false käme nur von der Attrappe. Gelänge die Verschlüsselung trotzdem, liefe der
	// INSERT durch und der Test würde rot.
	t.Setenv("APP_ENCRYPTION_KEY", "")
	mock.ExpectExec("INSERT INTO schueler_fotos").WillReturnResult(pgxmock.NewResult("INSERT", 1))
	ok := migriereFoto(mock, wurzel, "123.jpg", "123", "uuid")
	if ok {
		t.Error("ohne Schlüssel darf keine Verschlüsselung gelingen")
	}
}

func TestMigriereFoto_Datenbankfehler(t *testing.T) {
	t.Setenv("APP_ENCRYPTION_KEY", "01234567890123456789012345678901")

	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool: %v", err)
	}
	defer mock.Close()

	testVerzeichnis := t.TempDir()
	datei1, err := os.Create(testVerzeichnis + "/123.jpg")
	if err != nil {
		t.Fatalf("os.Create: %v", err)
	}
	_, err = datei1.WriteString("fake data")
	if err != nil {
		t.Fatalf("datei1.WriteString: %v", err)
	}
	if err := datei1.Close(); err != nil {
		t.Fatalf("datei1.Close: %v", err)
	}

	wurzel, err := os.OpenRoot(testVerzeichnis)
	if err != nil {
		t.Fatalf("os.OpenRoot: %v", err)
	}
	defer wurzel.Close() //nolint:errcheck

	mock.ExpectExec("INSERT INTO schueler_fotos").
		WithArgs("uuid", pgxmock.AnyArg()).
		WillReturnError(os.ErrPermission)

	ok := migriereFoto(mock, wurzel, "123.jpg", "123", "uuid")
	if ok {
		t.Error("migriereFoto(mock, wurzel, \"123.jpg\", \"123\", \"uuid\") = true, erwartet false")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("offene Erwartungen: %v", err)
	}
}

// Die Quelldateien sind unverschlüsselte Schülerfotos: Das Werkzeug löscht sie selbst,
// sobald die Gegenprobe bestanden ist, statt das Löschen nur zu empfehlen.
func TestMigriereAlleFotos_LoeschtQuelldateienNachGegenprobe(t *testing.T) {
	t.Setenv("APP_ENCRYPTION_KEY", "01234567890123456789012345678901")

	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool: %v", err)
	}
	defer mock.Close()

	testVerzeichnis := t.TempDir()
	pfad := testVerzeichnis + "/12345.jpg"
	if err := os.WriteFile(pfad, []byte("fake image data"), 0o600); err != nil {
		t.Fatalf("os.WriteFile: %v", err)
	}

	wurzel, err := os.OpenRoot(testVerzeichnis)
	if err != nil {
		t.Fatalf("os.OpenRoot: %v", err)
	}
	defer wurzel.Close() //nolint:errcheck

	ordner, err := wurzel.Open(".")
	if err != nil {
		t.Fatalf("wurzel.Open: %v", err)
	}
	eintraege, err := ordner.ReadDir(-1)
	if err != nil {
		t.Fatalf("ordner.ReadDir: %v", err)
	}
	if err := ordner.Close(); err != nil {
		t.Fatalf("ordner.Close: %v", err)
	}

	mock.ExpectQuery("SELECT barcode_id, id FROM schueler WHERE barcode_id = ANY").
		WithArgs(pgxmock.AnyArg()).
		WillReturnRows(pgxmock.NewRows([]string{"barcode_id", "id"}).AddRow("12345", "uuid-12345"))
	mock.ExpectExec("INSERT INTO schueler_fotos").
		WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg()).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))
	erwarteGegenprobe(t, mock, "uuid-12345", "fake image data")

	_, migriert, geloescht := migriereAlleFotos(mock, wurzel, eintraege, false)

	if migriert != 1 {
		t.Errorf("migriert = %v, erwartet %v", migriert, 1)
	}
	if geloescht != 1 {
		t.Errorf("geloescht = %v, erwartet %v", geloescht, 1)
	}
	if _, err := os.Stat(pfad); err == nil {
		t.Errorf("Datei %s besteht noch: das unverschlüsselte Foto liegt noch da", pfad)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("offene Erwartungen: %v", err)
	}
}

// Bis zur bestandenen Gegenprobe ist die Datei die einzige Kopie des Bildes. Ein "INSERT ohne
// Fehler" heißt noch nicht, dass sich das Foto je wieder anzeigen lässt; ein falsch abgeleiteter
// Schlüssel fällt erst beim Entschlüsseln auf, und dann wäre die Quelle weg.
func TestMigriereAlleFotos_MisslungeneGegenprobeBehaeltDieDatei(t *testing.T) {
	t.Setenv("APP_ENCRYPTION_KEY", "01234567890123456789012345678901")

	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool: %v", err)
	}
	defer mock.Close()

	testVerzeichnis := t.TempDir()
	pfad := testVerzeichnis + "/12345.jpg"
	if err := os.WriteFile(pfad, []byte("fake image data"), 0o600); err != nil {
		t.Fatalf("os.WriteFile: %v", err)
	}

	wurzel, err := os.OpenRoot(testVerzeichnis)
	if err != nil {
		t.Fatalf("os.OpenRoot: %v", err)
	}
	defer wurzel.Close() //nolint:errcheck

	ordner, err := wurzel.Open(".")
	if err != nil {
		t.Fatalf("wurzel.Open: %v", err)
	}
	eintraege, err := ordner.ReadDir(-1)
	if err != nil {
		t.Fatalf("ordner.ReadDir: %v", err)
	}
	if err := ordner.Close(); err != nil {
		t.Fatalf("ordner.Close: %v", err)
	}

	mock.ExpectQuery("SELECT barcode_id, id FROM schueler WHERE barcode_id = ANY").
		WithArgs(pgxmock.AnyArg()).
		WillReturnRows(pgxmock.NewRows([]string{"barcode_id", "id"}).AddRow("12345", "uuid-12345"))
	mock.ExpectExec("INSERT INTO schueler_fotos").
		WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg()).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))
	// Zurückgelesen wird ein anderes Bild; die Gegenprobe muss scheitern.
	erwarteGegenprobe(t, mock, "uuid-12345", "ein anderes bild")

	_, migriert, geloescht := migriereAlleFotos(mock, wurzel, eintraege, false)

	if migriert != 0 {
		t.Errorf("migriert = %v, erwartet %v: eine misslungene Gegenprobe darf nicht als Erfolg zählen", migriert, 0)
	}
	if geloescht != 0 {
		t.Errorf("geloescht = %v, erwartet %v", geloescht, 0)
	}
	if _, err := os.Stat(pfad); err != nil {
		t.Errorf("Datei %s fehlt: %v: die einzige Kopie des Bildes wurde gelöscht", pfad, err)
	}
}

// Die Abfrage nennt genau die Barcodes der Fotos, in der Reihenfolge der Dateinamen: Die
// Endung zählt in jeder Schreibweise, ein Verzeichnis und eine andere Datei zählen nicht.
// Stimmt die Liste nicht, findet die Datenbank keinen Schüler, und jedes Foto bliebe liegen.
func TestMigriereAlleFotos_FragtNachDenBarcodesDerFotos(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool: %v", err)
	}
	defer mock.Close()

	testVerzeichnis := t.TempDir()
	for _, name := range []string{"B-2.JPG", "a-1.jpg", "notiz.txt"} {
		if err := os.WriteFile(testVerzeichnis+"/"+name, []byte("fake"), 0o600); err != nil {
			t.Fatalf("os.WriteFile: %v", err)
		}
	}
	if err := os.Mkdir(testVerzeichnis+"/ordner.jpg", 0o700); err != nil {
		t.Fatalf("os.Mkdir: %v", err)
	}

	wurzel, err := os.OpenRoot(testVerzeichnis)
	if err != nil {
		t.Fatalf("os.OpenRoot: %v", err)
	}
	defer wurzel.Close() //nolint:errcheck

	ordner, err := wurzel.Open(".")
	if err != nil {
		t.Fatalf("wurzel.Open: %v", err)
	}
	eintraege, err := ordner.ReadDir(-1)
	if err != nil {
		t.Fatalf("ordner.ReadDir: %v", err)
	}
	if err := ordner.Close(); err != nil {
		t.Fatalf("ordner.Close: %v", err)
	}

	mock.ExpectQuery("SELECT barcode_id, id FROM schueler WHERE barcode_id = ANY").
		WithArgs([]string{"B-2", "a-1"}).
		WillReturnRows(pgxmock.NewRows([]string{"barcode_id", "id"}))

	gefunden, uebernommen, geloescht := migriereAlleFotos(mock, wurzel, eintraege, false)
	if gefunden != 2 {
		t.Errorf("gefunden = %v, erwartet %v: zwei Fotos liegen im Verzeichnis", gefunden, 2)
	}
	if uebernommen != 0 {
		t.Errorf("uebernommen = %v, erwartet %v", uebernommen, 0)
	}
	if geloescht != 0 {
		t.Errorf("geloescht = %v, erwartet %v", geloescht, 0)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("offene Erwartungen: %v", err)
	}
}

// Bricht die Liste der Schüler beim Lesen ab, ist sie unvollständig: Das Werkzeug migriert
// dann nichts, statt die fehlenden Schüler als „kein Schüler für Barcode" zu melden.
func TestMigriereAlleFotos_AbgebrocheneSchuelerlisteMigriertNichts(t *testing.T) {
	t.Setenv("APP_ENCRYPTION_KEY", "01234567890123456789012345678901")

	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool: %v", err)
	}
	defer mock.Close()

	testVerzeichnis := t.TempDir()
	pfad := testVerzeichnis + "/12345.jpg"
	if err := os.WriteFile(pfad, []byte("fake image data"), 0o600); err != nil {
		t.Fatalf("os.WriteFile: %v", err)
	}

	wurzel, err := os.OpenRoot(testVerzeichnis)
	if err != nil {
		t.Fatalf("os.OpenRoot: %v", err)
	}
	defer wurzel.Close() //nolint:errcheck

	ordner, err := wurzel.Open(".")
	if err != nil {
		t.Fatalf("wurzel.Open: %v", err)
	}
	eintraege, err := ordner.ReadDir(-1)
	if err != nil {
		t.Fatalf("ordner.ReadDir: %v", err)
	}
	if err := ordner.Close(); err != nil {
		t.Fatalf("ordner.Close: %v", err)
	}

	// Die erste Zeile kommt an, danach reißt die Antwort ab: Next liefert false, Err den Fehler.
	mock.ExpectQuery("SELECT barcode_id, id FROM schueler WHERE barcode_id = ANY").
		WithArgs(pgxmock.AnyArg()).
		WillReturnRows(pgxmock.NewRows([]string{"barcode_id", "id"}).
			AddRow("12345", "uuid-12345").
			RowError(1, errors.New("verbindung abgerissen")))

	gefunden, uebernommen, geloescht := migriereAlleFotos(mock, wurzel, eintraege, false)
	if gefunden != 0 {
		t.Errorf("gefunden = %v, erwartet %v: nach einer abgebrochenen Liste wird kein Foto angefasst", gefunden, 0)
	}
	if uebernommen != 0 {
		t.Errorf("uebernommen = %v, erwartet %v", uebernommen, 0)
	}
	if geloescht != 0 {
		t.Errorf("geloescht = %v, erwartet %v", geloescht, 0)
	}
	if _, err := os.Stat(pfad); err != nil {
		t.Errorf("Datei %s fehlt: %v", pfad, err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("offene Erwartungen: %v", err)
	}
}

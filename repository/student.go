package repository

import (
	"context"
	"time"

	"bibliothek/db"
	"bibliothek/internal/ausweis"
)

// BorrowedBook represents a currently checked out book copy detail for the student.
type BorrowedBook struct {
	ID         string `json:"id"`
	AusleiheID string `json:"ausleihe_id"`
	BarcodeID  string `json:"barcode_id"`
	Titel      string `json:"titel"`
	Autor      string `json:"autor"`
	// ISBN wird nicht angezeigt, sondern gebraucht: Ohne sie kann die Ausleihliste kein
	// Cover nachladen, wenn keins am Titel gespeichert ist (CoverPeek fragt darüber den
	// Cover-Proxy) — und das ist bei importierten Beständen der Normalfall.
	ISBN     string `json:"isbn,omitempty"`
	CoverURL string `json:"cover_url,omitempty"`
	// IstLernmittel steuert die Marke „Lernmittel" an der Ausleihzeile — aus der
	// Spalte (Migration 093), nicht mehr aus einem LMF-Präfix in Titel oder Signatur.
	IstLernmittel  bool      `json:"ist_lernmittel"`
	AusgeliehenAm  time.Time `json:"ausgeliehen_am"`
	RueckgabeFrist time.Time `json:"rueckgabe_frist"`
}

// StudentListStat represents a student along with their current loan statistics.
type StudentListStat struct {
	ID                string `json:"id"`
	BarcodeID         string `json:"barcode_id"`
	Vorname           string `json:"vorname"`
	Nachname          string `json:"nachname"`
	Klasse            string `json:"klasse"`
	AbgaengerJahr     int    `json:"abgaenger_jahr"`
	IstGesperrt       bool   `json:"ist_gesperrt"`
	IsManuallyBlocked bool   `json:"is_manually_blocked"`
	HasFoto           bool   `json:"-"`
	FotoURL           string `json:"foto_url"`
	AusgeliehenCount  int    `json:"ausgeliehen_count"`
	UeberfaelligCount int    `json:"ueberfaellig_count"`
	// AusweisGueltigBis ist das Jahr, bis zu dem der Schülerausweis gilt — aus dem
	// Bildungsgang der Klasse gerechnet, NICHT aus AbgaengerJahr (siehe
	// internal/ausweis). nil heißt: aus dieser Klassenbezeichnung nicht ableitbar,
	// dann fragt der Druckdialog nach.
	AusweisGueltigBis *int `json:"ausweis_gueltig_bis,omitempty"`
}

// Scanner kapselt die Scan-Schnittstelle von pgx.Row und pgx.Rows,
// um gemeinsame Helferfunktionen zum Einlesen von Zeilen zu ermöglichen.
type Scanner interface {
	Scan(dest ...any) error
}

// StudentRepository definiert die Operationen zur Abfrage und zum Abgleich von Schülern in der Datenbank.
type StudentRepository interface {
	// GetByBarcode sucht einen Schüler anhand seiner Barcode-ID (Schülerausweis).
	// Liefert nil zurück, wenn kein Schüler gefunden wurde.
	GetByBarcode(ctx context.Context, barcode string) (*Student, error)

	// GetByID sucht einen Schüler anhand seiner UUID (Primärschlüssel).
	// Liefert nil zurück, wenn kein Schüler gefunden wurde.
	GetByID(ctx context.Context, id string) (*Student, error)

	// SearchStudentsFuzzy sucht tokenweise und diakritikfrei über Vorname, Nachname
	// und Barcode-ID. Zweiter Rückgabewert ist die Gesamtzahl der Treffer vor dem
	// Limit — ohne sie kann die Oberfläche eine abgeschnittene Liste nicht als solche
	// kennzeichnen und zeigt stillschweigend nur die ersten Namen.
	SearchStudentsFuzzy(ctx context.Context, queryText string, limit int) ([]Student, int, error)

	// Hinweis: Der LUSD-Abgleich läuft ausschließlich über den Handler-Pfad in
	// api/lusd.go (ladeAktiveSchueler → wendeLusdAenderungenAn). Eine frühere
	// Massen-Pipeline (BulkSyncLUSD/GetAllLUSDStudents) wurde entfernt: ungenutzt und
	// mit latenten Fehlern (u. a. Ghost-Block bei Rückkehrern).

	// HasPhoto checks if an encrypted photo exists for the student.
	HasPhoto(ctx context.Context, studentID string) (bool, error)

	// HasOpenDamages checks if the student has any unpaid damage fees.
	HasOpenDamages(ctx context.Context, studentID string) (bool, error)

	// GetActiveBorrowedBooks retrieves all books currently borrowed by the student.
	GetActiveBorrowedBooks(ctx context.Context, studentID string) ([]BorrowedBook, error)

	// GetDistinctClasses returns a list of all active classes.
	GetDistinctClasses(ctx context.Context) ([]string, error)

	// EtikettenZeilen liefert Barcode, Name und Klasse zu den angegebenen IDs — die vier
	// Angaben, die auf einem Klebeetikett stehen. Sortiert nach Nachname, Vorname.
	EtikettenZeilen(ctx context.Context, ids []string) ([]SchuelerEtikettZeile, error)

	// ListStudentsWithStats liefert Schüler samt Ausleihzahlen, optional nach Klasse
	// und/oder Suchbegriff eingegrenzt. Die Suche läuft über dieselben SQL-Bausteine
	// wie SearchStudentsFuzzy und kennt keine 500er-Grenze.
	ListStudentsWithStats(ctx context.Context, klasse, suche string) ([]StudentListStat, error)
}

// pgStudentRepository implementiert das StudentRepository für PostgreSQL.
type pgStudentRepository struct {
	db db.PgxPoolIface
}

// NewStudentRepository erzeugt eine neue Instanz des PostgreSQL-basierten StudentRepositorys.
func NewStudentRepository(db db.PgxPoolIface) StudentRepository {
	return &pgStudentRepository{db: db}
}

// scanStudent ist eine Hilfsfunktion zum Einlesen einer Datenbankzeile in das Student-Modell.
func scanStudent(row Scanner) (*Student, error) {
	return scanStudentMitZusatz(row)
}

// scanStudentMitZusatz scannt die Standard-Spaltenliste und danach beliebige
// Zusatzspalten (z. B. count(*) OVER () für die Gesamttrefferzahl). So bleibt die
// Feldreihenfolge an genau einer Stelle — eine zweite Kopie würde beim nächsten
// Spaltenzuwachs still auseinanderlaufen.
func scanStudentMitZusatz(row Scanner, zusatz ...any) (*Student, error) {
	var s Student
	ziele := []any{
		&s.ID, &s.BarcodeID, &s.Vorname, &s.Nachname, &s.Klasse, &s.AbgaengerJahr, &s.IstGesperrt, &s.LusdID, &s.IstAbgaenger, &s.Geburtsdatum, &s.ErstelltAm, &s.AktualisiertAm, &s.IsManuallyBlocked, &s.BlockReason,
		&s.Strasse, &s.Hausnummer, &s.Plz, &s.Ort, &s.ElternEmail,
	}
	if len(zusatz) > 0 {
		ziele = append(ziele, zusatz...)
	}
	if err := row.Scan(ziele...); err != nil {
		return nil, err
	}
	s.AusweisGueltigBis = ausweisGueltigBis(s.Klasse)
	return &s, nil
}

// ausweisGueltigBis rechnet die Ausweisgültigkeit aus der Klasse. Kein Datenbankfeld:
// Die Regel darf sich ändern (Schulform, Zweigwechsel), ohne dass Altbestände
// nachgezogen werden müssen — und ein gespeicherter Wert wäre nach jedem
// Schuljahreswechsel still veraltet.
//
// Rückgabe nil bei einer Klassenbezeichnung, aus der sich nichts ableiten lässt. Der
// Druckdialog fragt dann nach, statt ein Datum zu erfinden.
func ausweisGueltigBis(klasse string) *int {
	jahr, ok := ausweis.GueltigBisJahr(klasse, ausweis.SchuljahrEnde(time.Now()))
	if !ok {
		return nil
	}
	return &jahr
}

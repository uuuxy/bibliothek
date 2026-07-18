package repository

import (
	"bibliothek/db"
	"context"
	"time"
)

// BorrowedBook represents a currently checked out book copy detail for the student.
type BorrowedBook struct {
	ID             string    `json:"id"`
	AusleiheID     string    `json:"ausleihe_id"`
	BarcodeID      string    `json:"barcode_id"`
	Titel          string    `json:"titel"`
	Autor          string    `json:"autor"`
	CoverURL       string    `json:"cover_url,omitempty"`
	AusgeliehenAm  time.Time `json:"ausgeliehen_am"`
	RueckgabeFrist time.Time `json:"rueckgabe_frist"`
}

// StudentListStat represents a student along with their current loan statistics.
type StudentListStat struct {
	ID                 string `json:"id"`
	BarcodeID          string `json:"barcode_id"`
	Vorname            string `json:"vorname"`
	Nachname           string `json:"nachname"`
	Klasse             string `json:"klasse"`
	AbgaengerJahr      int    `json:"abgaenger_jahr"`
	IstGesperrt        bool   `json:"ist_gesperrt"`
	HasFoto            bool   `json:"-"`
	FotoURL            string `json:"foto_url"`
	AusgeliehenCount   int    `json:"ausgeliehen_count"`
	UeberfaelligCount  int    `json:"ueberfaellig_count"`
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

	// SearchStudentsFuzzy führt eine Teilstring-Suche über Vorname, Nachname und Barcode-ID aus.
	SearchStudentsFuzzy(ctx context.Context, queryText string, limit int) ([]Student, error)

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

	// ListStudentsWithStats returns a list of students with loan statistics.
	ListStudentsWithStats(ctx context.Context, klasse string) ([]StudentListStat, error)
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
	var s Student
	err := row.Scan(
		&s.ID, &s.BarcodeID, &s.Vorname, &s.Nachname, &s.Klasse, &s.AbgaengerJahr, &s.IstGesperrt, &s.LusdID, &s.IstAbgaenger, &s.Geburtsdatum, &s.ErstelltAm, &s.AktualisiertAm, &s.IsManuallyBlocked, &s.BlockReason,
		&s.Strasse, &s.Hausnummer, &s.Plz, &s.Ort, &s.ElternEmail,
	)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

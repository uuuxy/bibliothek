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

	// GetNextSequence ermittelt die nächste freie Barcode-Nummer für neue Schülerausweise (Format: "S-1xxxx").
	GetNextSequence(ctx context.Context) (int, error)

	// GetAllLUSDStudents lädt alle Schüler-IDs, LUSD-IDs, Namen und Geburtsdaten zur Vorbereitung eines LUSD-Abgleichs.
	GetAllLUSDStudents(ctx context.Context) ([]Student, error)

	// BulkSyncLUSD führt den LUSD-Datenabgleich (Massen-Update und Massen-Insert) in einer Transaktion durch.
	// Schueler, die nicht mehr im LUSD-Datenbestand gelistet sind, werden automatisch als Schulabgänger (ist_abgaenger = true) markiert
	// und deren Vormerkungen gelöscht.
	// Gibt die Anzahl der Abgänger zurück, die noch offene Ausleihen haben.
	// BulkSyncLUSD führt den LUSD-Datenabgleich durch.
	BulkSyncLUSD(ctx context.Context, updates []StudentUpdate, inserts []StudentInsert, allLusdIDs []string) (int, error)

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

// StudentUpdate definiert die Datenstruktur für Aktualisierungen eines Schülers während des LUSD-Imports.
type StudentUpdate struct {
	ID           string
	Vorname      string
	Nachname     string
	Klasse       string
	Geburtsdatum *string // Format: YYYY-MM-DD
	LusdID       *string
}

// StudentInsert definiert die Datenstruktur für neu anzulegende Schüler während des LUSD-Imports.
type StudentInsert struct {
	BarcodeID     string
	Vorname       string
	Nachname      string
	Klasse        string
	Geburtsdatum  *string // Format: YYYY-MM-DD
	AbgaengerJahr int
	LusdID        *string
	IstAbgaenger  bool
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
	)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

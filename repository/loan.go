package repository

import (
	"bibliothek/db"
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
)

// LoanRepository verwaltet alle Datenbank-Interaktionen für Ausleihen und Rückgaben (Bücher und Geräte).
type LoanRepository interface {
	// GetActiveLoanByCopyID sucht die aktuell aktive (nicht zurückgegebene) Ausleihe für ein Buchexemplar.
	// Gibt nil zurück, wenn das Exemplar aktuell nicht verliehen ist.
	GetActiveLoanByCopyID(ctx context.Context, copyID string) (*Loan, error)

	// GetActiveLoanByCopyIDTx sucht die aktive Ausleihe innerhalb einer Transaktion und setzt
	// einen Row-Level-Lock (SELECT ... FOR UPDATE). Dies verhindert Race Conditions bei zeitgleichen Scans.
	GetActiveLoanByCopyIDTx(ctx context.Context, tx pgx.Tx, copyID string) (*Loan, error)

	// BeginTx startet eine neue Datenbanktransaktion mit dem Isolationslevel 'Read Committed'.
	// Aufrufer müssen defer tx.Rollback(ctx) aufrufen und bei Erfolg tx.Commit(ctx) ausführen.
	BeginTx(ctx context.Context) (pgx.Tx, error)

	// CreateLoan legt einen neuen Ausleihdatensatz für einen Schüler an.
	CreateLoan(ctx context.Context, exemplarID, schuelerID, bearbeiterID string, rueckgabeFrist time.Time) (*Loan, error)

	// CreateLoanTx legt einen neuen Ausleihdatensatz für einen Schüler innerhalb einer laufenden Transaktion an.
	CreateLoanTx(ctx context.Context, tx pgx.Tx, exemplarID, schuelerID, bearbeiterID string, rueckgabeFrist time.Time) (*Loan, error)

	// CreateUserLoan legt einen neuen Ausleihdatensatz für einen Systembenutzer (z. B. Lehrer) an.
	CreateUserLoan(ctx context.Context, exemplarID, ausleiherBenutzerID, bearbeiterID string, rueckgabeFrist time.Time, istHandapparat bool) (*Loan, error)

	// CreateUserLoanTx legt einen neuen Ausleihdatensatz für einen Systembenutzer innerhalb einer Transaktion an.
	CreateUserLoanTx(ctx context.Context, tx pgx.Tx, exemplarID, ausleiherBenutzerID, bearbeiterID string, rueckgabeFrist time.Time, istHandapparat bool) (*Loan, error)

	// ReturnLoan markiert eine aktive Ausleihe als zurückgegeben.
	ReturnLoan(ctx context.Context, loanID, bearbeiterID string, isFremdrueckgabe bool) error

	// ReturnLoanTx markiert eine aktive Ausleihe als zurückgegeben innerhalb einer Transaktion.
	ReturnLoanTx(ctx context.Context, tx pgx.Tx, loanID, bearbeiterID string, isFremdrueckgabe bool) error
}

// pgLoanRepository implementiert das LoanRepository für PostgreSQL.
type pgLoanRepository struct {
	db db.PgxPoolIface
}

// NewLoanRepository erstellt eine neue Instanz des PostgreSQL-basierten Loan-Repositorys.
func NewLoanRepository(db db.PgxPoolIface) LoanRepository {
	return &pgLoanRepository{db: db}
}

// scanLoan liest eine Tabellenzeile in ein Loan-Modellobjekt ein.
func scanLoan(row Scanner) (*Loan, error) {
	var l Loan
	err := row.Scan(
		&l.ID, &l.ExemplarID, &l.SchuelerID, &l.AusleiherBenutzerID, &l.AusgeliehenAm, &l.RueckgabeFrist, &l.RueckgabeAm, &l.BearbeiterID, &l.RueckgabeBearbeiterID, &l.IstFremdrueckgabe, &l.IstHandapparat,
	)
	if err != nil {
		return nil, err
	}
	return &l, nil
}

// BeginTx startet eine Transaktion mit dem Isolationslevel 'Read Committed'.
// Dieses Level ist ideal für das Ausleihsystem, da es Schmutzdaten (Dirty Reads) verhindert,
// gleichzeitig aber hohen Durchsatz bei parallelen Scanner-Anfragen ermöglicht.
func (r *pgLoanRepository) BeginTx(ctx context.Context) (pgx.Tx, error) {
	return r.db.BeginTx(ctx, pgx.TxOptions{
		IsoLevel:   pgx.ReadCommitted,
		AccessMode: pgx.ReadWrite,
	})
}

// GetActiveLoanByCopyID ruft die aktive Ausleihe ohne Sperre (schreibgeschützt) ab.
func (r *pgLoanRepository) GetActiveLoanByCopyID(ctx context.Context, copyID string) (*Loan, error) {
	query := `
		SELECT id, exemplar_id, schueler_id, ausleiher_benutzer_id, ausgeliehen_am, rueckgabe_frist, rueckgabe_am, bearbeiter_id, rueckgabe_bearbeiter_id, ist_fremdrueckgabe, ist_handapparat
		FROM ausleihen
		WHERE exemplar_id = $1 AND rueckgabe_am IS NULL
		LIMIT 1
	`
	l, err := scanLoan(r.db.QueryRow(ctx, query, copyID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return l, nil
}

// GetActiveLoanByCopyIDTx ruft die aktive Ausleihe innerhalb einer Transaktion mit 'SELECT ... FOR UPDATE' ab.
// Diese Zeilensperrung (Row-Level-Lock) verhindert, dass parallele Scanner-Anfragen
// dasselbe Exemplar innerhalb desselben Millisekundenfensters doppelt verarbeiten.
func (r *pgLoanRepository) GetActiveLoanByCopyIDTx(ctx context.Context, tx pgx.Tx, copyID string) (*Loan, error) {
	query := `
		SELECT id, exemplar_id, schueler_id, ausleiher_benutzer_id, ausgeliehen_am, rueckgabe_frist, rueckgabe_am, bearbeiter_id, rueckgabe_bearbeiter_id, ist_fremdrueckgabe, ist_handapparat
		FROM ausleihen
		WHERE exemplar_id = $1 AND rueckgabe_am IS NULL
		LIMIT 1
		FOR UPDATE
	`
	l, err := scanLoan(tx.QueryRow(ctx, query, copyID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return l, nil
}

// CreateLoan erzeugt einen neuen Ausleiheintrag.
// Durch 'ON CONFLICT DO NOTHING' auf dem eindeutigen Index (exemplar_id, rueckgabe_am IS NULL)
// werden Duplikate verhindert, die durch WLAN-Verzögerungen oder doppeltes Scannen entstehen.
func (r *pgLoanRepository) CreateLoan(ctx context.Context, exemplarID, schuelerID, bearbeiterID string, rueckgabeFrist time.Time) (*Loan, error) {
	query := `
		INSERT INTO ausleihen (exemplar_id, schueler_id, rueckgabe_frist, bearbeiter_id)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT DO NOTHING
		RETURNING id, exemplar_id, schueler_id, ausleiher_benutzer_id, ausgeliehen_am, rueckgabe_frist, rueckgabe_am, bearbeiter_id, rueckgabe_bearbeiter_id, ist_fremdrueckgabe, ist_handapparat
	`
	l, err := scanLoan(r.db.QueryRow(ctx, query, exemplarID, schuelerID, rueckgabeFrist, bearbeiterID))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) || errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return l, nil
}

// CreateLoanTx erzeugt einen neuen Ausleiheintrag innerhalb einer Transaktion.
func (r *pgLoanRepository) CreateLoanTx(ctx context.Context, tx pgx.Tx, exemplarID, schuelerID, bearbeiterID string, rueckgabeFrist time.Time) (*Loan, error) {
	query := `
		INSERT INTO ausleihen (exemplar_id, schueler_id, rueckgabe_frist, bearbeiter_id)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT DO NOTHING
		RETURNING id, exemplar_id, schueler_id, ausleiher_benutzer_id, ausgeliehen_am, rueckgabe_frist, rueckgabe_am, bearbeiter_id, rueckgabe_bearbeiter_id, ist_fremdrueckgabe, ist_handapparat
	`
	l, err := scanLoan(tx.QueryRow(ctx, query, exemplarID, schuelerID, rueckgabeFrist, bearbeiterID))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) || errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return l, nil
}

// CreateUserLoan erzeugt einen neuen Ausleiheintrag für einen Systembenutzer (Lehrkraft).
func (r *pgLoanRepository) CreateUserLoan(ctx context.Context, exemplarID, ausleiherBenutzerID, bearbeiterID string, rueckgabeFrist time.Time, istHandapparat bool) (*Loan, error) {
	query := `
		INSERT INTO ausleihen (exemplar_id, ausleiher_benutzer_id, rueckgabe_frist, bearbeiter_id, ist_handapparat)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT DO NOTHING
		RETURNING id, exemplar_id, schueler_id, ausleiher_benutzer_id, ausgeliehen_am, rueckgabe_frist, rueckgabe_am, bearbeiter_id, rueckgabe_bearbeiter_id, ist_fremdrueckgabe, ist_handapparat
	`
	l, err := scanLoan(r.db.QueryRow(ctx, query, exemplarID, ausleiherBenutzerID, rueckgabeFrist, bearbeiterID, istHandapparat))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) || errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return l, nil
}

// CreateUserLoanTx erzeugt einen neuen Ausleiheintrag für einen Systembenutzer innerhalb einer Transaktion.
func (r *pgLoanRepository) CreateUserLoanTx(ctx context.Context, tx pgx.Tx, exemplarID, ausleiherBenutzerID, bearbeiterID string, rueckgabeFrist time.Time, istHandapparat bool) (*Loan, error) {
	query := `
		INSERT INTO ausleihen (exemplar_id, ausleiher_benutzer_id, rueckgabe_frist, bearbeiter_id, ist_handapparat)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT DO NOTHING
		RETURNING id, exemplar_id, schueler_id, ausleiher_benutzer_id, ausgeliehen_am, rueckgabe_frist, rueckgabe_am, bearbeiter_id, rueckgabe_bearbeiter_id, ist_fremdrueckgabe, ist_handapparat
	`
	l, err := scanLoan(tx.QueryRow(ctx, query, exemplarID, ausleiherBenutzerID, rueckgabeFrist, bearbeiterID, istHandapparat))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) || errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return l, nil
}

// ReturnLoan bucht ein ausgeliehenes Buch zurück.
func (r *pgLoanRepository) ReturnLoan(ctx context.Context, loanID, bearbeiterID string, isFremdrueckgabe bool) error {
	query := `
		UPDATE ausleihen
		SET rueckgabe_am = CURRENT_TIMESTAMP, rueckgabe_bearbeiter_id = $1, ist_fremdrueckgabe = $2
		WHERE id = $3 AND rueckgabe_am IS NULL
	`
	tag, err := r.db.Exec(ctx, query, bearbeiterID, isFremdrueckgabe, loanID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return errors.New("loan not active or already returned")
	}
	return nil
}

// ReturnLoanTx bucht ein ausgeliehenes Buch innerhalb einer Transaktion zurück.
func (r *pgLoanRepository) ReturnLoanTx(ctx context.Context, tx pgx.Tx, loanID, bearbeiterID string, isFremdrueckgabe bool) error {
	query := `
		UPDATE ausleihen
		SET rueckgabe_am = CURRENT_TIMESTAMP, rueckgabe_bearbeiter_id = $1, ist_fremdrueckgabe = $2
		WHERE id = $3 AND rueckgabe_am IS NULL
	`
	tag, err := tx.Exec(ctx, query, bearbeiterID, isFremdrueckgabe, loanID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return errors.New("loan not active or already returned")
	}
	return nil
}

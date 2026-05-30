package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// LoanRepository handles the transactional check-out and check-in database procedures.
type LoanRepository interface {
	// GetActiveLoanByCopyID returns the current unreturned loan for a physical book copy. Returns nil if not borrowed.
	GetActiveLoanByCopyID(ctx context.Context, copyID string) (*Loan, error)
	// GetActiveLoanByCopyIDTx returns the current unreturned loan for a physical book copy within a transaction,
	// using SELECT ... FOR UPDATE to prevent concurrent modifications (race conditions with parallel scanners).
	GetActiveLoanByCopyIDTx(ctx context.Context, tx pgx.Tx, copyID string) (*Loan, error)
	// BeginTx starts a new Read Committed transaction on the underlying pool.
	// Callers must defer tx.Rollback(ctx) and call tx.Commit(ctx) on success.
	BeginTx(ctx context.Context) (pgx.Tx, error)
	// CreateLoan creates a new loan record.
	CreateLoan(ctx context.Context, exemplarID, schuelerID, bearbeiterID string, rueckgabeFrist time.Time) (*Loan, error)
	// CreateLoanTx performs CreateLoan inside a transaction context.
	CreateLoanTx(ctx context.Context, tx pgx.Tx, exemplarID, schuelerID, bearbeiterID string, rueckgabeFrist time.Time) (*Loan, error)
	// CreateUserLoan creates a new loan record for a system user (e.g. teacher/handapparat).
	CreateUserLoan(ctx context.Context, exemplarID, ausleiherBenutzerID, bearbeiterID string, rueckgabeFrist time.Time, istHandapparat bool) (*Loan, error)
	// CreateUserLoanTx performs CreateUserLoan inside a transaction context.
	CreateUserLoanTx(ctx context.Context, tx pgx.Tx, exemplarID, ausleiherBenutzerID, bearbeiterID string, rueckgabeFrist time.Time, istHandapparat bool) (*Loan, error)
	// ReturnLoan flags an active loan as returned.
	ReturnLoan(ctx context.Context, loanID, bearbeiterID string, isFremdrueckgabe bool) error
	// ReturnLoanTx flags an active loan as returned inside a transaction context.
	ReturnLoanTx(ctx context.Context, tx pgx.Tx, loanID, bearbeiterID string, isFremdrueckgabe bool) error
	// UndoReturn reverses a return (within 1 hour) by nullifying rueckgabe_am.
	UndoReturn(ctx context.Context, loanID string) error
}

type pgLoanRepository struct {
	db *pgxpool.Pool
}

// NewLoanRepository constructs a PostgreSQL-backed LoanRepository.
func NewLoanRepository(db *pgxpool.Pool) LoanRepository {
	return &pgLoanRepository{db: db}
}

// BeginTx starts a new READ COMMITTED transaction.
// Read Committed is the correct isolation level for library operations:
// it prevents dirty reads while allowing concurrent scanner throughput.
// Callers MUST defer tx.Rollback(ctx) immediately after calling BeginTx.
func (r *pgLoanRepository) BeginTx(ctx context.Context) (pgx.Tx, error) {
	return r.db.BeginTx(ctx, pgx.TxOptions{
		IsoLevel:   pgx.ReadCommitted,
		AccessMode: pgx.ReadWrite,
	})
}

// GetActiveLoanByCopyID gets the active loan without locking (read-only lookup).
func (r *pgLoanRepository) GetActiveLoanByCopyID(ctx context.Context, copyID string) (*Loan, error) {
	query := `
		SELECT id, exemplar_id, schueler_id, ausleiher_benutzer_id, ausgeliehen_am, rueckgabe_frist, rueckgabe_am, bearbeiter_id, rueckgabe_bearbeiter_id, ist_fremdrueckgabe, ist_handapparat
		FROM ausleihen
		WHERE exemplar_id = $1 AND rueckgabe_am IS NULL
		LIMIT 1
	`
	var l Loan
	err := r.db.QueryRow(ctx, query, copyID).Scan(
		&l.ID, &l.ExemplarID, &l.SchuelerID, &l.AusleiherBenutzerID, &l.AusgeliehenAm, &l.RueckgabeFrist, &l.RueckgabeAm, &l.BearbeiterID, &l.RueckgabeBearbeiterID, &l.IstFremdrueckgabe, &l.IstHandapparat,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &l, nil
}

// GetActiveLoanByCopyIDTx gets the active loan within a transaction using SELECT ... FOR UPDATE.
// This row-level lock prevents concurrent scanners from double-processing the same book copy
// within the same millisecond window (e.g., WLAN lag causing duplicate scan events).
func (r *pgLoanRepository) GetActiveLoanByCopyIDTx(ctx context.Context, tx pgx.Tx, copyID string) (*Loan, error) {
	query := `
		SELECT id, exemplar_id, schueler_id, ausleiher_benutzer_id, ausgeliehen_am, rueckgabe_frist, rueckgabe_am, bearbeiter_id, rueckgabe_bearbeiter_id, ist_fremdrueckgabe, ist_handapparat
		FROM ausleihen
		WHERE exemplar_id = $1 AND rueckgabe_am IS NULL
		LIMIT 1
		FOR UPDATE
	`
	var l Loan
	err := tx.QueryRow(ctx, query, copyID).Scan(
		&l.ID, &l.ExemplarID, &l.SchuelerID, &l.AusleiherBenutzerID, &l.AusgeliehenAm, &l.RueckgabeFrist, &l.RueckgabeAm, &l.BearbeiterID, &l.RueckgabeBearbeiterID, &l.IstFremdrueckgabe, &l.IstHandapparat,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &l, nil
}

// CreateLoan inserts a new loan.
// ON CONFLICT DO NOTHING on the unique index (exemplar_id, rueckgabe_am IS NULL) prevents
// duplicate checkout records from WLAN-lag-induced double-scans; returns nil if already exists.
func (r *pgLoanRepository) CreateLoan(ctx context.Context, exemplarID, schuelerID, bearbeiterID string, rueckgabeFrist time.Time) (*Loan, error) {
	query := `
		INSERT INTO ausleihen (exemplar_id, schueler_id, rueckgabe_frist, bearbeiter_id)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT DO NOTHING
		RETURNING id, exemplar_id, schueler_id, ausleiher_benutzer_id, ausgeliehen_am, rueckgabe_frist, rueckgabe_am, bearbeiter_id, rueckgabe_bearbeiter_id, ist_fremdrueckgabe, ist_handapparat
	`
	var l Loan
	err := r.db.QueryRow(ctx, query, exemplarID, schuelerID, rueckgabeFrist, bearbeiterID).Scan(
		&l.ID, &l.ExemplarID, &l.SchuelerID, &l.AusleiherBenutzerID, &l.AusgeliehenAm, &l.RueckgabeFrist, &l.RueckgabeAm, &l.BearbeiterID, &l.RueckgabeBearbeiterID, &l.IstFremdrueckgabe, &l.IstHandapparat,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) || errors.Is(err, pgx.ErrNoRows) {
			// ON CONFLICT DO NOTHING: duplicate scan suppressed, idempotent success
			return nil, nil
		}
		return nil, err
	}
	return &l, nil
}

// CreateLoanTx inserts a new loan inside a transaction.
func (r *pgLoanRepository) CreateLoanTx(ctx context.Context, tx pgx.Tx, exemplarID, schuelerID, bearbeiterID string, rueckgabeFrist time.Time) (*Loan, error) {
	query := `
		INSERT INTO ausleihen (exemplar_id, schueler_id, rueckgabe_frist, bearbeiter_id)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT DO NOTHING
		RETURNING id, exemplar_id, schueler_id, ausleiher_benutzer_id, ausgeliehen_am, rueckgabe_frist, rueckgabe_am, bearbeiter_id, rueckgabe_bearbeiter_id, ist_fremdrueckgabe, ist_handapparat
	`
	var l Loan
	err := tx.QueryRow(ctx, query, exemplarID, schuelerID, rueckgabeFrist, bearbeiterID).Scan(
		&l.ID, &l.ExemplarID, &l.SchuelerID, &l.AusleiherBenutzerID, &l.AusgeliehenAm, &l.RueckgabeFrist, &l.RueckgabeAm, &l.BearbeiterID, &l.RueckgabeBearbeiterID, &l.IstFremdrueckgabe, &l.IstHandapparat,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) || errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &l, nil
}

// CreateUserLoan inserts a new user loan.
func (r *pgLoanRepository) CreateUserLoan(ctx context.Context, exemplarID, ausleiherBenutzerID, bearbeiterID string, rueckgabeFrist time.Time, istHandapparat bool) (*Loan, error) {
	query := `
		INSERT INTO ausleihen (exemplar_id, ausleiher_benutzer_id, rueckgabe_frist, bearbeiter_id, ist_handapparat)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT DO NOTHING
		RETURNING id, exemplar_id, schueler_id, ausleiher_benutzer_id, ausgeliehen_am, rueckgabe_frist, rueckgabe_am, bearbeiter_id, rueckgabe_bearbeiter_id, ist_fremdrueckgabe, ist_handapparat
	`
	var l Loan
	err := r.db.QueryRow(ctx, query, exemplarID, ausleiherBenutzerID, rueckgabeFrist, bearbeiterID, istHandapparat).Scan(
		&l.ID, &l.ExemplarID, &l.SchuelerID, &l.AusleiherBenutzerID, &l.AusgeliehenAm, &l.RueckgabeFrist, &l.RueckgabeAm, &l.BearbeiterID, &l.RueckgabeBearbeiterID, &l.IstFremdrueckgabe, &l.IstHandapparat,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) || errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &l, nil
}

// CreateUserLoanTx inserts a new user loan inside a transaction.
func (r *pgLoanRepository) CreateUserLoanTx(ctx context.Context, tx pgx.Tx, exemplarID, ausleiherBenutzerID, bearbeiterID string, rueckgabeFrist time.Time, istHandapparat bool) (*Loan, error) {
	query := `
		INSERT INTO ausleihen (exemplar_id, ausleiher_benutzer_id, rueckgabe_frist, bearbeiter_id, ist_handapparat)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT DO NOTHING
		RETURNING id, exemplar_id, schueler_id, ausleiher_benutzer_id, ausgeliehen_am, rueckgabe_frist, rueckgabe_am, bearbeiter_id, rueckgabe_bearbeiter_id, ist_fremdrueckgabe, ist_handapparat
	`
	var l Loan
	err := tx.QueryRow(ctx, query, exemplarID, ausleiherBenutzerID, rueckgabeFrist, bearbeiterID, istHandapparat).Scan(
		&l.ID, &l.ExemplarID, &l.SchuelerID, &l.AusleiherBenutzerID, &l.AusgeliehenAm, &l.RueckgabeFrist, &l.RueckgabeAm, &l.BearbeiterID, &l.RueckgabeBearbeiterID, &l.IstFremdrueckgabe, &l.IstHandapparat,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) || errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &l, nil
}

// ReturnLoan sets return fields.
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

// ReturnLoanTx sets return fields inside a transaction.
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

// UndoReturn reverses a recent return (within 1 hour) by nullifying rueckgabe_am.
func (r *pgLoanRepository) UndoReturn(ctx context.Context, loanID string) error {
	tag, err := r.db.Exec(ctx, `
		UPDATE ausleihen
		SET rueckgabe_am = NULL, rueckgabe_bearbeiter_id = NULL
		WHERE id = $1
		  AND rueckgabe_am IS NOT NULL
		  AND rueckgabe_am > CURRENT_TIMESTAMP - INTERVAL '1 hour'
	`, loanID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return errors.New("loan not found, not yet returned, or return window exceeded")
	}
	return nil
}

package repository

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// LoanRepository handles the transactional check-out and check-in database procedures.
type LoanRepository interface {
	// GetActiveLoanByCopyID returns the current unreturned loan for a physical book copy. Returns nil if not borrowed.
	GetActiveLoanByCopyID(ctx context.Context, copyID string) (*Loan, error)
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
}

type pgLoanRepository struct {
	db *pgxpool.Pool
}

// NewLoanRepository constructs a PostgreSQL-backed LoanRepository.
func NewLoanRepository(db *pgxpool.Pool) LoanRepository {
	return &pgLoanRepository{db: db}
}

// GetActiveLoanByCopyID gets the active loan.
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

// CreateLoan inserts a new loan.
func (r *pgLoanRepository) CreateLoan(ctx context.Context, exemplarID, schuelerID, bearbeiterID string, rueckgabeFrist time.Time) (*Loan, error) {
	query := `
		INSERT INTO ausleihen (exemplar_id, schueler_id, rueckgabe_frist, bearbeiter_id)
		VALUES ($1, $2, $3, $4)
		RETURNING id, exemplar_id, schueler_id, ausleiher_benutzer_id, ausgeliehen_am, rueckgabe_frist, rueckgabe_am, bearbeiter_id, rueckgabe_bearbeiter_id, ist_fremdrueckgabe, ist_handapparat
	`
	var l Loan
	err := r.db.QueryRow(ctx, query, exemplarID, schuelerID, rueckgabeFrist, bearbeiterID).Scan(
		&l.ID, &l.ExemplarID, &l.SchuelerID, &l.AusleiherBenutzerID, &l.AusgeliehenAm, &l.RueckgabeFrist, &l.RueckgabeAm, &l.BearbeiterID, &l.RueckgabeBearbeiterID, &l.IstFremdrueckgabe, &l.IstHandapparat,
	)
	if err != nil {
		return nil, err
	}
	return &l, nil
}

// CreateLoanTx inserts a new loan inside a transaction.
func (r *pgLoanRepository) CreateLoanTx(ctx context.Context, tx pgx.Tx, exemplarID, schuelerID, bearbeiterID string, rueckgabeFrist time.Time) (*Loan, error) {
	query := `
		INSERT INTO ausleihen (exemplar_id, schueler_id, rueckgabe_frist, bearbeiter_id)
		VALUES ($1, $2, $3, $4)
		RETURNING id, exemplar_id, schueler_id, ausleiher_benutzer_id, ausgeliehen_am, rueckgabe_frist, rueckgabe_am, bearbeiter_id, rueckgabe_bearbeiter_id, ist_fremdrueckgabe, ist_handapparat
	`
	var l Loan
	err := tx.QueryRow(ctx, query, exemplarID, schuelerID, rueckgabeFrist, bearbeiterID).Scan(
		&l.ID, &l.ExemplarID, &l.SchuelerID, &l.AusleiherBenutzerID, &l.AusgeliehenAm, &l.RueckgabeFrist, &l.RueckgabeAm, &l.BearbeiterID, &l.RueckgabeBearbeiterID, &l.IstFremdrueckgabe, &l.IstHandapparat,
	)
	if err != nil {
		return nil, err
	}
	return &l, nil
}

// CreateUserLoan inserts a new user loan.
func (r *pgLoanRepository) CreateUserLoan(ctx context.Context, exemplarID, ausleiherBenutzerID, bearbeiterID string, rueckgabeFrist time.Time, istHandapparat bool) (*Loan, error) {
	query := `
		INSERT INTO ausleihen (exemplar_id, ausleiher_benutzer_id, rueckgabe_frist, bearbeiter_id, ist_handapparat)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, exemplar_id, schueler_id, ausleiher_benutzer_id, ausgeliehen_am, rueckgabe_frist, rueckgabe_am, bearbeiter_id, rueckgabe_bearbeiter_id, ist_fremdrueckgabe, ist_handapparat
	`
	var l Loan
	err := r.db.QueryRow(ctx, query, exemplarID, ausleiherBenutzerID, rueckgabeFrist, bearbeiterID, istHandapparat).Scan(
		&l.ID, &l.ExemplarID, &l.SchuelerID, &l.AusleiherBenutzerID, &l.AusgeliehenAm, &l.RueckgabeFrist, &l.RueckgabeAm, &l.BearbeiterID, &l.RueckgabeBearbeiterID, &l.IstFremdrueckgabe, &l.IstHandapparat,
	)
	if err != nil {
		return nil, err
	}
	return &l, nil
}

// CreateUserLoanTx inserts a new user loan inside a transaction.
func (r *pgLoanRepository) CreateUserLoanTx(ctx context.Context, tx pgx.Tx, exemplarID, ausleiherBenutzerID, bearbeiterID string, rueckgabeFrist time.Time, istHandapparat bool) (*Loan, error) {
	query := `
		INSERT INTO ausleihen (exemplar_id, ausleiher_benutzer_id, rueckgabe_frist, bearbeiter_id, ist_handapparat)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, exemplar_id, schueler_id, ausleiher_benutzer_id, ausgeliehen_am, rueckgabe_frist, rueckgabe_am, bearbeiter_id, rueckgabe_bearbeiter_id, ist_fremdrueckgabe, ist_handapparat
	`
	var l Loan
	err := tx.QueryRow(ctx, query, exemplarID, ausleiherBenutzerID, rueckgabeFrist, bearbeiterID, istHandapparat).Scan(
		&l.ID, &l.ExemplarID, &l.SchuelerID, &l.AusleiherBenutzerID, &l.AusgeliehenAm, &l.RueckgabeFrist, &l.RueckgabeAm, &l.BearbeiterID, &l.RueckgabeBearbeiterID, &l.IstFremdrueckgabe, &l.IstHandapparat,
	)
	if err != nil {
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

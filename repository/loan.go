package repository

import (
	"bibliothek/db"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

// ErrAusleiheKonflikt meldet, dass der INSERT am eindeutigen Index
// (exemplar_id, rueckgabe_am IS NULL) abgeprallt ist: Zwischen der Prüfung
// "Exemplar ist frei" und dem Schreiben hat ein anderer Vorgang es verbucht.
//
// Vorher gaben die Create*-Funktionen in diesem Fall (nil, nil) zurück — kein Fehler,
// keine Zeile. Der Aufrufer prüfte nur err, hielt das für Erfolg, committete und
// meldete dem Arbeitsplatz eine Ausleihe, die nie geschrieben wurde. Genau diese
// Bugklasse: HTTP 200 trotz Datenverlust.
var ErrAusleiheKonflikt = errors.New("exemplar wurde zwischenzeitlich anderweitig verbucht")

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

	// CreateLoanTx legt einen neuen Ausleihdatensatz für einen Schüler innerhalb einer laufenden Transaktion an.
	CreateLoanTx(ctx context.Context, tx pgx.Tx, exemplarID, schuelerID, bearbeiterID string, rueckgabeFrist time.Time) (*Loan, error)

	// CreateUserLoanTx legt einen neuen Ausleihdatensatz für einen Systembenutzer innerhalb einer Transaktion an.
	CreateUserLoanTx(ctx context.Context, tx pgx.Tx, exemplarID, ausleiherBenutzerID, bearbeiterID string, rueckgabeFrist time.Time, istHandapparat bool) (*Loan, error)

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

// Die drei Schreiber der Ausleihe teilen sich je EINE SQL-Formulierung mit dem Online-Scan
// und dem Nachbuchen (Stufe 2 des Offline-Baus, Commit 11): Der Zeitpunkt ist ein Parameter.
// nil heißt „jetzt" (Online-Scan, Default der Spalten); das Nachbuchen gibt den Scan-Zeitpunkt
// vom Theken-Rechner mit — höchstens Serverzeit, das prüft der Aufrufer. Zwei Formulierungen
// derselben Buchung (eine mit, eine ohne Zeit) wären zwei Türen zum selben Zustand.

const sqlAusleiheSchueler = `
		INSERT INTO ausleihen (exemplar_id, schueler_id, rueckgabe_frist, bearbeiter_id, ausgeliehen_am, erfasst_am)
		VALUES ($1, $2, $3, $4, COALESCE($5::timestamptz, CURRENT_TIMESTAMP), COALESCE($5::timestamptz, CURRENT_TIMESTAMP))
		ON CONFLICT DO NOTHING
		RETURNING id, exemplar_id, schueler_id, ausleiher_benutzer_id, ausgeliehen_am, rueckgabe_frist, rueckgabe_am, bearbeiter_id, rueckgabe_bearbeiter_id, ist_fremdrueckgabe, ist_handapparat
	`

const sqlAusleiheBenutzer = `
		INSERT INTO ausleihen (exemplar_id, ausleiher_benutzer_id, rueckgabe_frist, bearbeiter_id, ist_handapparat, ausgeliehen_am, erfasst_am)
		VALUES ($1, $2, $3, $4, $5, COALESCE($6::timestamptz, CURRENT_TIMESTAMP), COALESCE($6::timestamptz, CURRENT_TIMESTAMP))
		ON CONFLICT DO NOTHING
		RETURNING id, exemplar_id, schueler_id, ausleiher_benutzer_id, ausgeliehen_am, rueckgabe_frist, rueckgabe_am, bearbeiter_id, rueckgabe_bearbeiter_id, ist_fremdrueckgabe, ist_handapparat
	`

// CreateLoanTx erzeugt einen neuen Ausleiheintrag innerhalb einer Transaktion — jetzt.
func (r *pgLoanRepository) CreateLoanTx(ctx context.Context, tx pgx.Tx, exemplarID, schuelerID, bearbeiterID string, rueckgabeFrist time.Time) (*Loan, error) {
	return CreateLoanZumTx(ctx, tx, exemplarID, schuelerID, bearbeiterID, rueckgabeFrist, nil)
}

// CreateLoanZumTx erzeugt die Ausleihe eines Schülers zum gegebenen Zeitpunkt (nil = jetzt)
// und stempelt die Bewegung des Exemplars mit demselben Zeitpunkt.
func CreateLoanZumTx(ctx context.Context, tx pgx.Tx, exemplarID, schuelerID, bearbeiterID string, rueckgabeFrist time.Time, zeitpunkt *time.Time) (*Loan, error) {
	return schreibeAusleihe(ctx, tx, exemplarID, zeitpunkt, sqlAusleiheSchueler, exemplarID, schuelerID, rueckgabeFrist, bearbeiterID, zeitpunkt)
}

// CreateUserLoanTx erzeugt einen neuen Ausleiheintrag für einen Systembenutzer innerhalb einer Transaktion — jetzt.
func (r *pgLoanRepository) CreateUserLoanTx(ctx context.Context, tx pgx.Tx, exemplarID, ausleiherBenutzerID, bearbeiterID string, rueckgabeFrist time.Time, istHandapparat bool) (*Loan, error) {
	return CreateUserLoanZumTx(ctx, tx, exemplarID, ausleiherBenutzerID, bearbeiterID, rueckgabeFrist, istHandapparat, nil)
}

// CreateUserLoanZumTx erzeugt die Ausleihe einer Lehrkraft zum gegebenen Zeitpunkt (nil = jetzt).
func CreateUserLoanZumTx(ctx context.Context, tx pgx.Tx, exemplarID, ausleiherBenutzerID, bearbeiterID string, rueckgabeFrist time.Time, istHandapparat bool, zeitpunkt *time.Time) (*Loan, error) {
	return schreibeAusleihe(ctx, tx, exemplarID, zeitpunkt, sqlAusleiheBenutzer, exemplarID, ausleiherBenutzerID, rueckgabeFrist, bearbeiterID, istHandapparat, zeitpunkt)
}

func schreibeAusleihe(ctx context.Context, tx pgx.Tx, exemplarID string, zeitpunkt *time.Time, query string, args ...any) (*Loan, error) {
	l, err := scanLoan(tx.QueryRow(ctx, query, args...))
	if err != nil {
		// Keine Zeile heißt hier NICHT "nichts gefunden", sondern "ON CONFLICT hat
		// den INSERT verworfen" — ein Konflikt, kein Normalfall.
		if errors.Is(err, sql.ErrNoRows) || errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrAusleiheKonflikt
		}
		return nil, err
	}
	if err := StempleBewegungZum(ctx, tx, exemplarID, zeitpunkt); err != nil {
		return nil, err
	}
	return l, nil
}

// StempleBewegungZum setzt buecher_exemplare.letzte_bewegung_am (Migration 116) auf den
// Zeitpunkt; nil heißt jetzt (Online-Scan), das Nachbuchen gibt den Scan-Zeitpunkt mit.
// Der Wächter des Nachbuchens weist Scans ab, die älter sind als die letzte Bewegung.
// Aufgerufen NACH der Zeile in ausleihen — Sperrreihenfolge Schüler → Ausleihe → Exemplar
// (docs/invarianten.md, Abschnitt 1); das Nachbuchen hält dieselbe Reihenfolge.
func StempleBewegungZum(ctx context.Context, q DBQueryer, exemplarID string, zeitpunkt *time.Time) error {
	tag, err := q.Exec(ctx, `UPDATE buecher_exemplare SET letzte_bewegung_am = COALESCE($2::timestamptz, CURRENT_TIMESTAMP) WHERE id = $1`, exemplarID, zeitpunkt)
	if err != nil {
		return fmt.Errorf("bewegung stempeln: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrExemplarNichtGefunden
	}
	return nil
}

// ReturnLoanTx bucht ein ausgeliehenes Buch innerhalb einer Transaktion zurück — jetzt.
func (r *pgLoanRepository) ReturnLoanTx(ctx context.Context, tx pgx.Tx, loanID, bearbeiterID string, isFremdrueckgabe bool) error {
	return ReturnLoanZumTx(ctx, tx, loanID, bearbeiterID, isFremdrueckgabe, nil)
}

// ReturnLoanZumTx bucht die Rückgabe zum gegebenen Zeitpunkt (nil = jetzt). Liegt der
// Zeitpunkt vor der Ausleihe, weist check_return_date (23514) die Rückgabe ab — beim
// Nachbuchen ist das ein veralteter Scan, kein Serverfehler.
func ReturnLoanZumTx(ctx context.Context, tx pgx.Tx, loanID, bearbeiterID string, isFremdrueckgabe bool, zeitpunkt *time.Time) error {
	tag, err := tx.Exec(ctx, `
		UPDATE ausleihen
		SET rueckgabe_am = COALESCE($4::timestamptz, CURRENT_TIMESTAMP), rueckgabe_bearbeiter_id = $1, ist_fremdrueckgabe = $2
		WHERE id = $3 AND rueckgabe_am IS NULL
	`, bearbeiterID, isFremdrueckgabe, loanID, zeitpunkt)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return errors.New("loan not active or already returned")
	}
	// Bewegungsstempel am Exemplar (Migration 116). Diese Funktion bucht Bücher; Geräte gehen
	// über device_service — die Ausleihe hier trägt also immer ein Exemplar, 0 Zeilen wäre ein
	// Widerspruch und kein Normalfall.
	stempel, err := tx.Exec(ctx, `
		UPDATE buecher_exemplare e SET letzte_bewegung_am = COALESCE($2::timestamptz, CURRENT_TIMESTAMP)
		FROM ausleihen a WHERE a.id = $1 AND e.id = a.exemplar_id`, loanID, zeitpunkt)
	if err != nil {
		return fmt.Errorf("bewegung stempeln: %w", err)
	}
	if stempel.RowsAffected() == 0 {
		return ErrExemplarNichtGefunden
	}
	return nil
}

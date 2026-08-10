package repository

import (
	"context"
	"errors"
	"fmt"

	"bibliothek/db"

	"github.com/jackc/pgx/v5"
)

// UserRepository definiert die Datenbankoperationen zur Verwaltung von Systembenutzern (Lehrer, Admins, Helfer).
type UserRepository interface {
	// GetUsers ruft alle registrierten Systembenutzer sortiert nach Nachname und Vorname ab.
	GetUsers(ctx context.Context) ([]User, error)

	// CheckEmailExists prüft, ob eine E-Mail-Adresse bereits einem Benutzer zugeordnet ist.
	// Mit excludeID kann die ID des aktuell bearbeiteten Benutzers von der Prüfung ausgeschlossen werden.
	CheckEmailExists(ctx context.Context, email string, excludeID string) (bool, error)

	// CheckBarcodeExists prüft, ob eine Barcode-ID (Mitarbeiter-/Lehrerausweis) bereits vergeben ist.
	// Mit excludeID kann die ID des aktuell bearbeiteten Benutzers ausgeschlossen werden.
	CheckBarcodeExists(ctx context.Context, barcode string, excludeID string) (bool, error)

	// CreateUser legt einen neuen Systembenutzer in der Datenbank an und gibt dessen generierte ID (UUID) zurück.
	CreateUser(ctx context.Context, barcode *string, vorname, nachname, email, rolle string) (string, error)

	// UpdateUser aktualisiert die Daten eines bestehenden Systembenutzers.
	UpdateUser(ctx context.Context, p UpdateUserParams) error

	// GetRolleByID liefert die AKTUELL gespeicherte Rolle eines Benutzers in
	// Großschreibung. Kein Treffer ergibt ("", nil).
	//
	// Die Rolle aus dem Request-Rumpf taugt dafür nicht: Wer einen Admin-Datensatz
	// bearbeitet, schickt selbst mit, welche Rolle daraus werden soll. Ob das Ziel
	// HEUTE ein Admin ist — und die Bearbeitung damit Admin-Rechte des Aufrufers
	// verlangt — steht nur in der Datenbank.
	GetRolleByID(ctx context.Context, id string) (string, error)

	// GetLehrerByBarcode sucht eine AKTIVE Lehrkraft anhand ihres Ausweis-Barcodes.
	// Kein Treffer liefert (nil, nil) — wie GetCopyByBarcode und GetByBarcode, damit die
	// Omnibox „nicht gefunden" von einem echten Datenbankfehler unterscheiden kann.
	GetLehrerByBarcode(ctx context.Context, barcode string) (*User, error)
}

// postgresUserRepo implementiert das UserRepository für PostgreSQL.
type postgresUserRepo struct {
	pool db.PgxPoolIface
}

// NewUserRepository erzeugt eine neue Instanz des PostgreSQL-basierten UserRepositorys.
func NewUserRepository(pool db.PgxPoolIface) UserRepository {
	return &postgresUserRepo{pool: pool}
}

// GetLehrerByBarcode sucht eine aktive Lehrkraft über ihren Ausweis.
//
// rolle ist das ENUM benutzer_rolle ('admin','kollegium','mitarbeiter','helfer') und wird
// kleingeschrieben verglichen; rolle::text vermeidet „invalid input value for enum".
func (r *postgresUserRepo) GetLehrerByBarcode(ctx context.Context, barcode string) (*User, error) {
	var u User
	err := r.pool.QueryRow(ctx, `
		SELECT id, coalesce(barcode_id, ''), vorname, nachname, rolle
		FROM benutzer
		WHERE barcode_id = $1 AND lower(rolle::text) = 'kollegium' AND aktiv = true
		LIMIT 1
	`, barcode).Scan(&u.ID, &u.BarcodeID, &u.Vorname, &u.Nachname, &u.Rolle)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("konnte die Lehrkraft über ihren Barcode nicht lesen: %w", err)
	}
	return &u, nil
}

// GetUsers fragt alle registrierten Benutzer ab.
func (r *postgresUserRepo) GetUsers(ctx context.Context) ([]User, error) {
	query := `
		SELECT id, coalesce(barcode_id, ''), vorname, nachname, email, rolle, aktiv, erstellt_am
		FROM benutzer
		ORDER BY nachname, vorname
	`
	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	users := []User{}
	for rows.Next() {
		var u User
		err := rows.Scan(&u.ID, &u.BarcodeID, &u.Vorname, &u.Nachname, &u.Email, &u.Rolle, &u.Aktiv, &u.ErstelltAm)
		if err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return users, nil
}

// CheckEmailExists prüft das Vorhandensein einer E-Mail-Adresse im System.
func (r *postgresUserRepo) CheckEmailExists(ctx context.Context, email string, excludeID string) (bool, error) {
	var exists bool
	var err error
	if excludeID == "" {
		err = r.pool.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM benutzer WHERE email = $1)", email).Scan(&exists)
	} else {
		err = r.pool.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM benutzer WHERE email = $1 AND id != $2)", email, excludeID).Scan(&exists)
	}
	return exists, err
}

// CheckBarcodeExists prüft das Vorhandensein eines Barcodes im System.
func (r *postgresUserRepo) CheckBarcodeExists(ctx context.Context, barcode string, excludeID string) (bool, error) {
	var exists bool
	var err error
	if excludeID == "" {
		err = r.pool.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM benutzer WHERE barcode_id = $1)", barcode).Scan(&exists)
	} else {
		err = r.pool.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM benutzer WHERE barcode_id = $1 AND id != $2)", barcode, excludeID).Scan(&exists)
	}
	return exists, err
}

// CreateUser fügt einen neuen Benutzer hinzu.
func (r *postgresUserRepo) CreateUser(ctx context.Context, barcode *string, vorname, nachname, email, rolle string) (string, error) {
	var userID string
	query := `
		INSERT INTO benutzer (barcode_id, vorname, nachname, email, rolle, aktiv)
		VALUES ($1, $2, $3, $4, $5::benutzer_rolle, true)
		RETURNING id
	`
	err := r.pool.QueryRow(ctx, query, barcode, vorname, nachname, email, rolle).Scan(&userID)
	return userID, err
}

// UpdateUserParams bündelt die aktualisierbaren Felder eines Benutzers.
type UpdateUserParams struct {
	ID       string
	Barcode  *string
	Vorname  string
	Nachname string
	Email    string
	Rolle    string
	Aktiv    bool
}

func (r *postgresUserRepo) UpdateUser(ctx context.Context, p UpdateUserParams) error {
	query := `
		UPDATE benutzer
		SET barcode_id = $1, vorname = $2, nachname = $3, email = $4, rolle = $5::benutzer_rolle, aktiv = $6, aktualisiert_am = CURRENT_TIMESTAMP
		WHERE id = $7
	`
	_, err := r.pool.Exec(ctx, query, p.Barcode, p.Vorname, p.Nachname, p.Email, p.Rolle, p.Aktiv, p.ID)
	return err
}

// GetRolleByID liest die gespeicherte Rolle eines Benutzers.
//
// rolle::text wie in GetLehrerByBarcode — die Spalte ist das ENUM benutzer_rolle.
func (r *postgresUserRepo) GetRolleByID(ctx context.Context, id string) (string, error) {
	var rolle string
	err := r.pool.QueryRow(ctx, `SELECT UPPER(rolle::text) FROM benutzer WHERE id = $1`, id).Scan(&rolle)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("konnte die Rolle des Benutzers nicht lesen: %w", err)
	}
	return rolle, nil
}

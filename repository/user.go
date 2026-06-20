package repository

import (
	"context"

	"bibliothek/db"
)

// UserRepository handles user persistence.
type UserRepository interface {
	GetUsers(ctx context.Context) ([]User, error)
	CheckEmailExists(ctx context.Context, email string, excludeID string) (bool, error)
	CheckBarcodeExists(ctx context.Context, barcode string, excludeID string) (bool, error)
	CreateUser(ctx context.Context, barcode *string, vorname, nachname, email, rolle string) (string, error)
	UpdateUser(ctx context.Context, id string, barcode *string, vorname, nachname, email, rolle string, aktiv bool) error
}

type postgresUserRepo struct {
	pool db.PgxPoolIface
}

func NewUserRepository(pool db.PgxPoolIface) UserRepository {
	return &postgresUserRepo{pool: pool}
}

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
	return users, nil
}

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

func (r *postgresUserRepo) UpdateUser(ctx context.Context, id string, barcode *string, vorname, nachname, email, rolle string, aktiv bool) error {
	query := `
		UPDATE benutzer
		SET barcode_id = $1, vorname = $2, nachname = $3, email = $4, rolle = $5::benutzer_rolle, aktiv = $6, aktualisiert_am = CURRENT_TIMESTAMP
		WHERE id = $7
	`
	_, err := r.pool.Exec(ctx, query, barcode, vorname, nachname, email, rolle, aktiv, id)
	return err
}

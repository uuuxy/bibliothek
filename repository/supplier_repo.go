package repository

import (
	"bibliothek/db"
	"context"
)

// Supplier repräsentiert einen Lieferanten (z. B. eine Buchhandlung).
type Supplier struct {
	ID           string
	Name         string
	Email        string
	Kundennummer string
}

// SupplierRepository definiert die Datenbank-Zugriffe für Lieferanten.
type SupplierRepository interface {
	GetSupplierByID(ctx context.Context, id string) (*Supplier, error)
}

type pgSupplierRepository struct {
	db db.PgxPoolIface
}

// NewSupplierRepository erstellt eine neue Instanz des SupplierRepositorys.
func NewSupplierRepository(pool db.PgxPoolIface) SupplierRepository {
	return &pgSupplierRepository{db: pool}
}

// GetSupplierByID lädt einen Lieferanten anhand seiner ID.
func (r *pgSupplierRepository) GetSupplierByID(ctx context.Context, id string) (*Supplier, error) {
	var s Supplier
	s.ID = id
	err := r.db.QueryRow(ctx, `
		SELECT name, email, kundennummer 
		FROM lieferanten 
		WHERE id = $1
	`, id).Scan(&s.Name, &s.Email, &s.Kundennummer)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

package repository

import (
	"context"
	"bibliothek/db"
)

// Supplier repräsentiert einen Lieferanten (z. B. eine Buchhandlung).
type Supplier struct {
	ID           string
	Name         string
	Email        string
	Kundennummer string

	// LiefertMitBarcode: Der Händler beklebt die Bücher vor der Lieferung mit UNSEREN
	// Barcodes. Der Barcodebogen geht weiterhin mit dem Bestellbrief mit — er braucht ihn
	// ja dafür. Die Exemplare entstehen dann aber bereits als „Etikett vorhanden" und
	// erscheinen nicht auf der Nachdruck-Liste (siehe api/order_service.go).
	LiefertMitBarcode bool
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
		SELECT name, email, kundennummer, liefert_mit_barcode
		FROM lieferanten
		WHERE id = $1
	`, id).Scan(&s.Name, &s.Email, &s.Kundennummer, &s.LiefertMitBarcode)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

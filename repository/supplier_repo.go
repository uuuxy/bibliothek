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

	// IstHauptlieferant: Der EINE Händler, über den die Schule bestellt (z. B. Naacher).
	// Er bekommt statt der reinen Bestellmail den Bestelllink, wählt darüber die
	// Etikettengröße, beklebt die Bücher selbst und bestätigt damit die Bestellung.
	// Deshalb hängen drei Dinge an diesem einen Merkmal:
	//
	//   - der Bestätigungs-Token und das große Lernmittel-Etikett im Mailanhang
	//     (api/order_service.go, api/pdf_service.go),
	//   - die Exemplare entstehen als „Etikett vorhanden" und stehen nicht auf der
	//     Nachdruck-Liste (api/order_service.go),
	//   - die Vorauswahl im Bestellformular.
	//
	// Vorher waren das drei einzelne Schalter. Sie beschrieben denselben Händler, mussten
	// aber einzeln gesetzt werden — und „Bestelllink, aber nicht beklebt" hiess: Der
	// Händler beklebt, die Bibliothek druckt trotzdem noch einmal. Siehe Migration 066.
	IstHauptlieferant bool
}

// SupplierRepository definiert die Datenbank-Zugriffe für Lieferanten.
type SupplierRepository interface {
	GetSupplierByID(ctx context.Context, id string) (*Supplier, error)
	HatHauptlieferant(ctx context.Context) (bool, error)
}

type pgSupplierRepository struct {
	db db.PgxPoolIface
}

// NewSupplierRepository erstellt eine neue Instanz des SupplierRepositorys.
func NewSupplierRepository(pool db.PgxPoolIface) SupplierRepository {
	return &pgSupplierRepository{db: pool}
}

// HatHauptlieferant meldet, ob überhaupt ein Hauptlieferant eingerichtet ist.
//
// Nur diese eine Frage, kein geladener Datensatz: Die Oberfläche braucht sie, um den
// Hinweis auf die fehlende öffentliche Adresse zu zeigen — der ergibt ohne Hauptlieferant
// keinen Sinn, weil dann ohnehin niemand einen Bestätigungs-Link bekäme.
func (r *pgSupplierRepository) HatHauptlieferant(ctx context.Context) (bool, error) {
	var vorhanden bool
	err := r.db.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM lieferanten WHERE ist_hauptlieferant)`).Scan(&vorhanden)
	return vorhanden, err
}

// GetSupplierByID lädt einen Lieferanten anhand seiner ID.
func (r *pgSupplierRepository) GetSupplierByID(ctx context.Context, id string) (*Supplier, error) {
	var s Supplier
	s.ID = id
	err := r.db.QueryRow(ctx, `
		SELECT name, email, kundennummer, ist_hauptlieferant
		FROM lieferanten
		WHERE id = $1
	`, id).Scan(&s.Name, &s.Email, &s.Kundennummer, &s.IstHauptlieferant)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

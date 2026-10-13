package repository

// InventurAbfragen handles all database interactions required during a physical
// book inventory (stock-take). Der Fortschritt lebt seit Migration 045 in Sessions
// (inventur_sessions / inventur_erfassungen) statt in einer globalen Spalte; die
// Lebenszyklus-Methoden dazu stehen in inventurlauf.go und
// inventurlauf_abschluss.go.
type InventurAbfragen struct {
	db Datenbank
}

// NewInventurAbfragen initializes a new InventurAbfragen.
func NewInventurAbfragen(db Datenbank) *InventurAbfragen {
	return &InventurAbfragen{db: db}
}

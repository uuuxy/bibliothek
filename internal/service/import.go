package service

import (
	"bibliothek/db"
	"bibliothek/repository"
)

// ImportDienst liest Titel und Exemplare aus den Dateien von Littera und aus Tabellen (CSV,
// XLSX) in den Katalog.
type ImportDienst struct {
	katalogAbfragen repository.KatalogAbfragen
	pool            db.PgxPoolIface
}

// NewImportDienst baut den Dienst für die Importe.
func NewImportDienst(katalogAbfragen repository.KatalogAbfragen, pool db.PgxPoolIface) *ImportDienst {
	return &ImportDienst{katalogAbfragen: katalogAbfragen, pool: pool}
}

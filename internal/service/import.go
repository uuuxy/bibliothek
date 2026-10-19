package service

import (
	"bibliothek/db"
)

// ImportDienst liest Titel und Exemplare aus den Dateien von Littera und aus Tabellen (CSV,
// XLSX) in den Katalog.
type ImportDienst struct {
	katalogAbfragen titelImport
	pool            db.Pool
}

// NewImportDienst baut den Dienst für die Importe.
func NewImportDienst(katalogAbfragen titelImport, pool db.Pool) *ImportDienst {
	return &ImportDienst{katalogAbfragen: katalogAbfragen, pool: pool}
}

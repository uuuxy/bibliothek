package service

import (
	"bibliothek/db"
	"bibliothek/repository"
)

// ImportDienst stellt Geschäftslogik für Datenimporte bereit.
type ImportDienst struct {
	katalogAbfragen repository.KatalogAbfragen
	pool            db.PgxPoolIface
}

// NewImportDienst erstellt einen neuen ImportDienst.
func NewImportDienst(katalogAbfragen repository.KatalogAbfragen, pool db.PgxPoolIface) *ImportDienst {
	return &ImportDienst{katalogAbfragen: katalogAbfragen, pool: pool}
}

package repository

import (
	"bibliothek/internal/db"
)

// KatalogAbfragen liest und schreibt Titel und Exemplare: die Suche, das Anlegen und den Status
// eines Exemplars.
type KatalogAbfragen struct {
	db db.Pool
}

// NewKatalogAbfragen liefert die Abfragen des Katalogs an Postgres.
func NewKatalogAbfragen(db db.Pool) *KatalogAbfragen {
	return &KatalogAbfragen{db: db}
}

package service

import (
	"context"

	"bibliothek/db"
	"bibliothek/pkg/bestelllink"
	"bibliothek/repository"
)

// BestelllinkTage liest die Frist, die ein neuer Bestätigungs-Link bekommt. Lässt sich die
// Einstellung nicht lesen, gilt die Vorgabe: Die Bestellung scheitert daran nicht, und ein Link
// ohne Frist entsteht nicht.
func BestelllinkTage(ctx context.Context, pool db.PgxPoolIface) int {
	einstellungen, err := repository.NewEinstellungsAbfragen(pool).Lade(ctx)
	if err != nil || einstellungen == nil {
		return bestelllink.TageVorgabe
	}
	return bestelllink.Tage(einstellungen.BestelllinkGueltigkeitTage)
}

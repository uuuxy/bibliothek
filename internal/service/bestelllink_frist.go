package service

import (
	"context"

	"bibliothek/internal/db"
	"bibliothek/internal/pkg/bestelllink"
	"bibliothek/internal/repository"
)

// LadeBestelllinkTage liest die Frist, die ein neuer Bestätigungs-Link bekommt. Lässt sich die
// Einstellung nicht lesen, gilt die Vorgabe: Die Bestellung scheitert daran nicht, und ein Link
// ohne Frist entsteht nicht.
func LadeBestelllinkTage(ctx context.Context, pool db.Pool) int {
	einstellungen, err := repository.NewEinstellungsAbfragen(pool).Lade(ctx)
	if err != nil || einstellungen == nil {
		return bestelllink.TageVorgabe
	}
	return bestelllink.Tage(einstellungen.BestelllinkGueltigkeitTage)
}

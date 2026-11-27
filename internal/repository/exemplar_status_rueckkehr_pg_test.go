package repository

import (
	"context"
	"testing"

	"bibliothek/internal/pgtest"
)

// Ein verlorenes Buch, das in der Buchakte wieder auf „Verfügbar“ oder „Gesperrt“ gesetzt wird,
// ist zurück wie an der Theke: Die offene Forderung für das nicht zurückgegebene Buch wird
// storniert. Die Buchakte ist der Weg zurück für ein ausgesondertes Exemplar (ExemplarKarte).
func TestAendereExemplarStatus_HoltVerlorenesBuchZurueck(t *testing.T) {
	faelle := map[string]ExemplarStatusStand{
		"Verfügbar": {IstAusleihbar: true},
		"Gesperrt":  {},
	}
	for name, neu := range faelle {
		t.Run(name, func(t *testing.T) {
			pool := pgtest.Pool(t)
			ctx := context.Background()
			exemplarID, schadensfallID, bearbeiterID := rueckholenAufbau(t, pool)

			befund, err := NewKatalogAbfragen(pool).AendereExemplarStatus(ctx, exemplarID, nil, neu, "", nil, bearbeiterID)
			if err != nil {
				t.Fatalf("Status ändern: %v", err)
			}
			if befund.StornierteForderungen != 1 {
				t.Errorf("%d stornierte Forderungen gemeldet, erwartet 1", befund.StornierteForderungen)
			}
			var offen bool
			if err := pool.QueryRow(ctx, `SELECT storniert_am IS NULL FROM schadensfaelle WHERE id = $1`,
				schadensfallID).Scan(&offen); err != nil {
				t.Fatalf("Forderung lesen: %v", err)
			}
			if offen {
				t.Error("Die Forderung für das nicht zurückgegebene Buch ist noch offen, das Buch ist aber zurück")
			}
		})
	}
}

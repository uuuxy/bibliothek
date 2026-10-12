package testdaten

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Werte für Tests am Eigentumsvermerk eines Buchetiketts.
const (
	VermerkLand  = "Eigentum des Landes Hessen" // Werksvorgabe, nichts hinterlegt
	VermerkStadt = "Eigentum der Stadt Friedrichsdorf"
	// 70 × 36 mm: ab 30 mm Höhe steht der Eigentumsvermerk auf dem kleinen Etikett.
	EtikettFormatMitVermerk = "avery_3475"
)

// SetzeVermerkSchuelerbuecherei hinterlegt den Eigentumsvermerk für Bücher aus Mitteln des
// Schulträgers direkt in der Tabelle und nimmt ihn am Ende des Tests zurück.
func SetzeVermerkSchuelerbuecherei(t *testing.T, pool *pgxpool.Pool, wert string) {
	t.Helper()
	ctx := context.Background()
	if _, err := pool.Exec(ctx, `
		INSERT INTO system_einstellungen (schluessel, wert)
		VALUES ('etikett_eigentumsvermerk_schuelerbuecherei', $1)
		ON CONFLICT (schluessel) DO UPDATE SET wert = EXCLUDED.wert`, wert); err != nil {
		t.Fatalf("Einstellung setzen: %v", err)
	}
	t.Cleanup(func() {
		// Ein liegen gebliebener Wert färbte jeden späteren Etiketten-Test.
		if _, err := pool.Exec(ctx, `DELETE FROM system_einstellungen
			WHERE schluessel = 'etikett_eigentumsvermerk_schuelerbuecherei'`); err != nil {
			t.Errorf("Einstellung zurücknehmen: %v", err)
		}
	})
}

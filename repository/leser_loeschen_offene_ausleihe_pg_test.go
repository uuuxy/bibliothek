package repository

import (
	"context"
	"testing"
	"time"

	"bibliothek/db"
)

// Das endgültige Löschen eines Lesers löst eine offene Ausleihe nicht von ihm: Der Fremdschlüssel
// hält das Löschen auf, und die Ausleihe steht weiter auf ihm. Die Rufer prüfen vorher; der Test
// ruft die Funktion an ihnen vorbei.
func TestTilgeSpurenUndLoescheLeser_OffeneAusleiheHaeltAuf(t *testing.T) {
	pool := pgTestPool(t)
	ctx := context.Background()
	f := stempelAufbau(t, pool)
	vorher := beginne(t, pool)
	defer db.Verwerfe(ctx, vorher)
	if _, err := LegeAusleiheAn(ctx, vorher, AusleiheNeu{
		ExemplarID: f.exemplarID, LeserID: f.schuelerID, BearbeiterID: f.bearbeiterID,
		RueckgabeFrist: time.Now().AddDate(0, 0, 21),
	}); err != nil {
		t.Fatalf("LegeAusleiheAn: %v", err)
	}
	if err := vorher.Commit(ctx); err != nil {
		t.Fatalf("Ausleihe festschreiben: %v", err)
	}

	loeschen := beginne(t, pool)
	defer db.Verwerfe(ctx, loeschen)
	if err := tilgeSpurenUndLoescheLeser(ctx, loeschen, f.schuelerID, f.bearbeiterID, "Test"); err == nil {
		t.Errorf("tilgeSpurenUndLoescheLeser bei offener Ausleihe = nil, erwartet den Fehler des Fremdschlüssels")
	}
	db.Verwerfe(ctx, loeschen)

	var leser *string
	if err := pool.QueryRow(ctx, `SELECT schueler_id::text FROM ausleihen WHERE exemplar_id = $1 AND rueckgabe_am IS NULL`,
		f.exemplarID).Scan(&leser); err != nil {
		t.Fatalf("Ausleihe lesen: %v", err)
	}
	if leser == nil || *leser != f.schuelerID {
		t.Errorf("Leser der offenen Ausleihe = %v, erwartet %s", leser, f.schuelerID)
	}
}

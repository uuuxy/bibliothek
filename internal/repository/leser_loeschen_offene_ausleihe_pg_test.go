package repository

import (
	"context"
	"testing"
	"time"

	"bibliothek/internal/db"
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

// Zwilling für die Forderung: Eine unbezahlte Forderung, die nach der Prüfung der Rufer entstand
// (etwa „beschädigt zurückgegeben“ an einer beendeten Ausleihe), bleibt stehen, und ihr
// Fremdschlüssel hält das Löschen auf. Eine erledigte fällt mit dem Leser.
func TestTilgeSpurenUndLoescheLeser_UnbezahlteForderungHaeltAuf(t *testing.T) {
	pool := pgTestPool(t)
	ctx := context.Background()
	f := stempelAufbau(t, pool)
	var offen, erledigt string
	if err := pool.QueryRow(ctx, `INSERT INTO schadensfaelle (exemplar_id, schueler_id, beschreibung, betrag, art)
		VALUES ($1, $2, 'offen', 5, 'beschaedigt') RETURNING id::text`, f.exemplarID, f.schuelerID).Scan(&offen); err != nil {
		t.Fatalf("offene Forderung anlegen: %v", err)
	}
	if err := pool.QueryRow(ctx, `INSERT INTO schadensfaelle (exemplar_id, schueler_id, beschreibung, betrag, art, ist_bezahlt)
		VALUES ($1, $2, 'erledigt', 5, 'beschaedigt', true) RETURNING id::text`, f.exemplarID, f.schuelerID).Scan(&erledigt); err != nil {
		t.Fatalf("erledigte Forderung anlegen: %v", err)
	}

	loeschen := beginne(t, pool)
	defer db.Verwerfe(ctx, loeschen)
	if err := tilgeSpurenUndLoescheLeser(ctx, loeschen, f.schuelerID, f.bearbeiterID, "Test"); err == nil {
		t.Errorf("tilgeSpurenUndLoescheLeser bei unbezahlter Forderung = nil, erwartet den Fehler des Fremdschlüssels")
	}
	db.Verwerfe(ctx, loeschen)

	var schuldner *string
	if err := pool.QueryRow(ctx, `SELECT schueler_id::text FROM schadensfaelle WHERE id = $1 AND NOT ist_bezahlt`,
		offen).Scan(&schuldner); err != nil {
		t.Fatalf("offene Forderung lesen: %v", err)
	}
	if schuldner == nil || *schuldner != f.schuelerID {
		t.Errorf("Schuldner der offenen Forderung = %v, erwartet %s", schuldner, f.schuelerID)
	}

	// Gegenprobe: Ist die Forderung bezahlt, fällt sie mit dem Leser, und das Löschen gelingt.
	if _, err := pool.Exec(ctx, `UPDATE schadensfaelle SET ist_bezahlt = true WHERE id = $1`, offen); err != nil {
		t.Fatalf("Forderung bezahlen: %v", err)
	}
	loeschen2 := beginne(t, pool)
	defer db.Verwerfe(ctx, loeschen2)
	if err := tilgeSpurenUndLoescheLeser(ctx, loeschen2, f.schuelerID, f.bearbeiterID, "Test"); err != nil {
		t.Fatalf("tilgeSpurenUndLoescheLeser ohne offene Forderung: %v", err)
	}
	if err := loeschen2.Commit(ctx); err != nil {
		t.Fatalf("Löschen festschreiben: %v", err)
	}
	var uebrig int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM schadensfaelle WHERE id = ANY($1)`, []string{offen, erledigt}).Scan(&uebrig); err != nil {
		t.Fatalf("Forderungen zählen: %v", err)
	}
	if uebrig != 0 {
		t.Errorf("%d erledigte Forderungen stehen nach dem Löschen noch da, erwartet keine", uebrig)
	}
}

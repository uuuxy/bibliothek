package api

import (
	"context"
	"testing"
	"time"

	"bibliothek/repository"
)

// Die Zählung des Wächters am echten Postgres: Es zählt, wer weg ist, seit über einem
// Jahr, und noch ein offenes Buch ODER eine unbezahlte Forderung hat. Nicht: der frisch
// Weggegangene, der mit bezahlter Forderung, der Aktive, der Gelöschte.
func TestZaehleEhemaligeMitOffenenVorgaengen(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	ctx := context.Background()

	langWeg := time.Now().AddDate(0, 0, -400)
	kurzWeg := time.Now().AddDate(0, 0, -30)
	weg := func(barcode, vorname string, seit time.Time) string {
		id := seedSchueler(t, pool, barcode, vorname, "ABG")
		if _, err := pool.Exec(ctx, `UPDATE schueler SET ist_abgaenger = true, abgaenger_seit = $2 WHERE id = $1`, id, seit); err != nil {
			t.Fatal(err)
		}
		return id
	}
	buchOffen := weg("W-1", "BuchOffen", langWeg)
	seedAusleihe(t, pool, buchOffen, "Buch W1", time.Now().AddDate(0, 0, -300))
	forderungOffen := weg("W-2", "ForderungOffen", langWeg)
	forderungBezahlt := weg("W-3", "ForderungBezahlt", langWeg)
	for _, f := range []struct {
		id      string
		barcode string
		bezahlt bool
	}{{forderungOffen, "BC-W2", false}, {forderungBezahlt, "BC-W3", true}} {
		// Eine Forderung hängt immer an einem Exemplar (check_damage_item).
		eid := exemplar(t, pool, titelMitMeldebestand(t, pool, "Titel "+f.barcode, 1), f.barcode, true, "")
		if _, err := pool.Exec(ctx, `
			INSERT INTO schadensfaelle (exemplar_id, schueler_id, beschreibung, betrag, ist_bezahlt)
			VALUES ($1, $2, 'Verlust', 12.50, $3)`, eid, f.id, f.bezahlt); err != nil {
			t.Fatalf("Schadensfall: %v", err)
		}
	}
	frisch := weg("W-4", "FrischWeg", kurzWeg)
	seedAusleihe(t, pool, frisch, "Buch W4", time.Now().AddDate(0, 0, -10))
	aktiv := seedSchueler(t, pool, "W-5", "Aktiv", "9H1")
	seedAusleihe(t, pool, aktiv, "Buch W5", time.Now().AddDate(0, 0, -400))
	geloescht := weg("W-6", "Geloescht", langWeg)
	seedAusleihe(t, pool, geloescht, "Buch W6", time.Now().AddDate(0, 0, -300))
	if _, err := pool.Exec(ctx, `UPDATE schueler SET deleted_at = now() WHERE id = $1`, geloescht); err != nil {
		t.Fatal(err)
	}

	n, err := repository.NewBetriebszustandRepository(pool).ZaehleEhemaligeMitOffenenVorgaengen(ctx, ehemaligeOffenSeitTagen)
	if err != nil {
		t.Fatalf("zählen: %v", err)
	}
	if n != 2 {
		t.Errorf("erwartet 2 (BuchOffen, ForderungOffen), gezählt %d", n)
	}
}

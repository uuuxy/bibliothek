package repository

import (
	"context"
	"testing"

	"bibliothek/internal/pgtest"
)

// Eine ISBN, die kein Dienst kennt, fragt der Cover-Abgleich nicht wieder (NOT_FOUND). Wird sie
// in der Titel-Verwaltung berichtigt, gilt dieser Stand nicht mehr: Der nächste Lauf fragt die
// neue ISBN. Ein anderes Feld und dieselbe ISBN lassen den Stand stehen.
func TestAendereTitel_NeueISBNFragtDenCoverAbgleichWieder(t *testing.T) {
	pool := pgtest.Pool(t)
	ctx := context.Background()
	abfragen := NewTitelAbfragen(pool)

	const alt, neu = "9780000141411", "9780000141428"
	t.Cleanup(func() {
		if _, err := pool.Exec(context.Background(),
			`DELETE FROM buecher_titel WHERE isbn IN ($1, $2)`, alt, neu); err != nil {
			t.Errorf("aufräumen: %v", err)
		}
	})
	id, err := abfragen.LegeTitelAn(ctx, KatalogTitel{ISBN: alt, Titel: "Cover-Probe", Autor: "Test"})
	if err != nil {
		t.Fatalf("Titel anlegen: %v", err)
	}
	if _, err := pool.Exec(ctx, `UPDATE buecher_titel SET cover_status = 'NOT_FOUND' WHERE id = $1`, id); err != nil {
		t.Fatal(err)
	}
	stand := func() string {
		t.Helper()
		var status string
		if err := pool.QueryRow(ctx, `SELECT cover_status FROM buecher_titel WHERE id = $1`, id).Scan(&status); err != nil {
			t.Fatal(err)
		}
		return status
	}

	for _, schritt := range []struct {
		name    string
		eintrag KatalogTitel
		felder  []string
		stand   string
	}{
		{"ein anderes Feld", KatalogTitel{ISBN: neu, Titel: "Cover-Probe, berichtigt"}, []string{"title"}, "NOT_FOUND"},
		{"dieselbe ISBN", KatalogTitel{ISBN: alt}, []string{"isbn"}, "NOT_FOUND"},
		{"dieselbe ISBN mit Bindestrichen", KatalogTitel{ISBN: "978-0-00-014141-1"}, []string{"isbn"}, "NOT_FOUND"},
		{"eine neue ISBN", KatalogTitel{ISBN: neu}, []string{"isbn"}, "PENDING"},
	} {
		if err := abfragen.AendereTitel(ctx, id, schritt.eintrag, schritt.felder, nil); err != nil {
			t.Fatalf("%s: AendereTitel(…) = %v", schritt.name, err)
		}
		if got := stand(); got != schritt.stand {
			t.Errorf("%s: cover_status = %q, erwartet %q", schritt.name, got, schritt.stand)
		}
	}
}

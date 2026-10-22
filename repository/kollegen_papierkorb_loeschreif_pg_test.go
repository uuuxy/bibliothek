package repository_test

import (
	"context"
	"testing"

	"bibliothek/internal/pgtest"
	"bibliothek/repository"
)

// Der nächtliche Lauf liest die löschreifen Kollegen vorher und löscht danach einzeln. Wer
// dazwischen wiederhergestellt und neu gelöscht wurde, liegt erst Sekunden im Papierkorb; die
// Löschung stellt deshalb die Frage der Liste noch einmal (praedikatKollegenPapierkorb) und lässt
// ihn stehen.
func TestLoescheKollegenAusDemPapierkorb_FragtBeimLoeschenNochEinmal(t *testing.T) {
	pool := pgtest.Pool(t)
	ctx := context.Background()
	repo := repository.NewLeserAbfragen(pool)

	kollege := func(name string, tage int) string {
		t.Helper()
		var id string
		if err := pool.QueryRow(ctx, `
			INSERT INTO leser (art, vorname, nachname, deleted_at)
			VALUES ('lehrkraft', 'Probe', $1, NOW() - make_interval(days => $2::int)) RETURNING id`,
			name, tage).Scan(&id); err != nil {
			t.Fatalf("Kollege %s anlegen: %v", name, err)
		}
		t.Cleanup(func() {
			if _, err := pool.Exec(context.Background(), `DELETE FROM leser WHERE id = $1`, id); err != nil {
				t.Errorf("Aufräumen: %v", err)
			}
		})
		return id
	}
	gibtEs := func(id string) bool {
		t.Helper()
		var da bool
		if err := pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM leser WHERE id = $1)`, id).Scan(&da); err != nil {
			t.Fatal(err)
		}
		return da
	}

	frisch := kollege("Frisch-Geloescht", 0)
	if weg, err := repo.LoescheKollegenAusDemPapierkorb(ctx, frisch); err != nil || weg {
		t.Errorf("LoescheKollegenAusDemPapierkorb(seit heute im Papierkorb) = %v, %v, erwartet false ohne Fehler", weg, err)
	}
	if !gibtEs(frisch) {
		t.Error("ein Kollege, der erst seit heute im Papierkorb liegt, ist endgültig gelöscht")
	}

	alt := kollege("Lange-Geloescht", 200)
	if weg, err := repo.LoescheKollegenAusDemPapierkorb(ctx, alt); err != nil || !weg {
		t.Errorf("LoescheKollegenAusDemPapierkorb(200 Tage im Papierkorb) = %v, %v, erwartet true ohne Fehler", weg, err)
	}
	if gibtEs(alt) {
		t.Error("ein Kollege nach 200 Tagen im Papierkorb ist noch da")
	}
}

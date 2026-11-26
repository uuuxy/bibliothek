package repository

import (
	"context"
	"testing"

	"bibliothek/internal/testdaten"
)

// Ein Konto ohne Leserzeile ist ein gültiger Zustand: Die Tilgung eines Lesers setzt die
// Verknüpfung auf NULL, die Anmeldung bleibt. Sein Name lässt sich trotzdem ändern; eine
// Leserzeile, die es nicht gibt, wird nicht angefasst.
func TestAendere_NameEinesKontosOhneLeserzeile(t *testing.T) {
	pool := pgTestPool(t)
	ctx := context.Background()
	if _, err := pool.Exec(ctx, `DELETE FROM benutzer WHERE email = 'ohneleser@kontoname.test'`); err != nil {
		t.Fatal(err)
	}
	id := testdaten.LegeKontoAn(t, pool, "Ohne", "Leser", "ohneleser@kontoname.test", "helfer", true)
	// Das Anlegen gibt dem Konto eine Leserzeile; die Verknüpfung fällt wie bei der Tilgung.
	var leser string
	if err := pool.QueryRow(ctx, `SELECT leser_id::text FROM benutzer WHERE id = $1`, id).Scan(&leser); err != nil {
		t.Fatalf("Leserzeile des Kontos lesen: %v", err)
	}
	t.Cleanup(func() {
		if _, err := pool.Exec(context.Background(), `DELETE FROM benutzer WHERE id = $1`, id); err != nil {
			t.Logf("Aufräumen: %v", err)
		}
		if _, err := pool.Exec(context.Background(), `DELETE FROM leser WHERE id = $1`, leser); err != nil {
			t.Logf("Aufräumen: %v", err)
		}
	})
	if _, err := pool.Exec(ctx, `UPDATE benutzer SET leser_id = NULL WHERE id = $1`, id); err != nil {
		t.Fatalf("Verknüpfung lösen: %v", err)
	}

	neu := "Umbenannt"
	if _, err := NewKontoAbfragen(pool).Aendere(ctx, KontoAenderung{ID: id, Vorname: &neu}); err != nil {
		t.Fatalf("Aendere(Vorname) = %v, erwartet Erfolg", err)
	}
	var vorname string
	if err := pool.QueryRow(ctx, `SELECT vorname FROM benutzer WHERE id = $1`, id).Scan(&vorname); err != nil || vorname != neu {
		t.Errorf("Vorname = %q, %v; erwartet %q", vorname, err, neu)
	}
}

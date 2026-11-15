package repository

import (
	"context"
	"errors"
	"testing"

	"bibliothek/internal/testdaten"
)

// Der Adressabgleich trägt eine Adresse nur ein, solange das Konto noch seinen Platzhalter trägt
// (KontoAenderung.EmailBisher). Hat jemand die Adresse zwischen Berechnen und Eintragen von Hand
// gesetzt, bleibt sie stehen.
func TestAendere_EmailBisherSchuetztEineInzwischenGeaenderteAdresse(t *testing.T) {
	pool := pgTestPool(t)
	ctx := context.Background()
	id := testdaten.LegeKontoAn(t, pool, "Bisher", "Probe", "littera-9101@littera.invalid", "kollegium", true)
	t.Cleanup(func() {
		if _, err := pool.Exec(context.Background(), `DELETE FROM benutzer WHERE id = $1`, id); err != nil {
			t.Logf("Aufräumen: %v", err)
		}
	})
	repo := NewKontoAbfragen(pool)

	// Von Hand eingetragen, während der Abgleich noch mit dem Platzhalter rechnete.
	hand := "bisher.hand@schule.example"
	if _, err := repo.Aendere(ctx, KontoAenderung{ID: id, Email: &hand}); err != nil {
		t.Fatal(err)
	}
	abgleich, platzhalter := "bisher@schule.example", "littera-9101@littera.invalid"
	_, err := repo.Aendere(ctx, KontoAenderung{ID: id, Email: &abgleich, EmailBisher: &platzhalter})
	if !errors.Is(err, ErrKontoNichtGefunden) {
		t.Errorf("Aendere mit veraltetem EmailBisher: Fehler %v, erwartet ErrKontoNichtGefunden", err)
	}
	var email string
	if err := pool.QueryRow(ctx, `SELECT email FROM benutzer WHERE id = $1`, id).Scan(&email); err != nil {
		t.Fatal(err)
	}
	if email != hand {
		t.Errorf("Adresse nach dem Abgleich %q, erwartet die von Hand gesetzte %q", email, hand)
	}

	// Mit passendem EmailBisher wird geändert.
	if _, err := repo.Aendere(ctx, KontoAenderung{ID: id, Email: &abgleich, EmailBisher: &hand}); err != nil {
		t.Errorf("Aendere mit passendem EmailBisher: %v", err)
	}
}

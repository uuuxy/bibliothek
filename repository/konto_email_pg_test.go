package repository

import (
	"bibliothek/internal/testdaten"
	"context"
	"testing"
)

// Die Anmeldung findet ein Konto über LOWER(email) = LOWER($1) LIMIT 1 (LadeAnmeldeKonto). Die
// Eindeutigkeit gilt deshalb in derselben Normalform: Sonst entstehen zwei Konten, die sich nur
// in der Schreibweise unterscheiden, und welches die Anmeldung öffnet, entscheidet die
// Reihenfolge in der Tabelle. Bei einem genauen Vergleich legte eine Selbstanmeldung
// „erika.muster@…" an (inaktiv) und die Bibliothek daneben „Erika.Muster@…" als Mitarbeiterin.
func TestIstEmailVergeben_IgnoriertGrossKleinschreibung(t *testing.T) {
	pool := pgTestPool(t)
	ctx := context.Background()
	if _, err := pool.Exec(ctx, `DELETE FROM benutzer WHERE email ILIKE 'erika.muster@%'`); err != nil {
		t.Fatal(err)
	}
	id := testdaten.LegeKontoAn(t, pool, "Erika", "Muster", "erika.muster@schule.example", "kollegium", false)
	t.Cleanup(func() {
		if _, err := pool.Exec(context.Background(), `DELETE FROM benutzer WHERE id = $1`, id); err != nil {
			t.Logf("Aufräumen: %v", err)
		}
	})

	repo := NewKontoAbfragen(pool)
	for _, variante := range []string{"Erika.Muster@schule.example", "ERIKA.MUSTER@SCHULE.EXAMPLE"} {
		gibt, err := repo.IstEmailVergeben(ctx, variante, "")
		if err != nil {
			t.Fatal(err)
		}
		if !gibt {
			t.Errorf("%q gilt als frei, obwohl %q existiert", variante, "erika.muster@schule.example")
		}
	}
	// Die eigene Adresse in anderer Schreibweise ist beim Bearbeiten kein Konflikt.
	if gibt, err := repo.IstEmailVergeben(ctx, "Erika.Muster@schule.example", id); err != nil || gibt {
		t.Errorf("eigene Adresse in anderer Schreibweise: vergeben=%v err=%v, erwartet false", gibt, err)
	}
	// Die zweite Sicherung ist die Datenbank selbst (Index uniq_benutzer_email_lower).
	if _, err := pool.Exec(ctx, `INSERT INTO benutzer (vorname, nachname, email, rolle, aktiv)
		VALUES ('Erika', 'Doppelt', 'Erika.Muster@schule.example', 'mitarbeiter', true)`); err == nil {
		t.Errorf("die Datenbank nahm eine zweite Schreibweise derselben Adresse an")
		if _, err := pool.Exec(ctx, `DELETE FROM benutzer WHERE email = 'Erika.Muster@schule.example'`); err != nil {
			t.Logf("Aufräumen: %v", err)
		}
	}
}

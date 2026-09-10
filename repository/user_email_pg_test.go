package repository

import (
	"context"
	"testing"
)

// Die Anmeldung findet ein Konto über LOWER(email) = LOWER($1) LIMIT 1 (auth/handlers.go).
// Die Eindeutigkeit muss deshalb in DERSELBEN Normalform gelten — sonst entstehen zwei
// Konten, die sich nur in der Schreibweise unterscheiden, und welches der Login öffnet,
// entscheidet die Speicherreihenfolge der Tabelle (ohne ORDER BY).
//
// Bis zum 10.09.2026 prüfte CheckEmailExists exakt, und UNIQUE lag auf dem Rohtext: Eine
// Selbstanmeldung legte „erika.muster@…" an (inaktiv), die Bibliothek legte daneben
// „Erika.Muster@…" als Mitarbeiterin an — beides ging durch (Bestands-Durchgang,
// Bugklasse „Normalform-Asymmetrie Prüfer ↔ Leser").
func TestCheckEmailExists_IgnoriertGrossKleinschreibung(t *testing.T) {
	pool := pgTestPool(t)
	ctx := context.Background()
	if _, err := pool.Exec(ctx, `DELETE FROM benutzer WHERE email ILIKE 'erika.muster@%'`); err != nil {
		t.Fatal(err)
	}
	var id string
	if err := pool.QueryRow(ctx, `INSERT INTO benutzer (vorname, nachname, email, rolle, aktiv)
		VALUES ('Erika', 'Muster', 'erika.muster@schule.example', 'kollegium', false) RETURNING id`).Scan(&id); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if _, err := pool.Exec(context.Background(), `DELETE FROM benutzer WHERE id = $1`, id); err != nil {
			t.Logf("Aufräumen: %v", err)
		}
	})

	repo := NewUserRepository(pool)
	for _, variante := range []string{"Erika.Muster@schule.example", "ERIKA.MUSTER@SCHULE.EXAMPLE"} {
		gibt, err := repo.CheckEmailExists(ctx, variante, "")
		if err != nil {
			t.Fatal(err)
		}
		if !gibt {
			t.Errorf("%q gilt als frei, obwohl %q existiert", variante, "erika.muster@schule.example")
		}
	}
	// Die eigene Adresse in anderer Schreibweise ist beim Bearbeiten kein Konflikt.
	if gibt, err := repo.CheckEmailExists(ctx, "Erika.Muster@schule.example", id); err != nil || gibt {
		t.Errorf("eigene Adresse in anderer Schreibweise: exists=%v err=%v, want false", gibt, err)
	}
	// Die zweite Tür: die Datenbank selbst (Migration 113).
	if _, err := pool.Exec(ctx, `INSERT INTO benutzer (vorname, nachname, email, rolle, aktiv)
		VALUES ('Erika', 'Doppelt', 'Erika.Muster@schule.example', 'mitarbeiter', true)`); err == nil {
		t.Errorf("die Datenbank nahm eine zweite Schreibweise derselben Adresse an")
		if _, err := pool.Exec(ctx, `DELETE FROM benutzer WHERE email = 'Erika.Muster@schule.example'`); err != nil {
			t.Logf("Aufräumen: %v", err)
		}
	}
}

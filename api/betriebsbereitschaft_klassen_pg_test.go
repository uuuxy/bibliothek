package api

import (
	"context"
	"fmt"
	"slices"
	"testing"

	"bibliothek/db"
	"bibliothek/repository"
)

// Der Live-Pfad der Klassen-Prüfung: Schüler, Zuordnungen und Buchlisten anlegen, dann
// sammleLage. Die Oberstufe (ET, 12T, 13T) hat keine Klassenleitung, ihre Klassen sind aber
// Klassen: Eine Buchliste oder Zuordnung dafür ist nicht verwaist.
func TestKlassenZuordnungKenntDieOberstufe(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	ctx := context.Background()
	srv := &Server{DB: &db.Database{Pool: pool}}

	for i, klasse := range []string{"07G1", "10R2", "ET1", "12T1", "13T2"} {
		seedSchueler(t, pool, fmt.Sprintf("KL-%d", i), "Kind", klasse)
	}
	titel := titelMitMeldebestand(t, pool, "Buch einer Klassenliste", 1)
	if _, err := pool.Exec(ctx,
		`INSERT INTO class_books (class_name, book_id) VALUES ('ET1', $1), ('09H9', $1)`, titel); err != nil {
		t.Fatalf("Buchlisten anlegen: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO klassen_lehrer_mapping (klasse, lehrer_email) VALUES
			('10R2', 'leitung@schule.test'), ('ET1', 'tutor@schule.test'), ('08G5', 'leer@schule.test')`); err != nil {
		t.Fatalf("Zuordnungen anlegen: %v", err)
	}

	lage := srv.sammleLage(ctx, repository.NewSystemSettingsRepository(pool),
		repository.NewMailSettingsRepository(pool), repository.NewBetriebszustandRepository(pool))

	for _, fall := range []struct {
		name      string
		ist, soll []string
	}{
		{"Klassen ohne Lehrkraft-Zuordnung", lage.KlassenOhneLehrkraft, []string{"07G1"}},
		{"Zuordnungen ohne aktive Schüler", lage.VerwaisteZuordnungen, []string{"08G5"}},
		{"Bücherlisten für unbekannte Klassen", lage.VerwaisteBuecherliste, []string{"09H9"}},
	} {
		if !slices.Equal(fall.ist, fall.soll) {
			t.Errorf("%s: %v, erwartet %v", fall.name, fall.ist, fall.soll)
		}
	}
}

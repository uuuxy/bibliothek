package repository

import (
	"context"
	"errors"
	"testing"
)

// Warum die Aussonderung nichts traf: „noch verliehen" gilt nur, wenn genau dieses Exemplar offen
// verliehen ist. Eine offene Ausleihe irgendeines anderen Buchs, die es im Betrieb immer gibt, zählt
// nicht; ein unbekanntes Exemplar hieße sonst immer „noch verliehen".
func TestDeuteAussonderungsHindernis(t *testing.T) {
	pool := pgTestPool(t)
	resetInventurDaten(t, pool)
	ctx := context.Background()

	ex := legeSignaturMitExemplarenAn(t, pool, "Hindernis", 2)
	legeAusleiheAn(t, pool, ex[0], legeSchuelerAn(t, pool, "HIND-A", "Cora", "8a"), legeBearbeiterAn(t, pool))

	if err := deuteAussonderungsHindernis(ctx, pool, ex[0]); !errors.Is(err, ErrExemplarNochVerliehen) {
		t.Errorf("verliehenes Exemplar: %v, erwartet ErrExemplarNochVerliehen", err)
	}
	if err := deuteAussonderungsHindernis(ctx, pool, ex[1]); !errors.Is(err, ErrExemplarNichtGefunden) {
		t.Errorf("freies Exemplar neben einer offenen Ausleihe: %v, erwartet ErrExemplarNichtGefunden", err)
	}
}

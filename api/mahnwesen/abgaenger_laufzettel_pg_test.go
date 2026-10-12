package mahnwesen

import (
	"context"
	"testing"
	"time"

	"bibliothek/api/tuer"
	"bibliothek/db"
	"bibliothek/internal/pgtest"
	"bibliothek/internal/testdaten"
)

// TestLaufzettel_NurAbgaengerMitBuechern sichert Audit-Bug #2 ab: Der Laufzettel-Massendruck
// darf NUR Abgänger mit noch offenen Büchern enthalten. Vorher (LEFT JOIN) erschien jeder
// Abgänger — 150 Abgänger, davon 140 ohne Buch, ergaben 140 leere Laufzettel.
// Abgänger heißt hier seit dem 05.09.2026 wieder: Abschlussklasse, noch an der Schule.
func TestLaufzettel_NurAbgaengerMitBuechern(t *testing.T) {
	pool := pgtest.Pool(t)
	testdaten.ResetBestandsdaten(t, pool)
	ctx := context.Background()

	// Abschlussklasse MIT offenem Buch.
	mitBuch := testdaten.SeedSchueler(t, pool, "AB-1", "Anna", "10R1")
	testdaten.SeedAusleihe(t, pool, mitBuch, "Physikbuch 10", time.Now().AddDate(0, 0, -3))
	// Abschlussklasse OHNE Buch — darf keinen (leeren) Laufzettel bekommen.
	testdaten.SeedSchueler(t, pool, "AB-2", "Bodo", "10R1")
	// Kein Abschlussjahrgang, mit Buch — darf nicht auftauchen.
	kein := testdaten.SeedSchueler(t, pool, "AB-3", "Cleo", "8a")
	testdaten.SeedAusleihe(t, pool, kein, "Chemiebuch 8", time.Now().AddDate(0, 0, -3))

	srv := &Tueren{Basis: &tuer.Basis{DB: &db.Database{Pool: pool}}}
	students, err := srv.queryAbgaengerKontoauszug(ctx, "", nil)
	if err != nil {
		t.Fatalf("queryAbgaengerKontoauszug: %v", err)
	}

	if len(students) != 1 {
		t.Fatalf("erwartet genau 1 Laufzettel (nur Abgänger mit Buch), waren %d", len(students))
	}
	if students[0].Schueler.Vorname != "Anna" || len(students[0].Buecher) != 1 {
		t.Errorf("falscher/leerer Laufzettel: %+v", students[0])
	}
}

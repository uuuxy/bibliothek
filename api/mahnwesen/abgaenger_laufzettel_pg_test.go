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

// TestLaufzettel_NurAbgaengerMitBuechern: Der Massendruck der Kontoauszüge darf nur Abgänger
// mit noch offenen Büchern enthalten. Mit einem LEFT JOIN erschiene jeder Abgänger, und
// 150 Abgänger, davon 140 ohne Buch, ergäben 140 leere Seiten. Abgänger heißt hier:
// Abschlussklasse, noch an der Schule.
func TestLaufzettel_NurAbgaengerMitBuechern(t *testing.T) {
	pool := pgtest.Pool(t)
	testdaten.LeereBestandsdaten(t, pool)
	ctx := context.Background()

	// Abschlussklasse mit offenem Buch.
	mitBuch := testdaten.LegeSchuelerAn(t, pool, "AB-1", "Anna", "10R1")
	testdaten.LegeAusleiheMitFristAn(t, pool, mitBuch, "Physikbuch 10", time.Now().AddDate(0, 0, -3))
	// Abschlussklasse ohne Buch — darf keinen (leeren) Laufzettel bekommen.
	testdaten.LegeSchuelerAn(t, pool, "AB-2", "Bodo", "10R1")
	// Kein Abschlussjahrgang, mit Buch — darf nicht auftauchen.
	kein := testdaten.LegeSchuelerAn(t, pool, "AB-3", "Cleo", "8a")
	testdaten.LegeAusleiheMitFristAn(t, pool, kein, "Chemiebuch 8", time.Now().AddDate(0, 0, -3))

	srv := &Tueren{Basis: &tuer.Basis{DB: &db.Verbindung{Pool: pool}}}
	students, err := srv.ladeAbgaengerKontoauszuege(ctx, "", nil)
	if err != nil {
		t.Fatalf("ladeAbgaengerKontoauszuege: %v", err)
	}

	if len(students) != 1 {
		t.Fatalf("len(students) = %d, erwartet genau 1 Laufzettel (nur Abgänger mit Buch)", len(students))
	}
	if students[0].Schueler.Vorname != "Anna" || len(students[0].Buecher) != 1 {
		t.Errorf("falscher/leerer Laufzettel: %+v", students[0])
	}
}

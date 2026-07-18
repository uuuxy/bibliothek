package api

import (
	"context"
	"testing"
	"time"

	"bibliothek/db"

	"github.com/jackc/pgx/v5/pgxpool"
)

// TestLaufzettelStudents_AbgaengerFilter sichert den Abgänger-Filter ab: Der Laufzettel muss
// Abgänger (ist_abgaenger = true) unabhängig vom Klassennamen erfassen. Früher fing eine
// hartkodierte, case-sensitive Liste ('9h','10r','13') nur exakt diese Namen — Abgänger in
// '09h', '9H' oder '10a' fehlten auf dem PDF und verließen die Schule mit ihren Büchern.
// Seit Audit #2 gilt zusätzlich: nur Abgänger MIT offenem Buch bekommen einen Laufzettel,
// deshalb hat hier jeder ein Buch (der Nicht-Abgänger ebenfalls, um zu zeigen, dass ihn der
// ist_abgaenger-Filter — nicht die fehlende Ausleihe — aussortiert).
func TestLaufzettelStudents_AbgaengerFilter(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	ctx := context.Background()

	// Drei Abgänger mit unterschiedlich formatierten Klassen, jeweils mit offenem Buch.
	frist := time.Now().AddDate(0, 0, -3)
	anna := seedAbgaengerRet(t, pool, "S-1", "Anna", "09h") // führende Null
	bea := seedAbgaengerRet(t, pool, "S-2", "Bea", "9H")    // Großbuchstabe
	cem := seedAbgaengerRet(t, pool, "S-3", "Cem", "10a")   // anderer Zug
	seedAusleihe(t, pool, anna, "Buch Anna", frist)
	seedAusleihe(t, pool, bea, "Buch Bea", frist)
	seedAusleihe(t, pool, cem, "Buch Cem", frist)
	// Nicht-Abgänger MIT Buch — darf trotz Buch nicht erscheinen (ist_abgaenger-Filter).
	dana := seedSchueler(t, pool, "S-4", "Dana", "7b")
	seedAusleihe(t, pool, dana, "Buch Dana", frist)

	srv := &Server{DB: &db.Database{Pool: pool}}
	studenten, err := srv.queryLaufzettelStudents(ctx)
	if err != nil {
		t.Fatalf("queryLaufzettelStudents: %v", err)
	}

	namen := map[string]bool{}
	for _, s := range studenten {
		namen[s.Vorname] = true
	}

	for _, erwartet := range []string{"Anna", "Bea", "Cem"} {
		if !namen[erwartet] {
			t.Errorf("Abgänger %q fehlt auf dem Laufzettel (hartkodierte Klassenliste hätte ihn verschluckt)", erwartet)
		}
	}
	if namen["Dana"] {
		t.Error("Nicht-Abgänger Dana erscheint fälschlich auf dem Laufzettel")
	}
	if len(studenten) != 3 {
		t.Errorf("erwartet 3 Abgänger, waren %d", len(studenten))
	}
}

func seedAbgaenger(t *testing.T, pool *pgxpool.Pool, barcode, vorname, klasse string, istAbgaenger bool) {
	t.Helper()
	if _, err := pool.Exec(context.Background(),
		`INSERT INTO schueler (barcode_id, vorname, nachname, klasse, abgaenger_jahr, ist_abgaenger)
		 VALUES ($1, $2, 'Test', $3, 2030, $4)`,
		barcode, vorname, klasse, istAbgaenger); err != nil {
		t.Fatalf("Abgänger %q anlegen: %v", vorname, err)
	}
}

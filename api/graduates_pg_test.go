package api

import (
	"context"
	"testing"

	"bibliothek/db"

	"github.com/jackc/pgx/v5/pgxpool"
)

// TestLaufzettelStudents_AbgaengerFilter sichert #4 ab: Der Laufzettel muss ALLE
// Abgänger erfassen (ist_abgaenger = true), unabhängig vom Klassennamen. Früher fing
// eine hartkodierte, case-sensitive Liste ('9h','10r','13') nur exakt diese Namen —
// Abgänger in '09h', '9H' oder '10a' fehlten auf dem PDF und verließen die Schule mit
// ihren Büchern.
func TestLaufzettelStudents_AbgaengerFilter(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	ctx := context.Background()

	// Drei Abgänger mit unterschiedlich formatierten Klassen + ein Nicht-Abgänger.
	seedAbgaenger(t, pool, "S-1", "Anna", "09h", true) // führende Null
	seedAbgaenger(t, pool, "S-2", "Bea", "9H", true)   // Großbuchstabe
	seedAbgaenger(t, pool, "S-3", "Cem", "10a", true)  // anderer Zug
	seedAbgaenger(t, pool, "S-4", "Dana", "7b", false) // kein Abgänger

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

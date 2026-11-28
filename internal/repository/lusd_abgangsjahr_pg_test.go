package repository_test

import (
	"fmt"
	"testing"

	"bibliothek/internal/pgtest"
	"bibliothek/internal/repository"
	"bibliothek/internal/testdaten"
)

// Ein Klassenwechsel aus der LUSD-Datei rechnet das voraussichtliche Abgangsjahr neu, wie die
// Änderung der Klasse von Hand: Ein Wiederholer oder ein Rückkehrer trüge sonst das Jahr der
// alten Klasse in der Akte. Bleibt die Klasse, bleibt ein von Hand gesetztes Jahr; eine andere
// Schreibweise derselben Klasse ist kein Wechsel.
func TestAendereLUSDBestand_KlassenwechselRechnetAbgangsjahrNeu(t *testing.T) {
	pool := pgtest.Pool(t)
	testdaten.LeereBestandsdaten(t, pool)

	const vonHand = 2040
	faelle := []struct {
		name                   string
		alteKlasse, neueKlasse string
		abgaenger              bool
		soll                   int
	}{
		{"Klassenwechsel", "8a", "9R1", false, repository.AbgaengerJahr("9R1")},
		{"dieselbe Klasse", "8a", "8a", false, vonHand},
		{"andere Schreibweise derselben Klasse", "5a", "05A", false, vonHand},
		{"Rückkehrer nach der Versetzung", "ABG", "ET1", true, repository.AbgaengerJahr("ET1")},
		{"Rückkehrer in dieselbe Klasse", "10R1", "10R1", true, repository.AbgaengerJahr("10R1")},
	}
	for i, f := range faelle {
		t.Run(f.name, func(t *testing.T) {
			nachname := fmt.Sprintf("Muster%d", i)
			var id string
			if err := pool.QueryRow(t.Context(),
				`INSERT INTO schueler (barcode_id, vorname, nachname, klasse, abgaenger_jahr, ist_abgaenger)
				 VALUES ($1, 'Max', $2, $3, $4, $5) RETURNING id`,
				fmt.Sprintf("AJ-%d", i), nachname, f.alteKlasse, vonHand, f.abgaenger).Scan(&id); err != nil {
				t.Fatalf("Schüler anlegen: %v", err)
			}

			fuehreAktualisierungAus(t, pool, repository.LUSDAktualisierung{Vorname: "Max", Nachname: nachname, Klasse: f.neueKlasse}, id)

			var jahr int
			if err := pool.QueryRow(t.Context(), `SELECT abgaenger_jahr FROM schueler WHERE id = $1`, id).Scan(&jahr); err != nil {
				t.Fatalf("Abgangsjahr lesen: %v", err)
			}
			if jahr != f.soll {
				t.Errorf("abgaenger_jahr nach AendereLUSDBestand(%q zu %q) = %d, erwartet %d", f.alteKlasse, f.neueKlasse, jahr, f.soll)
			}
		})
	}
}

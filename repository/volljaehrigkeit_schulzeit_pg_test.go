package repository

import (
	"context"
	"os"
	"strings"
	"testing"

	"bibliothek/internal/pgtest"
)

// Volljährig wird man am Geburtstag — in Berlin, nicht in der Zeitzone der Sitzung.
//
// Der Bescheid-Vorschlag entscheidet daran, WER den Brief bekommt: die Eltern oder die
// Person selbst. Bis zum 17.09.2026 rechnete er mit CURRENT_DATE, also mit dem
// Kalendertag der Datenbank-Sitzung (im Image UTC). Am 18. Geburtstag galt das Kind
// damit bis 2 Uhr Berliner Zeit noch als minderjährig; ein in dieser Zeit erzeugter
// Bescheid ging an die Eltern eines Erwachsenen. Das ist kein Schönheitsfehler, sondern
// ein Brief an den falschen Empfänger — und niemandem wäre es aufgefallen.
//
// Der Test hängt nicht an der Uhrzeit: Er wertet die Bedingung in zwei Sitzungszonen aus,
// die 26 Stunden auseinanderliegen (UTC−12 und UTC+14). Zu jeder Stunde weicht mindestens
// eine von beiden vom Berliner Kalendertag ab. Dieselbe Bauart wie
// bescheid_frist_schulzeit_pg_test.go.
func TestVolljaehrigkeit_RechnetInDerSchulzeitzone(t *testing.T) {
	pool := pgtest.Pool(t)
	ctx := context.Background()

	// Genau der Ausdruck aus EmpfaengerFuerBescheid — abgeschrieben, also nachgeprüft:
	// Ohne diesen Wächter bliebe der Test grün, während die Abfrage längst wieder mit
	// CURRENT_DATE rechnet. Er misst dann seine eigene Kopie.
	ausdruck := `coalesce(geburtsdatum <= ` + sqlSchulHeute + ` - INTERVAL '18 years', false)`
	quelle, err := os.ReadFile("bescheid.go")
	if err != nil {
		t.Fatalf("bescheid.go nicht lesbar: %v", err)
	}
	if !strings.Contains(string(quelle), "geburtsdatum <= `+sqlSchulHeute+` - INTERVAL '18 years'") {
		t.Fatal("EmpfaengerFuerBescheid rechnet die Volljährigkeit nicht mehr mit sqlSchulHeute — " +
			"entweder ist der Test nachzuziehen oder die Abfrage ist zurückgefallen")
	}

	faelle := []struct {
		name        string
		tageVorher  int // Geburtstag relativ zum heutigen Berliner Tag, in Tagen
		volljaehrig bool
	}{
		{"18. Geburtstag ist heute", 0, true},
		{"18. Geburtstag ist morgen", 1, false},
		{"gestern 18 geworden", -1, true},
	}

	werteAus := func(zone string, tageVorher int) (bool, error) {
		tx := beginne(t, pool)
		var ergebnis bool
		_, err := tx.Exec(ctx, `SET LOCAL TIME ZONE '`+zone+`'`)
		if err == nil {
			err = tx.QueryRow(ctx, `
				SELECT `+ausdruck+`
				FROM (VALUES ((((now() AT TIME ZONE 'Europe/Berlin')::date - INTERVAL '18 years')::date + $1::int)))
				     AS s(geburtsdatum)`, tageVorher).Scan(&ergebnis)
		}
		if rbErr := tx.Rollback(ctx); rbErr != nil {
			t.Fatalf("zurückrollen: %v", rbErr)
		}
		return ergebnis, err
	}

	for _, zone := range []string{"Etc/GMT+12", "Etc/GMT-14"} {
		for _, f := range faelle {
			ergebnis, err := werteAus(zone, f.tageVorher)
			if err != nil {
				t.Fatalf("%s in %s: %v", f.name, zone, err)
			}
			if ergebnis != f.volljaehrig {
				t.Errorf("%s, Sitzungszone %s: volljährig = %v, erwartet %v — der Bescheid ginge "+
					"an den falschen Empfänger", f.name, zone, ergebnis, f.volljaehrig)
			}
		}
	}
}

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
// Der Bescheid-Vorschlag entscheidet daran, wer den Brief bekommt: die Eltern oder die
// Person selbst. Rechnete er mit CURRENT_DATE, also mit dem
// Kalendertag der Datenbank-Sitzung (im Image UTC), gälte das Kind am 18. Geburtstag
// bis 2 Uhr Berliner Zeit noch als minderjährig; ein in dieser Zeit erzeugter
// Bescheid ginge an die Eltern eines Erwachsenen. Das ist kein Schönheitsfehler, sondern
// ein Brief an den falschen Empfänger — und niemandem fiele es auf.
//
// Der Test hängt nicht an der Uhrzeit: Er wertet die Bedingung in zwei Sitzungszonen aus,
// die 26 Stunden auseinanderliegen (UTC−12 und UTC+14). Zu jeder Stunde weicht mindestens
// eine von beiden vom Berliner Kalendertag ab. Dieselbe Bauart wie
// bescheid_frist_schulzeit_pg_test.go.
func TestVolljaehrigkeit_RechnetInDerSchulzeitzone(t *testing.T) {
	pool := pgtest.Pool(t)
	ctx := context.Background()

	// Der Ausdruck ist sqlVolljaehrig, die eine Regel für Bescheid und Ersatzforderung. Der
	// Wächter prüft, dass beide Abfragen ihn lesen: Ohne ihn bliebe der Test grün, während eine
	// Abfrage längst wieder mit CURRENT_DATE rechnet.
	ausdruck := sqlVolljaehrig
	if !strings.Contains(ausdruck, sqlSchulHeute) {
		t.Fatal("sqlVolljaehrig rechnet nicht mit sqlSchulHeute")
	}
	for _, datei := range []string{"bescheid_vorschlag.go", "schadensbrief.go"} {
		quelle, err := os.ReadFile(datei)
		if err != nil {
			t.Fatalf("%s nicht lesbar: %v", datei, err)
		}
		if !strings.Contains(string(quelle), "`+sqlVolljaehrig+`") {
			t.Errorf("%s liest die Volljährigkeit nicht über sqlVolljaehrig — entweder ist der Test "+
				"nachzuziehen oder die Abfrage ist zurückgefallen", datei)
		}
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

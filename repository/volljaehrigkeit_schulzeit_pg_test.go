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
// Bescheid und Mahnbrief entscheiden daran, WER den Brief bekommt: die Eltern oder die
// Person selbst. Bis zum 17.09.2026 rechnete der Bescheid mit CURRENT_DATE, also mit dem
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

	// Der Ausdruck, nach dem Bescheid und Mahnbrief rechnen. Gemessen wird er selbst; die
	// zwei Abfragen müssen ihn einsetzen, sonst misst der Test eine Regel ohne Leser.
	ausdruck := sqlVolljaehrig
	for datei, einsatz := range map[string]string{
		"bescheid.go":          "`+sqlVolljaehrig+`",
		"mahnwesen_queries.go": "` + sqlVolljaehrig + `",
	} {
		quelle, err := os.ReadFile(datei)
		if err != nil {
			t.Fatalf("%s nicht lesbar: %v", datei, err)
		}
		if !strings.Contains(string(quelle), einsatz) {
			t.Fatalf("%s setzt sqlVolljaehrig nicht mehr ein — entweder ist der Test "+
				"nachzuziehen oder die Abfrage rechnet die Volljährigkeit auf eigene Weise", datei)
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
				t.Errorf("%s, Sitzungszone %s: volljährig = %v, erwartet %v — der Brief ginge "+
					"an den falschen Empfänger", f.name, zone, ergebnis, f.volljaehrig)
			}
		}
	}
}

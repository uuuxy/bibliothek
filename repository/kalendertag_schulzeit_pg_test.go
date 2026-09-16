package repository

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"bibliothek/internal/pgtest"
	"bibliothek/pkg/schulzeit"
)

// Wo ein TAG gemeint ist, muss der Tag der Schule gelten — nicht der der Sitzung.
//
// Die Datenbank läuft im Image in UTC. Ein Zeitpunkt um 00:30 Berliner Zeit gehört dort
// noch zum VORTAG, und zwei Prädikate haben genau daran gehangen (OFFEN.md 5.2):
//
//   - Der Mahnlauf („höchstens einmal am Tag", api/mahnwesen_bulk.go): Der Tag wechselte
//     um 2 Uhr Berliner Zeit statt um Mitternacht. Ein Lauf kurz nach Mitternacht galt als
//     „heute schon gemahnt" und wurde übersprungen.
//   - „Heute zurückgegeben" (CountReturnsToday): Eine Rückgabe zwischen 0 und 2 Uhr zählte
//     zum Vortag.
//
// Was dieser Test NICHT kann: den Fehler zur Laufzeit auslösen. Er tritt nur zwischen
// Mitternacht in Berlin und Mitternacht UTC ein, und `now()` lässt sich in Postgres nicht
// stellen — ein Test, der auf diese zwei Stunden wartet, liefe 22 Stunden am Tag grün und
// bewiese nichts. Beweisbar und von der Uhrzeit unabhängig ist dagegen die Aussage über
// die beiden FORMULIERUNGEN, und genau die steht hier:
//
//  1. Die neue Form bildet für einen Zeitpunkt um 00:30 Berliner Zeit den Berliner
//     Kalendertag — in JEDER Sitzungszone.
//  2. Die alte Form (`::date` in der Sitzungszone) tut das nicht: Sie liefert in
//     mindestens einer Zone einen anderen Tag. Sie KANN also nicht gleichwertig sein.
//  3. Die lebende Abfrage benutzt die neue Form — geprüft an der Quelle, damit dieser
//     Test nicht bloß seine eigene Kopie misst.

// mitternachtNachtsBerlin: heute um 00:30 Uhr Berliner Zeit, als Zeitpunkt.
// Genau die Stunde, in der die beiden Prädikate falsch lagen.
const mitternachtNachtsBerlin = `((((now() AT TIME ZONE 'Europe/Berlin')::date) + INTERVAL '30 minutes') AT TIME ZONE 'Europe/Berlin')`

// tagInZonen wertet einen Datums-Ausdruck über `wert` in zwei weit auseinanderliegenden
// Sitzungszonen aus (UTC−12 und UTC+14) und liefert die beiden Ergebnisse.
func tagInZonen(t *testing.T, ausdruck, spalte, wert string) map[string]string {
	t.Helper()
	pool := pgtest.Pool(t)
	ctx := context.Background()
	out := map[string]string{}
	for _, zone := range []string{"Etc/GMT+12", "Etc/GMT-14"} {
		tx := beginne(t, pool)
		var tag string
		_, err := tx.Exec(ctx, `SET LOCAL TIME ZONE '`+zone+`'`)
		if err == nil {
			err = tx.QueryRow(ctx,
				`SELECT (`+ausdruck+`)::text FROM (VALUES (`+wert+`)) AS z(`+spalte+`)`).Scan(&tag)
		}
		if rbErr := tx.Rollback(ctx); rbErr != nil {
			t.Fatalf("zurückrollen: %v", rbErr)
		}
		if err != nil {
			t.Fatalf("Auswertung in %s: %v", zone, err)
		}
		out[zone] = tag
	}
	return out
}

func TestKalendertag_SchulzeitStattSitzungszone(t *testing.T) {
	neu := `(x AT TIME ZONE '` + schulzeit.ZonenName + `')::date`
	alt := `x::date`
	berlinHeute := tagInZonen(t, schulzeit.SQLHeute, "x", `NULL::timestamptz`)

	// (1) Die neue Form trifft den Berliner Tag, gleich in welcher Sitzungszone.
	for zone, tag := range tagInZonen(t, neu, "x", mitternachtNachtsBerlin) {
		if tag != berlinHeute[zone] {
			t.Errorf("Sitzungszone %s: 00:30 Berliner Zeit ergibt den Tag %s, der Berliner Tag "+
				"ist %s", zone, tag, berlinHeute[zone])
		}
	}

	// (2) Die alte Form kann nicht gleichwertig sein — sie weicht in mindestens einer Zone ab.
	abweichungen := 0
	for zone, tag := range tagInZonen(t, alt, "x", mitternachtNachtsBerlin) {
		if tag != berlinHeute[zone] {
			abweichungen++
		}
	}
	if abweichungen == 0 {
		t.Error("die alte Form (::date in der Sitzungszone) traf in beiden Zonen den Berliner Tag — " +
			"dann misst dieser Test nichts mehr, und der Unterschied, um den es geht, wäre keiner")
	}
}

// (3) Die lebenden Abfragen benutzen die neue Form.
//
// Ohne diesen Wächter bliebe der Test oben grün, während die Abfragen längst wieder mit
// CURRENT_DATE rechnen — er misst dann nur noch seine eigene Kopie.
func TestKalendertag_DieLebendenAbfragenBenutzenDenSchultag(t *testing.T) {
	faelle := []struct{ pfad, klage string }{
		{filepath.Join("..", "api", "mahnwesen_bulk.go"),
			"der Mahnlauf vergleicht nicht mehr gegen den Kalendertag der Schule — eine Ausleihe " +
				"kann damit an einem Tag zweimal in der Mahnstufe steigen"},
		{"mahnwesen_repo.go",
			"CountReturnsToday zählt nicht mehr am Kalendertag der Schule — Rückgaben nach " +
				"Mitternacht fehlen dann auf dem Dashboard"},
		{"bescheid.go",
			"der Bescheid rechnet nicht mehr mit dem Kalendertag der Schule"},
	}
	for _, f := range faelle {
		roh, err := os.ReadFile(f.pfad)
		if err != nil {
			t.Fatalf("%s nicht lesbar: %v", f.pfad, err)
		}
		quelle := string(roh)
		if !strings.Contains(quelle, "schulzeit.SQLHeute") && !strings.Contains(quelle, "sqlSchulHeute") {
			t.Errorf("%s: %s", f.pfad, f.klage)
		}
	}
}

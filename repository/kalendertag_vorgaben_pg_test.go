package repository

import (
	"context"
	"testing"

	"bibliothek/internal/pgtest"
)

// Keine Spaltenvorgabe rechnet mit dem Kalendertag der Datenbank-Sitzung (im Image UTC,
// zwischen Mitternacht in Berlin und 2 Uhr der Vortag).
//
// docs/kalendertag_bestand_test.go hält CURRENT_DATE in Go-Quellen fest — eine Vorgabe in
// schema.sql oder in einer Migration sieht er nicht. Genau dort standen am 23.09.2026 die
// letzten zwei: buecher_exemplare.erworben_am (Migration 139) und
// schadensersatz_bescheide.brief_datum (Migration 142). Geprüft wird am fertigen Schema,
// nicht am Text: Postgres schreibt jede Vorgabe in eine Form, egal wie sie getippt war.
//
// Erlaubt ist der Kalendertag der Schule, ((now() AT TIME ZONE 'Europe/Berlin'))::date, und
// jeder Zeitpunkt (now(), CURRENT_TIMESTAMP) — ein Zeitpunkt ist in jeder Zone derselbe.
const vorgabeMitSitzungstag = `(v ILIKE '%current_date%'
	OR (v ~* '(now\(\)|current_timestamp|localtimestamp)' AND v ILIKE '%::date%'
	    AND v NOT ILIKE '%time zone%'))`

func TestKalendertag_KeineSpaltenvorgabeMitDemTagDerSitzung(t *testing.T) {
	pool := pgtest.Pool(t)
	rows, err := pool.Query(context.Background(), `
		SELECT table_name, column_name, v
		FROM (SELECT table_name, column_name, column_default AS v
		      FROM information_schema.columns
		      WHERE table_schema = 'public' AND column_default IS NOT NULL) c
		WHERE `+vorgabeMitSitzungstag+`
		ORDER BY table_name, column_name`)
	if err != nil {
		t.Fatalf("Spaltenvorgaben lesen: %v", err)
	}
	defer rows.Close()
	for rows.Next() {
		var tabelle, spalte, vorgabe string
		if err := rows.Scan(&tabelle, &spalte, &vorgabe); err != nil {
			t.Fatalf("lesen: %v", err)
		}
		t.Errorf("%s.%s: Vorgabe %s rechnet mit dem Tag der Sitzung — gemeint ist der Tag der "+
			"Schule, ((now() AT TIME ZONE 'Europe/Berlin'))::date wie pkg/schulzeit.SQLHeute",
			tabelle, spalte, vorgabe)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("lesen: %v", err)
	}
}

// Selbstprobe: Das Prädikat meldet die Formen, in denen Postgres einen Sitzungstag
// ausschreibt, und lässt den Tag der Schule und Zeitpunkte durch. Ohne diese Probe wäre ein
// Prädikat, das nichts findet, von einem sauberen Schema nicht zu unterscheiden.
func TestKalendertag_SelbstprobeDesPraedikats(t *testing.T) {
	pool := pgtest.Pool(t)
	for vorgabe, gemeldet := range map[string]bool{
		"CURRENT_DATE":              true,
		"(now())::date":             true,
		"(CURRENT_TIMESTAMP)::date": true,
		"((now() AT TIME ZONE 'Europe/Berlin'::text))::date": false,
		"now()":              false,
		"CURRENT_TIMESTAMP":  false,
		"'2026-01-01'::date": false,
	} {
		var treffer bool
		if err := pool.QueryRow(context.Background(),
			`SELECT `+vorgabeMitSitzungstag+` FROM (VALUES ($1::text)) AS z(v)`, vorgabe).Scan(&treffer); err != nil {
			t.Fatalf("%s: %v", vorgabe, err)
		}
		if treffer != gemeldet {
			t.Errorf("%s: gemeldet=%v, erwartet %v", vorgabe, treffer, gemeldet)
		}
	}
}

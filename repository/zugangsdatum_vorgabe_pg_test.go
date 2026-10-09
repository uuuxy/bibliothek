package repository

import (
	"context"
	"testing"

	"bibliothek/internal/pgtest"
)

// Das Zugangsdatum eines neuen Exemplars ist der Kalendertag der Schule (OFFEN.md 5.5,
// Migration 139). Handanlage, Sammelimport und Bestand-Nachziehen schreiben kein
// erworben_am und nehmen die Vorgabe der Spalte; bis zum 23.09.2026 war das CURRENT_DATE,
// also der Tag der Datenbank-Sitzung (im Image UTC) — zwischen Mitternacht in Berlin und
// 2 Uhr der Vortag. Der INSERT-Trigger aus Migration 129 übernimmt den Wert als Zugang,
// und das Zugangsbuch ist der Nachweis zum Stichtag 15.3./15.9. Littera trägt „das
// aktuelle Tagesdatum" ein — das des Arbeitsplatzes, also der Schule.
//
// Unabhängig von der Uhrzeit prüfbar wie in kalendertag_schulzeit_pg_test.go: Die beiden
// Sitzungszonen liegen 26 Stunden auseinander, ihre Kalendertage unterscheiden sich also
// immer. Eine Vorgabe, die dem Sitzungstag folgt, trifft den Berliner Tag deshalb zu jeder
// Stunde in mindestens einer Zone nicht.
func TestZugangsdatum_VorgabeIstDerKalendertagDerSchule(t *testing.T) {
	pool := pgtest.Pool(t)
	ctx := context.Background()

	for _, zone := range []string{"Etc/GMT+12", "Etc/GMT-14"} {
		tx := beginne(t, pool)
		var erworben, schultag string
		_, err := tx.Exec(ctx, `SET LOCAL TIME ZONE '`+zone+`'`)
		if err == nil {
			err = tx.QueryRow(ctx, `
				WITH t AS (INSERT INTO buecher_titel (titel) VALUES ('Zugangsdatum-Probe') RETURNING id)
				INSERT INTO buecher_exemplare (titel_id, barcode_id)
				SELECT id, 'ZUGANG-PROBE-1' FROM t
				RETURNING erworben_am::text, ((now() AT TIME ZONE 'Europe/Berlin')::date)::text`).
				Scan(&erworben, &schultag)
		}
		if rbErr := tx.Rollback(ctx); rbErr != nil {
			t.Fatalf("zurückrollen: %v", rbErr)
		}
		if err != nil {
			t.Fatalf("Sitzungszone %s: %v", zone, err)
		}
		if erworben != schultag {
			t.Errorf("Sitzungszone %s: erworben_am = %s, der Kalendertag der Schule ist %s — die "+
				"Vorgabe folgt der Zone der Sitzung", zone, erworben, schultag)
		}
	}
}

// Der Bestands-Import schreibt kein erworben_am und nimmt die Vorgabe der Spalte: eine Regel
// für den Tag des Zugangs, nicht zwei. Gemessen an dem Exemplar, das er anlegt, in denselben
// zwei Sitzungszonen wie oben.
func TestZugangsdatum_BestandsImportNimmtDieVorgabe(t *testing.T) {
	pool := pgtest.Pool(t)
	ctx := context.Background()
	const barcode = "ZUGANG-PROBE-IMPORT-1"

	for _, zone := range []string{"Etc/GMT+12", "Etc/GMT-14"} {
		tx := beginne(t, pool)
		var titelID, erworben, schultag string
		var angelegt int
		_, err := tx.Exec(ctx, `SET LOCAL TIME ZONE '`+zone+`'`)
		if err == nil {
			err = tx.QueryRow(ctx, `INSERT INTO buecher_titel (titel) VALUES ('Zugangsdatum-Probe Import')
				RETURNING id::text`).Scan(&titelID)
		}
		if err == nil {
			angelegt, _, err = LegeImportExemplareAn(ctx, tx, []ImportExemplar{
				{TitelID: titelID, Barcode: barcode, IstAusleihbar: true},
			})
		}
		if err == nil {
			err = tx.QueryRow(ctx, `
				SELECT erworben_am::text, ((now() AT TIME ZONE 'Europe/Berlin')::date)::text
				FROM buecher_exemplare WHERE barcode_id = $1`, barcode).Scan(&erworben, &schultag)
		}
		if rbErr := tx.Rollback(ctx); rbErr != nil {
			t.Fatalf("zurückrollen: %v", rbErr)
		}
		if err != nil {
			t.Fatalf("Sitzungszone %s: %v", zone, err)
		}
		if angelegt != 1 {
			t.Fatalf("Sitzungszone %s: %d Exemplare angelegt, erwartet 1", zone, angelegt)
		}
		if erworben != schultag {
			t.Errorf("Sitzungszone %s: erworben_am = %s, der Kalendertag der Schule ist %s — der "+
				"Import schreibt den Tag der Sitzung statt der Vorgabe der Spalte", zone, erworben, schultag)
		}
	}
}

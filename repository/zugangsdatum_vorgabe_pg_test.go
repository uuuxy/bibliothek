package repository

import (
	"context"
	"os"
	"path/filepath"
	"strings"
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

// Der Listenimport schrieb erworben_am selbst mit CURRENT_DATE, an der Vorgabe vorbei. Jetzt
// nimmt er die Vorgabe — eine Regel für „heute", nicht zwei. Geprüft an der Quelle, damit
// eine zweite Formulierung nicht unbemerkt zurückkommt.
func TestZugangsdatum_ListenimportNimmtDieVorgabe(t *testing.T) {
	roh, err := os.ReadFile(filepath.Join("..", "internal", "service", "import_dynamic.go"))
	if err != nil {
		t.Fatalf("Quelle lesen: %v", err)
	}
	// Kommentarzeilen fallen weg: Ein Kommentar, der die alte Form nennt, ist kein Rückfall —
	// und ein Code-Rückfall darf sich nicht hinter einem Kommentar verstecken.
	var code []string
	for _, zeile := range strings.Split(string(roh), "\n") {
		if !strings.HasPrefix(strings.TrimSpace(zeile), "//") {
			code = append(code, zeile)
		}
	}
	quelle := strings.Join(code, "\n")
	if strings.Contains(quelle, "CURRENT_DATE") {
		t.Error("internal/service/import_dynamic.go schreibt CURRENT_DATE — das ist der Tag der " +
			"Datenbank-Sitzung (UTC), nicht der der Schule; erworben_am kommt aus der Vorgabe der Spalte")
	}
	if !strings.Contains(quelle, "INSERT INTO buecher_exemplare (titel_id, barcode_id, ist_ausleihbar") {
		t.Error("die INSERT-Anweisung des Listenimports ist nicht mehr dort, wo dieser Test sie " +
			"sucht — Test nachziehen, sonst prüft er nichts")
	}
}

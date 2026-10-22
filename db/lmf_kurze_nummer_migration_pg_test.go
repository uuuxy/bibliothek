package db_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"testing"

	"bibliothek/internal/pgtest"
	"bibliothek/internal/testdaten"
)

// Migration 167 gibt jedem Exemplar mit LMF-Nummer eine kurze Nummer aus barcode_seq und legt die
// LMF-Nummer nach lmf_barcode. Geprüft wird die echte Datei an der Testdatenbank: Die kurzen
// Nummern liegen über einer von Hand eingetragenen, folgen der Reihenfolge der LMF-Nummern, und
// alles andere bleibt, auch etikett_gedruckt. Ein zweiter Lauf ändert nichts.
func TestMigration167_LMFExemplareBekommenKurzeNummern(t *testing.T) {
	pool := pgtest.Pool(t)
	testdaten.LeereBestandsdaten(t, pool)
	ctx := context.Background()

	var stand int64
	if err := pool.QueryRow(ctx, `SELECT last_value FROM barcode_seq`).Scan(&stand); err != nil {
		t.Fatalf("Stand der Sequenz: %v", err)
	}
	titelID := testdaten.LegeTitelAn(t, pool, "LMF-Umstellung", 0)
	// Eine von Hand eingetragene Nummer über dem Stand der Sequenz: Keine kurze Nummer darf sie
	// treffen.
	handNummer := fmt.Sprintf("B-%d", stand+500)
	hand := testdaten.LegeExemplarAn(t, pool, titelID, handNummer, true, "")
	littera := testdaten.LegeExemplarAn(t, pool, titelID, "101634", true, "")
	zweites := testdaten.LegeExemplarAn(t, pool, titelID, "LMF-9783060158423-002", true, "")
	erstes := testdaten.LegeExemplarAn(t, pool, titelID, "LMF-9783060158423-001", true, "")
	klein := testdaten.LegeExemplarAn(t, pool, titelID, "lmf-9783125487710-001", true, "")
	// Das UPDATE legt erstes und klein hinter zweites: Ohne ORDER BY bekäme zweites die erste
	// Nummer.
	if _, err := pool.Exec(ctx, `UPDATE buecher_exemplare SET etikett_gedruckt = true WHERE id = ANY($1)`,
		[]string{erstes, klein}); err != nil {
		t.Fatalf("Etikett vermerken: %v", err)
	}

	migration, err := os.ReadFile(filepath.Join("..", "migrations", "167_lmf_kurze_nummer.sql"))
	if err != nil {
		t.Fatalf("Migration lesen: %v", err)
	}
	if _, err := pool.Exec(ctx, string(migration)); err != nil {
		t.Fatalf("Migration 167 ausführen: %v", err)
	}

	type zeile struct {
		nummer, alt string
		gedruckt    bool
	}
	lies := func() map[string]zeile {
		t.Helper()
		rows, err := pool.Query(ctx, `
			SELECT id::text, barcode_id, coalesce(lmf_barcode, ''), etikett_gedruckt
			FROM buecher_exemplare WHERE titel_id = $1`, titelID)
		if err != nil {
			t.Fatalf("Exemplare lesen: %v", err)
		}
		defer rows.Close()
		zeilen := map[string]zeile{}
		for rows.Next() {
			var id string
			var z zeile
			if err := rows.Scan(&id, &z.nummer, &z.alt, &z.gedruckt); err != nil {
				t.Fatalf("Exemplar lesen: %v", err)
			}
			zeilen[id] = z
		}
		if err := rows.Err(); err != nil {
			t.Fatalf("Exemplare lesen: %v", err)
		}
		return zeilen
	}
	nachher := lies()

	kurz := regexp.MustCompile(`^B-([0-9]+)$`)
	ziffern := func(id string) int64 {
		t.Helper()
		m := kurz.FindStringSubmatch(nachher[id].nummer)
		if m == nil {
			t.Fatalf("Exemplar %s trägt %q, erwartet eine kurze Nummer", id, nachher[id].nummer)
		}
		n, err := strconv.ParseInt(m[1], 10, 64)
		if err != nil {
			t.Fatal(err)
		}
		return n
	}
	for id, want := range map[string]zeile{
		erstes:  {alt: "LMF-9783060158423-001", gedruckt: true},
		zweites: {alt: "LMF-9783060158423-002", gedruckt: false},
		klein:   {alt: "LMF-9783125487710-001", gedruckt: true},
	} {
		if n := ziffern(id); n <= stand+500 {
			t.Errorf("Exemplar %s bekam %s, erwartet eine Nummer über der von Hand eingetragenen %s", id, nachher[id].nummer, handNummer)
		}
		if got := nachher[id]; got.alt != want.alt || got.gedruckt != want.gedruckt {
			t.Errorf("Exemplar %s: altes Etikett %q, gedruckt %v; erwartet %q, %v", id, got.alt, got.gedruckt, want.alt, want.gedruckt)
		}
	}
	if ziffern(erstes) >= ziffern(zweites) || ziffern(zweites) >= ziffern(klein) {
		t.Errorf("Nummern %s, %s, %s; erwartet aufsteigend in der Reihenfolge der LMF-Nummern",
			nachher[erstes].nummer, nachher[zweites].nummer, nachher[klein].nummer)
	}
	for id, want := range map[string]string{hand: handNummer, littera: "101634"} {
		if got := nachher[id]; got.nummer != want || got.alt != "" {
			t.Errorf("Exemplar %s: %q mit altem Etikett %q, erwartet unverändert %q ohne", id, got.nummer, got.alt, want)
		}
	}

	if _, err := pool.Exec(ctx, string(migration)); err != nil {
		t.Fatalf("zweiter Lauf: %v", err)
	}
	for id, z := range lies() {
		if z != nachher[id] {
			t.Errorf("zweiter Lauf änderte Exemplar %s: %+v, vorher %+v", id, z, nachher[id])
		}
	}
}

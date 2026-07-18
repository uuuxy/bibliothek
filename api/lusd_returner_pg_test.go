package api

import (
	"context"
	"testing"

	"bibliothek/db"

	"github.com/jackc/pgx/v5/pgxpool"
)

// seedGesperrterAbgaenger legt einen als Abgänger gesperrten, NICHT soft-gelöschten Schüler
// mit lusd_id an und liefert dessen ID.
func seedGesperrterAbgaenger(t *testing.T, pool *pgxpool.Pool, barcode, lusdID string) string {
	t.Helper()
	var id string
	if err := pool.QueryRow(context.Background(), `
		INSERT INTO schueler
			(barcode_id, vorname, nachname, klasse, abgaenger_jahr, lusd_id,
			 ist_abgaenger, ist_gesperrt, block_reason)
		VALUES ($1, 'Max', 'Muster', 'ABG', 2025, $2,
			 true, true, 'Automatisierte Abgänger-Sperre (offene Vorgänge)')
		RETURNING id`, barcode, lusdID).Scan(&id); err != nil {
		t.Fatalf("Abgänger anlegen: %v", err)
	}
	return id
}

// TestLusdImport_ReturningBlockedAbgaenger fährt die ECHTE Import-Pipeline
// (computeLusdChanges → wendeLusdAenderungenAn) mit einem zurückkehrenden, als Abgänger
// gesperrten Schüler. Der Unit-Test der Runde 6/7 prüfte nur aktualisiereBestandsschueler
// isoliert; er blieb aber unerreichbar, weil ladeAktiveSchueler Abgänger ausfiltert und der
// Rückkehrer so als Neuzugang am partiellen Unique-Index (lusd_id) kollidierte — der GESAMTE
// Import scheiterte. Dieser Test sichert die korrekte Weiterleitung end-to-end ab.
func TestLusdImport_ReturningBlockedAbgaenger(t *testing.T) {
	pool := pgTestPool(t)

	t.Run("beglichener Rückkehrer wird reaktiviert und entsperrt", func(t *testing.T) {
		resetBestandsdaten(t, pool)
		ctx := context.Background()
		id := seedGesperrterAbgaenger(t, pool, "OLD-1", "L-RETURN")

		s := &Server{DB: &db.Database{Pool: pool}}
		if _, err := s.computeLusdChanges(ctx, []parsedStudentRow{
			{LusdID: "L-RETURN", Vorname: "Max", Nachname: "Muster", Klasse: "E1"},
		}, true, true); err != nil {
			t.Fatalf("Import eines Rückkehrers darf nicht fehlschlagen: %v", err)
		}

		// Genau EINE aktive Zeile für die lusd_id (kein Duplikat, kein Crash).
		var count int
		if err := pool.QueryRow(ctx,
			`SELECT count(*) FROM schueler WHERE lusd_id = 'L-RETURN' AND deleted_at IS NULL`).Scan(&count); err != nil {
			t.Fatal(err)
		}
		if count != 1 {
			t.Errorf("erwartet genau 1 aktive Zeile für L-RETURN, waren %d", count)
		}

		gesperrt, abgaenger, reason, _ := leseSchuelerStatus(t, pool, id)
		if gesperrt {
			t.Error("beglichener Rückkehrer bleibt gesperrt (Ghost-Block) — Pipeline reaktiviert nicht")
		}
		if abgaenger {
			t.Error("ist_abgaenger wurde beim Rückkehrer nicht zurückgesetzt")
		}
		if reason != nil {
			t.Errorf("block_reason wurde nicht geräumt: %q", *reason)
		}
		var klasse string
		if err := pool.QueryRow(ctx, `SELECT klasse FROM schueler WHERE id = $1`, id).Scan(&klasse); err != nil {
			t.Fatal(err)
		}
		if klasse != "E1" {
			t.Errorf("Klasse aus dem Export nicht übernommen: %q", klasse)
		}
	})

	t.Run("Rückkehrer mit offener Ausleihe bleibt gesperrt, Grund umbenannt", func(t *testing.T) {
		resetBestandsdaten(t, pool)
		ctx := context.Background()
		id := seedGesperrterAbgaenger(t, pool, "OLD-2", "L-DEBT")

		// Noch offene Ausleihe des Rückkehrers.
		titelID := titelMitMeldebestand(t, pool, "Schuldbuch", 1)
		exID := exemplar(t, pool, titelID, "EX-DEBT", true, "")
		if _, err := pool.Exec(ctx,
			`INSERT INTO ausleihen (exemplar_id, schueler_id, rueckgabe_frist)
			 VALUES ($1, $2, now() + interval '14 days')`, exID, id); err != nil {
			t.Fatalf("offene Ausleihe anlegen: %v", err)
		}

		s := &Server{DB: &db.Database{Pool: pool}}
		if _, err := s.computeLusdChanges(ctx, []parsedStudentRow{
			{LusdID: "L-DEBT", Vorname: "Max", Nachname: "Muster", Klasse: "E1"},
		}, true, true); err != nil {
			t.Fatalf("Import darf nicht fehlschlagen: %v", err)
		}

		gesperrt, abgaenger, reason, _ := leseSchuelerStatus(t, pool, id)
		if !gesperrt {
			t.Error("Rückkehrer mit offener Ausleihe muss gesperrt bleiben")
		}
		if abgaenger {
			t.Error("ist_abgaenger muss dennoch zurückgesetzt sein (wieder aktiv)")
		}
		if reason == nil || *reason != "Sperre wegen offener Vorgänge" {
			t.Errorf("irreführender Abgänger-Grund muss umbenannt werden, war %v", reason)
		}
	})
}

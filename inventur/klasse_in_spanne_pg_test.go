package inventur

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"bibliothek/internal/pgtest"
)

// Migration 135 faltet „Klasse" (grade_level) in die Jahrgangsspanne und lässt die Spalte
// fallen. Geprüft wird die echte Migrationsdatei: Das Testschema (schema.sql) kennt die
// Spalte nicht mehr, deshalb holt die Probe sie in einer Transaktion zurück, legt die
// Fälle an, führt die Migration aus und rollt alles zurück.
//
// Die Faltregel hat drei Seiten, die je einen Fall brauchen: Eine Klasse 6–13 bei
// Vorgabe-Spanne 5–10 wird die Spanne dieses Jahrgangs; Klasse 5 ist die Vorgabe der
// Maske und keine Aussage; eine gepflegte Spanne gewinnt, weil sie die Angabe mit den
// vier Lesern ist (Mahnwesen, Klassen-Inventur, Buchakte, Portal-Filter).
func TestMigration135_KlasseFaelltInDieSpanne(t *testing.T) {
	pool := pgtest.Pool(t)
	ctx := context.Background()
	migration, err := os.ReadFile(filepath.Join("..", "migrations", "135_klasse_in_spanne.sql"))
	if err != nil {
		t.Fatalf("Migration lesen: %v", err)
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback(ctx) }() //nolint:errcheck

	if _, err := tx.Exec(ctx, `ALTER TABLE buecher_titel ADD COLUMN grade_level smallint`); err != nil {
		t.Fatalf("Spalte für die Probe zurückholen: %v", err)
	}

	faelle := []struct {
		name             string
		klasse           *int16
		von, bis         int
		wantVon, wantBis int
	}{
		{"Klasse 7 bei Vorgabe wird 7–7", zeigerAuf16(7), 5, 10, 7, 7},
		{"Klasse 13 bei Vorgabe wird 13–13", zeigerAuf16(13), 5, 10, 13, 13},
		{"Klasse 5 ist die Vorgabe der Maske, keine Aussage", zeigerAuf16(5), 5, 10, 5, 10},
		{"gepflegte Spanne gewinnt gegen die Klasse", zeigerAuf16(8), 7, 9, 7, 9},
		{"ohne Klasse bleibt die Vorgabe", nil, 5, 10, 5, 10},
		{"Klasse 0 (unkategorisiert) bleibt die Vorgabe", zeigerAuf16(0), 5, 10, 5, 10},
	}
	ids := make([]string, len(faelle))
	for i, f := range faelle {
		if err := tx.QueryRow(ctx, `
			INSERT INTO buecher_titel (titel, grade_level, jahrgang_von, jahrgang_bis)
			VALUES ($1, $2, $3, $4) RETURNING id`, "Probe 135: "+f.name, f.klasse, f.von, f.bis).Scan(&ids[i]); err != nil {
			t.Fatalf("Fall %q anlegen: %v", f.name, err)
		}
	}

	if _, err := tx.Exec(ctx, string(migration)); err != nil {
		t.Fatalf("Migration 135 ausführen: %v", err)
	}

	for i, f := range faelle {
		var von, bis int
		if err := tx.QueryRow(ctx, `SELECT jahrgang_von, jahrgang_bis FROM buecher_titel WHERE id = $1`, ids[i]).Scan(&von, &bis); err != nil {
			t.Fatal(err)
		}
		if von != f.wantVon || bis != f.wantBis {
			t.Errorf("%s: Spanne %d–%d, want %d–%d", f.name, von, bis, f.wantVon, f.wantBis)
		}
	}

	var spalten int
	if err := tx.QueryRow(ctx, `
		SELECT count(*) FROM information_schema.columns
		WHERE table_name = 'buecher_titel' AND column_name = 'grade_level'`).Scan(&spalten); err != nil {
		t.Fatal(err)
	}
	if spalten != 0 {
		t.Errorf("grade_level steht nach der Migration noch in buecher_titel")
	}
}

func zeigerAuf16(v int16) *int16 { return &v }

// Die Lesetür ?gradeLevel= der Verwaltungsliste fragt seit 135 die Spanne: Ein Titel
// „7 bis 9" trifft die 8 und nicht die 10. Vorher verglich sie die Klasse, die es nicht
// mehr gibt.
func TestListBooks_JahrgangsfilterLiestDieSpanne(t *testing.T) {
	pool := pgtest.Pool(t)
	ctx := context.Background()
	repo := NewBookRepository(pool)

	var id string
	if err := pool.QueryRow(ctx, `
		INSERT INTO buecher_titel (titel, jahrgang_von, jahrgang_bis) VALUES ('Spanne 7–9', 7, 9) RETURNING id`).Scan(&id); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if _, err := pool.Exec(context.Background(), `DELETE FROM buecher_titel WHERE id = $1`, id); err != nil {
			t.Logf("Aufräumen: %v", err)
		}
	})
	if _, err := pool.Exec(ctx, `INSERT INTO buecher_exemplare (titel_id, barcode_id) VALUES ($1, 'B-135-PROBE')`, id); err != nil {
		t.Fatal(err)
	}

	enthaelt := func(jahrgang int16) bool {
		buecher, err := repo.ListBooks(ctx, "", &jahrgang, "Spanne 7–9", false)
		if err != nil {
			t.Fatal(err)
		}
		for _, b := range buecher {
			if b.ID == id {
				return true
			}
		}
		return false
	}
	if !enthaelt(8) {
		t.Errorf("Jahrgang 8 liegt in 7–9, der Filter findet den Titel nicht")
	}
	if enthaelt(10) {
		t.Errorf("Jahrgang 10 liegt außerhalb von 7–9, der Filter findet den Titel trotzdem")
	}
}

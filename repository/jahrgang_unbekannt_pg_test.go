package repository

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"

	"bibliothek/internal/pgtest"
)

// Migration 162: Die Vorgabe 5 bis 10 wird „unbekannt"; eine eigene Spanne und die Spanne eines
// Mehrjahresbands bleiben. Geprüft wird die echte Migrationsdatei an Zeilen jeder Form, am
// Stand davor (beide Spalten Pflicht mit Vorgabe, ohne die Regel), in einer Transaktion, die
// zurückgerollt wird.
func TestJahrgangUnbekannt_Migration162(t *testing.T) {
	pool := pgtest.Pool(t)
	ctx := context.Background()
	tx := beginne(t, pool)
	defer func() {
		if err := tx.Rollback(ctx); err != nil {
			t.Errorf("zurückrollen: %v", err)
		}
	}()

	for _, sql := range []string{
		`ALTER TABLE buecher_titel DROP CONSTRAINT chk_jahrgang_spanne`,
		`UPDATE buecher_titel SET jahrgang_von = 5, jahrgang_bis = 10 WHERE jahrgang_von IS NULL`,
		`ALTER TABLE buecher_titel
			ALTER COLUMN jahrgang_von SET DEFAULT 5, ALTER COLUMN jahrgang_von SET NOT NULL,
			ALTER COLUMN jahrgang_bis SET DEFAULT 10, ALTER COLUMN jahrgang_bis SET NOT NULL`,
	} {
		if _, err := tx.Exec(ctx, sql); err != nil {
			t.Fatalf("Stand vor der Migration herstellen: %v", err)
		}
	}
	titel := func(name, spalten, werte string) string {
		t.Helper()
		var id string
		if err := tx.QueryRow(ctx, `INSERT INTO buecher_titel (titel`+spalten+`) VALUES ($1`+werte+`) RETURNING id`, name).Scan(&id); err != nil {
			t.Fatalf("Titel %q: %v", name, err)
		}
		return id
	}
	ohne := titel("M162 ohne Angabe", "", "")
	eigen := titel("M162 eigene Spanne", ", jahrgang_von, jahrgang_bis", ", 7, 9")
	einJahr := titel("M162 ein Jahrgang", ", jahrgang_von, jahrgang_bis", ", 6, 6")
	band := titel("M162 Mehrjahresband", ", ist_lernmittel, mehrjahresband", ", true, true")

	migration, err := os.ReadFile(filepath.Join("..", "migrations", "162_jahrgang_unbekannt.sql"))
	if err != nil {
		t.Fatalf("Migration lesen: %v", err)
	}
	// Zweimal: Der zweite Lauf ändert nichts mehr.
	for lauf := 1; lauf <= 2; lauf++ {
		if _, err := tx.Exec(ctx, string(migration)); err != nil {
			t.Fatalf("Lauf %d: Migration scheitert: %v", lauf, err)
		}
	}

	spanne := func(id string) [2]int {
		t.Helper()
		var von, bis int
		if err := tx.QueryRow(ctx, `SELECT coalesce(jahrgang_von, 0), coalesce(jahrgang_bis, 0)
			FROM buecher_titel WHERE id = $1`, id).Scan(&von, &bis); err != nil {
			t.Fatal(err)
		}
		return [2]int{von, bis}
	}
	for name, fall := range map[string]struct {
		id   string
		soll [2]int
	}{
		"ohne Angabe (Vorgabe 5 bis 10)": {ohne, [2]int{0, 0}},
		"eigene Spanne 7 bis 9":          {eigen, [2]int{7, 9}},
		"ein Jahrgang 6":                 {einJahr, [2]int{6, 6}},
		"Mehrjahresband mit 5 bis 10":    {band, [2]int{5, 10}},
	} {
		if ist := spanne(fall.id); ist != fall.soll {
			t.Errorf("%s: nach der Migration %d bis %d, erwartet %d bis %d (0 = unbekannt)", name, ist[0], ist[1], fall.soll[0], fall.soll[1])
		}
	}

	// Danach gibt es keine Vorgabe mehr, und die Regel weist eine halbe Spanne ab.
	if ist := spanne(titel("M162 neu nach der Migration", "", "")); ist != [2]int{0, 0} {
		t.Errorf("ein neuer Titel bekommt nach der Migration %d bis %d, erwartet unbekannt", ist[0], ist[1])
	}
	halb, err := tx.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	_, err = halb.Exec(ctx, `UPDATE buecher_titel SET jahrgang_von = 7 WHERE id = $1`, ohne)
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) || pgErr.ConstraintName != "chk_jahrgang_spanne" {
		t.Errorf("halbe Spanne: die Datenbank muss abweisen (chk_jahrgang_spanne), bekam %v", err)
	}
	if err := halb.Rollback(ctx); err != nil {
		t.Fatalf("Sicherungspunkt zurückrollen: %v", err)
	}
}

// Migration 163: Die Spalte grade_level („Klasse") und ihre Bedingung entfallen. In die Spanne
// wird nichts übernommen: Ein Titel, der nur eine Klasse trug, bleibt ohne Jahrgang, eine
// eingetragene Spanne bleibt, wie sie ist. Geprüft wird die echte Migrationsdatei am Stand
// davor (Spalte und Bedingung wie in Migration 040), in einer Transaktion, die zurückgerollt
// wird. Dass sich die Spalte anlegen lässt, belegt zugleich, dass schema.sql sie nicht mehr
// führt.
func TestKlasseEntfaellt_Migration163(t *testing.T) {
	pool := pgtest.Pool(t)
	ctx := context.Background()
	tx := beginne(t, pool)
	defer func() {
		if err := tx.Rollback(ctx); err != nil {
			t.Errorf("zurückrollen: %v", err)
		}
	}()

	for _, sql := range []string{
		`ALTER TABLE buecher_titel ADD COLUMN grade_level SMALLINT`,
		`ALTER TABLE buecher_titel ADD CONSTRAINT chk_grade_level_bereich
			CHECK (grade_level IS NULL OR grade_level BETWEEN 0 AND 13)`,
	} {
		if _, err := tx.Exec(ctx, sql); err != nil {
			t.Fatalf("Stand vor der Migration herstellen: %v", err)
		}
	}
	titel := func(name, spalten, werte string) string {
		t.Helper()
		var id string
		if err := tx.QueryRow(ctx, `INSERT INTO buecher_titel (titel`+spalten+`) VALUES ($1`+werte+`) RETURNING id`, name).Scan(&id); err != nil {
			t.Fatalf("Titel %q: %v", name, err)
		}
		return id
	}
	nurKlasse := titel("M163 Klasse ohne Spanne", ", grade_level", ", 7")
	beides := titel("M163 Klasse und Spanne", ", grade_level, jahrgang_von, jahrgang_bis", ", 7, 7, 10")
	nichts := titel("M163 ohne Angabe", "", "")

	migration, err := os.ReadFile(filepath.Join("..", "migrations", "163_klasse_entfaellt.sql"))
	if err != nil {
		t.Fatalf("Migration lesen: %v", err)
	}
	// Zweimal: Der zweite Lauf findet nichts mehr vor und ändert nichts.
	for lauf := 1; lauf <= 2; lauf++ {
		if _, err := tx.Exec(ctx, string(migration)); err != nil {
			t.Fatalf("Lauf %d: Migration scheitert: %v", lauf, err)
		}
	}

	var spalten, bedingungen int
	if err := tx.QueryRow(ctx, `
		SELECT (SELECT count(*) FROM information_schema.columns
		        WHERE table_schema = current_schema() AND table_name = 'buecher_titel' AND column_name = 'grade_level'),
		       (SELECT count(*) FROM pg_constraint WHERE conname = 'chk_grade_level_bereich')`).
		Scan(&spalten, &bedingungen); err != nil {
		t.Fatal(err)
	}
	if spalten != 0 || bedingungen != 0 {
		t.Errorf("nach der Migration: %d Spalte grade_level, %d Bedingung chk_grade_level_bereich; erwartet 0 und 0", spalten, bedingungen)
	}

	for name, fall := range map[string]struct {
		id   string
		soll [2]int
	}{
		"nur eine Klasse: bleibt ohne Jahrgang": {nurKlasse, [2]int{0, 0}},
		"Klasse und Spanne: die Spanne bleibt":  {beides, [2]int{7, 10}},
		"ohne Angabe: bleibt ohne Jahrgang":     {nichts, [2]int{0, 0}},
	} {
		var von, bis int
		if err := tx.QueryRow(ctx, `SELECT coalesce(jahrgang_von, 0), coalesce(jahrgang_bis, 0)
			FROM buecher_titel WHERE id = $1`, fall.id).Scan(&von, &bis); err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if ist := [2]int{von, bis}; ist != fall.soll {
			t.Errorf("%s: Spanne %v, erwartet %v", name, ist, fall.soll)
		}
	}
}

// Die Theke liest zu jedem gescannten Buch, bis zu welchem Jahrgang ein Mehrjahresband beim
// Kind bleibt. An einem Titel ohne Jahrgang ist das 0: An der leeren Spalte darf der Scan
// nicht scheitern.
func TestGetCopyByBarcode_TitelOhneJahrgang(t *testing.T) {
	pool := pgtest.Pool(t)
	ctx := context.Background()
	const barcode = "JAHRGANG-162-THEKE"

	var titelID string
	if err := pool.QueryRow(ctx, `
		WITH t AS (INSERT INTO buecher_titel (titel) VALUES ('Jahrgang 162 an der Theke') RETURNING id)
		INSERT INTO buecher_exemplare (titel_id, barcode_id, ist_ausleihbar)
		SELECT id, $1, true FROM t RETURNING titel_id::text`, barcode).Scan(&titelID); err != nil {
		t.Fatalf("Titel und Exemplar anlegen: %v", err)
	}
	t.Cleanup(func() {
		for _, sql := range []string{
			`DELETE FROM buecher_exemplare WHERE titel_id::text = $1`,
			`DELETE FROM buecher_titel WHERE id::text = $1`,
		} {
			if _, err := pool.Exec(context.Background(), sql, titelID); err != nil {
				t.Errorf("aufräumen: %v", err)
			}
		}
	})

	exemplar, err := NewBookRepository(pool).GetCopyByBarcode(ctx, barcode)
	if err != nil || exemplar == nil {
		t.Fatalf("Exemplar eines Titels ohne Jahrgang lesen: %v (gefunden: %v)", err, exemplar != nil)
	}
	if exemplar.JahrgangBis != 0 || exemplar.Mehrjahresband {
		t.Errorf("gelesen: bis Jahrgang %d, Mehrjahresband %v; erwartet 0 und false", exemplar.JahrgangBis, exemplar.Mehrjahresband)
	}
}

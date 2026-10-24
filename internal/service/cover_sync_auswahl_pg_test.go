package service

import (
	"context"
	"os"
	"regexp"
	"testing"

	"bibliothek/internal/pgtest"
	"bibliothek/repository"
)

// Der Cover-Abgleich darf ein lokal liegendes Cover nicht anfassen, egal, was cover_status
// sagt: Sonst überschreibt der Treffer der Katalogdienste ein von Hand hochgeladenes
// /uploads/-Cover. Die Wege von Hand setzen den Status selbst
// (repository/titel_metadaten.go); die Auswahl schützt auch Titel, deren Status nicht dazu
// passt (Bugklasse „Maschine gegen Hand").
func TestCoverAbgleichAuswahl_LaesstLokaleCoverInRuhe(t *testing.T) {
	pool := pgtest.Pool(t)
	ctx := context.Background()
	faelle := map[string]struct {
		isbn, status, url string
		gewaehlt          bool
	}{
		"hand-hochgeladen, noch PENDING": {"978-9-99-100000-1", "PENDING", "/uploads/covers/hand.webp", false},
		"unversucht ohne Cover":          {"978-9-99-100000-2", "PENDING", "", true},
		"fehlgeschlagen ohne Cover":      {"978-9-99-100000-3", "FAILED", "", true},
		"extern, zu migrieren":           {"978-9-99-100000-4", "FOUND", "https://portal.dnb.de/x", true},
		"lokal gefunden":                 {"978-9-99-100000-5", "FOUND", "/uploads/covers/ok.webp", false},
	}
	ids := map[string]string{}
	for name, f := range faelle {
		var id string
		if err := pool.QueryRow(ctx, `INSERT INTO buecher_titel (titel, isbn, cover_status, cover_url)
			VALUES ($1, $2, $3, NULLIF($4, '')) ON CONFLICT (isbn) DO UPDATE SET cover_status = EXCLUDED.cover_status,
			cover_url = EXCLUDED.cover_url RETURNING id`, name, f.isbn, f.status, f.url).Scan(&id); err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		ids[id] = name
		t.Cleanup(func() {
			if _, err := pool.Exec(context.Background(), `DELETE FROM buecher_titel WHERE id = $1`, id); err != nil {
				t.Logf("Aufräumen: %v", err)
			}
		})
	}
	rows, err := pool.Query(ctx, repository.CoverAbgleichAuswahl)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	gewaehlt := map[string]bool{}
	for rows.Next() {
		var id, isbn string
		if err := rows.Scan(&id, &isbn); err != nil {
			t.Fatal(err)
		}
		if name, ok := ids[id]; ok {
			gewaehlt[name] = true
		}
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	for name, f := range faelle {
		if gewaehlt[name] != f.gewaehlt {
			t.Errorf("%s: vom Sync gewählt = %v, erwartet %v", name, gewaehlt[name], f.gewaehlt)
		}
	}
}

// Das Rezept aus docs/DEPLOYMENT.md („nach einem Restore ohne Volume"), wörtlich ausgeführt: Es
// muss die Titel in die Auswahl bringen. cover_status = 'PENDING' allein genügt nicht, weil die
// Auswahl oben lokale Pfade ausschließt; der nächste Lauf heilte dann nichts.
//
// In einer Transaktion, die zurückgerollt wird: Das Rezept trifft jeden Titel mit lokalem Pfad,
// und die Pakete teilen sich eine Datenbank.
func TestCoverRezeptNachRestore_WirktAufDieAuswahl(t *testing.T) {
	roh, err := os.ReadFile("../../docs/DEPLOYMENT.md")
	if err != nil {
		t.Fatalf("DEPLOYMENT.md lesen: %v", err)
	}
	treffer := regexp.MustCompile("(?s)```sql\\s*(UPDATE buecher_titel SET cover_status[^`]*?)```").FindSubmatch(roh)
	if treffer == nil {
		t.Fatal("DEPLOYMENT.md trägt das Cover-Rezept nicht mehr als ```sql-Block, der mit " +
			"„UPDATE buecher_titel SET cover_status“ beginnt — Rezept oder Gate nachziehen")
	}
	rezept := string(treffer[1])

	pool := pgtest.Pool(t)
	ctx := context.Background()
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := tx.Rollback(ctx); err != nil {
			t.Logf("Rollback: %v", err)
		}
	}()

	var id string
	if err := tx.QueryRow(ctx, `INSERT INTO buecher_titel (titel, isbn, cover_status, cover_url)
		VALUES ('Rezeptprobe', '978-9-99-100000-9', 'FOUND', '/uploads/covers/weg.webp') RETURNING id`).Scan(&id); err != nil {
		t.Fatalf("Titel anlegen: %v", err)
	}
	gewaehlt := func() bool {
		var n int
		if err := tx.QueryRow(ctx, `SELECT count(*) FROM (`+repository.CoverAbgleichAuswahl+`) a WHERE a.id = $1`, id).Scan(&n); err != nil {
			t.Fatalf("Auswahl lesen: %v", err)
		}
		return n == 1
	}
	if gewaehlt() {
		t.Fatal("Gegenprobe: Der Titel mit lokalem Pfad steht schon VOR dem Rezept in der Auswahl — der Test misst nichts")
	}
	if _, err := tx.Exec(ctx, rezept); err != nil {
		t.Fatalf("Rezept aus DEPLOYMENT.md ausführen: %v\n%s", err, rezept)
	}
	if !gewaehlt() {
		t.Errorf("Nach dem Rezept aus DEPLOYMENT.md fasst der Cover-Sync den Titel weiter nicht an — "+
			"das Rezept ist wirkungslos:\n%s", rezept)
	}
}

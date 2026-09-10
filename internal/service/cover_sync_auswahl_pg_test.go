package service

import (
	"context"
	"testing"

	"bibliothek/internal/pgtest"
)

// Der Cover-Sync darf ein lokal liegendes Cover nicht anfassen — egal, was cover_status
// sagt. Bis zum 10.09.2026 wählte er jeden Titel auf 'PENDING'/'FAILED', auch mit einem
// von Hand hochgeladenen /uploads/-Cover, und überschrieb es mit dem Treffer der
// Katalogdienste. Die Hand-Wege setzen den Status seitdem (inventur/repository_metadata.go);
// dieses Prädikat schützt zusätzlich die Titel, die schon vorher so hochgeladen wurden
// (Bestands-Durchgang, Bugklasse „Maschine gegen Hand").
func TestCoverSyncAuswahl_LaesstLokaleCoverInRuhe(t *testing.T) {
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
	rows, err := pool.Query(ctx, coverSyncAuswahl)
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
			t.Errorf("%s: vom Sync gewählt = %v, want %v", name, gewaehlt[name], f.gewaehlt)
		}
	}
}

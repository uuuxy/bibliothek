package inventur

import (
	"context"
	"testing"

	"bibliothek/internal/pgtest"
)

// Ein von Hand hochgeladenes Cover ist die Entscheidung eines Menschen. Der Abgleich der Cover
// (internal/service/cover.go) wählt seine Titel über cover_status; setzten die Wege von Hand den
// Status nicht, stünde ein neuer Titel auf 'PENDING', die Bibliothekarin lüde das richtige Cover
// hoch, und beim nächsten Lauf (alle 6 Stunden, beim Neustart) stünde wieder das Cover der DNB von
// der falschen Auflage da, ohne Meldung.
func TestHandUpload_SetztCoverStatusFound(t *testing.T) {
	pool := pgtest.Pool(t)
	ctx := context.Background()
	var id string
	if err := pool.QueryRow(ctx, `INSERT INTO buecher_titel (titel, isbn) VALUES ('Hand-Cover-Titel', '978-9-99-999999-1')
		ON CONFLICT (isbn) DO UPDATE SET cover_status = 'PENDING', cover_url = NULL RETURNING id`).Scan(&id); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if _, err := pool.Exec(context.Background(), `DELETE FROM buecher_titel WHERE id = $1`, id); err != nil {
			t.Logf("Aufräumen: %v", err)
		}
	})

	if err := NewTitelAbfragen(pool).AendereTitelMetadaten(ctx, id, "", "", "/uploads/covers/hand.webp"); err != nil {
		t.Fatal(err)
	}
	var status string
	if err := pool.QueryRow(ctx, `SELECT cover_status FROM buecher_titel WHERE id = $1`, id).Scan(&status); err != nil {
		t.Fatal(err)
	}
	if status != "FOUND" {
		t.Errorf("nach Hand-Upload cover_status = %q — der Cover-Sync hält den Titel für unversucht und überschreibt das Cover", status)
	}
}

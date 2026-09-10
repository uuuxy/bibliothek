package inventur

import (
	"context"
	"testing"

	"bibliothek/internal/pgtest"
)

// Ein von Hand hochgeladenes Cover ist die Entscheidung eines Menschen. Der Cover-Sync
// (internal/service/cover_service.go) wählt seine Titel aber über cover_status — und den
// setzten die Hand-Wege nie: Ein neuer Titel steht auf 'PENDING', die Bibliothekarin lädt
// das richtige Cover hoch, und beim nächsten Lauf (alle 6 Stunden, beim Neustart) stand
// wieder das DNB-Cover der falschen Auflage da, ohne Meldung (Bestands-Durchgang
// 10.09.2026, neue Bugklasse „Maschine gegen Hand").
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

	if err := NewBookRepository(pool).UpdateBookMetadata(ctx, id, "", "", "/uploads/covers/hand.webp"); err != nil {
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

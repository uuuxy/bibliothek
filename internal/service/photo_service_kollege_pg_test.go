package service

import (
	"context"
	"testing"

	"bibliothek/internal/crypto"
	"bibliothek/internal/pgtest"
)

// Das Passbild eines Kollegen: Die Akte ist eine Maske für jeden und zeigt auch ihm den
// Kamera-Knopf; die Auslieferung (api/photo_serve.go) liest seit dem 17.09.2026 die Tabelle
// `leser`, damit sein Bild herauskommt. Der Upload las bis zum 22.09.2026 aber noch die
// Sicht `schueler` und antwortete einem Kollegen „schüler nicht gefunden" (404) — die eine
// Richtung war repariert, die andere nicht (OFFEN.md 5.19).
func TestUploadStudentPhoto_AuchFuerKollegen(t *testing.T) {
	pool := pgtest.Pool(t)
	ctx := context.Background()
	t.Setenv(crypto.SchluesselVariable, "12345678901234567890123456789012")

	var kollegeID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO leser (barcode_id, vorname, nachname, art)
		VALUES ('A-FOTO-K1', 'Kim', 'Fotoprobe', 'lehrkraft') RETURNING id`).Scan(&kollegeID); err != nil {
		t.Fatalf("Kollege anlegen: %v", err)
	}
	t.Cleanup(func() {
		if _, err := pool.Exec(ctx, `DELETE FROM leser WHERE id = $1`, kollegeID); err != nil {
			t.Errorf("aufräumen: %v", err)
		}
	})

	// 10×10 Pixel PNG, dieselbe Probe wie photo_service_test.go.
	const bild = "data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAoAAAAKCAIAAAACUFjqAAAAF0lEQVR4nGL5z4APMOGVHbHSgAAAAP//RM4BFjLZ0j4AAAAASUVORK5CYII="
	url, err := UploadStudentPhoto(ctx, pool, kollegeID, bild)
	if err != nil {
		t.Fatalf("Upload für einen Kollegen: %v", err)
	}
	if url != "/api/schueler/A-FOTO-K1/photo" {
		t.Errorf("Bild-URL: %q", url)
	}
	var n int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM schueler_fotos WHERE schueler_id = $1`, kollegeID).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Errorf("%d Zeile(n) in schueler_fotos für den Kollegen, erwartet 1", n)
	}
}

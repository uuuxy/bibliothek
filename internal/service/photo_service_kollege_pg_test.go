package service

import (
	"bytes"
	"context"
	"encoding/base64"
	"image"
	"image/color"
	"image/draw"
	"image/png"
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

// Ein zweites Foto ersetzt das erste: Es bleibt bei einer Zeile je Leser, und in ihr steht
// das neue Bild. Bliebe das alte stehen, zeigte der Ausweis nach einem neuen Foto weiter das
// vom Vorjahr, und die Antwort des Uploads hätte trotzdem Erfolg gemeldet.
func TestUploadStudentPhoto_ZweitesFotoErsetztDasErste(t *testing.T) {
	pool := pgtest.Pool(t)
	ctx := context.Background()
	t.Setenv(crypto.SchluesselVariable, "12345678901234567890123456789012")

	var leserID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO leser (barcode_id, vorname, nachname, art)
		VALUES ('A-FOTO-K2', 'Kim', 'Zweitfoto', 'lehrkraft') RETURNING id`).Scan(&leserID); err != nil {
		t.Fatalf("Leser anlegen: %v", err)
	}
	t.Cleanup(func() {
		if _, err := pool.Exec(ctx, `DELETE FROM leser WHERE id = $1`, leserID); err != nil {
			t.Errorf("aufräumen: %v", err)
		}
	})

	// bild liefert ein einfarbiges PNG als Data-URL; zwei Farben ergeben zwei verschiedene Bilder.
	bild := func(farbe color.RGBA) string {
		t.Helper()
		flaeche := image.NewRGBA(image.Rect(0, 0, 12, 12))
		draw.Draw(flaeche, flaeche.Bounds(), &image.Uniform{C: farbe}, image.Point{}, draw.Src)
		var puffer bytes.Buffer
		if err := png.Encode(&puffer, flaeche); err != nil {
			t.Fatalf("PNG erzeugen: %v", err)
		}
		return "data:image/png;base64," + base64.StdEncoding.EncodeToString(puffer.Bytes())
	}
	gespeichert := func() []byte {
		t.Helper()
		var zeilen int
		if err := pool.QueryRow(ctx, `SELECT count(*) FROM schueler_fotos WHERE schueler_id = $1`, leserID).Scan(&zeilen); err != nil {
			t.Fatal(err)
		}
		if zeilen != 1 {
			t.Fatalf("%d Zeilen in schueler_fotos, erwartet 1", zeilen)
		}
		var verschluesselt []byte
		if err := pool.QueryRow(ctx, `SELECT foto_encrypted FROM schueler_fotos WHERE schueler_id = $1`, leserID).Scan(&verschluesselt); err != nil {
			t.Fatal(err)
		}
		klar, err := crypto.Decrypt(verschluesselt)
		if err != nil {
			t.Fatalf("gespeichertes Foto nicht entschlüsselbar: %v", err)
		}
		return klar
	}

	if _, err := UploadStudentPhoto(ctx, pool, leserID, bild(color.RGBA{R: 200, A: 255})); err != nil {
		t.Fatalf("erstes Foto: %v", err)
	}
	erstes := gespeichert()
	if _, err := UploadStudentPhoto(ctx, pool, leserID, bild(color.RGBA{B: 200, A: 255})); err != nil {
		t.Fatalf("zweites Foto: %v", err)
	}
	zweites := gespeichert()
	if bytes.Equal(erstes, zweites) {
		t.Error("nach dem zweiten Upload steht noch das erste Foto in der Tabelle")
	}
}

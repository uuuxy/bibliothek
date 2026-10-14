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
// Kamera-Knopf. Auslieferung (api/leser/foto.go) und Upload lesen deshalb beide die
// Tabelle `leser`, nicht die Sicht `schueler`; sonst antwortet der Upload einem Kollegen
// „schüler nicht gefunden" (404).
func TestSchreibeLeserFoto_AuchFuerKollegen(t *testing.T) {
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

	// 10×10 Pixel PNG, dieselbe Probe wie foto_test.go.
	const bild = "data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAoAAAAKCAIAAAACUFjqAAAAF0lEQVR4nGL5z4APMOGVHbHSgAAAAP//RM4BFjLZ0j4AAAAASUVORK5CYII="
	url, err := SchreibeLeserFoto(ctx, pool, kollegeID, bild)
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
func TestSchreibeLeserFoto_ZweitesFotoErsetztDasErste(t *testing.T) {
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
		klar, err := crypto.Entschluessele(verschluesselt)
		if err != nil {
			t.Fatalf("gespeichertes Foto nicht entschlüsselbar: %v", err)
		}
		return klar
	}

	if _, err := SchreibeLeserFoto(ctx, pool, leserID, bild(color.RGBA{R: 200, A: 255})); err != nil {
		t.Fatalf("erstes Foto: %v", err)
	}
	erstes := gespeichert()
	if _, err := SchreibeLeserFoto(ctx, pool, leserID, bild(color.RGBA{B: 200, A: 255})); err != nil {
		t.Fatalf("zweites Foto: %v", err)
	}
	zweites := gespeichert()
	if bytes.Equal(erstes, zweites) {
		t.Error("nach dem zweiten Upload steht noch das erste Foto in der Tabelle")
	}
}

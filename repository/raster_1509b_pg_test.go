//go:build raster

package repository

// Nachstellung Rasterdurchgang 15.09.2026 spät, über Migration 118 und 119. Build-Tag raster:
// läuft nur mit -tags raster. Rot heißt „bestätigt".

import (
	"context"
	"testing"

	"bibliothek/internal/pgtest"
)

// Verdacht (Frage 3, zwei Wahrheitsquellen): Seit Migration 119 sagt die Personenart, WER eine
// Lehrkraft ist. Theke (GetLehrerByBarcode) und Geräteausleihe (ladeAktiveLehrkraft) erkennen
// eine Lehrkraft aber weiter an der Rolle kollegium. Eine Lehrkraft, die in der Bibliothek
// mitarbeitet, hat die Rolle Mitarbeiter — ihr Ausweis findet an der Theke niemanden.
func TestRaster_LehrkraftMitRolleMitarbeiterFindetDieThekeNicht(t *testing.T) {
	pool := pgtest.Pool(t)
	ctx := context.Background()
	t.Cleanup(func() {
		if _, err := pool.Exec(ctx, `DELETE FROM benutzer WHERE email LIKE '%@raster1509b.invalid'`); err != nil {
			t.Errorf("aufräumen: %v", err)
		}
	})
	anlegen := func(barcode, email, rolle string) {
		if _, err := pool.Exec(ctx, `INSERT INTO benutzer (barcode_id, vorname, nachname, email, rolle, aktiv, personenart)
			VALUES ($1, 'Lehr', 'Kraft', $2, $3::benutzer_rolle, true, 'lehrkraft')`, barcode, email, rolle); err != nil {
			t.Fatalf("Konto anlegen: %v", err)
		}
	}
	repo := NewUserRepository(pool)

	// Gegenprobe: dieselbe Lehrkraft mit Rolle kollegium findet die Theke.
	anlegen("RB-KOL", "kol@raster1509b.invalid", "kollegium")
	if u, err := repo.GetLehrerByBarcode(ctx, "RB-KOL"); err != nil || u == nil {
		t.Fatalf("Gegenprobe: Lehrkraft mit Rolle kollegium nicht gefunden (%v, %v)", u, err)
	}

	anlegen("RB-MIT", "mit@raster1509b.invalid", "mitarbeiter")
	u, err := repo.GetLehrerByBarcode(ctx, "RB-MIT")
	if err != nil {
		t.Fatalf("GetLehrerByBarcode: %v", err)
	}
	if u == nil {
		t.Error("bestätigt: Eine Lehrkraft mit Rolle Mitarbeiter findet die Theke über ihren Ausweis nicht")
	}
}

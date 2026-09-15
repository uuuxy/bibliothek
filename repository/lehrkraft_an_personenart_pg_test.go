package repository

// Wer an der Theke als Lehrkraft gilt, entscheidet die Personenart (Lehrkraft, LiV), nicht die
// Rolle (Peter, 15.09.2026). Bis dahin nur die Rolle kollegium: Eine Lehrkraft, die in der
// Bibliothek mitarbeitet (Rolle Mitarbeiter), fand die Theke über ihren Ausweis nicht
// (Rasterdurchgang 15.09.2026, nachgestellt unter dem Build-Tag raster).

import (
	"context"
	"testing"

	"bibliothek/internal/pgtest"
)

func TestGetLehrerByBarcode_EntscheidetDiePersonenart(t *testing.T) {
	pool := pgtest.Pool(t)
	ctx := context.Background()
	t.Cleanup(func() {
		if _, err := pool.Exec(ctx, `DELETE FROM benutzer WHERE email LIKE '%@lehrkraft-personenart.invalid'`); err != nil {
			t.Errorf("aufräumen: %v", err)
		}
	})
	anlegen := func(barcode, rolle string, personenart *string, aktiv bool) {
		t.Helper()
		if _, err := pool.Exec(ctx, `INSERT INTO benutzer (barcode_id, vorname, nachname, email, rolle, aktiv, personenart)
			VALUES ($1, 'Lehr', 'Kraft', $2, $3::benutzer_rolle, $4, $5)`,
			barcode, barcode+"@lehrkraft-personenart.invalid", rolle, aktiv, personenart); err != nil {
			t.Fatalf("Konto %s anlegen: %v", barcode, err)
		}
	}
	lehrkraft, liv := "lehrkraft", "liv"
	anlegen("LP-KOL", "kollegium", &lehrkraft, true)
	anlegen("LP-MIT-LK", "mitarbeiter", &lehrkraft, true)
	anlegen("LP-HELF-LIV", "helfer", &liv, true)
	anlegen("LP-MIT", "mitarbeiter", nil, true)
	anlegen("LP-ALT", "kollegium", &lehrkraft, false)

	repo := NewUserRepository(pool)
	for barcode, gefunden := range map[string]bool{
		"LP-KOL":      true,  // Lehrkraft mit Portal-Rolle
		"LP-MIT-LK":   true,  // Lehrkraft, die in der Bibliothek mitarbeitet
		"LP-HELF-LIV": true,  // LiV mit Helfer-Rolle
		"LP-MIT":      false, // Mitarbeiterin ohne Personenart ist keine Lehrkraft
		"LP-ALT":      false, // deaktiviertes Konto
	} {
		u, err := repo.GetLehrerByBarcode(ctx, barcode)
		if err != nil {
			t.Fatalf("%s: %v", barcode, err)
		}
		if (u != nil) != gefunden {
			t.Errorf("%s: gefunden=%v, erwartet %v", barcode, u != nil, gefunden)
		}
	}
}

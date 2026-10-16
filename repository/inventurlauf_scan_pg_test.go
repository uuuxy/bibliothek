package repository

import (
	"context"
	"testing"
)

// NewInventurAbfragen liefert die Abfragen der Inventur.
func TestNewInventurAbfragen(t *testing.T) {
	pool := pgTestPool(t)
	repo := NewInventurAbfragen(pool)
	if repo == nil {
		t.Fatal("NewInventurAbfragen(pool) = nil, erwartet die Abfragen")
	}
	if repo.db == nil {
		t.Fatal("NewInventurAbfragen(pool).db = nil, erwartet den übergebenen Pool")
	}
}

// LadeExemplar lädt, was der Scan der Inventur vom Exemplar braucht.
func TestInventurLadeExemplar(t *testing.T) {
	pool := pgTestPool(t)
	resetInventurDaten(t, pool)
	ctx := context.Background()
	repo := NewInventurAbfragen(pool)

	// 1. Titel anlegen
	var titelID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO buecher_titel (titel, autor, signatur, cover_url)
		VALUES ('Der Hobbit', 'J.R.R. Tolkien', 'Fantasie', 'http://example.com/cover.jpg')
		RETURNING id
	`).Scan(&titelID); err != nil {
		t.Fatalf("Titel anlegen: %v", err)
	}

	var titelOhneCoverID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO buecher_titel (titel, autor, signatur)
		VALUES ('Herr der Ringe', 'J.R.R. Tolkien', 'Fantasie')
		RETURNING id
	`).Scan(&titelOhneCoverID); err != nil {
		t.Fatalf("Titel ohne Cover anlegen: %v", err)
	}

	// 2. Exemplare anlegen
	// Fall 1: Normales Exemplar (nicht ausgeliehen, nicht ausgesondert)
	var normalExID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO buecher_exemplare (titel_id, barcode_id, ist_ausleihbar)
		VALUES ($1, $2, true) RETURNING id
	`, titelID, "SCAN-NORM").Scan(&normalExID); err != nil {
		t.Fatalf("Normales Exemplar anlegen: %v", err)
	}

	// Fall 2: Ohne CoverURL
	var ohneCoverExID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO buecher_exemplare (titel_id, barcode_id, ist_ausleihbar)
		VALUES ($1, $2, true) RETURNING id
	`, titelOhneCoverID, "SCAN-NOCOVER").Scan(&ohneCoverExID); err != nil {
		t.Fatalf("Exemplar ohne Cover anlegen: %v", err)
	}

	// Fall 3: Ausgesondert. chk_aussonderung_grund verlangt bei ist_ausgesondert = true einen
	// Grund; ohne ihn scheitert schon das Anlegen mit SQLSTATE 23514.
	var ausgesondertExID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO buecher_exemplare (titel_id, barcode_id, ist_ausgesondert, aussonderung_grund, ist_ausleihbar)
		VALUES ($1, $2, true, 'AUSSORTIERT', false) RETURNING id
	`, titelID, "SCAN-AUSG").Scan(&ausgesondertExID); err != nil {
		t.Fatalf("Ausgesondertes Exemplar anlegen: %v", err)
	}

	// Fall 4: Ausgeliehen
	var verliehenExID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO buecher_exemplare (titel_id, barcode_id, ist_ausleihbar)
		VALUES ($1, $2, true) RETURNING id
	`, titelID, "SCAN-VERL").Scan(&verliehenExID); err != nil {
		t.Fatalf("Verliehenes Exemplar anlegen: %v", err)
	}

	// Für das verliehene Exemplar eine laufende Ausleihe anlegen; dafür braucht es einen
	// Schüler. klasse und abgaenger_jahr sind NOT NULL ohne Vorgabe.
	var schuelerID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO schueler (vorname, nachname, barcode_id, klasse, abgaenger_jahr)
		VALUES ('Max', 'Mustermann', 'S-1234', '5a', 2030) RETURNING id
	`).Scan(&schuelerID); err != nil {
		t.Fatalf("Schüler anlegen: %v", err)
	}

	// Die Ausleihe hängt an schueler_id (nicht an einer benutzer_id) und braucht eine
	// Rückgabefrist; ausgeliehen_am setzt die Tabelle selbst.
	if _, err := pool.Exec(ctx, `
		INSERT INTO ausleihen (exemplar_id, schueler_id, rueckgabe_frist)
		VALUES ($1, $2, CURRENT_DATE + 14)
	`, verliehenExID, schuelerID); err != nil {
		t.Fatalf("Ausleihe anlegen: %v", err)
	}

	t.Run("Normales Exemplar mit Cover", func(t *testing.T) {
		res, err := repo.LadeExemplar(ctx, "SCAN-NORM")
		if err != nil {
			t.Fatalf("Fehler beim Laden: %v", err)
		}
		if res.ExemplarID != normalExID {
			t.Errorf("erwartete CopyID %q, war %q", normalExID, res.ExemplarID)
		}
		if res.Titel != "Der Hobbit" {
			t.Errorf("erwarteter Titel 'Der Hobbit', war %q", res.Titel)
		}
		if res.CoverURL != "http://example.com/cover.jpg" {
			t.Errorf("erwartete CoverURL 'http://example.com/cover.jpg', war %q", res.CoverURL)
		}
		if res.IstAusgesondert {
			t.Error("erwartete IsAusgesondert = false, war true")
		}
		if res.IstVerliehen {
			t.Error("erwartete IsLent = false, war true")
		}
	})

	t.Run("Titel ohne Cover - coalesce to empty string", func(t *testing.T) {
		res, err := repo.LadeExemplar(ctx, "SCAN-NOCOVER")
		if err != nil {
			t.Fatalf("Fehler beim Laden: %v", err)
		}
		if res.ExemplarID != ohneCoverExID {
			t.Errorf("erwartete CopyID %q, war %q", ohneCoverExID, res.ExemplarID)
		}
		if res.CoverURL != "" {
			t.Errorf("erwartete leere CoverURL (COALESCE greift nicht?), war %q", res.CoverURL)
		}
	})

	t.Run("Ausgesondertes Exemplar", func(t *testing.T) {
		res, err := repo.LadeExemplar(ctx, "SCAN-AUSG")
		if err != nil {
			t.Fatalf("Fehler beim Laden: %v", err)
		}
		if res.ExemplarID != ausgesondertExID {
			t.Errorf("erwartete CopyID %q, war %q", ausgesondertExID, res.ExemplarID)
		}
		if !res.IstAusgesondert {
			t.Error("erwartete IsAusgesondert = true, war false")
		}
	})

	t.Run("Verliehenes Exemplar", func(t *testing.T) {
		res, err := repo.LadeExemplar(ctx, "SCAN-VERL")
		if err != nil {
			t.Fatalf("Fehler beim Laden: %v", err)
		}
		if res.ExemplarID != verliehenExID {
			t.Errorf("erwartete CopyID %q, war %q", verliehenExID, res.ExemplarID)
		}
		if !res.IstVerliehen {
			t.Error("erwartete IsLent = true (EXISTS Subquery greift nicht?), war false")
		}
	})

	t.Run("Nicht existierendes Exemplar", func(t *testing.T) {
		res, err := repo.LadeExemplar(ctx, "GIBT-ES-NICHT")
		if err == nil {
			t.Error("erwartete einen Fehler bei unbekanntem Barcode, war nil")
		}
		if res != nil {
			t.Errorf("erwartete res = nil, war %+v", res)
		}
	})
}

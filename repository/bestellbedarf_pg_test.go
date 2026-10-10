package repository

import (
	"context"
	"testing"

	"bibliothek/internal/pgtest"
)

// Der Bestellbedarf nennt für einen Titel ohne eigenes Cover die Cover-Adresse der DNB zu
// seiner ISBN, wie Zulauf und Bestellsuche (sqlCoverOderDNB).
func TestListeBestellbedarf_OhneEigenesCoverDieAdresseDerDNB(t *testing.T) {
	pool := pgtest.Pool(t)
	ctx := context.Background()
	tx := beginne(t, pool)
	defer func() {
		if err := tx.Rollback(ctx); err != nil {
			t.Errorf("zurückrollen: %v", err)
		}
	}()

	var id string
	if err := tx.QueryRow(ctx, `INSERT INTO buecher_titel (titel, isbn) VALUES ('Bedarf-Cover Probe', '9780000577092')
		RETURNING id::text`).Scan(&id); err != nil {
		t.Fatalf("Titel anlegen: %v", err)
	}

	liste, err := ListeBestellbedarf(ctx, tx, "", 1)
	if err != nil {
		t.Fatalf("ListeBestellbedarf: %v", err)
	}
	for _, zeile := range liste {
		if zeile.ID != id {
			continue
		}
		if want := "https://portal.dnb.de/opac/mvb/cover?isbn=9780000577092"; zeile.CoverURL != want {
			t.Errorf("Cover %q, erwartet %q", zeile.CoverURL, want)
		}
		return
	}
	t.Errorf("der Titel ohne Exemplar fehlt unter %d Zeilen des Bestellbedarfs", len(liste))
}

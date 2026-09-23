package api

import (
	"context"
	"testing"

	"bibliothek/db"
	"bibliothek/inventur"
)

// Bestellen per ISBN legt den Titel aus der DNB an und verwarf dabei Untertitel und
// Ladenpreis, obwohl es für beide eine Spalte gibt (docs/OFFEN.md 5.5; listenpreis seit
// Migration 127). Beim Anlegen über das Buchformular füllt ergaenzeBuchMetadaten den
// Listenpreis aus derselben Quelle — der Bestellweg nimmt jetzt dieselbe Regel
// (inventur.ListenpreisAusNachschlagen): Ein Preis von 0 füllt nichts, und was schon
// erfasst ist, gewinnt.
func TestUpsertTitelAusMetadaten_UntertitelUndLadenpreis(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	ctx := context.Background()
	srv := &Server{DB: &db.Database{Pool: pool}}

	lies := func(titelID string) (untertitel *string, listenpreis *float64) {
		t.Helper()
		if err := pool.QueryRow(ctx,
			`SELECT untertitel, listenpreis::float8 FROM buecher_titel WHERE id = $1`, titelID).
			Scan(&untertitel, &listenpreis); err != nil {
			t.Fatalf("lesen: %v", err)
		}
		return untertitel, listenpreis
	}

	// 1. Neuer Titel: beides kommt an.
	resp, err := srv.upsertTitelAusMetadaten(ctx, "9783551555551", &inventur.MetadatenErgebnis{
		Titel: "Wolkenkind", Untertitel: "Ein Roman über Freundschaft", Autor: "Muster, Erika", Preis: 14.99,
	})
	if err != nil {
		t.Fatalf("anlegen: %v", err)
	}
	if u, p := lies(resp.TitelID); u == nil || *u != "Ein Roman über Freundschaft" || p == nil || *p != 14.99 {
		t.Errorf("neuer Titel: untertitel=%v listenpreis=%v, want „Ein Roman über Freundschaft“ und 14.99", u, p)
	}

	// 2. Ein Preis von 0 heißt „nicht ermittelbar", nicht „kostet nichts".
	resp0, err := srv.upsertTitelAusMetadaten(ctx, "9783551555568", &inventur.MetadatenErgebnis{
		Titel: "Ohne Preis", Preis: 0,
	})
	if err != nil {
		t.Fatalf("anlegen ohne Preis: %v", err)
	}
	if u, p := lies(resp0.TitelID); u != nil || p != nil {
		t.Errorf("ohne Preis und Untertitel: untertitel=%v listenpreis=%v, want beide NULL", u, p)
	}

	// 3. Der Titel ist schon da (ON CONFLICT): Was erfasst ist, gewinnt.
	if _, err := pool.Exec(ctx,
		`UPDATE buecher_titel SET untertitel = 'Gepflegt', listenpreis = 20.00 WHERE id = $1`, resp.TitelID); err != nil {
		t.Fatalf("pflegen: %v", err)
	}
	if _, err := srv.upsertTitelAusMetadaten(ctx, "9783551555551", &inventur.MetadatenErgebnis{
		Titel: "Wolkenkind", Untertitel: "Anderer Untertitel", Preis: 9.99,
	}); err != nil {
		t.Fatalf("erneut: %v", err)
	}
	if u, p := lies(resp.TitelID); u == nil || *u != "Gepflegt" || p == nil || *p != 20.00 {
		t.Errorf("vorhandener Titel: untertitel=%v listenpreis=%v, want „Gepflegt“ und 20.00", u, p)
	}
}

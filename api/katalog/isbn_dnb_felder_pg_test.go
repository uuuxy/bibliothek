package katalog

import (
	"context"
	"testing"

	"bibliothek/api/tuer"
	"bibliothek/db"
	"bibliothek/internal/katalogdienste"
	"bibliothek/internal/pgtest"
	"bibliothek/internal/testdaten"
)

// Bestellen per ISBN legt den Titel aus der DNB an, mit Untertitel und Ladenpreis. Den Preis
// übernimmt der Bestellweg nach der Regel aus inventur.ListenpreisAusNachschlagen: Ein Preis
// von 0 füllt nichts, und was schon erfasst ist, gewinnt.
func TestSchreibeTitelAusMetadaten_UntertitelUndLadenpreis(t *testing.T) {
	pool := pgtest.Pool(t)
	testdaten.LeereBestandsdaten(t, pool)
	ctx := context.Background()
	srv := &Tueren{Basis: &tuer.Basis{DB: &db.Verbindung{Pool: pool}}}

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
	resp, err := srv.schreibeTitelAusMetadaten(ctx, "9783551555551", &katalogdienste.MetadatenErgebnis{
		Titel: "Wolkenkind", Untertitel: "Ein Roman über Freundschaft", Autor: "Muster, Erika", Preis: 14.99,
	})
	if err != nil {
		t.Fatalf("anlegen: %v", err)
	}
	if u, p := lies(resp.TitelID); u == nil || *u != "Ein Roman über Freundschaft" || p == nil || *p != 14.99 {
		t.Errorf("neuer Titel: untertitel=%v listenpreis=%v, erwartet „Ein Roman über Freundschaft“ und 14.99", u, p)
	}

	// 2. Ein Preis von 0 heißt „nicht ermittelbar", nicht „kostet nichts".
	resp0, err := srv.schreibeTitelAusMetadaten(ctx, "9783551555568", &katalogdienste.MetadatenErgebnis{
		Titel: "Ohne Preis", Preis: 0,
	})
	if err != nil {
		t.Fatalf("anlegen ohne Preis: %v", err)
	}
	if u, p := lies(resp0.TitelID); u != nil || p != nil {
		t.Errorf("ohne Preis und Untertitel: untertitel=%v listenpreis=%v, erwartet beide NULL", u, p)
	}

	// 3. Der Titel ist schon da (ON CONFLICT): Was erfasst ist, gewinnt.
	if _, err := pool.Exec(ctx,
		`UPDATE buecher_titel SET untertitel = 'Gepflegt', listenpreis = 20.00 WHERE id = $1`, resp.TitelID); err != nil {
		t.Fatalf("pflegen: %v", err)
	}
	if _, err := srv.schreibeTitelAusMetadaten(ctx, "9783551555551", &katalogdienste.MetadatenErgebnis{
		Titel: "Wolkenkind", Untertitel: "Anderer Untertitel", Preis: 9.99,
	}); err != nil {
		t.Fatalf("erneut: %v", err)
	}
	if u, p := lies(resp.TitelID); u == nil || *u != "Gepflegt" || p == nil || *p != 20.00 {
		t.Errorf("vorhandener Titel: untertitel=%v listenpreis=%v, erwartet „Gepflegt“ und 20.00", u, p)
	}
}

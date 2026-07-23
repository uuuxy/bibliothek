package api

import (
	"context"
	"testing"

	"bibliothek/db"

	"github.com/jackc/pgx/v5/pgxpool"
)

// TestQueryReorders_GesamtNichtVerfuegbar sichert den Meldebestand-Fix ab: Der
// Bestellbedarf richtet sich nach dem BESITZ (gesamt, nicht ausgesondert), nicht nach
// dem gerade verfügbaren Bestand. Sonst würde jeder verliehene Lernmittel-Klassensatz
// (im Schuljahr der Normalfall) als "kritisch nachbestellen" gemeldet — die Liste war
// mit Fehlalarmen geflutet und die vorgeschlagene Menge (Meldebestand − verfügbar) zu hoch.
func TestQueryReorders_GesamtNichtVerfuegbar(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	ctx := context.Background()

	schueler := seedSchueler(t, pool, "R-1", "Ida", "5a")
	srv := &Server{DB: &db.Database{Pool: pool}}

	// A) Voll verliehener Klassensatz: 30 Exemplare, alle ausgeliehen (verfügbar 0,
	//    gesamt 30), Meldebestand 5 → KEIN Bestellgrund.
	tVoll := titelMitMeldebestand(t, pool, "LMF-Mathe 7", 5)
	for i := 0; i < 30; i++ {
		e := exemplar(t, pool, tVoll, barcodeN("V", i), true, "")
		lendExemplar(t, pool, e, schueler)
	}

	// B) Echter Fehlbestand: nur 3 eigene Exemplare (gesamt 3 < Meldebestand 5).
	tKnapp := titelMitMeldebestand(t, pool, "LMF-Deutsch 5", 5)
	for i := 0; i < 3; i++ {
		exemplar(t, pool, tKnapp, barcodeN("K", i), true, "")
	}

	// C) Bereits bestellt: 2 eigene + 3 "bestellt"-Platzhalter = gesamt 5 → gedeckt,
	//    kein erneuter Bedarf (die "bestellt"-Platzhalter zählen in gesamt mit).
	tBestellt := titelMitMeldebestand(t, pool, "LMF-Bio 6", 5)
	exemplar(t, pool, tBestellt, "BE-1", true, "")
	exemplar(t, pool, tBestellt, "BE-2", true, "")
	for i := 0; i < 3; i++ {
		exemplar(t, pool, tBestellt, barcodeN("BE", i+3), false, "bestellt")
	}

	// Schwelle 5 = früherer Meldebestand-Default; die Auswahllogik bleibt so unverändert.
	reorders, err := srv.queryReorders(ctx, reorderFilterFragmentLMF(), 5)
	if err != nil {
		t.Fatalf("queryReorders: %v", err)
	}

	got := map[string]ReorderTitle{}
	for _, r := range reorders {
		got[r.Titel] = r
	}

	if _, drin := got["LMF-Mathe 7"]; drin {
		t.Error("voll verliehener Klassensatz (gesamt 30) wurde als Bestellbedarf gemeldet — Fehlalarm")
	}
	if _, drin := got["LMF-Bio 6"]; drin {
		t.Error("bereits vollständig bestellter Titel (gesamt 5) wurde erneut gemeldet")
	}
	k, drin := got["LMF-Deutsch 5"]
	if !drin {
		t.Fatal("Titel mit echtem Fehlbestand (gesamt 3 < Meldebestand 5) fehlt in der Liste")
	}
	if k.GesamtBestand != 3 || k.Meldebestand != 5 {
		t.Errorf("Fehlbestand-Zahlen falsch: gesamt=%d meldebestand=%d (erwartet 3 / 5)", k.GesamtBestand, k.Meldebestand)
	}
}

// TestQueryReorders_SchwelleSteuert beweist, dass jetzt die konfigurierbare Schwelle die
// Aufnahme steuert — nicht mehr der pauschale Meldebestand-Default 5. Ein Titel mit
// gesamt 3 erscheint bei Schwelle 5, verschwindet aber bei Schwelle 2.
func TestQueryReorders_SchwelleSteuert(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	ctx := context.Background()
	srv := &Server{DB: &db.Database{Pool: pool}}

	tKnapp := titelMitMeldebestand(t, pool, "LMF-Physik 8", 5) // meldebestand pauschal 5
	for i := 0; i < 3; i++ {
		exemplar(t, pool, tKnapp, barcodeN("P", i), true, "")
	}

	filter := reorderFilterFragmentLMF()
	r, err := srv.queryReorders(ctx, filter, 5)
	if err != nil {
		t.Fatalf("queryReorders error: %v", err)
	}
	if len(r) != 1 {
		t.Errorf("Schwelle 5: erwartet 1 Treffer (gesamt 3 < 5), waren %d", len(r))
	}

	r2, err := srv.queryReorders(ctx, filter, 2)
	if err != nil {
		t.Fatalf("queryReorders error: %v", err)
	}
	if len(r2) != 0 {
		t.Errorf("Schwelle 2: erwartet 0 Treffer (gesamt 3 ≥ 2) — Schwelle steuert, nicht Meldebestand 5, waren %d", len(r2))
	}
}

// reorderFilterFragmentLMF liefert das Default-(LMF-)Filterfragment ohne HTTP-Request.
func reorderFilterFragmentLMF() string {
	frag, _ := resolveBestandsFilter("lmf")
	return frag
}

func lendExemplar(t *testing.T, pool *pgxpool.Pool, exemplarID, schuelerID string) {
	t.Helper()
	if _, err := pool.Exec(context.Background(),
		`INSERT INTO ausleihen (exemplar_id, schueler_id, rueckgabe_frist)
		 VALUES ($1, $2, CURRENT_DATE)`, exemplarID, schuelerID); err != nil {
		t.Fatalf("Ausleihe anlegen: %v", err)
	}
}

func barcodeN(prefix string, i int) string {
	return prefix + "-" + string(rune('a'+i%26)) + string(rune('0'+i/26))
}

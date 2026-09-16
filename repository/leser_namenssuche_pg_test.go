package repository

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

// seedSuchLeser legt einen Leser beliebiger Art an — ohne Klasse, ohne Abgängerjahr,
// wie ein Kollege in der Leserdatei steht. Geschrieben wird in die TABELLE `leser`:
// Durch die Sicht `schueler` ließe WITH CHECK OPTION diese Zeile gar nicht entstehen.
func seedSuchLeser(t *testing.T, pool *pgxpool.Pool, barcode, vorname, nachname, art string) string {
	t.Helper()
	var id string
	var barcodeWert any
	if barcode != "" {
		barcodeWert = barcode
	}
	if err := pool.QueryRow(context.Background(),
		`INSERT INTO leser (barcode_id, vorname, nachname, art)
		 VALUES ($1, $2, $3, $4) RETURNING id`, barcodeWert, vorname, nachname, art).Scan(&id); err != nil {
		t.Fatalf("Leser %q %q (%s) anlegen: %v", vorname, nachname, art, err)
	}
	return id
}

// TestSearchStudentsFuzzy_FindetKollegium ist der Rot-Test für die Namenssuche an der
// Theke: Ein Kollege lässt sich seit Migration 125 über seinen Ausweis laden, über
// seinen NAMEN aber nicht — die Suche las die Sicht `schueler`, und die zeigt ihn nicht.
// Wer den Ausweis gerade nicht zur Hand hat, findet ihn damit an der Theke nicht.
//
// Geprüft wird zugleich die Art am Treffer: Ohne sie stünde ein Kollege ohne Klasse in
// der Trefferliste wie ein Schüler mit fehlender Angabe.
func TestSearchStudentsFuzzy_FindetKollegium(t *testing.T) {
	pool := pgTestPool(t)
	resetInventurDaten(t, pool)
	ctx := context.Background()

	seedSuchSchueler(t, pool, "LSUCH-1", "Katrin", "Wendland")
	seedSuchLeser(t, pool, "LSUCH-2", "Katrin", "Wendlandt", "lehrkraft")
	// Ein Kollege ohne gedruckten Ausweis: Seine Nummer ist NULL. Er muss trotzdem
	// gefunden werden — und der Scan der Spalten darf an der NULL nicht zerbrechen.
	seedSuchLeser(t, pool, "", "Hendrik", "Wendlandt", "liv")

	repo := NewStudentRepository(pool)
	treffer, gesamt, err := repo.SearchStudentsFuzzy(ctx, "Wendlandt", 10)
	if err != nil {
		t.Fatalf("Suche nach Kollegium: %v", err)
	}
	if gesamt != 2 {
		t.Errorf("Gesamtzahl = %d, erwartet 2 (Lehrkraft und LiV) — Treffer: %v", gesamt, namenDerTreffer(treffer))
	}

	arten := map[string]string{}
	for _, s := range treffer {
		arten[s.Vorname+" "+s.Nachname] = s.Art
	}
	if arten["Katrin Wendlandt"] != "lehrkraft" {
		t.Errorf("Katrin Wendlandt: Art = %q, erwartet \"lehrkraft\" — ohne die Art steht ein Kollege in der Trefferliste wie ein Schüler ohne Klasse", arten["Katrin Wendlandt"])
	}
	if arten["Hendrik Wendlandt"] != "liv" {
		t.Errorf("Hendrik Wendlandt: Art = %q, erwartet \"liv\"", arten["Hendrik Wendlandt"])
	}

	// Die Schüler-Suche daneben muss weiter Schüler finden — und an ihnen steht
	// 'schueler', nicht der leere String.
	schuelerTreffer, _, err := repo.SearchStudentsFuzzy(ctx, "Wendland", 10)
	if err != nil {
		t.Fatalf("Suche nach Schüler: %v", err)
	}
	for _, s := range schuelerTreffer {
		if s.Nachname == "Wendland" && s.Art != "schueler" {
			t.Errorf("Schüler Wendland: Art = %q, erwartet \"schueler\"", s.Art)
		}
	}
}

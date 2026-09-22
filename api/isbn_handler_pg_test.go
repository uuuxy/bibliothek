package api

import (
	"context"
	"testing"

	"bibliothek/db"
	"bibliothek/inventur"
)

// Ein neu über die DNB-Bestellsuche angelegter Titel bekommt das Fach aus der Heuristik —
// und KEINE Signatur. Die Signaturen der Schülerbücherei sind die Littera-Codes am Regal
// („Sk", „JF", „MANGA"); bis zum 22.09.2026 stand hier „BIB Jugendbuch", ein Wort, das in
// keinem Regal vorkommt (gemessen am Testserver am 22.09.2026: 0 Titel mit „BIB").
func TestUpsertTitelAusMetadaten_SchreibtFachUndKeineErfundeneSignatur(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	ctx := context.Background()
	srv := &Server{DB: &db.Database{Pool: pool}}

	meta := &inventur.MetadatenErgebnis{
		Titel:  "Die Tribute von Panem",
		Autor:  "Collins, Suzanne",
		Verlag: "Oetinger",
		Jahr:   "2009",
		Fach:   "Deutsch",
	}

	resp, err := srv.upsertTitelAusMetadaten(ctx, "9783841421001", meta)
	if err != nil {
		t.Fatalf("upsertTitelAusMetadaten: %v", err)
	}
	if resp.Exists {
		t.Error("Exists = true, want false (neu angelegter Titel)")
	}
	if resp.Signatur != "" {
		t.Errorf("Signatur = %q — ein neuer Titel bekommt keine erfundene Signatur", resp.Signatur)
	}

	var subject string
	if err := pool.QueryRow(ctx, `SELECT coalesce(subject, '') FROM buecher_titel WHERE id = $1`, resp.TitelID).Scan(&subject); err != nil {
		t.Fatalf("subject lesen: %v", err)
	}
	if subject != "Deutsch" {
		t.Errorf("subject in DB = %q, want %q", subject, "Deutsch")
	}
}

// TestUpsertTitelAusMetadaten_OhneKategorieBleibtSignaturLeer stellt sicher, dass
// ohne DNB-Genre-Treffer kein sinnloser "BIB "-Wert entsteht — das Feld bleibt leer,
// genau wie beim manuellen Anlegen über das Buchformular.
func TestUpsertTitelAusMetadaten_OhneKategorieBleibtSignaturLeer(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	ctx := context.Background()
	srv := &Server{DB: &db.Database{Pool: pool}}

	meta := &inventur.MetadatenErgebnis{Titel: "Ein Fachbuch ohne Genre-Treffer"}

	resp, err := srv.upsertTitelAusMetadaten(ctx, "9783150009999", meta)
	if err != nil {
		t.Fatalf("upsertTitelAusMetadaten: %v", err)
	}
	if resp.Signatur != "" {
		t.Errorf("Signatur = %q, want leer", resp.Signatur)
	}
}

// TestFindeLokalenTitel_LiefertVorhandeneSignaturUnveraendert sichert die Erwartung
// aus dem Gespräch ab: Existiert der Titel schon, wird die vorhandene Systematik
// unverändert übernommen (angezeigt), nicht durch einen frischen Vorschlag ersetzt.
func TestFindeLokalenTitel_LiefertVorhandeneSignaturUnveraendert(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	ctx := context.Background()
	srv := &Server{DB: &db.Database{Pool: pool}}

	titelMitSignatur(t, pool, "Effi Briest", "Pg", 5)
	if _, err := pool.Exec(ctx, `UPDATE buecher_titel SET isbn = $1 WHERE titel = $2`, "9783150001", "Effi Briest"); err != nil {
		t.Fatalf("ISBN nachtragen: %v", err)
	}

	resp, err := srv.findeLokalenTitel(ctx, "9783150001")
	if err != nil {
		t.Fatalf("findeLokalenTitel: %v", err)
	}
	if resp == nil {
		t.Fatal("Titel nicht gefunden, obwohl er existiert")
	}
	if !resp.Exists {
		t.Error("Exists = false, want true")
	}
	if resp.Signatur != "Pg" {
		t.Errorf("Signatur = %q, want %q (unverändert übernommen)", resp.Signatur, "Pg")
	}
}

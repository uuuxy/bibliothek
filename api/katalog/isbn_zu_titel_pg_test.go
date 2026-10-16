package katalog

import (
	"context"
	"testing"

	"bibliothek/api/tuer"
	"bibliothek/db"
	"bibliothek/internal/pgtest"
	"bibliothek/internal/testdaten"
	"bibliothek/inventur"
)

// Ein über die Bestellsuche neu aus der DNB angelegter Titel bekommt das Fach aus der
// Faustregel, aber keine Signatur: Die Signaturen der Schülerbücherei sind die Littera-Codes am
// Regal („Sk", „JF", „MANGA"), und eine Kategorie der DNB wie „BIB Jugendbuch" kommt in keinem
// Regal vor.
func TestSchreibeTitelAusMetadaten_SchreibtFachUndKeineErfundeneSignatur(t *testing.T) {
	pool := pgtest.Pool(t)
	testdaten.LeereBestandsdaten(t, pool)
	ctx := context.Background()
	srv := &Tueren{Basis: &tuer.Basis{DB: &db.Verbindung{Pool: pool}}}

	meta := &inventur.MetadatenErgebnis{
		Titel:  "Die Tribute von Panem",
		Autor:  "Collins, Suzanne",
		Verlag: "Oetinger",
		Jahr:   "2009",
		Fach:   "Deutsch",
	}

	resp, err := srv.schreibeTitelAusMetadaten(ctx, "9783841421001", meta)
	if err != nil {
		t.Fatalf("schreibeTitelAusMetadaten: %v", err)
	}
	if resp.Vorhanden {
		t.Error("Vorhanden = true, erwartet false (neu angelegter Titel)")
	}
	if resp.Signatur != "" {
		t.Errorf("Signatur = %q — ein neuer Titel bekommt keine erfundene Signatur", resp.Signatur)
	}

	var subject string
	if err := pool.QueryRow(ctx, `SELECT coalesce(subject, '') FROM buecher_titel WHERE id = $1`, resp.TitelID).Scan(&subject); err != nil {
		t.Fatalf("subject lesen: %v", err)
	}
	if subject != "Deutsch" {
		t.Errorf("subject in DB = %q, erwartet %q", subject, "Deutsch")
	}
}

// TestSchreibeTitelAusMetadaten_OhneKategorieBleibtSignaturLeer stellt sicher, dass
// ohne DNB-Genre-Treffer kein sinnloser "BIB "-Wert entsteht — das Feld bleibt leer,
// genau wie beim manuellen Anlegen über das Buchformular.
func TestSchreibeTitelAusMetadaten_OhneKategorieBleibtSignaturLeer(t *testing.T) {
	pool := pgtest.Pool(t)
	testdaten.LeereBestandsdaten(t, pool)
	ctx := context.Background()
	srv := &Tueren{Basis: &tuer.Basis{DB: &db.Verbindung{Pool: pool}}}

	meta := &inventur.MetadatenErgebnis{Titel: "Ein Fachbuch ohne Genre-Treffer"}

	resp, err := srv.schreibeTitelAusMetadaten(ctx, "9783150009999", meta)
	if err != nil {
		t.Fatalf("schreibeTitelAusMetadaten: %v", err)
	}
	if resp.Signatur != "" {
		t.Errorf("Signatur = %q, erwartet leer", resp.Signatur)
	}
}

// TestFindeLokalenTitel_LiefertVorhandeneSignaturUnveraendert sichert die Erwartung
// aus dem Gespräch ab: Existiert der Titel schon, wird die vorhandene Systematik
// unverändert übernommen (angezeigt), nicht durch einen frischen Vorschlag ersetzt.
func TestFindeLokalenTitel_LiefertVorhandeneSignaturUnveraendert(t *testing.T) {
	pool := pgtest.Pool(t)
	testdaten.LeereBestandsdaten(t, pool)
	ctx := context.Background()
	srv := &Tueren{Basis: &tuer.Basis{DB: &db.Verbindung{Pool: pool}}}

	testdaten.LegeTitelMitSignaturAn(t, pool, "Effi Briest", "Pg", 5)
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
	if !resp.Vorhanden {
		t.Error("Vorhanden = false, erwartet true")
	}
	if resp.Signatur != "Pg" {
		t.Errorf("Signatur = %q, erwartet %q (unverändert übernommen)", resp.Signatur, "Pg")
	}
}

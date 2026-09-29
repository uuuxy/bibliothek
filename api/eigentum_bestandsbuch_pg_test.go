package api

import (
	"context"
	"testing"
	"time"

	"bibliothek/pkg/schulzeit"
	"bibliothek/repository"
)

// Zugangs- und Abgangsbuch lesen das Eigentum nach derselben Regel wie das Etikett
// (repository.ExemplarTopfSQL, docs/OFFEN.md 4.24, Stufe 2): Eigentum am Exemplar, sonst Topf
// der Bestellung, sonst der Titel. Das Zugangsbuch nimmt davon nur die belegten Teile.
//
// Die Fälle sind die, in denen die alten Regeln danebenlagen: ein Lernmittel-Titel, den der
// Schulträger bezahlt hat, und ein Büchereibuch, das Littera dem Land zuschreibt.

// topfBestellung legt eine Bestellung mit Topf an.
func topfBestellung(t *testing.T, mittel string) string {
	t.Helper()
	var id string
	if err := pgTestPool(t).QueryRow(context.Background(), `
		INSERT INTO bestellungen_verlauf (lieferant_name, lieferant_email, mittel)
		VALUES ('Buchhandlung', 'haendler@example.org', $1) RETURNING id`, mittel).Scan(&id); err != nil {
		t.Fatalf("Bestellung: %v", err)
	}
	return id
}

func TestAbgangsbuch_TopfFolgtDemEigentum(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	ctx := context.Background()
	loc := schulzeit.Zone()

	lernmittel := titelMitSignatur(t, pool, "Mathebuch 7", "Mat 7", 0)
	if _, err := pool.Exec(ctx, `UPDATE buecher_titel SET ist_lernmittel = true WHERE id = $1`, lernmittel); err != nil {
		t.Fatalf("Lernmittel setzen: %v", err)
	}
	lektuere := titelMitSignatur(t, pool, "Nathan der Weise", "Ga Les", 0)
	traeger := topfBestellung(t, repository.MittelSchultraeger)

	abgang := func(titelID, barcode string, bestellung *string, eigentum string) {
		t.Helper()
		if _, err := pool.Exec(ctx, `
			INSERT INTO buecher_exemplare (titel_id, barcode_id, bestellung_id, eigentum, eigentum_quelle,
			                               ist_ausleihbar, ist_ausgesondert, aussonderung_grund)
			VALUES ($1, $2, $3, NULLIF($4, ''), CASE WHEN $4 = '' THEN NULL ELSE 'littera' END,
			        false, true, 'VERLUST')`,
			titelID, barcode, bestellung, eigentum); err != nil {
			t.Fatalf("Abgang %s: %v", barcode, err)
		}
		if _, err := pool.Exec(ctx, `UPDATE buecher_exemplare SET ausgesondert_am = $2 WHERE barcode_id = $1`,
			barcode, time.Date(2026, time.May, 4, 10, 0, 0, 0, loc)); err != nil {
			t.Fatalf("datieren %s: %v", barcode, err)
		}
	}
	abgang(lernmittel, "EIG-LMF-TRAEGER", &traeger, "")               // bis 29.09.: Land
	abgang(lektuere, "EIG-LEKTUERE-LAND", nil, repository.MittelLand) // bis 29.09.: Schulträger
	abgang(lernmittel, "EIG-LMF-ALT", nil, "")                        // Faustregel: Land
	abgang(lektuere, "EIG-ROMAN-ALT", nil, "")                        // Faustregel: Schulträger

	buch, err := repository.LadeAbgangsbuch(ctx, pool,
		time.Date(2026, time.March, 16, 0, 0, 0, 0, loc), time.Date(2026, time.September, 15, 0, 0, 0, 0, loc))
	if err != nil {
		t.Fatalf("Abgangsbuch laden: %v", err)
	}
	erwartet := map[string]string{
		"EIG-LMF-TRAEGER":   repository.MittelSchultraeger,
		"EIG-LEKTUERE-LAND": repository.MittelLand,
		"EIG-LMF-ALT":       repository.MittelLand,
		"EIG-ROMAN-ALT":     repository.MittelSchultraeger,
	}
	if len(buch.Zeilen) != len(erwartet) {
		t.Fatalf("Zeilen: %+v — erwartet %d", buch.Zeilen, len(erwartet))
	}
	for i, z := range buch.Zeilen {
		if z.Topf != erwartet[z.Barcode] {
			t.Errorf("%s: Topf %q, erwartet %q", z.Barcode, z.Topf, erwartet[z.Barcode])
		}
		// Land zuerst — die Reihenfolge des Ausdrucks.
		if i > 0 && buch.Zeilen[i-1].Topf == repository.MittelSchultraeger && z.Topf == repository.MittelLand {
			t.Errorf("Reihenfolge: %s (Land) steht hinter einer Zeile des Schulträgers", z.Barcode)
		}
	}
}

func TestZugangsbuch_EigentumAmExemplarIstEinBeleg(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	ctx := context.Background()
	loc := schulzeit.Zone()

	titelID := titelMitSignatur(t, pool, "Nathan der Weise", "Ga Les", 0)
	land := topfBestellung(t, repository.MittelLand)

	zugang := func(barcode string, bestellung *string, eigentum string) {
		t.Helper()
		if _, err := pool.Exec(ctx, `
			INSERT INTO buecher_exemplare (titel_id, barcode_id, erworben_am, bestellung_id, eigentum, eigentum_quelle)
			VALUES ($1, $2, '2026-05-04', $3, NULLIF($4, ''), CASE WHEN $4 = '' THEN NULL ELSE 'littera' END)`,
			titelID, barcode, bestellung, eigentum); err != nil {
			t.Fatalf("Zugang %s: %v", barcode, err)
		}
	}
	zugang("ZEIG-LITTERA", nil, repository.MittelLand)             // bis 29.09.: ohne Zuordnung
	zugang("ZEIG-UMGESETZT", &land, repository.MittelSchultraeger) // bis 29.09.: Land
	zugang("ZEIG-BESTELLT", &land, "")                             // Topf der Bestellung
	zugang("ZEIG-OHNE", nil, "")                                   // nicht geraten

	buch, err := repository.LadeZugangsbuch(ctx, pool,
		time.Date(2026, time.March, 16, 0, 0, 0, 0, loc), time.Date(2026, time.September, 15, 0, 0, 0, 0, loc))
	if err != nil {
		t.Fatalf("Zugangsbuch laden: %v", err)
	}
	erwartet := map[string]string{
		"ZEIG-LITTERA":   repository.MittelLand,
		"ZEIG-UMGESETZT": repository.MittelSchultraeger,
		"ZEIG-BESTELLT":  repository.MittelLand,
		"ZEIG-OHNE":      "",
	}
	if len(buch.Zeilen) != len(erwartet) {
		t.Fatalf("Zeilen: %+v — erwartet %d", buch.Zeilen, len(erwartet))
	}
	for _, z := range buch.Zeilen {
		if z.Topf != erwartet[z.Barcode] {
			t.Errorf("%s: Topf %q, erwartet %q", z.Barcode, z.Topf, erwartet[z.Barcode])
		}
	}
	// Der Lieferant hängt weiter an der Bestellung, nicht am Eigentum.
	for _, z := range buch.Zeilen {
		mitBestellung := z.Barcode == "ZEIG-BESTELLT" || z.Barcode == "ZEIG-UMGESETZT"
		if mitBestellung != (z.Lieferant == "Buchhandlung") {
			t.Errorf("%s: Lieferant %q", z.Barcode, z.Lieferant)
		}
	}
}

package api

import (
	"context"
	"strings"
	"testing"
	"time"

	"bibliothek/internal/pdftest"
	"bibliothek/pdf"
	"bibliothek/pkg/schulzeit"
	"bibliothek/repository"
)

// Das Zugangsbuch ist wie das Abgangsbuch ein Nachweis zum Abheften. Gemessen wird an den
// Rändern des Zeitraums, am Topf und am fertigen Blatt.
func TestZugangsbuch_ZeitraumToepfeUndBestellung(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	ctx := context.Background()
	loc := schulzeit.Zone()

	titelID := titelMitSignatur(t, pool, "Zugangs-Titel", "Zug 1", 0)

	// Zwei Bestellungen, je ein Topf — der Topf des ZUGANGS steht an der Bestellung, nicht
	// am Titel: Ein Titel darf im Warenkorb in den anderen Topf geschoben werden.
	bestellung := func(name, mittel string) string {
		t.Helper()
		var id string
		if err := pool.QueryRow(ctx, `
			INSERT INTO bestellungen_verlauf (lieferant_name, lieferant_email, mittel)
			VALUES ($1, 'haendler@example.org', $2) RETURNING id`, name, mittel).Scan(&id); err != nil {
			t.Fatalf("Bestellung %s: %v", name, err)
		}
		return id
	}
	land := bestellung("Buchhandlung Land", repository.MittelLand)
	traeger := bestellung("Buchhandlung Traeger", repository.MittelSchultraeger)

	zugang := func(barcode string, wann time.Time, bestellID *string) string {
		t.Helper()
		var id string
		if err := pool.QueryRow(ctx, `
			INSERT INTO buecher_exemplare (titel_id, barcode_id, erworben_am, bestellung_id)
			VALUES ($1, $2, $3::date, $4) RETURNING id`,
			titelID, barcode, wann.Format("2006-01-02"), bestellID).Scan(&id); err != nil {
			t.Fatalf("Zugang %s: %v", barcode, err)
		}
		return id
	}
	tag := func(j int, m time.Month, d int) time.Time { return time.Date(j, m, d, 0, 0, 0, 0, loc) }

	zugang("ZUG-LAND", tag(2026, time.April, 12), &land)
	zugang("ZUG-TRAEGER", tag(2026, time.September, 15), &traeger) // letzter Tag des Halbjahres
	zugang("ZUG-OHNE", tag(2026, time.May, 2), nil)                // ohne Bestellung
	zugang("ZUG-FRUEH", tag(2026, time.March, 15), &land)          // Tag davor
	zugang("ZUG-SPAET", tag(2026, time.September, 16), &land)      // Tag danach

	// Ein Zugang, der später ausgesondert wurde, BLEIBT im Zugangsbuch: Er ist trotzdem
	// zugegangen. Sonst änderte sich rückwirkend eine Zahl, die jemand unterschrieben hat.
	spaeterWeg := zugang("ZUG-WEG", tag(2026, time.June, 1), &land)
	if _, err := pool.Exec(ctx, `
		UPDATE buecher_exemplare SET ist_ausgesondert = true, ist_ausleihbar = false,
		       aussonderung_grund = 'VERLUST' WHERE id = $1`, spaeterWeg); err != nil {
		t.Fatalf("aussondern: %v", err)
	}

	buch, err := repository.LadeZugangsbuch(ctx, pool, tag(2026, time.March, 16), tag(2026, time.September, 15))
	if err != nil {
		t.Fatalf("Zugangsbuch laden: %v", err)
	}

	gefunden := map[string]repository.ZugangsZeile{}
	for _, z := range buch.Zeilen {
		gefunden[z.Barcode] = z
	}
	if len(buch.Zeilen) != 4 {
		t.Fatalf("Zeilen im Halbjahr: %v — erwartet LAND, TRAEGER, OHNE und WEG", gefunden)
	}
	if _, drin := gefunden["ZUG-FRUEH"]; drin {
		t.Error("der Zugang vom 15.03. gehört ins vorige Halbjahr")
	}
	if _, drin := gefunden["ZUG-SPAET"]; drin {
		t.Error("der Zugang vom 16.09. gehört ins nächste Halbjahr")
	}
	if _, drin := gefunden["ZUG-WEG"]; !drin {
		t.Error("ein später ausgesondertes Exemplar fehlt — es ist trotzdem zugegangen")
	}
	if z := gefunden["ZUG-LAND"]; z.Topf != repository.MittelLand || z.Lieferant != "Buchhandlung Land" {
		t.Errorf("Topf/Lieferant aus der Bestellung: %+v", z)
	}
	if z := gefunden["ZUG-TRAEGER"]; z.Topf != repository.MittelSchultraeger {
		t.Errorf("Topf des Schulträgers: %+v", z)
	}
	// Ohne Bestellung wird NICHT geraten — auch nicht aus ist_lernmittel des Titels.
	if z := gefunden["ZUG-OHNE"]; z.Topf != "" || z.Lieferant != "" {
		t.Errorf("Exemplar ohne Bestellung: %+v — erwartet leeren Topf und keinen Lieferanten", z)
	}
}

// Das Blatt: drei Abschnitte mit Summen und der Hinweis zu „ohne Zuordnung".
func TestZugangsbuchPDF_AbschnitteUndHinweis(t *testing.T) {
	loc := schulzeit.Zone()
	buch := repository.Zugangsbuch{
		Von: time.Date(2026, time.March, 16, 0, 0, 0, 0, loc),
		Bis: time.Date(2026, time.September, 15, 0, 0, 0, 0, loc),
		Zeilen: []repository.ZugangsZeile{
			{Datum: time.Date(2026, time.April, 12, 0, 0, 0, 0, loc), Barcode: "B-00100",
				Titel: "Mathebuch 7", Lieferant: "Buchhandlung Land", Topf: repository.MittelLand},
			{Datum: time.Date(2026, time.May, 3, 0, 0, 0, 0, loc), Barcode: "B-00200",
				Titel: "Gregs Tagebuch", Lieferant: "Buchhandlung Traeger", Topf: repository.MittelSchultraeger},
			{Datum: time.Date(2026, time.May, 4, 0, 0, 0, 0, loc), Barcode: "B-00300",
				Titel: "Fundstueck aus dem Schrank"},
		},
	}

	roh, err := generateZugangsbuchPDF(buch, pdf.SchuleInfo{Name: "Philipp-Reis-Schule", Strasse: "Schulstr. 1", PLZ: "61440", Ort: "Oberursel"})
	if err != nil {
		t.Fatalf("Zugangsbuch drucken: %v", err)
	}
	blatt := strings.Join(pdftest.Texte(t, roh), " ")

	for _, muss := range []string{
		"Zugangsbuch",
		"16.03.2026", "15.09.2026",
		"B-00100", "B-00200", "B-00300",
		"Buchhandlung Land",
		"Summe " + mittelBeschriftung(repository.MittelLand) + ": 1 Exemplare",
		"Summe " + mittelBeschriftung(repository.MittelSchultraeger) + ": 1 Exemplare",
		"Summe " + mittelOhneZuordnung + ": 1 Exemplare",
		"keine Bestellung hinterlegt", // der Hinweis, der die Einschränkung nennt
		"Zugänge im Zeitraum: 3 Exemplare",
	} {
		if !strings.Contains(blatt, muss) {
			t.Errorf("auf dem Blatt fehlt %q:\n%s", muss, blatt)
		}
	}
}

// Ohne Zugänge gibt es keinen Hinweis auf „ohne Zuordnung" — und das Blatt sagt, dass
// nichts kam, statt ein leeres Gerüst zu zeigen.
func TestZugangsbuchPDF_LeererZeitraum(t *testing.T) {
	loc := schulzeit.Zone()
	buch := repository.Zugangsbuch{
		Von:    time.Date(2026, time.March, 16, 0, 0, 0, 0, loc),
		Bis:    time.Date(2026, time.September, 15, 0, 0, 0, 0, loc),
		Zeilen: []repository.ZugangsZeile{},
	}
	roh, err := generateZugangsbuchPDF(buch, pdf.SchuleInfo{Name: "Philipp-Reis-Schule"})
	if err != nil {
		t.Fatalf("Zugangsbuch drucken: %v", err)
	}
	blatt := strings.Join(pdftest.Texte(t, roh), " ")
	if !strings.Contains(blatt, "kein Exemplar in den Bestand gekommen") {
		t.Errorf("leerer Zeitraum sagt es nicht:\n%s", blatt)
	}
	if strings.Contains(blatt, "keine Bestellung hinterlegt") {
		t.Errorf("Hinweis auf „ohne Zuordnung\" ohne solche Zeilen:\n%s", blatt)
	}
}

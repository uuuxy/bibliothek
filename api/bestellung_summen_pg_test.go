package api

import (
	"context"
	"errors"
	"testing"

	"bibliothek/db"
	"bibliothek/internal/service"
	"bibliothek/pkg/mitteltopf"
	"bibliothek/repository"
)

// Ein Warenkorb mit zwei Positionen: Betrag und Menge der Bestellung sind die Summe über
// beide, und Zusammenfassung, Etiketten und Positionen nennen jede von ihnen.
func TestProcessOrder_ZweiPositionenZaehlenZusammen(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	ctx := context.Background()
	srv := &Server{DB: &db.Database{Pool: pool}}
	svc := service.NewOrderService(srv.DB, repository.NewBookRepository(pool))

	lieferant := haendler(t, pool, "Summen", false)
	erster := titelMitMeldebestand(t, pool, "LMF-Summe Eins", 0)
	zweiter := titelMitMeldebestand(t, pool, "LMF-Summe Zwei", 0)

	res, err := svc.ProcessOrder(ctx, service.BestellAuftrag{
		Mittel: mitteltopf.Land, SupplierID: lieferant,
		Items: []service.BestellAuftragPosition{
			{TitelID: erster, Menge: 3, Preis: 10, GenerateBarcodes: true},
			{TitelID: zweiter, Menge: 2, Preis: 4.5, GenerateBarcodes: true},
		},
	})
	if err != nil {
		t.Fatalf("Bestellung: %v", err)
	}
	if res.TotalAllocated != 5 || len(res.SummaryItems) != 2 || len(res.Labels) != 5 {
		t.Errorf("Ergebnis: %d Exemplare, %d Zeilen, %d Etiketten; erwartet 5, 2 und 5",
			res.TotalAllocated, len(res.SummaryItems), len(res.Labels))
	}

	var betrag float64
	var anzahl int
	if err := pool.QueryRow(ctx, `
		SELECT gesamtbetrag::float8, anzahl_exemplare FROM bestellungen_verlauf WHERE id = $1`,
		res.BestellungID).Scan(&betrag, &anzahl); err != nil {
		t.Fatalf("Bestellkopf lesen: %v", err)
	}
	if betrag != 39 || anzahl != 5 {
		t.Errorf("Bestellkopf: %.2f Euro und %d Exemplare, erwartet 39,00 Euro und 5", betrag, anzahl)
	}
	if n := zaehleZeilen(t, pool, `SELECT count(*) FROM bestellungen_positionen WHERE bestellung_id = $1`, res.BestellungID); n != 2 {
		t.Errorf("%d Positionen geschrieben, erwartet 2", n)
	}
	if n := zaehleZeilen(t, pool, `SELECT count(*) FROM buecher_exemplare WHERE bestellung_id = $1`, res.BestellungID); n != 5 {
		t.Errorf("%d Exemplare an der Bestellung, erwartet 5", n)
	}
}

// Ohne gültigen Topf legt das Anlegen nichts an, auch wenn kein Aufrufer vorher prüft: Die
// Tür prüft den Topf selbst, das Anlegen ist die zweite Stelle. Abgelehnt wird mit dem Fehler
// des Vokabulars, bevor ein Barcode reserviert ist.
func TestProcessOrder_OhneGueltigenTopfEntstehtNichts(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	ctx := context.Background()
	svc := service.NewOrderService(&db.Database{Pool: pool}, repository.NewBookRepository(pool))
	lieferant := haendler(t, pool, "Ohne Topf", true)
	titel := titelMitMeldebestand(t, pool, "LMF-Ohne-Topf", 0)

	for _, mittel := range []string{"", "kreis"} {
		res, err := svc.ProcessOrder(ctx, service.BestellAuftrag{
			Mittel: mittel, SupplierID: lieferant,
			Items: []service.BestellAuftragPosition{{TitelID: titel, Menge: 2, Preis: 10, GenerateBarcodes: true}},
		})
		if !errors.Is(err, mitteltopf.ErrUngueltig) {
			t.Errorf("Topf %q: Fehler %v und Ergebnis %+v, erwartet mitteltopf.ErrUngueltig", mittel, err, res)
		}
	}
	var bestellungen, exemplare int
	if err := pool.QueryRow(ctx, `
		SELECT (SELECT count(*) FROM bestellungen_verlauf WHERE lieferant_id = $1),
		       (SELECT count(*) FROM buecher_exemplare WHERE titel_id = $2)`, lieferant, titel).Scan(&bestellungen, &exemplare); err != nil {
		t.Fatalf("zählen: %v", err)
	}
	if bestellungen != 0 || exemplare != 0 {
		t.Errorf("ohne gültigen Topf entstanden %d Bestellungen und %d Exemplare", bestellungen, exemplare)
	}
}

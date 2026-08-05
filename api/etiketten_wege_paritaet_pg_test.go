package api

import (
	"context"
	"testing"

	"bibliothek/db"
	"bibliothek/repository"
)

// Dieselben Daten auf beiden Wegen zum Etikett.
//
// Diese Bugklasse hat das Projekt zweimal getroffen, und beide Male fiel sie erst im
// Betrieb auf, weil jeder Weg für sich funktionierte:
//
//   - Die Signatur stand im Selbstdruck, fehlte aber im Lieferanten-Mailanhang
//     (Commit „Signatur erreicht den Lieferanten-Mailanhang").
//   - Das Anschaffungsjahr stand im Selbstdruck, fehlte aber auf der Lieferantenseite
//     (05.08.2026, gefunden erst beim Vergleich mit dem Foto der physischen Vorlage).
//
// Es gibt zwei getrennte Abfragen für dieselbe Frage — queryLabelItems (Druck-Center)
// und ladeBestellEtiketten (Lieferanten-Link). Wer eine erweitert und die andere
// vergisst, bekommt kein rotes Gate, sondern zwei verschiedene Aufkleber für dasselbe
// Buch. Dieser Test vergleicht sie Feld für Feld.
//
// Verglichen wird, was auf dem Etikett LANDET: Barcode, Titel, Autor, Anschaffungsjahr,
// Signatur. Die ISBN bleibt außen vor — sie wird von keinem der Generatoren gezeichnet.
func TestEtikettenDatenSindAufBeidenWegenGleich(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	ctx := context.Background()

	srv := &Server{DB: &db.Database{Pool: pool}}
	svc := NewOrderService(srv.DB, repository.NewBookRepository(pool))

	lieferant := haendlerMitBestaetigung(t, pool, "Naacher-Paritaet", true)
	titel := titelMitMeldebestand(t, pool, "LMF-Paritaet", 0)

	res, err := svc.ProcessOrder(ctx, SubmitOrderRequest{
		SupplierID: lieferant,
		Items:      []OrderItemRequest{{TitelID: titel, Menge: 3, Preis: 10, GenerateBarcodes: true}},
	})
	if err != nil {
		t.Fatalf("Bestellung: %v", err)
	}

	// Weg 1: Druck-Center (Selbstdruck aus dem Bestand)
	selbstdruck, err := srv.queryLabelItems(ctx, titel)
	if err != nil {
		t.Fatalf("Selbstdruck-Etiketten laden: %v", err)
	}
	// Weg 2: Lieferantenseite hinter dem Bestätigungs-Link
	lieferantenweg, err := srv.ladeBestellEtiketten(ctx, res.BestellungID)
	if err != nil {
		t.Fatalf("Lieferanten-Etiketten laden: %v", err)
	}

	if len(selbstdruck) != len(lieferantenweg) || len(selbstdruck) != 3 {
		t.Fatalf("Selbstdruck %d, Lieferantenweg %d Etiketten — erwartet je 3",
			len(selbstdruck), len(lieferantenweg))
	}

	nachBarcode := map[string]BarcodeLabelDetail{}
	for _, e := range lieferantenweg {
		nachBarcode[e.BarcodeID] = e
	}

	for _, eigen := range selbstdruck {
		fremd, ok := nachBarcode[eigen.BarcodeID]
		if !ok {
			t.Fatalf("Exemplar %s fehlt auf dem Lieferantenweg", eigen.BarcodeID)
		}
		felder := []struct{ name, a, b string }{
			{"Titel", eigen.Titel, fremd.Titel},
			{"Autor", eigen.Autor, fremd.Autor},
			{"Anschaffungsjahr", eigen.AnschaffungsJahr, fremd.AnschaffungsJahr},
			{"Signatur", eigen.Signatur, fremd.Signatur},
		}
		for _, f := range felder {
			if f.a != f.b {
				t.Errorf("Exemplar %s: %s unterscheidet sich — Selbstdruck %q, Lieferantenweg %q.\n"+
					"→ Beide Abfragen müssen dieselben Felder liefern, sonst trägt dasselbe Buch "+
					"je nach Druckweg einen anderen Aufkleber.", eigen.BarcodeID, f.name, f.a, f.b)
			}
		}
	}
}

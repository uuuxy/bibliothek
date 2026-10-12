package buchetikett

import (
	"context"
	"testing"

	"bibliothek/db"
	"bibliothek/internal/pgtest"
	"bibliothek/internal/service"
	"bibliothek/internal/testdaten"
	"bibliothek/pkg/mitteltopf"
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
// Es gibt zwei getrennte Abfragen für dieselbe Frage — DesTitels (Druck-Center)
// und DerBestellung (Lieferanten-Link). Wer eine erweitert und die andere
// vergisst, bekommt kein rotes Gate, sondern zwei verschiedene Aufkleber für dasselbe
// Buch. Dieser Test vergleicht sie Feld für Feld.
//
// Verglichen wird, was auf dem Etikett LANDET: Barcode, Titel, Autor, Anschaffungsjahr,
// Signatur. Die ISBN bleibt außen vor — sie wird von keinem der Generatoren gezeichnet.
func TestEtikettenDatenSindAufBeidenWegenGleich(t *testing.T) {
	pool := pgtest.Pool(t)
	testdaten.ResetBestandsdaten(t, pool)
	ctx := context.Background()

	svc := service.NewOrderService(&db.Database{Pool: pool}, repository.NewBookRepository(pool))

	lieferant := testdaten.Haendler(t, pool, "Naacher-Paritaet", true)
	// Signatur und Autor sind gesetzt: Zwei leere Felder wären auf beiden Wegen gleich, auch
	// wenn eine der Abfragen das Feld nicht läse.
	titel := testdaten.TitelMitSignatur(t, pool, "LMF-Paritaet", "LMF-Deutsch 5", 0)
	if _, err := pool.Exec(ctx, `UPDATE buecher_titel SET autor = 'Autorin der Probe' WHERE id = $1`, titel); err != nil {
		t.Fatalf("Autor setzen: %v", err)
	}

	res, err := svc.ProcessOrder(ctx, service.BestellAuftrag{
		Mittel:     mitteltopf.Land,
		SupplierID: lieferant,
		Items:      []service.BestellAuftragPosition{{TitelID: titel, Menge: 3, Preis: 10, GenerateBarcodes: true}},
	})
	if err != nil {
		t.Fatalf("Bestellung: %v", err)
	}

	// Weg 1: Druck-Center (Selbstdruck aus dem Bestand)
	selbstdruck, err := DesTitels(ctx, pool, titel)
	if err != nil {
		t.Fatalf("Selbstdruck-Etiketten laden: %v", err)
	}
	// Weg 2: Lieferantenseite hinter dem Bestätigungs-Link
	lieferantenweg, err := DerBestellung(ctx, pool, res.BestellungID)
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
			if f.a == "" {
				t.Errorf("Exemplar %s: %s ist im Selbstdruck leer, der Vergleich prüft an diesem Feld nichts",
					eigen.BarcodeID, f.name)
			}
			if f.a != f.b {
				t.Errorf("Exemplar %s: %s unterscheidet sich — Selbstdruck %q, Lieferantenweg %q.\n"+
					"→ Beide Abfragen müssen dieselben Felder liefern, sonst trägt dasselbe Buch "+
					"je nach Druckweg einen anderen Aufkleber.", eigen.BarcodeID, f.name, f.a, f.b)
			}
		}
	}
}

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

// Dieselben Daten auf beiden Wegen zum Etikett. Das Druck-Center bekommt Anschaffungsjahr,
// Signatur und Topf von ErgaenzeServerfelder, der Lieferanten-Link von LadeDerBestellung: zwei
// Abfragen für dieselbe Frage. Wer eine erweitert und die andere vergisst, bekommt zwei
// verschiedene Aufkleber für dasselbe Buch, und jeder Weg für sich funktioniert.
//
// Verglichen wird, was der Server auf das Etikett bringt; Titel und Autor schickt im
// Druck-Center der Browser mit. Die ISBN zeichnet keiner der Erzeuger.
func TestEtikettenDatenSindAufBeidenWegenGleich(t *testing.T) {
	pool := pgtest.Pool(t)
	testdaten.LeereBestandsdaten(t, pool)
	ctx := context.Background()

	svc := service.NewBestellDienst(&db.Verbindung{Pool: pool}, repository.NewKatalogAbfragen(pool))

	lieferant := testdaten.LegeLieferantAn(t, pool, "Naacher-Paritaet", true)
	// Die Signatur ist gesetzt: Zwei leere Felder wären auf beiden Wegen gleich, auch wenn eine
	// der Abfragen das Feld nicht läse.
	titel := testdaten.LegeTitelMitSignaturAn(t, pool, "LMF-Paritaet", "LMF-Deutsch 5", 0)

	res, err := svc.LegeAn(ctx, service.BestellAuftrag{
		Mittel:      mitteltopf.Land,
		LieferantID: lieferant,
		Positionen:  []service.BestellAuftragPosition{{TitelID: titel, Menge: 3, Preis: 10, MitVorabBarcode: true}},
	})
	if err != nil {
		t.Fatalf("Bestellung: %v", err)
	}

	// Weg 1: Lieferantenseite hinter dem Bestätigungs-Link
	lieferantenweg, err := LadeDerBestellung(ctx, pool, res.BestellungID)
	if err != nil {
		t.Fatalf("Lieferanten-Etiketten laden: %v", err)
	}
	if len(lieferantenweg) != 3 {
		t.Fatalf("Lieferantenweg %d Etiketten, erwartet 3", len(lieferantenweg))
	}

	// Weg 2: ein Druckauftrag des Druck-Centers mit denselben Nummern, wie der Browser ihn schickt
	druckcenter := make([]Etikett, 0, len(lieferantenweg))
	for _, e := range lieferantenweg {
		druckcenter = append(druckcenter, Etikett{BarcodeID: e.BarcodeID, Titel: e.Titel, Autor: e.Autor})
	}
	ErgaenzeServerfelder(ctx, pool, druckcenter)

	for i, eigen := range druckcenter {
		fremd := lieferantenweg[i]
		felder := []struct{ name, a, b string }{
			{"Anschaffungsjahr", eigen.AnschaffungsJahr, fremd.AnschaffungsJahr},
			{"Signatur", eigen.Signatur, fremd.Signatur},
			{"Topf", eigen.Topf, fremd.Topf},
		}
		for _, f := range felder {
			if f.a == "" {
				t.Errorf("Exemplar %s: %s ist im Druck-Center leer, der Vergleich prüft an diesem Feld nichts",
					eigen.BarcodeID, f.name)
			}
			if f.a != f.b {
				t.Errorf("Exemplar %s: %s unterscheidet sich — Druck-Center %q, Lieferantenweg %q.\n"+
					"→ Beide Abfragen müssen dieselben Felder liefern, sonst trägt dasselbe Buch "+
					"je nach Druckweg einen anderen Aufkleber.", eigen.BarcodeID, f.name, f.a, f.b)
			}
		}
	}
}

package bestellung

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"bibliothek/api/tuer"
	"bibliothek/db"
	"bibliothek/internal/buchetikett"
	"bibliothek/internal/pgtest"
	"bibliothek/internal/service"
	"bibliothek/internal/testdaten"
	"bibliothek/pkg/mitteltopf"
	"bibliothek/repository"
)

// Der ganze Weg an echten Daten: bestellen → Token entsteht → Lieferant öffnet den Link
// → druckt die Etiketten seiner Lieferung. Die Einzelteile sind anderswo geprüft; hier
// geht es um die Übergänge, an denen sie sonst aneinander vorbeilaufen.

// TestBestellablauf_LinkUndEtiketten deckt die Stelle ab, an der die Etikettenseite
// falsch liegen könnte: Sie darf nur die Exemplare DIESER Bestellung und nur die
// Positionen MIT Vorab-Barcode enthalten — sonst bekleben Lieferant und Bibliothek
// dasselbe Buch doppelt.
func TestBestellablauf_LinkUndEtiketten(t *testing.T) {
	pool := pgtest.Pool(t)
	testdaten.LeereBestandsdaten(t, pool)
	ctx := context.Background()

	srv := &Tueren{Basis: &tuer.Basis{DB: &db.Database{Pool: pool}}}
	svc := service.NewBestellDienst(srv.DB, repository.NewKatalogAbfragen(pool))

	lieferant := testdaten.LegeLieferantAn(t, pool, "Naacher", true)
	mitBarcode := testdaten.LegeTitelAn(t, pool, "LMF-Mit-Barcode", 0)
	ohneBarcode := testdaten.LegeTitelAn(t, pool, "LMF-Ohne-Barcode", 0)

	res, err := svc.LegeAn(ctx, service.BestellAuftrag{
		Mittel:      mitteltopf.Land,
		LieferantID: lieferant,
		Positionen: []service.BestellAuftragPosition{
			{TitelID: mitBarcode, Menge: 3, Preis: 10, MitVorabBarcode: true},
			{TitelID: ohneBarcode, Menge: 2, Preis: 10, MitVorabBarcode: false},
		},
	})
	if err != nil {
		t.Fatalf("Bestellung: %v", err)
	}
	if res.BestaetigungsToken == "" {
		t.Fatal("kein Bestätigungs-Token erzeugt — der Lieferant bekäme eine Mail ohne Link")
	}

	// Die Exemplare müssen ihre Bestellung kennen; sonst kann die Seite ihre Etiketten
	// nicht von denen anderer Lieferungen desselben Titels trennen.
	var zugeordnet int
	if err := pool.QueryRow(ctx,
		`SELECT count(*) FROM buecher_exemplare WHERE bestellung_id = $1`, res.BestellungID).Scan(&zugeordnet); err != nil {
		t.Fatalf("Exemplare zählen: %v", err)
	}
	if zugeordnet != 5 {
		t.Fatalf("Exemplare mit Bestellbezug = %d, want 5", zugeordnet)
	}

	// Nur die drei Exemplare der Position MIT Vorab-Barcode gehören auf den Bogen.
	etiketten, err := buchetikett.LadeDerBestellung(ctx, pool, res.BestellungID)
	if err != nil {
		t.Fatalf("Etiketten laden: %v", err)
	}
	if len(etiketten) != 3 {
		t.Fatalf("Etiketten = %d, want 3 (nur Positionen mit Vorab-Barcode)", len(etiketten))
	}
	// „Ansch.J." steht auf der physischen Etikettenvorlage der Schule (Foto vom
	// Lernmittel-Etikett: Schulname / Titel / Ansch.J. / Barcode / Exemplar-Nr. /
	// Eigentumsvermerk / Ausleihtabelle). Es fehlte hier zunächst, weil das Etikett dem
	// Mailanhang gleichen sollte — der trug es aber selbst nicht. Beide Wege tragen es jetzt.
	jahr := strconv.Itoa(time.Now().Year())
	for _, e := range etiketten {
		if e.AnschaffungsJahr != jahr {
			t.Fatalf("Etikett der Seite trägt Anschaffungsjahr %q, want %q", e.AnschaffungsJahr, jahr)
		}
	}
	for _, e := range res.Etiketten {
		if e.AnschaffungsJahr != jahr {
			t.Fatalf("Etikett des Mailanhangs trägt Anschaffungsjahr %q, want %q", e.AnschaffungsJahr, jahr)
		}
	}

	if len(etiketten) != len(res.Etiketten) {
		t.Fatalf("Seite zeigt %d Etiketten, die Bestellmail enthielt %d — beide Wege müssen dieselben sein",
			len(etiketten), len(res.Etiketten))
	}

	// Und die Seite muss über den echten Token erreichbar sein, in beiden Größen.
	for _, groesse := range []string{"klein", "gross"} {
		req := httptest.NewRequest(http.MethodGet,
			"/api/public/bestellung/"+res.BestaetigungsToken+"/etiketten/"+groesse, nil)
		req.SetPathValue("token", res.BestaetigungsToken)
		req.SetPathValue("groesse", groesse)
		rec := httptest.NewRecorder()
		srv.OeffentlicheEtikettenHandler().ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("Etiketten %s: Status = %d, want 200, body: %s", groesse, rec.Code, rec.Body.String())
		}
		if !bytes.HasPrefix(rec.Body.Bytes(), []byte("%PDF")) {
			t.Fatalf("Etiketten %s: Antwort ist kein PDF", groesse)
		}
	}
}

// Lieferanten ohne den externen Schritt bekommen KEINEN Token. Sonst läge zu jeder
// Bestellung ein gültiger Link ohne Empfänger herum — mehr Angriffsfläche ohne Nutzen.
func TestBestellablauf_OhneBestaetigungKeinToken(t *testing.T) {
	pool := pgtest.Pool(t)
	testdaten.LeereBestandsdaten(t, pool)
	ctx := context.Background()

	srv := &Tueren{Basis: &tuer.Basis{DB: &db.Database{Pool: pool}}}
	svc := service.NewBestellDienst(srv.DB, repository.NewKatalogAbfragen(pool))

	lieferant := testdaten.LegeLieferantAn(t, pool, "Normalo", false)
	titel := testdaten.LegeTitelAn(t, pool, "LMF-Normal", 0)

	res, err := svc.LegeAn(ctx, service.BestellAuftrag{
		Mittel:      mitteltopf.Land,
		LieferantID: lieferant,
		Positionen:  []service.BestellAuftragPosition{{TitelID: titel, Menge: 1, Preis: 10, MitVorabBarcode: true}},
	})
	if err != nil {
		t.Fatalf("Bestellung: %v", err)
	}
	if res.BestaetigungsToken != "" {
		t.Fatal("Token für einen Lieferanten ohne Bestätigungsschritt erzeugt")
	}

	var hash *string
	if err := pool.QueryRow(ctx,
		`SELECT bestaetigungs_token_hash FROM bestellungen_verlauf WHERE id = $1`, res.BestellungID).Scan(&hash); err != nil {
		t.Fatalf("Token-Hash lesen: %v", err)
	}
	if hash != nil {
		t.Fatalf("Token-Hash = %q, want NULL", *hash)
	}
}

package api

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"bibliothek/db"
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
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	ctx := context.Background()

	srv := &Server{DB: &db.Database{Pool: pool}}
	svc := NewOrderService(srv.DB, repository.NewBookRepository(pool))

	lieferant := haendlerMitBestaetigung(t, pool, "Naacher", true)
	mitBarcode := titelMitMeldebestand(t, pool, "LMF-Mit-Barcode", 0)
	ohneBarcode := titelMitMeldebestand(t, pool, "LMF-Ohne-Barcode", 0)

	res, err := svc.ProcessOrder(ctx, SubmitOrderRequest{
		SupplierID: lieferant,
		Items: []OrderItemRequest{
			{TitelID: mitBarcode, Menge: 3, Preis: 10, GenerateBarcodes: true},
			{TitelID: ohneBarcode, Menge: 2, Preis: 10, GenerateBarcodes: false},
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
	etiketten, err := srv.ladeBestellEtiketten(ctx, res.BestellungID)
	if err != nil {
		t.Fatalf("Etiketten laden: %v", err)
	}
	if len(etiketten) != 3 {
		t.Fatalf("Etiketten = %d, want 3 (nur Positionen mit Vorab-Barcode)", len(etiketten))
	}
	if len(etiketten) != len(res.Labels) {
		t.Fatalf("Seite zeigt %d Etiketten, die Bestellmail enthielt %d — beide Wege müssen dieselben sein",
			len(etiketten), len(res.Labels))
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
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	ctx := context.Background()

	srv := &Server{DB: &db.Database{Pool: pool}}
	svc := NewOrderService(srv.DB, repository.NewBookRepository(pool))

	lieferant := haendlerMitBestaetigung(t, pool, "Normalo", false)
	titel := titelMitMeldebestand(t, pool, "LMF-Normal", 0)

	res, err := svc.ProcessOrder(ctx, SubmitOrderRequest{
		SupplierID: lieferant,
		Items:      []OrderItemRequest{{TitelID: titel, Menge: 1, Preis: 10, GenerateBarcodes: true}},
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

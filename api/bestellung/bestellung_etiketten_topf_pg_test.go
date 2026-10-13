package bestellung

import (
	"context"
	"net/http"
	"testing"

	"bibliothek/api/tuer"
	"bibliothek/db"
	"bibliothek/internal/buchetikett"
	"bibliothek/internal/pdftest"
	"bibliothek/internal/pgtest"
	"bibliothek/internal/service"
	"bibliothek/internal/testdaten"
	"bibliothek/pkg/mitteltopf"
	"bibliothek/repository"
)

// Der Eigentumsvermerk folgt dem Topf des Exemplars, auch auf den zwei Wegen zum Händler. Die
// zwei Wege des Hauses prüft api/druck/etiketten_topf_pg_test.go.

// Die beiden Wege zum Händler: der Lieferanten-Link (buchetikett.DerBestellung) und der
// Mailanhang (Etiketten aus ProcessOrder). Bestellt wird ein LERNMITTEL-Titel für die
// Schülerbücherei — der Fall, in dem Titel und Bestellung einander widersprechen.
func TestEigentumsvermerkFolgtDemTopf_LieferantenLinkUndMailanhang(t *testing.T) {
	pool := pgtest.Pool(t)
	testdaten.ResetBestandsdaten(t, pool)
	testdaten.SetzeVermerkSchuelerbuecherei(t, pool, testdaten.VermerkStadt)
	ctx := context.Background()
	srv := &Tueren{Basis: &tuer.Basis{DB: &db.Database{Pool: pool}}}

	svc := service.NewOrderService(srv.DB, repository.NewBookRepository(pool))
	res, err := svc.ProcessOrder(ctx, service.BestellAuftrag{
		Mittel:     mitteltopf.Schultraeger,
		SupplierID: testdaten.Haendler(t, pool, "Naacher", true),
		Items: []service.BestellAuftragPosition{{
			TitelID: testdaten.TitelMitMeldebestand(t, pool, "LMF-Topfprobe", 0), Menge: 2, Preis: 10, GenerateBarcodes: true}},
	})
	if err != nil {
		t.Fatalf("Bestellung: %v", err)
	}

	// Weg 3: der Link. Das große Etikett gibt es zu dieser Bestellung nicht (4.11).
	rec := etikettenBogenHolen(t, srv, res.BestaetigungsToken, "klein", testdaten.EtikettFormatMitVermerk)
	if rec.Code != http.StatusOK {
		t.Fatalf("Lieferanten-Link: Status %d — %s", rec.Code, rec.Body.String())
	}
	pdftest.Aufdruck(t, "Lieferanten-Link", pdftest.Suchtext(t, rec.Body.Bytes()), testdaten.VermerkStadt, testdaten.VermerkLand)

	// Weg 4: der Mailanhang, mit dem Kopf, den der Bestell-Handler baut.
	einstellungen, err := repository.NewEinstellungsAbfragen(pool).Lade(ctx)
	if err != nil {
		t.Fatalf("Einstellungen lesen: %v", err)
	}
	// Die Etiketten gehen durch dieselbe Füll-Funktion wie an der Tür der Bestellung.
	boegen, err := etikettenboegen(bestellmailDatenAus(res).Etiketten, buchetikett.EtikettKopfAus(einstellungen), true, res.Mittel)
	if err != nil {
		t.Fatalf("etikettenboegen: %v", err)
	}
	if len(boegen) == 0 {
		t.Fatal("kein Bogen im Anhang")
	}
	for _, b := range boegen {
		pdftest.Aufdruck(t, "Mailanhang "+b.Name, pdftest.Suchtext(t, b.Data), testdaten.VermerkStadt, testdaten.VermerkLand)
	}
}

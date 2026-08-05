package api

import (
	"context"
	"testing"

	"bibliothek/db"
	"bibliothek/repository"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Händler, die die Bücher vor der Lieferung selbst mit UNSEREN Barcodes bekleben.
//
// Anlass: etikett_gedruckt am Exemplar bedeutet „WIR haben gedruckt", nicht „am Buch
// klebt ein Etikett". Ein fertig beklebt geliefertes Buch war deshalb von einem
// unbeklebten nicht zu unterscheiden — es stand dauerhaft auf der Nachdruck-Liste und
// entwertete den Hinweis im Bestellwesen mit einer Zahl, die niemand abarbeiten kann.
//
// Warum ein Go-Test und kein E2E: Der Weg über POST /api/bestellungen verschickt die
// Bestellung per Mail. Der lokale Stack trägt den echten Schulserver in der Umgebung —
// ein Test, der dort Bestellmails auslöst, ist das Risiko nicht wert. ProcessOrder ist
// ohnehin die Stelle, an der entschieden wird; hier wird sie direkt geprüft.

// haendler legt einen Lieferanten an und gibt seine ID zurück.
func haendler(t *testing.T, pool *pgxpool.Pool, name string, beklebt bool) string {
	t.Helper()
	var id string
	err := pool.QueryRow(context.Background(), `
		INSERT INTO lieferanten (name, email, kundennummer, liefert_mit_barcode)
		VALUES ($1, $2, $3, $4) RETURNING id
	`, name, name+"@example.invalid", "K-"+name, beklebt).Scan(&id)
	if err != nil {
		t.Fatalf("Lieferant %s anlegen: %v", name, err)
	}
	return id
}

// offeneEtiketten zählt, wie viele Exemplare des Titels auf der Nachdruck-Liste stünden.
// Bewusst dieselbe Bedingung wie etikettenOffenBedingung in etiketten_offen.go.
func offeneEtiketten(t *testing.T, pool *pgxpool.Pool, titelID string) int {
	t.Helper()
	var n int
	err := pool.QueryRow(context.Background(), `
		SELECT count(*) FROM buecher_exemplare e
		WHERE e.titel_id = $1 AND e.etikett_gedruckt = false AND e.ist_ausgesondert = false
	`, titelID).Scan(&n)
	if err != nil {
		t.Fatalf("offene Etiketten zählen: %v", err)
	}
	return n
}

func TestProcessOrder_HaendlerBeklebtSelbst(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	ctx := context.Background()

	srv := &Server{DB: &db.Database{Pool: pool}}
	svc := NewOrderService(srv.DB, repository.NewBookRepository(pool))

	beklebtID := haendler(t, pool, "Beklebt", true)
	selbstID := haendler(t, pool, "Selbst", false)

	titelBeklebt := titelMitMeldebestand(t, pool, "LMF-Beklebt", 0)
	titelSelbst := titelMitMeldebestand(t, pool, "LMF-Selbst", 0)

	if _, err := svc.ProcessOrder(ctx, SubmitOrderRequest{
		SupplierID: beklebtID,
		Items:      []OrderItemRequest{{TitelID: titelBeklebt, Menge: 2, Preis: 10, GenerateBarcodes: true}},
	}); err != nil {
		t.Fatalf("Bestellung beim beklebenden Händler: %v", err)
	}

	if _, err := svc.ProcessOrder(ctx, SubmitOrderRequest{
		SupplierID: selbstID,
		Items:      []OrderItemRequest{{TitelID: titelSelbst, Menge: 2, Preis: 10, GenerateBarcodes: true}},
	}); err != nil {
		t.Fatalf("Bestellung beim normalen Händler: %v", err)
	}

	// Der Kern: Wer beklebt liefert, erzeugt keine offenen Etiketten.
	if n := offeneEtiketten(t, pool, titelBeklebt); n != 0 {
		t.Errorf("beklebt gelieferte Exemplare gelten als erledigt: offen = %d, erwartet 0", n)
	}

	// Gegenprobe im selben Lauf — sonst bewiese der Test nur, dass die Abfrage nichts findet.
	if n := offeneEtiketten(t, pool, titelSelbst); n != 2 {
		t.Errorf("selbst zu beklebende Exemplare bleiben offen: offen = %d, erwartet 2", n)
	}

	// Die Barcodes müssen trotzdem entstehen: Ohne sie hätte der Händler nichts
	// aufzukleben, und die Annahme „kommt beklebt an" wäre falsch.
	var mitBarcode int
	if err := pool.QueryRow(ctx, `
		SELECT count(*) FROM buecher_exemplare WHERE titel_id = $1 AND barcode_id <> ''
	`, titelBeklebt).Scan(&mitBarcode); err != nil {
		t.Fatalf("Barcodes zählen: %v", err)
	}
	if mitBarcode != 2 {
		t.Errorf("Exemplare müssen Barcodes tragen: %d von 2", mitBarcode)
	}
}

// Die Etiketten im Lieferanten-Mailanhang entstehen aus ProcessOrder, NICHT aus dem
// Druck-Center-Pfad (queryLabelItems) — die Signatur muss deshalb hier ankommen, über
// die echte Kette GetTitleByIDTx → BarcodeLabelDetail. Ein Unit-Test, der Signatur
// direkt ans Struct schreibt, beweist das nicht (siehe Memory
// verify-audit-fixes-over-live-path: genau so blieb sie im Mailanhang unbemerkt leer).
func TestProcessOrder_LabelsTragenSignatur(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	ctx := context.Background()

	srv := &Server{DB: &db.Database{Pool: pool}}
	svc := NewOrderService(srv.DB, repository.NewBookRepository(pool))

	lieferant := haendler(t, pool, "MitSignatur", false)
	titel := titelMitSignatur(t, pool, "Deutschbuch 5", "LMF-Deutsch 5", 0)

	res, err := svc.ProcessOrder(ctx, SubmitOrderRequest{
		SupplierID: lieferant,
		Items:      []OrderItemRequest{{TitelID: titel, Menge: 1, Preis: 10, GenerateBarcodes: true}},
	})
	if err != nil {
		t.Fatalf("Bestellung: %v", err)
	}

	if len(res.Labels) != 1 {
		t.Fatalf("Labels = %d, erwartet 1", len(res.Labels))
	}
	if got := res.Labels[0].Signatur; got != "LMF-Deutsch 5" {
		t.Errorf("Label-Signatur = %q, erwartet %q — der Mailanhang an den Lieferanten bliebe ohne Signatur", got, "LMF-Deutsch 5")
	}
}

// Ohne erzeugten Barcode steht für die Position nichts auf dem Bogen — der Händler kann
// nichts aufkleben, also bleibt das Etikett unsere Aufgabe, auch wenn er sonst beklebt.
func TestProcessOrder_BeklebenderHaendlerOhneBarcodebogen(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	ctx := context.Background()

	srv := &Server{DB: &db.Database{Pool: pool}}
	svc := NewOrderService(srv.DB, repository.NewBookRepository(pool))

	lieferant := haendler(t, pool, "BeklebtOhneBogen", true)
	titel := titelMitMeldebestand(t, pool, "LMF-OhneBogen", 0)

	if _, err := svc.ProcessOrder(ctx, SubmitOrderRequest{
		SupplierID: lieferant,
		Items:      []OrderItemRequest{{TitelID: titel, Menge: 1, Preis: 10, GenerateBarcodes: false}},
	}); err != nil {
		t.Fatalf("Bestellung ohne Barcodebogen: %v", err)
	}

	if n := offeneEtiketten(t, pool, titel); n != 1 {
		t.Errorf("ohne Bogen bleibt das Etikett offen: offen = %d, erwartet 1", n)
	}
}

package katalog

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"bibliothek/api/tuer"
	"bibliothek/db"
	"bibliothek/internal/pgtest"
	"bibliothek/internal/testdaten"
	"bibliothek/repository"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Exemplarnummern dürfen sich nicht überschneiden und nach dem Löschen nicht
// zurückkehren: Die Nummer klebt physisch am Buch. Zwei Bücher mit derselben Nummer
// sind an der Theke nicht auseinanderzuhalten, und ein altes Etikett, dessen Nummer
// erneut vergeben wurde, bucht auf das falsche Buch.
//
// Warum ein Test an der Datenbank: Die Vergabe steckt ganz in SQL (nextval auf barcode_seq
// gegen einen Abgleich mit dem Bestand), und die Kollision zeigt sich erst am UNIQUE-Constraint
// der Tabelle. pgxmock spielt beides nur nach.

// naechsteInterneNummer ruft den Weg des Knopfes „Interne ID generieren" (GET /api/barcode/next
// in der Exemplarkarte), nicht den Generator direkt: An diesem Weg darf keine zweite Quelle für
// Nummern entstehen.
func naechsteInterneNummer(t *testing.T, srv *Tueren) string {
	t.Helper()
	rec := httptest.NewRecorder()
	srv.ExemplarBarcodeZiehen()(rec, httptest.NewRequest(http.MethodGet, "/api/barcode/next", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("/api/barcode/next: Status %d, Body %s", rec.Code, rec.Body.String())
	}
	var antwort struct {
		NextBarcode string `json:"next_barcode"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &antwort); err != nil {
		t.Fatalf("Antwort von /api/barcode/next lesen: %v", err)
	}
	if antwort.NextBarcode == "" {
		t.Fatal("/api/barcode/next lieferte eine leere Nummer")
	}
	return antwort.NextBarcode
}

// bestelleExemplare geht den Weg des Bestellwesens: Barcodes vorab ziehen und die
// Exemplare damit anlegen (BestellDienst.LegeAn). Schlägt der Insert
// fehl, kippt in der Anwendung die komplette Bestellung — deshalb ist der Insert hier
// Teil der Prüfung und nicht nur der gezogene Barcode.
func bestelleExemplare(t *testing.T, pool *pgxpool.Pool, repo *repository.KatalogAbfragen, titelID string, menge int) []string {
	t.Helper()
	ctx := context.Background()

	barcodes, err := repo.ErzeugeBarcodes(ctx, menge)
	if err != nil {
		t.Fatalf("Barcodes für %d Exemplare ziehen: %v", menge, err)
	}
	kopien := make([]repository.ExemplarNeu, menge)
	for i, bc := range barcodes {
		kopien[i] = repository.ExemplarNeu{TitelID: titelID, BarcodeID: bc}
	}
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("Tx starten: %v", err)
	}
	if err := repo.LegeExemplareAn(ctx, tx, kopien); err != nil {
		db.Verwerfe(ctx, tx)
		t.Fatalf("Bestellung über %d Exemplare anlegen (Barcodes %v): %v", menge, barcodes, err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatalf("Tx abschließen: %v", err)
	}
	return barcodes
}

// barcodeBestand liest alle vergebenen Nummern als Menge.
func barcodeBestand(t *testing.T, pool *pgxpool.Pool) map[string]bool {
	t.Helper()
	rows, err := pool.Query(context.Background(), `SELECT barcode_id FROM buecher_exemplare`)
	if err != nil {
		t.Fatalf("Barcodes lesen: %v", err)
	}
	defer rows.Close()

	belegt := map[string]bool{}
	for rows.Next() {
		var bc string
		if err := rows.Scan(&bc); err != nil {
			t.Fatalf("Barcode lesen: %v", err)
		}
		belegt[bc] = true
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("Barcodes lesen: %v", err)
	}
	return belegt
}

// TestBarcodeVergabe_HandvergabeKollidiertNichtMitBestellung: Das Bestellwesen und der Knopf
// „Interne ID generieren" ziehen aus derselben Sequenz barcode_seq. Rechnete der Knopf die
// höchste Nummer plus eins aus der Tabelle, lieferten beide dieselbe nächste Nummer, aber nur die
// Sequenz zählte weiter: Die von Hand vergebene Nummer käme bei der nächsten Bestellung erneut,
// die Bestellung scheiterte am UNIQUE, und weil die Exemplare in einer Transaktion per CopyFrom
// entstehen, ginge die ganze Bestellung verloren, nicht nur die Position.
func TestBarcodeVergabe_HandvergabeKollidiertNichtMitBestellung(t *testing.T) {
	pool := pgtest.Pool(t)
	testdaten.LeereBestandsdaten(t, pool)
	srv := &Tueren{Basis: &tuer.Basis{DB: &db.Verbindung{Pool: pool}}}
	bookRepo := repository.NewKatalogAbfragen(pool)

	titel := testdaten.LegeTitelAn(t, pool, "Barcode-Vergabe", 0)

	// 1. Eine Bestellung mit Vorab-Barcodes bringt den Bestand auf Stand.
	bestelleExemplare(t, pool, bookRepo, titel, 2)

	// 2. Ein Altbestands-Exemplar bekommt über den Knopf eine interne Nummer.
	platzhalter := testdaten.LegeExemplarAn(t, pool, titel, "SYS-1", true, "")
	handNummer := naechsteInterneNummer(t, srv)
	if err := bookRepo.SetzeExemplarBarcode(context.Background(), platzhalter, handNummer); err != nil {
		t.Fatalf("Handvergabe von %s: %v", handNummer, err)
	}

	// 3. Die nächste Bestellung darf diese Nummer nicht erneut ziehen.
	neue := bestelleExemplare(t, pool, bookRepo, titel, 1)
	if neue[0] == handNummer {
		t.Errorf("Bestellung zog erneut %s — dieselbe Nummer klebt dann an zwei Büchern", handNummer)
	}
}

// TestBarcodeVergabe_GeloeschteNummerWirdNichtRecycelt: Titel löschen, Massenlöschung
// in der Bestandsverwaltung und „endgültig löschen" im Fehlbestand entfernen die
// Exemplarzeile wirklich (kein Soft-Delete). Ein Generator, der die nächste Nummer aus
// dem Bestand ableitet, gibt danach genau die freigewordene Nummer wieder aus — und
// ein noch existierendes Etikett zeigt damit auf ein anderes Buch.
func TestBarcodeVergabe_GeloeschteNummerWirdNichtRecycelt(t *testing.T) {
	pool := pgtest.Pool(t)
	testdaten.LeereBestandsdaten(t, pool)
	srv := &Tueren{Basis: &tuer.Basis{DB: &db.Verbindung{Pool: pool}}}

	titel := testdaten.LegeTitelAn(t, pool, "Barcode-Recycling", 0)

	vergeben := naechsteInterneNummer(t, srv)
	id := testdaten.LegeExemplarAn(t, pool, titel, vergeben, true, "")

	if _, err := pool.Exec(context.Background(),
		`DELETE FROM buecher_exemplare WHERE id = $1`, id); err != nil {
		t.Fatalf("Exemplar hart löschen: %v", err)
	}

	if wieder := naechsteInterneNummer(t, srv); wieder == vergeben {
		t.Errorf("Nummer %s wurde nach dem Löschen erneut ausgegeben", vergeben)
	}
}

// TestBarcodeVergabe_UeberspringtVonHandVergebeneNummern sichert den Fall ab, in dem
// die Sequenz gegenüber dem Bestand zurückliegt — etwa weil in der Exemplarkarte
// direkt eine B-Nummer eingetippt wurde oder weil die Datenbank aus der Zeit der
// zweiten Vergabestelle stammt. Der Generator muss über den belegten Bereich
// hinweggehen, statt eine bereits vergebene Nummer auszugeben.
func TestBarcodeVergabe_UeberspringtVonHandVergebeneNummern(t *testing.T) {
	pool := pgtest.Pool(t)
	testdaten.LeereBestandsdaten(t, pool)
	ctx := context.Background()
	bookRepo := repository.NewKatalogAbfragen(pool)

	titel := testdaten.LegeTitelAn(t, pool, "Barcode-Drift", 0)

	// Von Hand weit über den Stand der Sequenz hinaus vergeben.
	var stand int64
	if err := pool.QueryRow(ctx, `SELECT last_value FROM barcode_seq`).Scan(&stand); err != nil {
		t.Fatalf("Sequenzstand lesen: %v", err)
	}
	for i := int64(1); i <= 5; i++ {
		testdaten.LegeExemplarAn(t, pool, titel, fmt.Sprintf("B-%05d", stand+i), true, "")
	}

	belegt := barcodeBestand(t, pool)
	neue := bestelleExemplare(t, pool, bookRepo, titel, 3)
	for _, bc := range neue {
		if belegt[bc] {
			t.Errorf("Generator gab die bereits vergebene Nummer %s aus", bc)
		}
	}
}

// TestBarcodeVergabe_SechsstelligeNummerBleibtGanz: Über 99999 wächst die Nummer um eine Stelle.
// Auf fünf Zeichen gekürzt, hießen 100000 und 100001 beide B-10000: Die Vergabe gäbe dieselbe
// Nummer zweimal aus oder die eines alten Buchs. Geprüft an der Bestellung und am Knopf
// „Interne ID generieren".
func TestBarcodeVergabe_SechsstelligeNummerBleibtGanz(t *testing.T) {
	pool := pgtest.Pool(t)
	testdaten.LeereBestandsdaten(t, pool)
	ctx := context.Background()
	srv := &Tueren{Basis: &tuer.Basis{DB: &db.Verbindung{Pool: pool}}}
	bookRepo := repository.NewKatalogAbfragen(pool)

	titel := testdaten.LegeTitelAn(t, pool, "Barcode-Sechsstellig", 0)
	var stand int64
	if err := pool.QueryRow(ctx, `SELECT setval('barcode_seq', GREATEST(last_value, 99998)) FROM barcode_seq`).
		Scan(&stand); err != nil {
		t.Fatalf("Sequenz an die Grenze stellen: %v", err)
	}

	pruefe := func(weg string, i int64, nummer string) {
		t.Helper()
		if want := fmt.Sprintf("B-%05d", stand+i); nummer != want {
			t.Errorf("%s: Nummer %d ist %s, erwartet %s", weg, i, nummer, want)
		}
	}
	for i, nummer := range bestelleExemplare(t, pool, bookRepo, titel, 2) {
		pruefe("Bestellung", int64(i)+1, nummer)
	}
	pruefe("Interne ID generieren", 3, naechsteInterneNummer(t, srv))
}

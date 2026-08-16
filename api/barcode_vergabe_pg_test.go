package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"bibliothek/db"
	"bibliothek/repository"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Exemplarnummern dürfen sich nicht überschneiden und nach dem Löschen nicht
// zurückkehren: Die Nummer klebt physisch am Buch. Zwei Bücher mit derselben Nummer
// sind an der Theke nicht auseinanderzuhalten, und ein altes Etikett, dessen Nummer
// erneut vergeben wurde, bucht auf das falsche Buch.
//
// Warum ein PG-Test: Die Vergabe steckt komplett in SQL (nextval auf barcode_seq gegen
// einen Bestandsabgleich) und die Kollision zeigt sich erst am UNIQUE-Constraint der
// Tabelle. pgxmock spielt beides nur nach.

// naechsteInterneNummer ruft den Live-Pfad des Knopfes „Interne ID generieren" auf
// (GET /api/barcode/next in der Exemplarkarte) statt den Generator direkt — genau
// dieser Weg vergab die Nummern früher aus einer zweiten, eigenen Quelle.
func naechsteInterneNummer(t *testing.T, srv *Server) string {
	t.Helper()
	rec := httptest.NewRecorder()
	srv.NextBarcodeHandler()(rec, httptest.NewRequest(http.MethodGet, "/api/barcode/next", nil))
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
// Exemplare damit anlegen (ProcessOrder, api/order_service.go). Schlägt der Insert
// fehl, kippt in der Anwendung die komplette Bestellung — deshalb ist der Insert hier
// Teil der Prüfung und nicht nur der gezogene Barcode.
func bestelleExemplare(t *testing.T, pool *pgxpool.Pool, repo repository.BookRepository, titelID string, menge int) []string {
	t.Helper()
	ctx := context.Background()

	barcodes, err := repo.GenerateBarcodes(ctx, menge)
	if err != nil {
		t.Fatalf("Barcodes für %d Exemplare ziehen: %v", menge, err)
	}
	kopien := make([]repository.BookCopyInsert, menge)
	for i, bc := range barcodes {
		kopien[i] = repository.BookCopyInsert{TitelID: titelID, BarcodeID: bc}
	}
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("Tx starten: %v", err)
	}
	if err := repo.BulkInsertCopiesTx(ctx, tx, kopien); err != nil {
		db.SafeRollback(ctx, tx)
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

// TestBarcodeVergabe_HandvergabeKollidiertNichtMitBestellung deckt den Bruch zwischen
// den beiden früheren Vergabestellen auf: Das Bestellwesen zog aus der Sequenz
// barcode_seq, der Knopf „Interne ID generieren" rechnete stattdessen MAX(Nummer)+1
// aus der Tabelle. Beide lieferten damit dieselbe nächste Nummer, aber nur die Sequenz
// zählte weiter — die von Hand vergebene Nummer kam bei der nächsten Bestellung
// erneut. Ergebnis: UNIQUE-Verletzung, und weil die Exemplare in EINER Transaktion per
// CopyFrom entstehen, verlor nicht die Position, sondern die ganze Bestellung.
func TestBarcodeVergabe_HandvergabeKollidiertNichtMitBestellung(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	srv := &Server{DB: &db.Database{Pool: pool}}
	bookRepo := repository.NewBookRepository(pool)

	titel := titelMitMeldebestand(t, pool, "Barcode-Vergabe", 0)

	// 1. Eine Bestellung mit Vorab-Barcodes bringt den Bestand auf Stand.
	bestelleExemplare(t, pool, bookRepo, titel, 2)

	// 2. Ein Altbestands-Exemplar bekommt über den Knopf eine interne Nummer.
	platzhalter := exemplar(t, pool, titel, "SYS-1", true, "")
	handNummer := naechsteInterneNummer(t, srv)
	if err := bookRepo.UpdateCopyBarcode(context.Background(), platzhalter, handNummer); err != nil {
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
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	srv := &Server{DB: &db.Database{Pool: pool}}

	titel := titelMitMeldebestand(t, pool, "Barcode-Recycling", 0)

	vergeben := naechsteInterneNummer(t, srv)
	id := exemplar(t, pool, titel, vergeben, true, "")

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
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	ctx := context.Background()
	bookRepo := repository.NewBookRepository(pool)

	titel := titelMitMeldebestand(t, pool, "Barcode-Drift", 0)

	// Von Hand weit über den Stand der Sequenz hinaus vergeben.
	var stand int64
	if err := pool.QueryRow(ctx, `SELECT last_value FROM barcode_seq`).Scan(&stand); err != nil {
		t.Fatalf("Sequenzstand lesen: %v", err)
	}
	for i := int64(1); i <= 5; i++ {
		exemplar(t, pool, titel, fmt.Sprintf("B-%05d", stand+i), true, "")
	}

	belegt := barcodeBestand(t, pool)
	neue := bestelleExemplare(t, pool, bookRepo, titel, 3)
	for _, bc := range neue {
		if belegt[bc] {
			t.Errorf("Generator gab die bereits vergebene Nummer %s aus", bc)
		}
	}
}

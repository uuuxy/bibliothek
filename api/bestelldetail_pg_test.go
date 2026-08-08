package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"bibliothek/db"
	"bibliothek/repository"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Die Detailansicht einer Bestellung steht und fällt mit zwei Verbindungen, die es
// beide erst seit kurzem gibt und die beide unsichtbar reissen können:
//
//  1. bestellungen_positionen → buecher_titel (Autor, Verlag, Cover)
//  2. buecher_exemplare.bestellung_id → die tatsächlich gelieferten Bücher
//
// Punkt 2 ist der eigentliche Zugewinn und war bis Migration 063 unmöglich. Ein Test,
// der nur den Statuscode prüft, würde 200 und eine leere Exemplarliste durchwinken —
// also genau den Zustand, wegen dem die aufklappende Zeile vorher nichts wert war.
//
// Nullbare Spalten sind hier kein Nebenschauplatz: autor, verlag und cover_url dürfen
// leer sein, und ein NULL in einem nicht-nullbaren Go-Typ ist in diesem Projekt eine
// wiederkehrende 500er-Quelle. Der Test legt deshalb einen Titel MIT und einen OHNE
// diese Angaben an.

func holeDetail(t *testing.T, srv *Server, pool *pgxpool.Pool, bestellungID string) repository.BestellungDetail {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/api/bestellhistorie/"+bestellungID, nil)
	req.SetPathValue("id", bestellungID)
	rec := httptest.NewRecorder()
	srv.GetBestelldetailHandler(repository.NewBestelldetailRepository(pool)).ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("Detail Status = %d, body: %s", rec.Code, rec.Body.String())
	}

	var d repository.BestellungDetail
	if err := json.Unmarshal(rec.Body.Bytes(), &d); err != nil {
		t.Fatalf("Detail lesen: %v", err)
	}
	return d
}

// exemplarAusBestellung legt ein Exemplar an, das seine Herkunft kennt — genau das, was
// order_service.go beim Bestellen tut.
func exemplarAusBestellung(t *testing.T, pool *pgxpool.Pool, titelID, bestellungID, barcode string, etikettGedruckt bool) {
	t.Helper()
	_, err := pool.Exec(context.Background(), `
		INSERT INTO buecher_exemplare (titel_id, barcode_id, bestellung_id, etikett_gedruckt)
		VALUES ($1, $2, $3, $4)
	`, titelID, barcode, bestellungID, etikettGedruckt)
	if err != nil {
		t.Fatalf("Exemplar %s anlegen: %v", barcode, err)
	}
}

func TestBestelldetail_ZeigtTitelangabenUndGelieferteExemplare(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	ctx := context.Background()
	srv := &Server{DB: &db.Database{Pool: pool}}

	lieferant := haendler(t, pool, "Naacher-Detail", true)
	bestellung := bestellungFuerLieferant(t, pool, lieferant)

	// Ein vollständig gepflegter Titel …
	var mitAngaben string
	if err := pool.QueryRow(ctx, `
		INSERT INTO buecher_titel (titel, autor, verlag, isbn, cover_url)
		VALUES ('Lambacher Schweizer 7', 'Ruth Zeidler', 'Klett', '9783127338713', '/uploads/cover/ls7.webp')
		RETURNING id`).Scan(&mitAngaben); err != nil {
		t.Fatalf("Titel mit Angaben: %v", err)
	}
	// … und einer, bei dem autor/verlag/cover_url NULL sind.
	ohneAngaben := titelMitMeldebestand(t, pool, "Titel ohne Angaben", 5)

	for _, p := range []struct {
		titelID string
		name    string
		menge   int
		preis   float64
	}{
		{mitAngaben, "Lambacher Schweizer 7", 2, 24.50},
		{ohneAngaben, "Titel ohne Angaben", 1, 10.00},
	} {
		if _, err := pool.Exec(ctx, `
			INSERT INTO bestellungen_positionen (bestellung_id, titel_id, titel_name, isbn, menge, einzelpreis)
			VALUES ($1, $2, $3, '', $4, $5)
		`, bestellung, p.titelID, p.name, p.menge, p.preis); err != nil {
			t.Fatalf("Position %s: %v", p.name, err)
		}
	}

	// Zwei Exemplare aus DIESER Bestellung, eines davon noch ohne Etikett …
	exemplarAusBestellung(t, pool, mitAngaben, bestellung, "DETAIL-0001", true)
	exemplarAusBestellung(t, pool, mitAngaben, bestellung, "DETAIL-0002", false)
	// … und eines desselben Titels OHNE Bestellbezug (Altbestand). Es darf nicht
	// auftauchen: Sonst zeigte der Beleg Bücher, die nie in diesem Karton lagen.
	if _, err := pool.Exec(ctx,
		`INSERT INTO buecher_exemplare (titel_id, barcode_id) VALUES ($1, 'ALTBESTAND-1')`,
		mitAngaben); err != nil {
		t.Fatalf("Altbestand anlegen: %v", err)
	}

	detail := holeDetail(t, srv, pool, bestellung)

	if len(detail.Positionen) != 2 {
		t.Fatalf("Positionen = %d, want 2", len(detail.Positionen))
	}

	// Sortiert nach titel_name: "Lambacher …" vor "Titel ohne Angaben".
	gepflegt := detail.Positionen[0]
	if gepflegt.Autor != "Ruth Zeidler" || gepflegt.Verlag != "Klett" {
		t.Errorf("Autor/Verlag = %q/%q, want Ruth Zeidler/Klett", gepflegt.Autor, gepflegt.Verlag)
	}
	if gepflegt.CoverURL != "/uploads/cover/ls7.webp" {
		t.Errorf("CoverURL = %q — ohne Cover ist die Ansicht nur eine schmalere Tabelle", gepflegt.CoverURL)
	}
	if gepflegt.Gesamtpreis != 49.00 {
		t.Errorf("Gesamtpreis = %v, want 49.00 (2 × 24,50)", gepflegt.Gesamtpreis)
	}

	// NULL-Spalten kommen als leere Zeichenkette an, nicht als 500.
	leer := detail.Positionen[1]
	if leer.Autor != "" || leer.Verlag != "" || leer.CoverURL != "" {
		t.Errorf("ungepflegter Titel: Autor=%q Verlag=%q Cover=%q, want jeweils leer",
			leer.Autor, leer.Verlag, leer.CoverURL)
	}

	// Der eigentliche Zugewinn: die Exemplare DIESER Lieferung, mit ihren Nummern.
	if len(detail.Exemplare) != 2 {
		t.Fatalf("Exemplare = %d, want 2 (Altbestand ohne bestellung_id gehört nicht dazu)",
			len(detail.Exemplare))
	}
	barcodes := map[string]bool{}
	for _, e := range detail.Exemplare {
		barcodes[e.BarcodeID] = e.EtikettGedruckt
		if e.TitelName != "Lambacher Schweizer 7" {
			t.Errorf("Exemplar %s trägt Titel %q", e.BarcodeID, e.TitelName)
		}
	}
	if gedruckt, ok := barcodes["DETAIL-0001"]; !ok || !gedruckt {
		t.Errorf("DETAIL-0001 fehlt oder gilt als ungedruckt: %v/%v", ok, gedruckt)
	}
	if gedruckt, ok := barcodes["DETAIL-0002"]; !ok || gedruckt {
		t.Errorf("DETAIL-0002 fehlt oder gilt als gedruckt: %v/%v", ok, gedruckt)
	}
	if barcodes["ALTBESTAND-1"] {
		t.Error("Altbestand ohne bestellung_id steht im Beleg dieser Bestellung")
	}
}

// Eine unbekannte ID ist ein veralteter Verweis, keine Störung. 404 statt 500, sonst
// ersetzt der Sanitizer die Meldung durch "interner Datenbankfehler" und der Benutzer
// erfährt nie, dass die Bestellung schlicht nicht mehr existiert.
func TestBestelldetail_UnbekannteBestellungIst404(t *testing.T) {
	pool := pgTestPool(t)
	srv := &Server{DB: &db.Database{Pool: pool}}

	const unbekannt = "00000000-0000-0000-0000-0000000000ff"
	req := httptest.NewRequest(http.MethodGet, "/api/bestellhistorie/"+unbekannt, nil)
	req.SetPathValue("id", unbekannt)
	rec := httptest.NewRecorder()
	srv.GetBestelldetailHandler(repository.NewBestelldetailRepository(pool)).ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("Status = %d, want 404 — body: %s", rec.Code, rec.Body.String())
	}
}

package bestellung

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"bibliothek/internal/api/tuer"
	"bibliothek/internal/auth"
	"bibliothek/internal/db"
	"bibliothek/internal/pgtest"
	"bibliothek/internal/pkg/mitteltopf"
	"bibliothek/internal/testdaten"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Den Preis einer Bestellposition nachtragen, am Live-Pfad über die Tür: Position, Einkaufspreis
// der Exemplare und Summe der Bestellung tragen danach denselben Preis, und das Protokoll nennt
// alten und neuen. Ein ungültiger Preis oder eine fremde Position ändert nichts.

func aenderePreisUeberTuer(t *testing.T, srv *Tueren, pool *pgxpool.Pool, bestellungID, positionID, rumpf string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPut,
		"/api/bestellungen/"+bestellungID+"/positionen/"+positionID+"/einzelpreis", strings.NewReader(rumpf))
	req.SetPathValue("id", bestellungID)
	req.SetPathValue("position_id", positionID)
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(context.WithValue(req.Context(), auth.ClaimsSchluessel,
		&auth.Claims{KontoID: testdaten.LegeBearbeiterAn(t, pool), Rolle: auth.RolleAdmin}))
	rec := httptest.NewRecorder()
	srv.PositionspreisAendern()(rec, req)
	return rec
}

// preiseInDB liest Einzelpreis der Position, die Einkaufspreise der Exemplare dieser Bestellung
// und die Summe der Bestellung als Text, wie Postgres sie schreibt.
func preiseInDB(t *testing.T, pool *pgxpool.Pool, bestellungID, positionID string) (einzel string, exemplare []string, summe string) {
	t.Helper()
	ctx := context.Background()
	if err := pool.QueryRow(ctx, `SELECT einzelpreis::text FROM bestellungen_positionen WHERE id = $1`, positionID).Scan(&einzel); err != nil {
		t.Fatalf("Einzelpreis lesen: %v", err)
	}
	if err := pool.QueryRow(ctx, `
		SELECT coalesce(array_agg(einkaufspreis::text ORDER BY barcode_id), '{}') FROM buecher_exemplare
		WHERE bestellung_id = $1`, bestellungID).Scan(&exemplare); err != nil {
		t.Fatalf("Einkaufspreise lesen: %v", err)
	}
	if err := pool.QueryRow(ctx, `SELECT gesamtbetrag::text FROM bestellungen_verlauf WHERE id = $1`, bestellungID).Scan(&summe); err != nil {
		t.Fatalf("Summe lesen: %v", err)
	}
	return einzel, exemplare, summe
}

func zaehlePreisProtokoll(t *testing.T, pool *pgxpool.Pool, positionID string) int {
	t.Helper()
	var n int
	if err := pool.QueryRow(context.Background(), `
		SELECT count(*) FROM audit_logs WHERE aktion = $1 AND details->>'position_id' = $2`,
		aktionBestellpreisGeaendert, positionID).Scan(&n); err != nil {
		t.Fatalf("Protokoll lesen: %v", err)
	}
	return n
}

// bestellungMitPosition legt über die Tür eine Bestellung mit einer Position an (2 × 9,50 €) und
// liefert die Kennungen von Bestellung und Position.
func bestellungMitPosition(t *testing.T, srv *Tueren, pool *pgxpool.Pool, name string) (string, string) {
	t.Helper()
	lieferant := testdaten.LegeLieferantAn(t, pool, name, false)
	titel := testdaten.LegeTitelAn(t, pool, "LMF-"+name)
	if rec := bestelleMitTopf(t, srv, lieferant, titel, mitteltopf.Land); rec.Code != http.StatusOK {
		t.Fatalf("Bestellung anlegen: %d %s", rec.Code, rec.Body.String())
	}
	var bestellungID, positionID string
	if err := pool.QueryRow(context.Background(), `
		SELECT v.id, p.id FROM bestellungen_verlauf v JOIN bestellungen_positionen p ON p.bestellung_id = v.id
		WHERE v.lieferant_id = $1`, lieferant).Scan(&bestellungID, &positionID); err != nil {
		t.Fatalf("Bestellung lesen: %v", err)
	}
	return bestellungID, positionID
}

func TestPositionspreis_ZiehtExemplareUndSummeMit(t *testing.T) {
	pool := pgtest.Pool(t)
	testdaten.LeereBestandsdaten(t, pool)
	srv := &Tueren{Basis: &tuer.Basis{DB: &db.Verbindung{Pool: pool}}}
	bestellungID, positionID := bestellungMitPosition(t, srv, pool, "Preis-Haendler")

	if einzel, exemplare, summe := preiseInDB(t, pool, bestellungID, positionID); einzel != "9.50" ||
		strings.Join(exemplare, ",") != "9.50,9.50" || summe != "19.00" {
		t.Fatalf("vorher: Position %s, Exemplare %v, Summe %s; erwartet 9.50, zweimal 9.50, 19.00", einzel, exemplare, summe)
	}

	// Der Händler rechnet 12,40 € ab statt der 9,50 € des Fachbereichs.
	if rec := aenderePreisUeberTuer(t, srv, pool, bestellungID, positionID, `{"einzelpreis":12.4}`); rec.Code != http.StatusOK {
		t.Fatalf("Preis ändern: %d %s", rec.Code, rec.Body.String())
	}
	if einzel, exemplare, summe := preiseInDB(t, pool, bestellungID, positionID); einzel != "12.40" ||
		strings.Join(exemplare, ",") != "12.40,12.40" || summe != "24.80" {
		t.Errorf("nachher: Position %s, Exemplare %v, Summe %s; erwartet 12.40, zweimal 12.40, 24.80", einzel, exemplare, summe)
	}
	var von, nach, exemplare string
	if err := pool.QueryRow(context.Background(), `
		SELECT details->>'von', details->>'nach', details->>'exemplare' FROM audit_logs
		WHERE aktion = $1 AND details->>'position_id' = $2`, aktionBestellpreisGeaendert, positionID).Scan(&von, &nach, &exemplare); err != nil {
		t.Fatalf("Protokolleintrag fehlt: %v", err)
	}
	if von != "9.50" || nach != "12.40" || exemplare != "2" {
		t.Errorf("Protokoll von %q nach %q, %q Exemplare; erwartet 9.50 nach 12.40, 2 Exemplare", von, nach, exemplare)
	}

	// Derselbe Preis noch einmal schreibt keinen zweiten Eintrag.
	if rec := aenderePreisUeberTuer(t, srv, pool, bestellungID, positionID, `{"einzelpreis":12.40}`); rec.Code != http.StatusOK {
		t.Fatalf("gleicher Preis: %d %s", rec.Code, rec.Body.String())
	}
	if n := zaehlePreisProtokoll(t, pool, positionID); n != 1 {
		t.Errorf("%d Protokolleinträge, erwartet 1", n)
	}
}

func TestPositionspreis_UngueltigesAendertNichts(t *testing.T) {
	pool := pgtest.Pool(t)
	testdaten.LeereBestandsdaten(t, pool)
	srv := &Tueren{Basis: &tuer.Basis{DB: &db.Verbindung{Pool: pool}}}
	bestellungID, positionID := bestellungMitPosition(t, srv, pool, "Preis-Pruefung")
	_, fremdePosition := bestellungMitPosition(t, srv, pool, "Preis-Fremd")

	for _, f := range []struct {
		name, position, rumpf string
		status                int
	}{
		{"ohne Preis", positionID, `{}`, http.StatusBadRequest},
		{"negativ", positionID, `{"einzelpreis":-1}`, http.StatusBadRequest},
		{"zu hoch", positionID, `{"einzelpreis":100000}`, http.StatusBadRequest},
		{"Position einer anderen Bestellung", fremdePosition, `{"einzelpreis":5}`, http.StatusNotFound},
	} {
		if rec := aenderePreisUeberTuer(t, srv, pool, bestellungID, f.position, f.rumpf); rec.Code != f.status {
			t.Errorf("%s: Status %d, erwartet %d (%s)", f.name, rec.Code, f.status, rec.Body.String())
		}
	}
	if einzel, exemplare, summe := preiseInDB(t, pool, bestellungID, positionID); einzel != "9.50" ||
		strings.Join(exemplare, ",") != "9.50,9.50" || summe != "19.00" {
		t.Errorf("nach den Ablehnungen: Position %s, Exemplare %v, Summe %s; erwartet unverändert", einzel, exemplare, summe)
	}
	if n := zaehlePreisProtokoll(t, pool, positionID) + zaehlePreisProtokoll(t, pool, fremdePosition); n != 0 {
		t.Errorf("%d Protokolleinträge, erwartet keinen", n)
	}
}

// Auch ein geänderter Preis hält die Summe der Bestellung in ihrer Spalte (NUMERIC(10,2)): Bis an
// die Grenze geht es, darüber kommt der Satz statt eines Serverfehlers, und nichts ändert sich.
func TestPositionspreis_SummeUeberDerGrenze(t *testing.T) {
	pool := pgtest.Pool(t)
	testdaten.LeereBestandsdaten(t, pool)
	srv := &Tueren{Basis: &tuer.Basis{DB: &db.Verbindung{Pool: pool}}}
	bestellungID, positionID := bestellungMitPosition(t, srv, pool, "Preis-Summe")
	// Fünf Positionen zu 200 × 99.999,99 € ergeben 99.999.990,00 €; die Position der Bestellung
	// (2 Stück) darf danach höchstens 4,99 € kosten.
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO bestellungen_positionen (bestellung_id, titel_name, menge, einzelpreis)
		SELECT $1, 'Füllposten', 200, 99999.99 FROM generate_series(1, 5)`, bestellungID); err != nil {
		t.Fatalf("Füllposten anlegen: %v", err)
	}

	if rec := aenderePreisUeberTuer(t, srv, pool, bestellungID, positionID, `{"einzelpreis":4.99}`); rec.Code != http.StatusOK {
		t.Fatalf("Preis an der Grenze: %d %s", rec.Code, rec.Body.String())
	}
	rec := aenderePreisUeberTuer(t, srv, pool, bestellungID, positionID, `{"einzelpreis":5}`)
	if rec.Code != http.StatusBadRequest || !strings.Contains(rec.Body.String(), "99.999.999,99") {
		t.Errorf("Preis über der Grenze: Status %d, %s, erwartet 400 mit der Grenze der Summe", rec.Code, rec.Body.String())
	}
	if einzel, exemplare, summe := preiseInDB(t, pool, bestellungID, positionID); einzel != "4.99" ||
		strings.Join(exemplare, ",") != "4.99,4.99" || summe != "99999999.98" {
		t.Errorf("nachher: Position %s, Exemplare %v, Summe %s; erwartet 4.99, zweimal 4.99, 99999999.98", einzel, exemplare, summe)
	}
}

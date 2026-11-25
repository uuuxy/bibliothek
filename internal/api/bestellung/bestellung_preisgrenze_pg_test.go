package bestellung

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"bibliothek/internal/api/tuer"
	"bibliothek/internal/db"
	"bibliothek/internal/pgtest"
	"bibliothek/internal/pkg/mitteltopf"
	"bibliothek/internal/repository"
	"bibliothek/internal/service"
	"bibliothek/internal/testdaten"
)

// Der Preis einer Position hat beim Anlegen dieselben Grenzen wie beim Ändern
// (PUT …/einzelpreis). Ohne sie nahm die Bestellung einen negativen Preis und jeden Betrag an.
func TestBestellungAnlegen_PreisAusserhalbDerGrenzen(t *testing.T) {
	pool := pgtest.Pool(t)
	testdaten.LeereBestandsdaten(t, pool)
	srv := &Tueren{Basis: &tuer.Basis{DB: &db.Verbindung{Pool: pool}}}
	lieferant := testdaten.LegeLieferantAn(t, pool, "Preisgrenze", false)
	titel := testdaten.LegeTitelAn(t, pool, "LMF-Preisgrenze")

	for _, preis := range []string{"-0.01", "100000"} {
		rumpf := fmt.Sprintf(`{"supplier_id":%q,"mittel":%q,"items":[{"titel_id":%q,"menge":2,"preis":%s,"generate_barcodes":false}]}`,
			lieferant, mitteltopf.Land, titel, preis)
		req := httptest.NewRequest(http.MethodPost, "/api/bestellungen", strings.NewReader(rumpf))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		srv.BestellungAnlegen(service.NewBestellDienst(srv.DB, repository.NewKatalogAbfragen(srv.DB.Pool)))(rec, req)
		if rec.Code != http.StatusBadRequest || !strings.Contains(rec.Body.String(), "99.999,99") {
			t.Errorf("Preis %s: Status %d, %s, erwartet 400 mit den Grenzen", preis, rec.Code, rec.Body.String())
		}
	}
	if n := pgtest.ZaehleZeilen(t, pool, `SELECT count(*) FROM bestellungen_verlauf WHERE lieferant_id = $1`, lieferant); n != 0 {
		t.Errorf("%d Bestellung(en) trotz Preis außerhalb der Grenzen angelegt", n)
	}
}

// Die Summe einer Bestellung fasst höchstens 99.999.999,99 € (NUMERIC(10,2)). Sechs Positionen zu
// 200 Stück zum höchsten Einzelpreis endeten mit einem Serverfehler statt mit dem Satz.
func TestBestellungAnlegen_SummeUeberDerGrenze(t *testing.T) {
	pool := pgtest.Pool(t)
	testdaten.LeereBestandsdaten(t, pool)
	srv := &Tueren{Basis: &tuer.Basis{DB: &db.Verbindung{Pool: pool}}}
	lieferant := testdaten.LegeLieferantAn(t, pool, "Summengrenze", false)
	var positionen []string
	for i := range 6 {
		titel := testdaten.LegeTitelAn(t, pool, fmt.Sprintf("LMF-Summengrenze-%d", i))
		positionen = append(positionen,
			fmt.Sprintf(`{"titel_id":%q,"menge":200,"preis":99999.99,"generate_barcodes":false}`, titel))
	}
	rumpf := fmt.Sprintf(`{"supplier_id":%q,"mittel":%q,"items":[%s]}`,
		lieferant, mitteltopf.Land, strings.Join(positionen, ","))
	req := httptest.NewRequest(http.MethodPost, "/api/bestellungen", strings.NewReader(rumpf))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	srv.BestellungAnlegen(service.NewBestellDienst(srv.DB, repository.NewKatalogAbfragen(srv.DB.Pool)))(rec, req)

	if rec.Code != http.StatusBadRequest || !strings.Contains(rec.Body.String(), "99.999.999,99") {
		t.Errorf("Status %d, %s, erwartet 400 mit der Grenze der Summe", rec.Code, rec.Body.String())
	}
	if n := pgtest.ZaehleZeilen(t, pool, `SELECT count(*) FROM bestellungen_verlauf WHERE lieferant_id = $1`, lieferant); n != 0 {
		t.Errorf("%d Bestellung(en) trotz Summe über der Grenze angelegt", n)
	}
}

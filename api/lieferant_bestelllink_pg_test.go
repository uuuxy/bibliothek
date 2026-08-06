package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"bibliothek/db"
)

// Den Bestelllink bekommt höchstens EINER.
//
// Über diesen Link wählt der Händler die Etikettengröße und bestätigt die Bestellung;
// die Bestätigung landet automatisch in bestellungen_verlauf. Trügen zwei Händler den
// Link, bekämen beide ihn zur selben Bestellung — wer zuerst bestätigt, gewinnt, der
// zweite läuft in ein 409. Nichts daran wäre sichtbar, bis es passiert.
//
// Geprüft wird über die HTTP-Handler, nicht über den Setzer allein: Die Falle liegt
// genau dort, wo der Handler die Spalte früher direkt ins UPDATE schrieb. Mit dem
// Teil-Index (Migration 065) hätte das beim zweiten Händler einen 500er geworfen —
// ein Fehler, den ein Test auf den Setzer niemals sieht.
func TestNurEinLieferantMitBestelllink(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	ctx := context.Background()
	srv := &Server{DB: &db.Database{Pool: pool}}

	anlegen := func(name string, mitLink bool) string {
		t.Helper()
		body := `{"name":"` + name + `","email":"` + name + `@test.invalid","customerNumber":"K-` + name +
			`","bietet_bestellbestaetigung":` + map[bool]string{true: "true", false: "false"}[mitLink] + `}`
		rec := httptest.NewRecorder()
		srv.CreateSupplierHandler()(rec, httptest.NewRequest(http.MethodPost, "/api/lieferanten", strings.NewReader(body)))
		if rec.Code != http.StatusCreated {
			t.Fatalf("Lieferant %s anlegen: Status %d — %s", name, rec.Code, rec.Body.String())
		}
		// ID aus der Antwort, NICHT über den Namen nachschlagen: resetBestandsdaten leert
		// die Lieferanten nicht, und andere Tests im selben Paketlauf legen Händler mit
		// denselben Namen an — die Suche träfe dann den älteren.
		var antwort SupplierResponse
		if err := json.Unmarshal(rec.Body.Bytes(), &antwort); err != nil {
			t.Fatalf("Antwort auf das Anlegen von %s lesen: %v", name, err)
		}
		return antwort.ID
	}

	aendern := func(id, name string, mitLink bool) *httptest.ResponseRecorder {
		t.Helper()
		body := `{"name":"` + name + `","email":"` + name + `@test.invalid","customerNumber":"K-` + name +
			`","bietet_bestellbestaetigung":` + map[bool]string{true: "true", false: "false"}[mitLink] + `}`
		req := httptest.NewRequest(http.MethodPut, "/api/lieferanten/x", strings.NewReader(body))
		req.SetPathValue("id", id)
		rec := httptest.NewRecorder()
		srv.UpdateSupplierHandler()(rec, req)
		return rec
	}

	traeger := func() (anzahl int, id string) {
		t.Helper()
		if err := pool.QueryRow(ctx,
			`SELECT count(*), coalesce(max(id::text), '') FROM lieferanten WHERE bietet_bestellbestaetigung`,
		).Scan(&anzahl, &id); err != nil {
			t.Fatalf("Träger des Bestelllinks lesen: %v", err)
		}
		return anzahl, id
	}

	ersterID := anlegen("Naacher", true)
	zweiterID := anlegen("Cornelsen", false)

	if anzahl, id := traeger(); anzahl != 1 || id != ersterID {
		t.Fatalf("nach dem Anlegen: %d Träger (%s), erwartet genau der erste (%s)", anzahl, id, ersterID)
	}

	// 1. Ein ZWEITER Link darf an der Datenbank gar nicht erst entstehen.
	if _, err := pool.Exec(ctx,
		`UPDATE lieferanten SET bietet_bestellbestaetigung = true WHERE id = $1`, zweiterID); err == nil {
		t.Fatal("zwei Lieferanten mit Bestelllink müssen an idx_lieferanten_ein_bestelllink scheitern")
	}

	// 2. Über die Oberfläche wandert der Link — ohne 500, weil der Handler erst räumt.
	if rec := aendern(zweiterID, "Cornelsen", true); rec.Code != http.StatusOK {
		t.Fatalf("Bestelllink umhängen: Status %d — %s\n"+
			"→ Genau so äußert sich ein Handler, der die Spalte direkt ins UPDATE schreibt: "+
			"Der Teil-Index bricht ab, sobald ein anderer den Link schon trägt.",
			rec.Code, rec.Body.String())
	}
	if anzahl, id := traeger(); anzahl != 1 || id != zweiterID {
		t.Fatalf("nach dem Umhängen: %d Träger (%s), erwartet genau der zweite (%s)", anzahl, id, zweiterID)
	}

	// 3. Wiederholbar — hier fällt eine Umsetzung auf, die erst setzt und dann räumt.
	if rec := aendern(ersterID, "Naacher", true); rec.Code != http.StatusOK {
		t.Fatalf("zweites Umhängen: Status %d — %s", rec.Code, rec.Body.String())
	}
	if anzahl, _ := traeger(); anzahl != 1 {
		t.Fatalf("%d Träger nach dem zweiten Umhängen, erwartet genau 1", anzahl)
	}

	// 4. Abschalten muss gehen: Wer aufhört, über diesen Händler zu bestellen, muss den
	//    Link loswerden können, ohne ihn erst jemand anderem zu geben. (Anders als beim
	//    Standardlieferanten — „kein Bestelllink" ist ein normaler Betriebszustand.)
	if rec := aendern(ersterID, "Naacher", false); rec.Code != http.StatusOK {
		t.Fatalf("Bestelllink abschalten: Status %d — %s", rec.Code, rec.Body.String())
	}
	if anzahl, _ := traeger(); anzahl != 0 {
		t.Fatalf("%d Träger nach dem Abschalten, erwartet 0 — der Link liesse sich sonst nie wieder loswerden", anzahl)
	}
}

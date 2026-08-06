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

// Hauptlieferant darf höchstens EINER sein.
//
// An diesem einen Merkmal hängen drei Dinge, die vorher drei getrennte Schalter waren
// (Migration 066): Vorauswahl im Bestellformular, der Bestelllink (Etikettengröße +
// Bestätigung) und ob die Exemplare als etikettiert gelten. Zwei Hauptlieferanten wären
// ein stiller Fehler — beide bekämen den Link zur selben Bestellung, wer zuerst bestätigt
// gewinnt, und bei der Vorauswahl entschiede die Sortierung.
//
// Geprüft wird über die HTTP-Handler, nicht über den Setzer allein: Die Falle liegt genau
// dort, wo der Handler das Merkmal direkt ins INSERT/UPDATE schreiben würde. Mit dem
// Teil-Index wirft das einen 500er, sobald ein anderer es schon trägt — ein Fehler, den
// ein Test auf den Setzer niemals sieht.
func TestNurEinHauptlieferant(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	ctx := context.Background()
	srv := &Server{DB: &db.Database{Pool: pool}}

	nutzlast := func(name string, haupt bool) string {
		return `{"name":"` + name + `","email":"` + name + `@test.invalid","customerNumber":"K-` + name +
			`","ist_hauptlieferant":` + map[bool]string{true: "true", false: "false"}[haupt] + `}`
	}

	anlegen := func(name string, haupt bool) string {
		t.Helper()
		rec := httptest.NewRecorder()
		srv.CreateSupplierHandler()(rec,
			httptest.NewRequest(http.MethodPost, "/api/lieferanten", strings.NewReader(nutzlast(name, haupt))))
		if rec.Code != http.StatusCreated {
			t.Fatalf("Lieferant %s anlegen: Status %d — %s", name, rec.Code, rec.Body.String())
		}
		// ID aus der Antwort, NICHT über den Namen nachschlagen: resetBestandsdaten leert die
		// Lieferanten nicht, und andere Tests im selben Paketlauf legen Händler mit denselben
		// Namen an — die Suche träfe dann den älteren.
		var antwort SupplierResponse
		if err := json.Unmarshal(rec.Body.Bytes(), &antwort); err != nil {
			t.Fatalf("Antwort auf das Anlegen von %s lesen: %v", name, err)
		}
		return antwort.ID
	}

	aendern := func(id, name string, haupt bool) *httptest.ResponseRecorder {
		t.Helper()
		req := httptest.NewRequest(http.MethodPut, "/api/lieferanten/x", strings.NewReader(nutzlast(name, haupt)))
		req.SetPathValue("id", id)
		rec := httptest.NewRecorder()
		srv.UpdateSupplierHandler()(rec, req)
		return rec
	}

	traeger := func() (anzahl int, id string) {
		t.Helper()
		if err := pool.QueryRow(ctx,
			`SELECT count(*), coalesce(max(id::text), '') FROM lieferanten WHERE ist_hauptlieferant`,
		).Scan(&anzahl, &id); err != nil {
			t.Fatalf("Hauptlieferanten lesen: %v", err)
		}
		return anzahl, id
	}

	naacher := anlegen("Naacher", true)
	cornelsen := anlegen("Cornelsen", false)

	if anzahl, id := traeger(); anzahl != 1 || id != naacher {
		t.Fatalf("nach dem Anlegen: %d Hauptlieferanten (%s), erwartet genau Naacher (%s)", anzahl, id, naacher)
	}

	// 1. Ein ZWEITER darf an der Datenbank gar nicht erst entstehen.
	if _, err := pool.Exec(ctx,
		`UPDATE lieferanten SET ist_hauptlieferant = true WHERE id = $1`, cornelsen); err == nil {
		t.Fatal("zwei Hauptlieferanten müssen an idx_lieferanten_ein_hauptlieferant scheitern")
	}

	// 2. Über die Oberfläche wandert die Rolle — ohne 500, weil der Handler erst räumt.
	if rec := aendern(cornelsen, "Cornelsen", true); rec.Code != http.StatusOK {
		t.Fatalf("Hauptlieferant umhängen: Status %d — %s\n"+
			"→ Genau so äußert sich ein Handler, der das Merkmal direkt ins UPDATE schreibt: "+
			"Der Teil-Index bricht ab, sobald ein anderer es schon trägt.",
			rec.Code, rec.Body.String())
	}
	if anzahl, id := traeger(); anzahl != 1 || id != cornelsen {
		t.Fatalf("nach dem Umhängen: %d Hauptlieferanten (%s), erwartet Cornelsen (%s)", anzahl, id, cornelsen)
	}

	// 3. Wiederholbar — hier fällt eine Umsetzung auf, die erst setzt und dann räumt.
	if rec := aendern(naacher, "Naacher", true); rec.Code != http.StatusOK {
		t.Fatalf("zweites Umhängen: Status %d — %s", rec.Code, rec.Body.String())
	}
	if anzahl, _ := traeger(); anzahl != 1 {
		t.Fatalf("%d Hauptlieferanten nach dem zweiten Umhängen, erwartet genau 1", anzahl)
	}

	// 4. Abschalten muss gehen: Wer aufhört, über diesen Händler zu bestellen, muss ihn
	//    zurückstufen können, ohne die Rolle erst jemand anderem zu geben.
	if rec := aendern(naacher, "Naacher", false); rec.Code != http.StatusOK {
		t.Fatalf("Hauptlieferant abschalten: Status %d — %s", rec.Code, rec.Body.String())
	}
	if anzahl, _ := traeger(); anzahl != 0 {
		t.Fatalf("%d Hauptlieferanten nach dem Abschalten, erwartet 0 — die Rolle liesse sich sonst nie wieder loswerden", anzahl)
	}
}

// Der Hauptlieferant steht in der Liste OBEN.
//
// Das Bestellformular wählt den ersten Eintrag vor. Käme die Liste alphabetisch, gewänne
// ein beliebiger Händler und die Vorauswahl wäre wirkungslos — ein Schalter, der nichts
// tut, ist schlimmer als keiner, weil man sich auf ihn verlässt.
func TestHauptlieferantStehtInDerListeOben(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	ctx := context.Background()
	srv := &Server{DB: &db.Database{Pool: pool}}

	// Absichtlich so benannt, dass der Hauptlieferant alphabetisch HINTEN stünde.
	haendler(t, pool, "Aaa-Cornelsen", false)
	haendler(t, pool, "Zzz-Naacher", true)

	rec := httptest.NewRecorder()
	srv.ListSuppliersHandler()(rec, httptest.NewRequest(http.MethodGet, "/api/lieferanten", nil).WithContext(ctx))
	if rec.Code != http.StatusOK {
		t.Fatalf("Liste laden: Status %d — %s", rec.Code, rec.Body.String())
	}

	var liste []SupplierResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &liste); err != nil {
		t.Fatalf("Liste lesen: %v", err)
	}
	if len(liste) < 2 {
		t.Fatalf("%d Lieferanten in der Liste, erwartet mindestens 2", len(liste))
	}
	if !liste[0].IstHauptlieferant {
		t.Errorf("erster Eintrag ist %q (Hauptlieferant: %v) — erwartet den Hauptlieferanten.\n"+
			"→ Das Bestellformular wählt den ersten vor; steht dort der falsche, geht die "+
			"Bestellung an den falschen Händler raus.", liste[0].Name, liste[0].IstHauptlieferant)
	}
}

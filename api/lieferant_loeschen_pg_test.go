package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"bibliothek/db"
)

// Der Hauptlieferant lässt sich nicht nebenbei wegklicken.
//
// Gefunden am 07.09.2026 beim Befragen des Fremdschlüssels
// `bestellungen_verlauf.lieferant_id -> lieferanten` (Frage 12, Gegenrichtung Schema).
// Der Fremdschlüssel selbst war die harmlose Hälfte: ON DELETE SET NULL, und die
// Bestellung hält Name und E-Mail des Händlers als eigene Abschrift — die Historie
// überlebt das Löschen unbeschadet.
//
// Der LÖSCHWEG daneben war das Problem. „Löschen" in der Lieferantenverwaltung fragt
// nicht nach (ein Klick, kein Dialog), und getroffen werden konnte auch der EINE
// Händler, an dem der ganze Bestellweg hängt: Bestellmail, Bestätigungs-Link und die
// Etiketten-Entscheidung „der Händler beklebt selbst". Danach hatte die Schule keinen
// Hauptlieferanten mehr — ohne Meldung, ohne Spur, die Liste zeigte nur einen Händler
// weniger. Dieselbe Klasse wie „Feature hängt an ungesetzter Einstellung", nur dass hier
// jemand den Schalter aktiv umlegt, ohne es zu merken.
//
// Die Regel sitzt im Handler, nicht im Formular: Die Verwaltung ist nicht die einzige
// Tür, und ein Hinweis, den nur ein Bildschirm kennt, ist keine Regel.
func TestHauptlieferantLoeschenWirdAbgewiesen(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	srv := &Server{DB: &db.Database{Pool: pool}}

	anlegen := func(name string, haupt bool) string {
		t.Helper()
		rumpf := `{"name":"` + name + `","email":"` + name + `@test.invalid","customerNumber":"K-` + name +
			`","ist_hauptlieferant":` + map[bool]string{true: "true", false: "false"}[haupt] + `}`
		rec := httptest.NewRecorder()
		srv.CreateSupplierHandler()(rec,
			httptest.NewRequest(http.MethodPost, "/api/lieferanten", strings.NewReader(rumpf)))
		if rec.Code != http.StatusCreated {
			t.Fatalf("Lieferant %s anlegen: Status %d — %s", name, rec.Code, rec.Body.String())
		}
		var antwort SupplierResponse
		if err := json.Unmarshal(rec.Body.Bytes(), &antwort); err != nil {
			t.Fatalf("Antwort lesen: %v", err)
		}
		return antwort.ID
	}

	loeschen := func(id string) *httptest.ResponseRecorder {
		t.Helper()
		rec := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodDelete, "/api/lieferanten/"+id, nil)
		r.SetPathValue("id", id)
		srv.DeleteSupplierHandler()(rec, r)
		return rec
	}

	haupt := anlegen("HauptLoesch", true)
	neben := anlegen("NebenLoesch", false)

	// 1. Der Hauptlieferant bleibt stehen — und die Meldung sagt, was zu tun ist.
	rec := loeschen(haupt)
	if rec.Code != http.StatusConflict {
		t.Errorf("Hauptlieferant gelöscht: Status %d, erwartet 409 — der Bestellweg hinge danach an nichts", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "Hauptlieferant") {
		t.Errorf("Meldung nennt den Grund nicht: %s", rec.Body.String())
	}
	var nochDa bool
	if err := pool.QueryRow(t.Context(),
		`SELECT EXISTS (SELECT 1 FROM lieferanten WHERE id = $1)`, haupt).Scan(&nochDa); err != nil {
		t.Fatal(err)
	}
	if !nochDa {
		t.Error("der Hauptlieferant ist trotz 409 aus der Tabelle verschwunden")
	}

	// 2. Gegenprobe: Ein gewöhnlicher Händler lässt sich weiterhin löschen. Ohne diese
	//    Hälfte wäre nicht zu unterscheiden, ob die Regel greift oder das Löschen kaputt ist.
	if rec := loeschen(neben); rec.Code != http.StatusNoContent {
		t.Errorf("gewöhnlicher Händler nicht löschbar: Status %d — %s", rec.Code, rec.Body.String())
	}

	// 3. Der Weg bleibt offen: Schalter abwählen, dann löschen.
	rumpf := `{"name":"HauptLoesch","email":"h@test.invalid","customerNumber":"K-1","ist_hauptlieferant":false}`
	up := httptest.NewRequest(http.MethodPut, "/api/lieferanten/"+haupt, strings.NewReader(rumpf))
	up.SetPathValue("id", haupt)
	upRec := httptest.NewRecorder()
	srv.UpdateSupplierHandler()(upRec, up)
	if upRec.Code != http.StatusOK {
		t.Fatalf("Schalter abwählen: Status %d — %s", upRec.Code, upRec.Body.String())
	}
	if rec := loeschen(haupt); rec.Code != http.StatusNoContent {
		t.Errorf("nach dem Abwählen immer noch nicht löschbar: Status %d — %s", rec.Code, rec.Body.String())
	}
}

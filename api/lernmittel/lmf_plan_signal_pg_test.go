package lernmittel

import (
	"context"
	"net/http"
	"testing"

	"bibliothek/api/tuer"
	"bibliothek/db"
	"bibliothek/internal/pgtest"
	"bibliothek/internal/testdaten"
	"bibliothek/internal/tuertest"
)

// Das Live-Signal des LMF-Plans folgt dem Schreiben: Es kommt nach jedem Speichern,
// Veröffentlichen und Verwerfen und bleibt aus, wo nichts geschrieben wurde (Vorschau,
// Ablehnung, ein zweites Veröffentlichen). Geprüft am Strom: Nach jedem Schritt geht eine
// Marke über die Leitung; was vor ihr ankommt, hat der Schritt gesendet.
func TestLMFPlan_SignalNurNachDemSchreiben(t *testing.T) {
	pool := pgtest.Pool(t)
	testdaten.LeereBestandsdaten(t, pool)
	raeumePlaene := func() {
		if _, err := pool.Exec(context.Background(), `DELETE FROM lmf_plaene`); err != nil {
			t.Logf("Pläne aufräumen: %v", err)
		}
	}
	raeumePlaene()
	t.Cleanup(raeumePlaene)
	broker, signale := tuertest.BeobachteLeitung(t)

	srv := &Tueren{Basis: &tuer.Basis{DB: &db.Verbindung{Pool: pool}, Broker: broker}}
	testdaten.LegeSchuelerAn(t, pool, "SIG-1", "Anna", "9H1")
	rahmen := `"letzte_stunde":4,"stunden_je_tag":6,"zeilen":[{"klassen":["9H1"]}]`
	plan := `{"letzter_tag":"2027-06-28",` + rahmen + `}`
	korrektur := `{"letzter_tag":"2027-06-30",` + rahmen + `}`
	vorschau := `{"letzter_tag":"2027-06-28",` + rahmen + `,"vorschau":true}`

	for _, s := range []struct {
		name, methode, art, body string
		status, signale          int
	}{
		{"unbekannte Art", http.MethodPut, "quatsch", plan, http.StatusBadRequest, 0},
		{"Vorschau", http.MethodPut, "rueckgabe", vorschau, http.StatusOK, 0},
		{"Verwerfen ohne Plan", http.MethodDelete, "rueckgabe", "", http.StatusNotFound, 0},
		{"Veröffentlichen ohne Plan", http.MethodPost, "rueckgabe", "", http.StatusNotFound, 0},
		{"Entwurf speichern", http.MethodPut, "rueckgabe", plan, http.StatusOK, 1},
		{"Veröffentlichen", http.MethodPost, "rueckgabe", "", http.StatusOK, 1},
		{"schon veröffentlicht", http.MethodPost, "rueckgabe", "", http.StatusOK, 0},
		{"veröffentlichten Plan korrigieren", http.MethodPut, "rueckgabe", korrektur, http.StatusOK, 1},
		{"Verwerfen", http.MethodDelete, "rueckgabe", "", http.StatusOK, 1},
	} {
		rec := lmfPlanAufruf(t, srv, s.methode, s.art, s.body)
		if rec.Code != s.status {
			t.Fatalf("%s: Status %d, erwartet %d: %s", s.name, rec.Code, s.status, rec.Body.String())
		}
		if n := signale("lmf-plan"); n != s.signale {
			t.Errorf("%s: %d Signale, erwartet %d", s.name, n, s.signale)
		}
	}
}

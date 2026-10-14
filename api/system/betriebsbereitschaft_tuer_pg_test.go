package system

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"bibliothek/api/tuer"
	"bibliothek/db"
	"bibliothek/internal/bereitschaft"
	"bibliothek/internal/pgtest"
	"bibliothek/internal/testdaten"
	"bibliothek/repository"
)

// Die Seite „Betriebsbereitschaft" über ihre Tür: Die Antwort nennt die Befunde und als Gesamt
// ihre schärfste Stufe. Die Oberfläche färbt danach den Kopf der Seite; stünde dort „ok" neben
// einem kritischen Befund, sähe die Seite auf den ersten Blick in Ordnung aus.
func TestBetriebsbereitschaft_GesamtIstDieSchaerfsteStufeDerBefunde(t *testing.T) {
	pool := pgtest.Pool(t)
	testdaten.LeereBestandsdaten(t, pool)
	srv := &Tueren{Basis: &tuer.Basis{DB: &db.Database{Pool: pool}}}
	rec := httptest.NewRecorder()
	srv.Betriebsbereitschaft(repository.NewEinstellungsAbfragen(pool),
		repository.NewMailEinstellungsAbfragen(pool), repository.NewBetriebszustandAbfragen(pool)).
		ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/admin/system/betriebsbereitschaft", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("Status %d: %s", rec.Code, rec.Body.String())
	}
	var antwort betriebsbereitschaftAntwort
	if err := json.Unmarshal(rec.Body.Bytes(), &antwort); err != nil {
		t.Fatal(err)
	}
	if len(antwort.Befunde) < 15 {
		t.Fatalf("nur %d Befunde in der Antwort, die Selbstprüfung hat rund zwanzig Bereiche", len(antwort.Befunde))
	}
	// Selbst gezählt, nicht über die geprüfte Funktion.
	stufen := map[string]int{}
	for _, b := range antwort.Befunde {
		stufen[b.Stufe]++
	}
	will := bereitschaft.StufeOK
	switch {
	case stufen[bereitschaft.StufeKritisch] > 0:
		will = bereitschaft.StufeKritisch
	case stufen[bereitschaft.StufeWarnung] > 0:
		will = bereitschaft.StufeWarnung
	}
	// Die Testumgebung ist kein eingerichteter Betrieb: Mindestens ein Bereich meldet etwas.
	// Wäre alles in Ordnung, träfe ein festes „ok" die Erwartung, und der Test mäße nichts.
	if will == bereitschaft.StufeOK {
		t.Fatalf("kein Befund über „ok“ in der Testumgebung (%v): der Test misst so nichts", stufen)
	}
	if antwort.Gesamt != will {
		t.Errorf("Gesamt %q bei den Stufen %v, erwartet %q", antwort.Gesamt, stufen, will)
	}
}

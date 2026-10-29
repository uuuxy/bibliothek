package benutzer

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"bibliothek/api/tuer"
	"bibliothek/db"
	"bibliothek/internal/pgtest"
	"bibliothek/internal/testdaten"
	"bibliothek/repository"
)

// Die Kontenliste trägt, was die Tabelle der Konten zeigt: die Ausweisnummer aus der
// Leserzeile, „aktiv", den offenen Antrag der Selbstanmeldung, an dem die Tabelle „wartet" von
// „deaktiviert" unterscheidet, und die Leserzeile, von der aus die Zugangsanfragen nach einem
// gleichnamigen Leser suchen.
func TestKonten_ZeigenDenStandJedesKontos(t *testing.T) {
	pool := pgtest.Pool(t)
	testdaten.LeereBestandsdaten(t, pool)
	ctx := context.Background()
	srv := &Tueren{Basis: &tuer.Basis{DB: &db.Verbindung{Pool: pool}}}

	aktiv := testdaten.LegeKontoAn(t, pool, "Anna", "Kontenliste", "anna@kontenliste.invalid", "mitarbeiter", true)
	wartet := testdaten.LegeKontoAn(t, pool, "Wim", "Kontenliste", "wim@kontenliste.invalid", "kollegium", false)
	aus := testdaten.LegeKontoAn(t, pool, "Dora", "Kontenliste", "dora@kontenliste.invalid", "kollegium", false)
	var leserAktiv string
	if err := pool.QueryRow(ctx, `
		UPDATE leser SET barcode_id = 'KL-4711' WHERE id = (SELECT leser_id FROM benutzer WHERE id = $1)
		RETURNING id::text`, aktiv).Scan(&leserAktiv); err != nil {
		t.Fatalf("Ausweisnummer setzen: %v", err)
	}
	if _, err := pool.Exec(ctx, `UPDATE benutzer SET zugang_beantragt_am = now() WHERE id = $1`, wartet); err != nil {
		t.Fatalf("Antrag setzen: %v", err)
	}

	rec := httptest.NewRecorder()
	srv.Konten(repository.NewKontoAbfragen(pool)).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/benutzer", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("Status %d, erwartet 200: %s", rec.Code, rec.Body.String())
	}
	var konten []kontoAntwort
	if err := json.Unmarshal(rec.Body.Bytes(), &konten); err != nil {
		t.Fatalf("Antwort lesen: %v", err)
	}
	jeKennung := map[string]kontoAntwort{}
	for _, k := range konten {
		jeKennung[k.ID] = k
	}

	if k := jeKennung[aktiv]; !k.Aktiv || k.BarcodeID != "KL-4711" || k.LeserID != leserAktiv || k.ZugangBeantragtAm != nil {
		t.Errorf("aktives Konto: aktiv %t, Ausweis %q, Leserzeile %q, Antrag %v; erwartet aktiv, KL-4711, %s, kein Antrag",
			k.Aktiv, k.BarcodeID, k.LeserID, k.ZugangBeantragtAm, leserAktiv)
	}
	if k := jeKennung[wartet]; k.Aktiv || k.ZugangBeantragtAm == nil {
		t.Errorf("Konto mit offenem Antrag: aktiv %t, Antrag %v; erwartet nicht aktiv, mit Antrag", k.Aktiv, k.ZugangBeantragtAm)
	}
	if k := jeKennung[aus]; k.Aktiv || k.ZugangBeantragtAm != nil {
		t.Errorf("deaktiviertes Konto: aktiv %t, Antrag %v; erwartet nicht aktiv, ohne Antrag", k.Aktiv, k.ZugangBeantragtAm)
	}
}

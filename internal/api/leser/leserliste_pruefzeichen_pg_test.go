package leser

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"bibliothek/internal/api/tuer"
	"bibliothek/internal/auth"
	"bibliothek/internal/db"
	"bibliothek/internal/pgtest"
	"bibliothek/internal/pkg/code39"
	"bibliothek/internal/repository"
)

// Ein alter Ausweis trägt im Strichcode ein Prüfzeichen von Code 39. Leserdatei und
// Schülersuche beim Vormerken gehen durch GET /api/schueler; ein Scan davon findet den Leser wie
// an der Theke, als zweiter Versuch, wenn der Scan, wie er kam, niemanden findet.
func TestLeserliste_FindetAltenAusweisMitPruefzeichen(t *testing.T) {
	pool := pgtest.Pool(t)
	ctx := context.Background()
	srv := &Tueren{Basis: &tuer.Basis{DB: &db.Verbindung{Pool: pool}}}
	claims := &auth.Claims{KontoID: "00000000-0000-0000-0000-00000000a128", Rolle: auth.RolleAdmin}

	ausweis := fmt.Sprintf("A-PZ%d", time.Now().UnixNano()%1000000)
	zeichen, ok := code39.Pruefzeichen(ausweis)
	if !ok {
		t.Fatalf("kein Prüfzeichen zu %q", ausweis)
	}
	t.Cleanup(func() {
		if _, err := pool.Exec(ctx, `DELETE FROM leser WHERE barcode_id = $1`, ausweis); err != nil {
			t.Errorf("aufräumen: %v", err)
		}
	})
	if _, err := pool.Exec(ctx, `INSERT INTO leser (barcode_id, vorname, nachname, klasse, abgaenger_jahr, art)
		VALUES ($1, 'Pia', 'Pruefzeichen', '05F1', 2032, 'schueler')`, ausweis); err != nil {
		t.Fatalf("Schüler anlegen: %v", err)
	}

	scan := url.QueryEscape(ausweis + string(zeichen))
	for _, adresse := range []string{"/api/schueler?q=" + scan, "/api/schueler?art=alle&q=" + scan} {
		req := httptest.NewRequest(http.MethodGet, adresse, nil)
		req = req.WithContext(context.WithValue(req.Context(), auth.ClaimsSchluessel, claims))
		rec := httptest.NewRecorder()
		srv.Leserliste(repository.NewLeserAbfragen(pool), repository.NewLMFAbfragen(pool)).ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("GET %s: Status %d, %s", adresse, rec.Code, rec.Body.String())
		}
		var zeilen []repository.LeserListenZeile
		if err := json.Unmarshal(rec.Body.Bytes(), &zeilen); err != nil {
			t.Fatalf("Antwort lesen: %v", err)
		}
		if len(zeilen) != 1 || zeilen[0].BarcodeID != ausweis {
			t.Errorf("GET %s: %+v, erwartet den Ausweis %s", adresse, zeilen, ausweis)
		}
	}
}

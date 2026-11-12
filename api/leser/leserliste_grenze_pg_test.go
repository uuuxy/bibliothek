package leser

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"testing"

	"bibliothek/api/tuer"
	"bibliothek/auth"
	"bibliothek/db"
	"bibliothek/internal/pgtest"
	"bibliothek/repository"
)

// TestLeserliste_NenntSeineGrenze belegt, dass die Leserliste ohne Suchtext bei
// repository.LeserListeGrenze gekappt ist und die Antwort genau diese Zahl im Kopf
// tuer.HeaderGrenze nennt, auch mit Klassenfilter und in der Leserdatei. Mit Suchtext gibt es
// weder Grenze noch Kopf. Aus dem Kopf liest die Schülerdatei, ob sie „Die ersten 500 Leser"
// sagt (schuelerSuche.svelte.js); führte sie die Zahl selbst, liefen beide auseinander, sobald
// sich eine ändert.
//
// Geprüft an einer echten Datenbank mit mehr Zeilen als die Grenze: Gegen eine kleine Tabelle
// wäre der Test auch ohne LIMIT grün.
func TestLeserliste_NenntSeineGrenze(t *testing.T) {
	pool := pgtest.Pool(t)
	ctx := context.Background()
	srv := &Tueren{Basis: &tuer.Basis{DB: &db.Verbindung{Pool: pool}}}
	leserAbfragen := repository.NewLeserAbfragen(pool)
	claims := &auth.Claims{KontoID: "00000000-0000-0000-0000-00000000a127", Rolle: auth.RolleAdmin}

	grenze := repository.LeserListeGrenze("")
	if grenze <= 0 {
		t.Fatalf("LeserListeGrenze(\"\") = %d, die Liste ohne Suchtext braucht eine Grenze", grenze)
	}

	t.Cleanup(func() {
		if _, err := pool.Exec(ctx, `DELETE FROM leser WHERE nachname LIKE 'Grenzleser%'`); err != nil {
			t.Errorf("aufräumen: %v", err)
		}
	})
	// Mehr Schüler als die Grenze, alle in einer Klasse: Auch die gefilterte Liste ist gekappt.
	ueberschuss := grenze + 20
	if _, err := pool.Exec(ctx, `
		INSERT INTO leser (barcode_id, vorname, nachname, klasse, abgaenger_jahr, art)
		SELECT 'GRZ-' || g, 'Paul', 'Grenzleser' || g, '5a', 2032, 'schueler'
		FROM generate_series(1, $1) AS g`, ueberschuss); err != nil {
		t.Fatalf("Schüler anlegen: %v", err)
	}
	// Die Datenbank schreibt die Klasse auf den Namen um, unter dem sie schon geführt wird
	// (klasse_kanonisieren); gefiltert wird nach diesem Namen.
	var klasse string
	if err := pool.QueryRow(ctx, `SELECT klasse FROM leser WHERE barcode_id = 'GRZ-1'`).Scan(&klasse); err != nil {
		t.Fatalf("Klasse lesen: %v", err)
	}

	faelle := []struct {
		name, adresse string
		zeilen        int
		kopf          string
	}{
		{"ohne Suchtext", "/api/schueler", grenze, strconv.Itoa(grenze)},
		{"mit Klassenfilter", "/api/schueler?klasse=" + url.QueryEscape(klasse), grenze, strconv.Itoa(grenze)},
		{"Leserdatei", "/api/schueler?art=alle", grenze, strconv.Itoa(grenze)},
		{"mit Suchtext", "/api/schueler?q=Grenzleser", ueberschuss, ""},
	}
	for _, f := range faelle {
		t.Run(f.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, f.adresse, nil)
			req = req.WithContext(context.WithValue(req.Context(), auth.ClaimsSchluessel, claims))
			rec := httptest.NewRecorder()
			srv.Leserliste(leserAbfragen, repository.NewLMFAbfragen(pool)).ServeHTTP(rec, req)
			if rec.Code != http.StatusOK {
				t.Fatalf("GET %s: Status %d, %s", f.adresse, rec.Code, rec.Body.String())
			}
			var zeilen []repository.LeserListenZeile
			if err := json.Unmarshal(rec.Body.Bytes(), &zeilen); err != nil {
				t.Fatalf("Antwort lesen: %v", err)
			}
			if len(zeilen) != f.zeilen {
				t.Errorf("GET %s: %d Zeilen, erwartet %d", f.adresse, len(zeilen), f.zeilen)
			}
			if kopf := rec.Header().Get(tuer.HeaderGrenze); kopf != f.kopf {
				t.Errorf("GET %s: Kopf %s = %q, erwartet %q", f.adresse, tuer.HeaderGrenze, kopf, f.kopf)
			}
		})
	}
}

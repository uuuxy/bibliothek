package leser

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"testing"
	"time"

	"bibliothek/api/tuer"
	"bibliothek/auth"
	"bibliothek/db"
	"bibliothek/internal/pgtest"
	"bibliothek/repository"
)

// TestLeserliste_NenntSeineGrenze belegt, dass die Leserliste ohne Suchtext bei
// repository.LeserListeGrenze gekappt ist und die Antwort genau diese Zahl im Kopf
// tuer.HeaderGrenze nennt, auch mit Klassenfilter, in der Leserdatei und bei den Ehemaligen.
// Mit Suchtext gibt es weder Grenze noch Kopf. Aus dem Kopf liest die Schülerdatei, ob sie „Die ersten 500 Leser"
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
	if _, err := pool.Exec(ctx, `
		INSERT INTO leser (barcode_id, vorname, nachname, klasse, abgaenger_jahr, art, ist_abgaenger)
		SELECT 'GRZ-E' || g, 'Paul', 'Grenzleser-E' || g, '5a', 2024, 'schueler', true
		FROM generate_series(1, $1) AS g`, ueberschuss); err != nil {
		t.Fatalf("Ehemalige anlegen: %v", err)
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
		{"Ehemalige", "/api/schueler?status=ehemalige", grenze, strconv.Itoa(grenze)},
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

// TestPapierkorb_NenntSeineGrenze belegt, dass der Papierkorb bei 1000 Lesern gekappt ist, die
// zuletzt gelöschten zuerst, und dass die Antwort die Zahl im Kopf tuer.HeaderGrenze nennt. Daran
// sagt die Ansicht „Die 1000 zuletzt gelöschten Leser" (Papierkorb.svelte).
func TestPapierkorb_NenntSeineGrenze(t *testing.T) {
	pool := pgtest.Pool(t)
	ctx := context.Background()
	srv := &Tueren{Basis: &tuer.Basis{DB: &db.Verbindung{Pool: pool}}}
	const grenze, ueberschuss = 1000, 1020

	t.Cleanup(func() {
		if _, err := pool.Exec(ctx, `DELETE FROM leser WHERE nachname LIKE 'Grenzleser-P%'`); err != nil {
			t.Errorf("aufräumen: %v", err)
		}
	})
	// Fallende Löschzeitpunkte, die jüngste Löschung vor einer Sekunde: Sie steht vorn.
	if _, err := pool.Exec(ctx, `
		INSERT INTO leser (barcode_id, vorname, nachname, klasse, abgaenger_jahr, art, deleted_at)
		SELECT 'GRZ-P' || g, 'Paul', 'Grenzleser-P' || g, '5a', 2032, 'schueler',
		       CURRENT_TIMESTAMP - (g || ' seconds')::interval
		FROM generate_series(1, $1) AS g`, ueberschuss); err != nil {
		t.Fatalf("Gelöschte Leser anlegen: %v", err)
	}

	rec := httptest.NewRecorder()
	srv.Papierkorb().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/schueler/deleted", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("Status %d, %s", rec.Code, rec.Body.String())
	}
	var zeilen []map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &zeilen); err != nil {
		t.Fatalf("Antwort lesen: %v", err)
	}
	if len(zeilen) != grenze {
		t.Errorf("%d Zeilen, erwartet %d — ohne LIMIT wären es mindestens %d", len(zeilen), grenze, ueberschuss)
	}
	if kopf := rec.Header().Get(tuer.HeaderGrenze); kopf != strconv.Itoa(grenze) {
		t.Errorf("Kopf %s = %q, erwartet %q", tuer.HeaderGrenze, kopf, strconv.Itoa(grenze))
	}
	// Die jüngsten zuerst: Eine Grenze, die die ältesten Löschungen zeigte, versteckte genau die,
	// die sich noch wiederherstellen lassen.
	if len(zeilen) > 1 {
		erste, zweite := loeschZeit(t, zeilen[0]), loeschZeit(t, zeilen[1])
		if erste.Before(zweite) {
			t.Errorf("Reihenfolge falsch: erste Löschung %s ist älter als die zweite %s", erste, zweite)
		}
	}
}

// loeschZeit liest den Zeitpunkt der Löschung aus einer Zeile des Papierkorbs.
func loeschZeit(t *testing.T, zeile map[string]any) time.Time {
	t.Helper()
	text, ok := zeile["deleted_at"].(string)
	if !ok {
		t.Fatalf("deleted_at fehlt oder ist kein Text: %v", zeile["deleted_at"])
	}
	zeit, err := time.Parse(time.RFC3339Nano, text)
	if err != nil {
		t.Fatalf("deleted_at %q: %v", text, err)
	}
	return zeit
}

package inventur

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"
	"time"

	"bibliothek/internal/pgtest"
)

// Die Titelliste (GET /api/books, Titel-Verwaltung und Medienkatalog) steht nach dem Titel.
// Am Mux des Moduls und an der echten Datenbank geprüft: Die Titel werden in einer
// Reihenfolge angelegt, die der erwarteten widerspricht, ihre laufende Nummer (sort_order)
// steigt also gegen den Titel.
//
// Die Reihenfolge schreibt niemand mehr um: PUT /api/admin/books/reorder gibt es nicht.
func TestTitelliste_StehtNachTitel(t *testing.T) {
	pool := pgtest.Pool(t)
	ctx := context.Background()
	marke := fmt.Sprintf("REIHE%d", time.Now().UnixNano())

	// Angelegt von hinten nach vorn.
	erwartet := []string{
		marke + " Ägypten",
		marke + " Ahlan",
		marke + " Mathe 5",
		marke + " Mathe 10",
		marke + " Ofen",
		marke + " Ökologie",
		marke + " zebra",
		marke + " Zoo",
	}
	t.Cleanup(func() {
		auf := context.Background()
		for _, sql := range []string{
			`DELETE FROM buecher_exemplare WHERE titel_id IN (SELECT id FROM buecher_titel WHERE autor = $1)`,
			`DELETE FROM buecher_titel WHERE autor = $1`,
		} {
			if _, err := pool.Exec(auf, sql, marke); err != nil {
				t.Errorf("Aufräumen: %v", err)
			}
		}
	})
	for i := len(erwartet) - 1; i >= 0; i-- {
		var id string
		if err := pool.QueryRow(ctx, `
			INSERT INTO buecher_titel (titel, autor, medientyp) VALUES ($1, $2, 'Buch') RETURNING id`,
			erwartet[i], marke).Scan(&id); err != nil {
			t.Fatalf("Titel anlegen: %v", err)
		}
		// Der Katalog zeigt nur Titel mit einem Exemplar im Bestand.
		if _, err := pool.Exec(ctx, `
			INSERT INTO buecher_exemplare (titel_id, barcode_id, ist_ausleihbar) VALUES ($1, $2, true)`,
			id, fmt.Sprintf("B-%s-%d", marke, i)); err != nil {
			t.Fatalf("Exemplar anlegen: %v", err)
		}
	}

	durch := func(h http.Handler) http.Handler { return h }
	handler := NewAPIHandler(APIHandlerConfig{
		Repo:                 NewBookRepository(pool),
		Metadaten:            &MetadatenClient{httpClient: &http.Client{}},
		RequireViewBooks:     durch,
		RequireEditBooks:     durch,
		RequireDeleteBooks:   durch,
		RequireAuthenticated: durch,
	})

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/books?q="+strings.ToLower(marke), nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /api/books: HTTP %d — %s", rec.Code, rec.Body.String())
	}
	var antwort struct {
		Data []struct {
			Title string `json:"title"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &antwort); err != nil {
		t.Fatalf("Antwort lesen: %v", err)
	}
	ist := make([]string, len(antwort.Data))
	for i, b := range antwort.Data {
		ist[i] = b.Title
	}
	if !slices.Equal(ist, erwartet) {
		t.Errorf("Reihenfolge der Titelliste:\n ist:\n  %s\n soll:\n  %s",
			strings.Join(ist, "\n  "), strings.Join(erwartet, "\n  "))
	}

	// Die Tür zum Umsortieren ist zu: Der Mux kennt für den Pfad kein PUT mehr.
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodPut, "/api/admin/books/reorder",
		strings.NewReader(`{"bookIds":[]}`)))
	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("PUT /api/admin/books/reorder: HTTP %d, erwartet 405", rec.Code)
	}
}

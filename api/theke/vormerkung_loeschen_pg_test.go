package theke

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"bibliothek/repository"
)

// Der Knopf „Vormerkung löschen": Danach ist sie weg, und ein zweiter Klick hört 404, statt einen
// Erfolg zu melden, der nichts mehr bewirkt hat.
func TestVormerkungLoeschen(t *testing.T) {
	w := baueNachbuchWelt(t)
	ctx := context.Background()
	var id string
	if err := w.pool.QueryRow(ctx, `
		INSERT INTO vormerkungen (titel_id, schueler_id)
		SELECT titel_id, $2 FROM buecher_exemplare WHERE id = $1 RETURNING id`, w.exemplarID, w.anna).Scan(&id); err != nil {
		t.Fatalf("Vormerkung: %v", err)
	}
	loesche := func(kennung string) int {
		req := httptest.NewRequest(http.MethodDelete, "/api/vormerkungen/"+kennung, nil)
		req.SetPathValue("id", kennung)
		rec := httptest.NewRecorder()
		w.srv.VormerkungLoeschen(repository.NewVormerkungsAbfragen(w.pool))(rec, w.sitzung(req))
		return rec.Code
	}

	if code := loesche(""); code != http.StatusBadRequest {
		t.Errorf("ohne Kennung: Status %d, erwartet 400", code)
	}
	if code := loesche(id); code != http.StatusOK {
		t.Errorf("Löschen: Status %d, erwartet 200", code)
	}
	var n int
	if err := w.pool.QueryRow(ctx, `SELECT count(*) FROM vormerkungen WHERE id = $1`, id).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Error("die Vormerkung steht nach dem Löschen noch da")
	}
	if code := loesche(id); code != http.StatusNotFound {
		t.Errorf("zweiter Klick: Status %d, erwartet 404", code)
	}
}

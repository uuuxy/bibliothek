package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"bibliothek/db"
	"bibliothek/internal/pgtest"
)

// Das Passbild eines Lesers im Papierkorb wird nicht mehr ausgeliefert.
//
// Fund (OFFEN.md 5.6): Die Auslieferung verband `schueler_fotos` mit `leser` und fragte
// nicht nach `deleted_at`. Für jeden sichtbaren Weg ist eine gelöschte Person weg — die
// Zeile steht nur noch da, damit ein Versehen zurückgeholt werden kann. Ein Passbild, das
// über die Ausweisnummer weiter herauskommt, macht aus dem Papierkorb ein Archiv.
//
// Die Gegenprobe gehört dazu: Ein aktiver Leser bekommt sein Bild weiterhin, sonst wäre
// der Fix eine Verschlechterung mit grünem Test. Und die Zeile in `schueler_fotos` bleibt
// stehen — wer die Person zurückholt, bekommt ihr Bild zurück.
func TestFotoAuslieferung_NichtAusDemPapierkorb(t *testing.T) {
	pool := pgtest.Pool(t)
	ctx := context.Background()
	srv := &Server{DB: &db.Database{Pool: pool}}

	anlegen := func(barcode string, geloescht bool) {
		t.Helper()
		var id string
		if err := pool.QueryRow(ctx, `
			INSERT INTO leser (barcode_id, vorname, nachname, klasse, abgaenger_jahr, art, deleted_at)
			VALUES ($1, 'Pass', 'Bild', '7a', 2031, 'schueler', CASE WHEN $2 THEN NOW() ELSE NULL END)
			RETURNING id`, barcode, geloescht).Scan(&id); err != nil {
			t.Fatalf("Leser anlegen: %v", err)
		}
		// Der Inhalt ist hier gleichgültig: Geprüft wird, OB etwas herauskommt. Ein
		// unlesbarer Geheimtext endet in 500, nicht in 404 — und 500 ist nicht 200.
		if _, err := pool.Exec(ctx,
			`INSERT INTO schueler_fotos (schueler_id, foto_encrypted) VALUES ($1, $2)`,
			id, []byte("kein echtes Bild")); err != nil {
			t.Fatalf("Foto anlegen: %v", err)
		}
		t.Cleanup(func() {
			auf := context.Background()
			if _, err := pool.Exec(auf, `DELETE FROM schueler_fotos WHERE schueler_id = $1`, id); err != nil {
				t.Errorf("Aufräumen Foto: %v", err)
			}
			if _, err := pool.Exec(auf, `DELETE FROM leser WHERE id = $1`, id); err != nil {
				t.Errorf("Aufräumen Leser: %v", err)
			}
		})
	}

	hole := func(barcode string) int {
		t.Helper()
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/api/schueler/"+barcode+"/photo", nil)
		r.SetPathValue("barcode_id", barcode)
		srv.ServeStudentPhotoHandler().ServeHTTP(w, r)
		return w.Code
	}

	anlegen("S-PAPIERKORB-1", true)
	anlegen("S-AKTIV-1", false)

	if code := hole("S-PAPIERKORB-1"); code != http.StatusNotFound {
		t.Errorf("das Passbild einer gelöschten Person kommt mit Status %d heraus, erwartet 404 — "+
			"der Papierkorb wäre damit ein Archiv", code)
	}
	if code := hole("S-AKTIV-1"); code == http.StatusNotFound {
		t.Error("das Passbild eines aktiven Lesers kommt nicht mehr heraus — der Fix hat die " +
			"Auslieferung ganz abgeschaltet")
	}
}

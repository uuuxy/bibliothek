package katalog

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"bibliothek/api/tuer"
	"bibliothek/db"
	"bibliothek/internal/pgtest"
	"bibliothek/internal/testdaten"
	"bibliothek/internal/tuertest"
	"bibliothek/repository"
)

// Ein Status-Dialog, der seit einer Schadensmeldung an einem anderen Platz offen stand, schreibt
// den alten Status nicht zurück: Sonst stünde das ausgesonderte Buch wieder im Bestand, während
// die Forderung an die Eltern bleibt. Mit dem heutigen Stand speichert er.
func TestExemplarStatus_LehntAbWennEinAndererPlatzInzwischenAusgesondertHat(t *testing.T) {
	pool := pgtest.Pool(t)
	testdaten.LeereBestandsdaten(t, pool)
	ctx := context.Background()
	katalog := repository.NewKatalogAbfragen(pool)
	srv := &Tueren{Basis: &tuer.Basis{DB: &db.Verbindung{Pool: pool}}}
	handler := srv.ExemplarStatusAendern(katalog, repository.NewBescheidAbfragen(pool))
	bearbeiter := testdaten.LegeBearbeiterAn(t, pool)

	titelID := testdaten.LegeTitelMitSignaturAn(t, pool, "Veraltet-Titel", "Vea 1", 0)
	exemplarID := testdaten.LegeExemplarAn(t, pool, titelID, "VEA-1", true, "")
	schuelerID := testdaten.LegeSchuelerAn(t, pool, "S-VEA-1", "Vea", "7a")
	testdaten.LegeAusleiheVorZweiTagenAn(t, pool, exemplarID, schuelerID)
	var ausleiheID string
	if err := pool.QueryRow(ctx,
		`SELECT id FROM ausleihen WHERE exemplar_id = $1 AND rueckgabe_am IS NULL`, exemplarID,
	).Scan(&ausleiheID); err != nil {
		t.Fatalf("Ausleihe lesen: %v", err)
	}

	// Der andere Platz meldet den Schaden; das Exemplar wird ausgesondert.
	if _, err := repository.NewSchadensAbfragen(pool).Melde(ctx, ausleiheID, bearbeiter, "Wasserschaden",
		repository.SchadensArtBeschaedigt, 1200); err != nil {
		t.Fatalf("Schaden melden: %v", err)
	}

	speichere := func(koerper string) *httptest.ResponseRecorder {
		t.Helper()
		req := httptest.NewRequest(http.MethodPut, "/api/buecher/exemplare/"+exemplarID+"/status", strings.NewReader(koerper))
		req.SetPathValue("id", exemplarID)
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, tuertest.AlsKonto(req, bearbeiter))
		return rec
	}
	ausgesondert := func() bool {
		t.Helper()
		var weg bool
		if err := pool.QueryRow(ctx, `SELECT ist_ausgesondert FROM buecher_exemplare WHERE id = $1`, exemplarID).Scan(&weg); err != nil {
			t.Fatalf("Exemplar lesen: %v", err)
		}
		return weg
	}

	// Der Dialog zeigte beim Öffnen „Verfügbar" und speichert nur den Beschädigungsgrad.
	rec := speichere(`{"ist_ausleihbar":true,"ist_ausgesondert":false,"zustand_notiz":"",` +
		`"zustand_abwertung_prozent":30,"bisher":{"ist_ausleihbar":true,"ist_ausgesondert":false}}`)
	if rec.Code != http.StatusConflict {
		t.Fatalf("veralteter Dialog: HTTP %d statt 409: %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "neu laden") {
		t.Errorf("die Ablehnung sagt nicht, was zu tun ist: %s", rec.Body.String())
	}
	if !ausgesondert() {
		t.Fatal("das Exemplar steht wieder im Bestand, obwohl an einem anderen Platz der Schaden gemeldet ist")
	}

	rec = speichere(`{"ist_ausleihbar":false,"ist_ausgesondert":true,"zustand_notiz":"Wasserschaden",` +
		`"zustand_abwertung_prozent":30,"bisher":{"ist_ausleihbar":false,"ist_ausgesondert":true}}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("Dialog mit dem heutigen Stand: HTTP %d: %s", rec.Code, rec.Body.String())
	}
	if !ausgesondert() {
		t.Error("das Speichern mit dem heutigen Stand hat das Exemplar zurückgeholt")
	}
}

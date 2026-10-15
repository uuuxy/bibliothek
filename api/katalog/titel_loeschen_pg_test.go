package katalog

import (
	"context"
	"net/http"
	"testing"

	"bibliothek/api/tuer"
	"bibliothek/db"
	"bibliothek/internal/pgtest"
	"bibliothek/internal/testdaten"
	"bibliothek/internal/tuertest"
	"bibliothek/repository"
)

// Ist ein Exemplar verliehen, lehnt die Tür das Löschen eines Titels mit 400 ab und nennt die
// Barcodes der verliehenen Exemplare: Daran sucht die Bibliothek, wo das Buch ist. Der Titel
// bleibt stehen.
func TestTitelLoeschen_VerliehenesExemplarNenntDenBarcode(t *testing.T) {
	pool := pgtest.Pool(t)
	testdaten.LeereBestandsdaten(t, pool)
	ctx := context.Background()

	var adminID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO benutzer (vorname, nachname, email, rolle, aktiv)
		VALUES ('Ada', 'Titelloesch', 'ada@titelloesch.invalid', 'admin', true) RETURNING id::text`).Scan(&adminID); err != nil {
		t.Fatalf("Konto anlegen: %v", err)
	}
	t.Cleanup(func() {
		pgtest.RaeumeAuf(t, pool, `DELETE FROM benutzer WHERE email = 'ada@titelloesch.invalid'`)
	})
	titelID := testdaten.LegeTitelAn(t, pool, "Titel mit verliehenem Exemplar", 0)
	exemplarID := testdaten.LegeExemplarAn(t, pool, titelID, "B-TITELLOESCH-1", true, "")
	testdaten.LegeExemplarAn(t, pool, titelID, "B-TITELLOESCH-2", true, "")
	schuelerID := testdaten.LegeSchuelerAn(t, pool, "S-TITELLOESCH-1", "Lotte", "07A")
	testdaten.LegeAusleiheVorZweiTagenAn(t, pool, exemplarID, schuelerID)

	srv := &Tueren{Basis: &tuer.Basis{DB: &db.Verbindung{Pool: pool}}}
	rec := tuertest.RufeAlsAdmin(t, adminID, srv.TitelLoeschen(repository.NewProtokollAbfragen(pool)),
		http.MethodDelete, "/api/buecher/titel/"+titelID, "", map[string]string{"id": titelID})

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("TitelLoeschen: Status %d, erwartet 400: %s", rec.Code, rec.Body.String())
	}
	const erwartet = "Löschen fehlgeschlagen: folgende Exemplare sind noch verliehen: [B-TITELLOESCH-1]"
	if got := tuertest.Fehlermeldung(t, rec); got != erwartet {
		t.Errorf("TitelLoeschen: Meldung %q, erwartet %q", got, erwartet)
	}
	if n := pgtest.ZaehleZeilen(t, pool, `SELECT count(*) FROM buecher_titel WHERE id = $1`, titelID); n != 1 {
		t.Errorf("nach der Ablehnung stehen %d Titel, erwartet 1", n)
	}
}

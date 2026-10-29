package inventurlauf

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"bibliothek/api/tuer"
	"bibliothek/db"
	"bibliothek/internal/pgtest"
	"bibliothek/internal/testdaten"
	"bibliothek/internal/tuertest"
	"bibliothek/repository"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Ältere Inventuren sind erreichbar: Die erste Seite nennt, wie viele danach kommen, und ihr
// Zeichen holt die nächste. Mit nur zehn Läufen in der Antwort war jeder elfte samt
// Fehlbestandsbericht aus der Oberfläche verschwunden.
func TestAbgeschlosseneInventuren_SeitenUeberDasZeichen(t *testing.T) {
	pool := pgtest.Pool(t)
	srv := &Tueren{Basis: &tuer.Basis{DB: &db.Verbindung{Pool: pool}}}
	ids := schliesseInventurenAb(t, pool, 12)

	erste := fruehereSeite(t, srv, "")
	if len(erste.Laeufe) != laeufeJeSeite || erste.Weitere != 2 || erste.NaechsteSeite == "" {
		t.Fatalf("erste Seite: %d Läufe, %d weitere, Zeichen %q; erwartet %d, 2 weitere und ein Zeichen",
			len(erste.Laeufe), erste.Weitere, erste.NaechsteSeite, laeufeJeSeite)
	}
	zweite := fruehereSeite(t, srv, erste.NaechsteSeite)
	if len(zweite.Laeufe) != 2 || zweite.Weitere != 0 || zweite.NaechsteSeite != "" {
		t.Fatalf("zweite Seite: %d Läufe, %d weitere, Zeichen %q; erwartet 2, keine weitere, kein Zeichen",
			len(zweite.Laeufe), zweite.Weitere, zweite.NaechsteSeite)
	}

	// Beide Seiten zusammen sind alle Läufe, jeder einmal, die neuesten zuerst.
	gesehen := map[string]bool{}
	for i, zeile := range append(erste.Laeufe, zweite.Laeufe...) {
		id := fmt.Sprint(zeile["session_id"])
		if gesehen[id] {
			t.Errorf("Lauf %s steht zweimal in der Liste", id)
		}
		gesehen[id] = true
		if want := ids[len(ids)-1-i]; id != want {
			t.Errorf("Zeile %d ist %s, erwartet %s (die neuesten zuerst)", i, id, want)
		}
	}
	if len(gesehen) != len(ids) {
		t.Errorf("%d von %d Läufen erreichbar", len(gesehen), len(ids))
	}

	rec := httptest.NewRecorder()
	srv.AbgeschlosseneInventuren().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/inventur/abgeschlossen?seite=kaputt", nil))
	if rec.Code != http.StatusBadRequest {
		t.Errorf("ein fremdes Zeichen ergab %d, erwartet 400: %s", rec.Code, rec.Body.String())
	}
}

// Eine frühere Inventur lässt sich löschen: Sie verschwindet aus der Liste, die Verlustbuchung am
// Buch bleibt. Eine laufende wird verworfen, nicht gelöscht, und ein zweites Löschen findet
// nichts mehr.
func TestAbgeschlosseneInventurLoeschen(t *testing.T) {
	pool := pgtest.Pool(t)
	ctx := context.Background()
	srv := &Tueren{Basis: &tuer.Basis{DB: &db.Verbindung{Pool: pool}}}
	bearbeiter := testdaten.LegeBearbeiterAn(t, pool)

	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	sigFertig, sigOffen := "INV-Loeschen-"+suffix, "INV-Offen-"+suffix
	verloren := "B-L-" + suffix
	testdaten.LegeExemplarMitSignaturAn(t, pool, sigFertig, verloren)
	testdaten.LegeExemplarMitSignaturAn(t, pool, sigOffen, "B-O-"+suffix)
	invRepo := repository.NewInventurAbfragen(pool)
	fertig, err := invRepo.LegeLaufAn(ctx, "signature", repository.InventurBereich{Signatur: &sigFertig}, "Löschen", "")
	if err != nil {
		t.Fatalf("Lauf anlegen: %v", err)
	}
	if _, err := invRepo.SchliesseLaufAb(ctx, fertig.ID, fertig.Bereich(), bearbeiter); err != nil {
		t.Fatalf("Abschluss: %v", err)
	}
	offen, err := invRepo.LegeLaufAn(ctx, "signature", repository.InventurBereich{Signatur: &sigOffen}, "Offen", "")
	if err != nil {
		t.Fatalf("Lauf anlegen: %v", err)
	}

	loesche := func(id string) int {
		return tuertest.RufeAlsAdmin(t, bearbeiter, srv.AbgeschlosseneInventurLoeschen(), http.MethodDelete,
			"/api/inventur/abgeschlossen/"+id, "", map[string]string{"id": id}).Code
	}
	if code := loesche(fertig.ID); code != http.StatusNoContent {
		t.Fatalf("Löschen der abgeschlossenen Inventur = %d, erwartet 204", code)
	}
	for _, zeile := range abgeschlosseneInventuren(t, srv) {
		if zeile["session_id"] == fertig.ID {
			t.Errorf("die gelöschte Inventur steht noch in der Liste: %+v", zeile)
		}
	}
	var grund string
	if err := pool.QueryRow(ctx, `SELECT coalesce(aussonderung_grund, '') FROM buecher_exemplare WHERE barcode_id = $1`,
		verloren).Scan(&grund); err != nil {
		t.Fatalf("Exemplar lesen: %v", err)
	}
	if grund != "VERLUST" {
		t.Errorf("das verlorene Buch hat nach dem Löschen den Grund %q, erwartet VERLUST", grund)
	}

	for name, id := range map[string]string{"zweites Löschen": fertig.ID, "laufende Inventur": offen.ID} {
		if code := loesche(id); code != http.StatusNotFound {
			t.Errorf("%s = %d, erwartet 404", name, code)
		}
	}
}

// schliesseInventurenAb legt n Läufe über je eine eigene Signatur an und schließt sie ab, einen
// nach dem anderen. Geliefert werden die Kennungen in der Reihenfolge des Abschlusses.
func schliesseInventurenAb(t *testing.T, pool *pgxpool.Pool, n int) []string {
	t.Helper()
	ctx := context.Background()
	if _, err := pool.Exec(ctx, `TRUNCATE inventur_sessions RESTART IDENTITY CASCADE`); err != nil {
		t.Fatalf("Läufe leeren: %v", err)
	}
	testdaten.LeereBestandsdaten(t, pool)
	bearbeiter := testdaten.LegeBearbeiterAn(t, pool)
	invRepo := repository.NewInventurAbfragen(pool)

	ids := make([]string, 0, n)
	for i := 0; i < n; i++ {
		sig := fmt.Sprintf("INV-Seite-%d-%d", time.Now().UnixNano(), i)
		testdaten.LegeExemplarMitSignaturAn(t, pool, sig, fmt.Sprintf("B-S-%d-%d", time.Now().UnixNano(), i))
		lauf, err := invRepo.LegeLaufAn(ctx, "signature", repository.InventurBereich{Signatur: &sig}, sig, "")
		if err != nil {
			t.Fatalf("Lauf %d anlegen: %v", i, err)
		}
		if _, err := invRepo.SchliesseLaufAb(ctx, lauf.ID, lauf.Bereich(), bearbeiter); err != nil {
			t.Fatalf("Abschluss %d: %v", i, err)
		}
		ids = append(ids, lauf.ID)
	}
	return ids
}

package lernmittel

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"bibliothek/internal/api/tuer"
	"bibliothek/internal/auth"
	"bibliothek/internal/db"
	"bibliothek/internal/pgtest"
	"bibliothek/internal/pkg/schulzeit"
	"bibliothek/internal/repository"
	"bibliothek/internal/service"
	"bibliothek/internal/testdaten"
)

// Verwerfen, während ein Veröffentlichen noch läuft: Das Verwerfen las den Plan vor seiner
// Transaktion, hielt ihn für einen Entwurf und ließ nach dem Löschen die Fristen des eben
// veröffentlichten Plans stehen. Jetzt sperrt es zuerst wie das Speichern, wartet auf das
// Veröffentlichen und setzt die Fristen zurück.
func TestLMFPlan_VerwerfenWartetAufEinLaufendesVeroeffentlichen(t *testing.T) {
	pool := pgtest.Pool(t)
	testdaten.LeereBestandsdaten(t, pool)
	ctx := context.Background()
	srv := &Tueren{Basis: &tuer.Basis{DB: &db.Verbindung{Pool: pool}}}
	t.Cleanup(func() {
		if _, err := pool.Exec(ctx, `DELETE FROM lmf_plaene`); err != nil {
			t.Logf("Aufräumen Pläne: %v", err)
		}
	})

	tag := func(d string) time.Time {
		x, err := time.ParseInLocation("2006-01-02", d, schulzeit.Zone())
		if err != nil {
			t.Fatalf("Testdatum %q: %v", d, err)
		}
		return schulzeit.TagesEnde(x)
	}
	schueler := testdaten.LegeSchuelerAn(t, pool, "WETT-1", "Wettlauf", "9H1")
	testdaten.LegeAusleiheAmTagAn(t, pool, schueler, "LMF Mathe 9 Wettlauf", ausleihtagImSchuljahr(), tag("2027-07-31"))
	frist := func() string {
		t.Helper()
		var d string
		if err := pool.QueryRow(ctx, `SELECT to_char(rueckgabe_frist AT TIME ZONE 'Europe/Berlin', 'YYYY-MM-DD')
			FROM ausleihen WHERE schueler_id = $1 AND rueckgabe_am IS NULL`, schueler).Scan(&d); err != nil {
			t.Fatalf("Frist lesen: %v", err)
		}
		return d
	}
	anfrage := func(methode, body string) *http.Request {
		req := httptest.NewRequest(methode, "/api/lmf-plan/rueckgabe", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.SetPathValue("art", "rueckgabe")
		return req.WithContext(context.WithValue(req.Context(), auth.ClaimsSchluessel, &auth.Claims{Rolle: auth.RolleAdmin}))
	}

	rec := httptest.NewRecorder()
	srv.LMFPlanSchreiben()(rec, anfrage(http.MethodPut, mitGeladenemStand(t, pool, "rueckgabe",
		`{"letzter_tag":"2027-06-28","letzte_stunde":4,"stunden_je_tag":6,"zeilen":[{"klassen":["9H1"]}]}`)))
	if rec.Code != http.StatusOK {
		t.Fatalf("Entwurf speichern: %d %s", rec.Code, rec.Body.String())
	}

	// Ein Veröffentlichen, das gestempelt und die Fristen gesetzt, aber noch nicht abgeschlossen hat.
	abfragen := repository.NewLMFAbfragen(pool)
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Verwerfe(ctx, tx)
	if _, err := abfragen.SperreNeuestenPlan(ctx, tx, "rueckgabe"); err != nil {
		t.Fatal(err)
	}
	entwurf, err := abfragen.LadeNeuestenPlan(ctx, "rueckgabe")
	if err != nil {
		t.Fatal(err)
	}
	veroeffentlicht, err := abfragen.VeroeffentlichePlan(ctx, tx, entwurf.Plan.ID, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.KoppleLMFPlanFristen(ctx, abfragen, tx, "rueckgabe", nil, veroeffentlicht.Zeilen); err != nil {
		t.Fatal(err)
	}

	fertig := make(chan *httptest.ResponseRecorder, 1)
	go func() {
		rec := httptest.NewRecorder()
		srv.LMFPlanVerwerfen()(rec, anfrage(http.MethodDelete, ""))
		fertig <- rec
	}()
	select {
	case rec := <-fertig:
		t.Fatalf("das Verwerfen wartete nicht auf das Veröffentlichen: %d %s", rec.Code, rec.Body.String())
	case <-time.After(300 * time.Millisecond):
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	if got := frist(); got != "2027-06-28" {
		t.Fatalf("nach dem Veröffentlichen steht die Frist auf %s, erwartet 2027-06-28 (Plan)", got)
	}

	rec = <-fertig
	if rec.Code != http.StatusOK {
		t.Fatalf("Verwerfen: %d %s", rec.Code, rec.Body.String())
	}
	var antwort struct {
		FristenAngepasst int64 `json:"fristen_angepasst"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &antwort); err != nil {
		t.Fatal(err)
	}
	// Die Frist kehrt zum Stichtag zurück wie beim Verwerfen eines veröffentlichten Plans; sie
	// bleibt nicht auf dem Termin des Plans, den es nicht mehr gibt.
	if got := frist(); got == "2027-06-28" || antwort.FristenAngepasst != 1 {
		t.Errorf("nach dem Verwerfen: Frist %s, %d Fristen angepasst; erwartet zurück zum Stichtag und 1", got, antwort.FristenAngepasst)
	}
}

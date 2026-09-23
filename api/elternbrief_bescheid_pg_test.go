package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"bibliothek/db"
)

// Der Elternbrief je Schadensfall (GET /api/schadensfaelle/{id}/pdf) prüfte nicht, ob die
// Forderung schon auf einem Schadensersatz-Bescheid steht (docs/OFFEN.md 5.2). Dann stünde
// dieselbe Forderung auf zwei Briefen mit zwei Fristen und zwei Zahlungswegen. Die
// Ersatzforderung hält diese Regel seit dem 13.09.2026 (api/print.go, bescheid_id IS NULL);
// die Oberfläche öffnet den Elternbrief nicht mehr, über die Adresse blieb er erreichbar.
func TestElternbrief_ForderungAufEinemBescheidBekommtKeinenBrief(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	bescheidReset(t, pool)
	bescheidAngabenSetzen(t, pool)
	srv := &Server{DB: &db.Database{Pool: pool}}
	ctx := context.Background()

	sid := seedSchueler(t, pool, "S-EBRIEF", "Briefkind", "08G2")
	titelID := bescheidLernmittel(t, pool, "Mathematik 8")
	fAuf := bescheidForderung(t, pool, sid, exemplar(t, pool, titelID, "EBRIEF-1", true, ""),
		"nicht_zurueckgegeben", "Mathematik 8 nicht zurück")
	fOhne := bescheidForderung(t, pool, sid, exemplar(t, pool, titelID, "EBRIEF-2", true, ""),
		"beschaedigt", "Mathematik 8 beschädigt")
	if rec := bescheidErstellenUeberHandler(t, srv, pool, sid,
		bescheidRumpf(in28Tagen(), map[string]float64{fAuf: 24.90})); rec.Code != http.StatusCreated {
		t.Fatalf("Bescheid anlegen: Status %d, want 201: %s", rec.Code, rec.Body.String())
	}

	brief := func(id string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodGet, "/api/schadensfaelle/"+id+"/pdf", nil)
		req.SetPathValue("id", id)
		rec := httptest.NewRecorder()
		srv.GenerateDamagePDFHandler().ServeHTTP(rec, req)
		return rec
	}

	rec := brief(fAuf)
	if rec.Code != http.StatusConflict {
		t.Fatalf("Status %d, want 409 — die Forderung steht auf einem Bescheid, ein Elternbrief "+
			"daneben nennte sie ein zweites Mal mit eigener Frist (Content-Type %q)",
			rec.Code, rec.Header().Get("Content-Type"))
	}
	if !strings.Contains(rec.Body.String(), "Bescheid") {
		t.Errorf("Meldung nennt den Bescheid nicht: %s", rec.Body.String())
	}
	// Ein abgelehnter Brief ist kein erstellter: Der Vermerk bleibt aus.
	var vermerkt bool
	if err := pool.QueryRow(ctx,
		`SELECT elternbrief_generiert FROM schadensfaelle WHERE id = $1`, fAuf).Scan(&vermerkt); err != nil {
		t.Fatalf("Vermerk lesen: %v", err)
	}
	if vermerkt {
		t.Error("elternbrief_generiert steht auf true, obwohl kein Brief entstanden ist")
	}

	// Gegenprobe: Die Forderung desselben Kindes ohne Bescheid bekommt ihren Brief.
	if rec := brief(fOhne); rec.Code != http.StatusOK {
		t.Fatalf("Forderung ohne Bescheid: Status %d, want 200: %s", rec.Code, rec.Body.String())
	}
}

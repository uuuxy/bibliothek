package api

import (
	"net/http"
	"strings"
	"testing"

	"bibliothek/db"
	"bibliothek/pkg/schulzeit"
)

// Die Frist eines Bescheids kommt aus der Anfrage (POST /api/schueler/{id}/bescheide) und
// wurde nur daraufhin geprüft, ob sie ein Datum ist (docs/OFFEN.md 5.2). Eine Frist von
// heute oder gestern machte den Bescheid sofort übergabefähig; eine Frist wie 2062 ging als
// Kassenjahr in die Referenznummer, und eine Zahlung wäre nicht zuzuordnen. Die Arbeitshilfe
// nennt eine „Vierwochen-Frist", die sich „in der Regel ab dem Briefdatum" berechnet.
func TestBescheid_FristMussNachHeuteUndHoechstensEinJahrVorausLiegen(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	bescheidReset(t, pool)
	bescheidAngabenSetzen(t, pool)
	srv := &Server{DB: &db.Database{Pool: pool}}

	sid := seedSchueler(t, pool, "S-FRIST", "Fristprobe", "08G2")
	titelID := bescheidLernmittel(t, pool, "Chemie 8")
	f := bescheidForderung(t, pool, sid, exemplar(t, pool, titelID, "FRIST-1", true, ""),
		"nicht_zurueckgegeben", "Chemie 8 nicht zurück")

	heute := schulzeit.Jetzt()
	tag := func(tage int) string { return heute.AddDate(0, 0, tage).Format("2006-01-02") }
	for _, fall := range []struct {
		name, frist, meldung string
	}{
		{"gestern", tag(-1), "nach dem heutigen Tag"},
		{"heute", tag(0), "nach dem heutigen Tag"},
		{"mehr als ein Jahr voraus", heute.AddDate(1, 0, 1).Format("2006-01-02"), "mehr als ein Jahr"},
	} {
		rec := bescheidErstellenUeberHandler(t, srv, pool, sid,
			bescheidRumpf(fall.frist, map[string]float64{f: 18.00}))
		if rec.Code != http.StatusBadRequest {
			t.Errorf("Frist %s (%s): Status %d, want 400 — %s", fall.name, fall.frist, rec.Code, rec.Body.String())
			continue
		}
		if !strings.Contains(rec.Body.String(), fall.meldung) {
			t.Errorf("Frist %s: Meldung ohne %q: %s", fall.name, fall.meldung, rec.Body.String())
		}
	}

	// Gegenprobe: morgen und vier Wochen sind erlaubt — der zweite Bescheid braucht eine
	// eigene Forderung, die erste steht dann schon auf einem.
	if rec := bescheidErstellenUeberHandler(t, srv, pool, sid,
		bescheidRumpf(tag(1), map[string]float64{f: 18.00})); rec.Code != http.StatusCreated {
		t.Fatalf("Frist morgen: Status %d, want 201: %s", rec.Code, rec.Body.String())
	}
	f2 := bescheidForderung(t, pool, sid, exemplar(t, pool, titelID, "FRIST-2", true, ""),
		"beschaedigt", "Chemie 8 beschädigt")
	if rec := bescheidErstellenUeberHandler(t, srv, pool, sid,
		bescheidRumpf(in28Tagen(), map[string]float64{f2: 9.00})); rec.Code != http.StatusCreated {
		t.Fatalf("Frist in vier Wochen: Status %d, want 201: %s", rec.Code, rec.Body.String())
	}
}

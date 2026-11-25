package lernmittel

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"bibliothek/internal/api/tuer"
	"bibliothek/internal/db"
	"bibliothek/internal/pgtest"
	"bibliothek/internal/pkg/schulzeit"
	"bibliothek/internal/testdaten"
)

// TestLernmittelDerKlasseVerlaengern_SchreibvariantenRobust: Die Verlängerung einer Klasse
// erkennt Lernmittel über internal/pkg/lmf wie Ausleihe und Frist, nicht über `ILIKE 'LMF-%'`. Sonst
// bliebe ein von Hand angelegtes "LMF - Deutsch 5" (Leerzeichen nach LMF) auf der alten Frist
// stehen, würde überfällig und landete in der Mahnung.
func TestLernmittelDerKlasseVerlaengern_SchreibvariantenRobust(t *testing.T) {
	pool := pgtest.Pool(t)
	testdaten.LeereBestandsdaten(t, pool)

	const klasse = "5a"
	sid := testdaten.LegeSchuelerAn(t, pool, "S-LMF-1", "Fritz", klasse)

	alteFrist := time.Date(2025, 9, 1, 23, 59, 59, 0, time.UTC)

	// Drei LMF-Schreibvarianten (alle müssen verlängert werden) + ein Freihand-Titel
	// (darf nicht angefasst werden).
	bindestrich := testdaten.LegeAusleiheMitFristAn(t, pool, sid, "LMF-Mathe 5", alteFrist)
	leerBindestrich := testdaten.LegeAusleiheMitFristAn(t, pool, sid, "LMF - Deutsch 5", alteFrist)
	leer := testdaten.LegeAusleiheMitFristAn(t, pool, sid, "LMF Bio 5", alteFrist)
	freihand := testdaten.LegeAusleiheMitFristAn(t, pool, sid, "Der Hobbit", alteFrist)

	// Ein Tag im Juli des nächsten Jahres: Die Verlängerung nimmt nur ein Datum in der Zukunft.
	juli := time.Date(time.Now().Year()+1, 7, 31, 0, 0, 0, 0, time.UTC)
	srv := &Tueren{Basis: &tuer.Basis{DB: &db.Verbindung{Pool: pool}}}
	req := httptest.NewRequest(http.MethodPost, "/api/ausleihen/lmf/global-extend",
		strings.NewReader(`{"klasse":"05A","neues_rueckgabe_datum":"`+juli.Format("2006-01-02")+`"}`))
	rec := httptest.NewRecorder()
	srv.LernmittelDerKlasseVerlaengern()(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("rec.Code = %d, erwartet 200: %s", rec.Code, rec.Body.String())
	}

	// Berliner Tagesende zum Datum (Sommer/CEST = 21:59:59 UTC), wie die Tür es setzt, nicht
	// roh 23:59:59 UTC.
	neu := schulzeit.TagesEnde(juli)
	for _, tc := range []struct {
		name       string
		id         string
		verlängert bool
	}{
		{"LMF-Mathe 5 (Bindestrich)", bindestrich, true},
		{"LMF - Deutsch 5 (Leer-Bindestrich)", leerBindestrich, true},
		{"LMF Bio 5 (Leerzeichen)", leer, true},
		{"Der Hobbit (Freihand)", freihand, false},
	} {
		frist := testdaten.LadeFrist(t, pool, tc.id)
		if tc.verlängert && !frist.Equal(neu) {
			t.Errorf("%s: Frist = %s, erwartet %s (LMF-Schreibvariante übersehen)", tc.name, frist, neu)
		}
		if !tc.verlängert && !frist.Equal(alteFrist) {
			t.Errorf("%s: Freihand-Buch fälschlich verlängert — war %s", tc.name, frist)
		}
	}
}

// Die Verlängerung einer Klasse verlängert und verkürzt nicht: Ein Band für mehrere Jahre, dessen
// Frist hinter dem neuen Datum liegt, behält sie. Ein Datum in der Vergangenheit oder hinter der
// längsten Frist des Programms ist ein Tippfehler: 400, und keine Frist ändert sich.
func TestLernmittelDerKlasseVerlaengern_VerkuerztNichtUndPrueftDasDatum(t *testing.T) {
	pool := pgtest.Pool(t)
	testdaten.LeereBestandsdaten(t, pool)

	sid := testdaten.LegeSchuelerAn(t, pool, "S-LMF-MJ", "Mia", "7a")
	jetzt := time.Now()
	einjaehrig := testdaten.LegeAusleiheMitFristAn(t, pool, sid, "LMF-Englisch 7", jetzt.AddDate(0, 1, 0))
	mehrjaehrig := testdaten.LegeAusleiheMitFristAn(t, pool, sid, "LMF-Atlas 7-9", jetzt.AddDate(2, 0, 0))
	fristEinjaehrig := testdaten.LadeFrist(t, pool, einjaehrig)
	fristMehrjaehrig := testdaten.LadeFrist(t, pool, mehrjaehrig)

	srv := &Tueren{Basis: &tuer.Basis{DB: &db.Verbindung{Pool: pool}}}
	verlaengere := func(datum time.Time) *httptest.ResponseRecorder {
		t.Helper()
		rec := httptest.NewRecorder()
		srv.LernmittelDerKlasseVerlaengern()(rec, httptest.NewRequest(http.MethodPost, "/api/ausleihen/global-extend-lmf",
			strings.NewReader(`{"klasse":"07A","neues_rueckgabe_datum":"`+datum.Format("2006-01-02")+`"}`)))
		return rec
	}

	for _, fall := range []struct {
		name  string
		datum time.Time
	}{
		{"gestern", jetzt.AddDate(0, 0, -1)},
		{"in vierzehn Jahren", jetzt.AddDate(14, 0, 0)},
	} {
		if rec := verlaengere(fall.datum); rec.Code != http.StatusBadRequest {
			t.Errorf("Datum %s: %d, erwartet 400: %s", fall.name, rec.Code, rec.Body.String())
		}
	}
	if !testdaten.LadeFrist(t, pool, einjaehrig).Equal(fristEinjaehrig) || !testdaten.LadeFrist(t, pool, mehrjaehrig).Equal(fristMehrjaehrig) {
		t.Error("ein abgewiesenes Datum hat eine Frist geändert")
	}

	neu := jetzt.AddDate(1, 0, 0)
	rec := verlaengere(neu)
	if rec.Code != http.StatusOK {
		t.Fatalf("Verlängerung um ein Jahr: %d: %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"updated_count":1`) {
		t.Errorf("Antwort %s, erwartet eine verlängerte Ausleihe", rec.Body.String())
	}
	soll := schulzeit.TagesEnde(time.Date(neu.Year(), neu.Month(), neu.Day(), 0, 0, 0, 0, time.UTC))
	if frist := testdaten.LadeFrist(t, pool, einjaehrig); !frist.Equal(soll) {
		t.Errorf("einjähriges Lernmittel: Frist %s, erwartet %s", frist, soll)
	}
	if frist := testdaten.LadeFrist(t, pool, mehrjaehrig); !frist.Equal(fristMehrjaehrig) {
		t.Errorf("Band für mehrere Jahre: Frist %s, erwartet unverändert %s", frist, fristMehrjaehrig)
	}
}

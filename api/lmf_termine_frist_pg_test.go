package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"bibliothek/db"
	"bibliothek/internal/service"
	"bibliothek/pkg/schulzeit"
	"bibliothek/repository"
)

// Die Kopplung Plan → Frist am echten Postgres, über den Plan-Handler (Register,
// Entscheidung 3a; seit Migration 097 als Reihenfolge): Ein Rückgabe-Plan setzt die
// Frist der offenen Schulbücher seiner Klassen; ein Umschreiben zieht nach; Klassen, die
// aus dem Plan fallen, kehren zum Stichtag zurück; Verwerfen ebenso. Nicht angefasst:
// Nicht-Lernmittel, gesperrte Schüler, mehrjährige Ausleihen, andere Klassen, Ausgabe-Pläne.
func TestLmfPlan_RueckgabeTerminIstDieFristDerKlasse(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	ctx := context.Background()
	srv := &Server{DB: &db.Database{Pool: pool}}
	t.Cleanup(func() {
		if _, err := pool.Exec(ctx, `DELETE FROM lmf_plaene`); err != nil {
			t.Logf("Aufräumen: %v", err)
		}
	})

	tag := func(d string) time.Time {
		x, err := time.ParseInLocation("2006-01-02", d, schulzeit.Zone())
		if err != nil {
			t.Fatalf("Testdatum %q: %v", d, err)
		}
		return service.TagesEndeInSchulzeitzone(x)
	}
	stichtag := tag("2027-07-31")

	anna := seedSchueler(t, pool, "F-1", "Anna", "9H1")
	annaLmf := seedAusleihe(t, pool, anna, "LMF Mathe 9 Anna", stichtag)
	annaRoman := seedAusleihe(t, pool, anna, "Roman Anna", tag("2026-10-01"))
	annaMehrjahr := seedAusleihe(t, pool, anna, "LMF Atlas Anna", tag("2028-07-31"))
	ben := seedSchueler(t, pool, "F-2", "Ben", "09h1") // andere Schreibweise, dieselbe Klasse
	benLmf := seedAusleihe(t, pool, ben, "LMF Mathe 9 Ben", stichtag)
	if _, err := pool.Exec(ctx, `UPDATE schueler SET ist_gesperrt = true, block_reason = 'Test' WHERE id = $1`, ben); err != nil {
		t.Fatal(err)
	}
	emil := seedSchueler(t, pool, "F-3", "Emil", "8G1")
	emilLmf := seedAusleihe(t, pool, emil, "LMF Bio 8 Emil", stichtag)

	frist := func(id string) time.Time { return fristVon(t, pool, id) }
	erwarte := func(was, id string, soll time.Time) {
		t.Helper()
		if ist := frist(id); !ist.Equal(soll) {
			t.Errorf("%s: Frist %v, erwartet %v", was, ist.In(schulzeit.Zone()), soll)
		}
	}

	// 0. Vorschau schreibt nichts: Plätze kommen, Fristen bleiben.
	rec := lmfPlanAufruf(t, srv, http.MethodPut, "rueckgabe",
		`{"erster_tag":"2027-06-28","startstunde":3,"stunden_je_tag":6,"vorschau":true,"zeilen":[{"klassen":["9H1"]}]}`)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"vorschau":true`) {
		t.Fatalf("Vorschau: %d %s", rec.Code, rec.Body.String())
	}
	erwarte("Annas Schulbuch nach der Vorschau", annaLmf, stichtag)

	// 1. Speichern: Montag 28.06.2027 ab 3. Stunde, 9H1 zuerst.
	rec = lmfPlanAufruf(t, srv, http.MethodPut, "rueckgabe",
		`{"erster_tag":"2027-06-28","startstunde":3,"stunden_je_tag":6,"zeilen":[{"klassen":["9H1"]},{"vermerk":"Bücher setzen"}]}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("speichern: %d %s", rec.Code, rec.Body.String())
	}
	var antwort LmfPlanSpeicherAntwort
	if err := json.Unmarshal(rec.Body.Bytes(), &antwort); err != nil {
		t.Fatal(err)
	}
	if antwort.FristenAngepasst != 1 {
		t.Errorf("genau Annas Schulbuch folgt dem Termin, gemeldet: %d", antwort.FristenAngepasst)
	}
	if len(antwort.Zeilen) != 2 || antwort.Zeilen[0].Datum != "2027-06-28" || antwort.Zeilen[0].Stunde != 3 {
		t.Errorf("Zeilen der Antwort: %+v", antwort.Zeilen)
	}
	erwarte("Annas Schulbuch", annaLmf, tag("2027-06-28"))
	erwarte("Annas Roman (kein Lernmittel)", annaRoman, tag("2026-10-01"))
	erwarte("Annas mehrjähriger Atlas", annaMehrjahr, tag("2028-07-31"))
	erwarte("Bens Schulbuch (gesperrt)", benLmf, stichtag)
	erwarte("Emils Schulbuch (8G1)", emilLmf, stichtag)

	// 2. Umschreiben mit späterem ersten Tag: die Frist zieht nach.
	rec = lmfPlanAufruf(t, srv, http.MethodPut, "rueckgabe",
		`{"erster_tag":"2027-06-30","startstunde":1,"stunden_je_tag":6,"zeilen":[{"klassen":["9H1"]}]}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("umschreiben: %d %s", rec.Code, rec.Body.String())
	}
	erwarte("Annas Schulbuch nach Verschiebung", annaLmf, tag("2027-06-30"))

	// 3. Klasse tauschen: 9H1 fällt aus dem Plan (zurück zum Stichtag), 8G1 kommt hinein
	//    — 8G1 in Zeile 7 liegt am Donnerstag 01.07. (6 Stunden je Tag).
	rec = lmfPlanAufruf(t, srv, http.MethodPut, "rueckgabe",
		`{"erster_tag":"2027-06-30","startstunde":1,"stunden_je_tag":6,"zeilen":[`+
			`{"vermerk":"1"},{"vermerk":"2"},{"vermerk":"3"},{"vermerk":"4"},{"vermerk":"5"},{"vermerk":"6"},{"klassen":["8G1"]}]}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("Klasse tauschen: %d %s", rec.Code, rec.Body.String())
	}
	erwarte("Annas Schulbuch ohne Termin", annaLmf, stichtag)
	erwarte("Emils Schulbuch mit Termin", emilLmf, tag("2027-07-01"))

	// 4. Ein Ausgabe-Plan setzt keine Frist — auch nicht für 8G1.
	rec = lmfPlanAufruf(t, srv, http.MethodPut, "ausgabe",
		`{"erster_tag":"2027-08-10","startstunde":2,"stunden_je_tag":6,"zeilen":[{"klassen":["8G1"],"vermerk":"neu"}]}`)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"fristen_angepasst":0`) {
		t.Fatalf("Ausgabe-Plan: %d %s", rec.Code, rec.Body.String())
	}
	erwarte("Emils Schulbuch nach dem Ausgabe-Plan", emilLmf, tag("2027-07-01"))

	// 5. Verwerfen: Rückweg, aber nur für Fristen, die auf dem Termin lagen.
	if _, err := pool.Exec(ctx, `UPDATE ausleihen SET rueckgabe_frist = $1 WHERE id = $2`, tag("2027-05-15"), emilLmf); err != nil {
		t.Fatal(err)
	}
	rec = lmfPlanAufruf(t, srv, http.MethodDelete, "rueckgabe", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("verwerfen: %d %s", rec.Code, rec.Body.String())
	}
	erwarte("Emils von Hand gesetzte Frist überlebt das Verwerfen", emilLmf, tag("2027-05-15"))
	if rec = lmfPlanAufruf(t, srv, http.MethodDelete, "rueckgabe", ""); rec.Code != http.StatusNotFound {
		t.Errorf("zweites Verwerfen: %d", rec.Code)
	}

	// Gegenprobe des Rückwegs: Frist auf dem Termin-Tag → Stichtag, und die Antwort zählt sie.
	rec = lmfPlanAufruf(t, srv, http.MethodPut, "rueckgabe",
		`{"erster_tag":"2027-06-28","startstunde":1,"stunden_je_tag":6,"zeilen":[{"klassen":["9H1"]}]}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("dritter Plan: %d %s", rec.Code, rec.Body.String())
	}
	erwarte("Annas Schulbuch am dritten Plan", annaLmf, tag("2027-06-28"))
	if rec = lmfPlanAufruf(t, srv, http.MethodDelete, "rueckgabe", ""); rec.Code != http.StatusOK {
		t.Fatalf("verwerfen: %d %s", rec.Code, rec.Body.String())
	}
	erwarte("Annas Schulbuch nach dem Verwerfen", annaLmf, stichtag)
	if !strings.Contains(rec.Body.String(), `"fristen_angepasst":1`) {
		t.Errorf("Verwerfen nennt die zurückgesetzten Fristen nicht: %s", rec.Body.String())
	}

	// 6. Fachliche Ablehnung: Zeile ohne Klasse und Vermerk, falsche Art, Startstunde > Stunden je Tag.
	for _, fall := range []struct{ art, body string }{
		{"rueckgabe", `{"erster_tag":"2027-06-28","startstunde":1,"stunden_je_tag":6,"zeilen":[{"klassen":[],"vermerk":""}]}`},
		{"egal", `{"erster_tag":"2027-06-28","startstunde":1,"stunden_je_tag":6,"zeilen":[]}`},
		{"rueckgabe", `{"erster_tag":"2027-06-28","startstunde":7,"stunden_je_tag":6,"zeilen":[]}`},
	} {
		if rec = lmfPlanAufruf(t, srv, http.MethodPut, fall.art, fall.body); rec.Code != http.StatusBadRequest {
			t.Errorf("%s %s: %d, erwartet 400", fall.art, fall.body, rec.Code)
		}
	}
}

// Der Vorschlag für einen neuen Plan: ohne Vorjahr aus der Regel (Abschluss zuerst,
// Oberstufe ausgelassen); mit vorbeigegangenem Vorjahr dessen Reihenfolge plus neue Klassen.
func TestLmfPlan_VorschlagAusVorjahrOderRegel(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	ctx := context.Background()
	srv := &Server{DB: &db.Database{Pool: pool}}
	t.Cleanup(func() {
		if _, err := pool.Exec(ctx, `DELETE FROM lmf_plaene`); err != nil {
			t.Logf("Aufräumen: %v", err)
		}
	})
	seedSchueler(t, pool, "V-1", "A", "8G1")
	seedSchueler(t, pool, "V-2", "B", "9H1")
	seedSchueler(t, pool, "V-3", "C", "Q1")
	seedSchueler(t, pool, "V-4", "D", "12T1")

	lies := func() LmfPlanStandAntwort {
		t.Helper()
		rec := lmfPlanAufruf(t, srv, http.MethodGet, "rueckgabe", "")
		if rec.Code != http.StatusOK {
			t.Fatalf("lesen: %d %s", rec.Code, rec.Body.String())
		}
		var a LmfPlanStandAntwort
		if err := json.Unmarshal(rec.Body.Bytes(), &a); err != nil {
			t.Fatal(err)
		}
		return a
	}
	a := lies()
	if a.Plan != nil || a.Vorschlag == nil || a.Vorschlag.Quelle != "regel" {
		t.Fatalf("ohne Plan: %+v", a)
	}
	// Das Vokabular zeigt Klassen in seiner Anzeigeform (Migration 087: „09H1").
	if klassenFolge(a.Vorschlag.Zeilen) != "09H1,08G1" {
		t.Errorf("Regel-Reihenfolge (Abschluss zuerst): %+v", a.Vorschlag.Zeilen)
	}
	if strings.Join(a.Vorschlag.Ausgelassen, ",") != "Q1,12T1" {
		t.Errorf("Oberstufe ausgelassen (Jahrgang absteigend, ohne Ziffer zuerst): %v", a.Vorschlag.Ausgelassen)
	}

	// Ein Plan in der Vergangenheit (2020): vorbei → Vorschlag aus dem Vorjahr, in dessen
	// Reihenfolge (8G1 vor 9H1), ergänzt um die neue Klasse 7R1 am Ende.
	rec := lmfPlanAufruf(t, srv, http.MethodPut, "rueckgabe",
		`{"erster_tag":"2020-06-15","startstunde":1,"stunden_je_tag":6,"ausgelassen":["Q1","12T1"],"zeilen":[{"klassen":["8G1"]},{"klassen":["9H1"],"vermerk":"bis 11. eingesammelt"}]}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("Vorjahresplan: %d %s", rec.Code, rec.Body.String())
	}
	seedSchueler(t, pool, "V-5", "E", "7R1")
	a = lies()
	if a.Plan == nil || !a.Vorbei || a.Vorschlag == nil || a.Vorschlag.Quelle != "vorjahr" {
		t.Fatalf("mit vergangenem Plan: plan=%v vorbei=%v vorschlag=%+v", a.Plan != nil, a.Vorbei, a.Vorschlag)
	}
	z := a.Vorschlag.Zeilen
	if klassenFolge(z) != "08G1,09H1,07R1" || z[1].Vermerk != "bis 11. eingesammelt" {
		t.Errorf("Vorjahres-Reihenfolge plus neue Klasse: %+v", z)
	}
	if strings.Join(a.Vorschlag.Ausgelassen, ",") != "12T1,Q1" {
		t.Errorf("Auslassungen übernommen (aus dem Plan, nach Normschlüssel): %v", a.Vorschlag.Ausgelassen)
	}
	if len(a.Klassen) != 5 {
		t.Errorf("Klassen des Vokabulars: %v", a.Klassen)
	}
}

// klassenFolge nennt die erste Klasse jeder Zeile, kommagetrennt.
func klassenFolge(zeilen []repository.LmfPlanZeile) string {
	var teile []string
	for _, z := range zeilen {
		teile = append(teile, strings.Join(z.Klassen, "/"))
	}
	return strings.Join(teile, ",")
}

// lmfPlanAufruf ruft die Plan-Handler wie der Router: {art} im Pfad.
func lmfPlanAufruf(t *testing.T, srv *Server, methode, art, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(methode, "/api/lmf-plan/"+art, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("art", art)
	rec := httptest.NewRecorder()
	switch methode {
	case http.MethodGet:
		srv.GetLmfPlanHandler()(rec, req)
	case http.MethodPut:
		srv.PutLmfPlanHandler()(rec, req)
	default:
		srv.DeleteLmfPlanHandler()(rec, req)
	}
	return rec
}

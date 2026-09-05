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
)

// Die Kopplung Plan → Frist am echten Postgres, über die Handler (Register, Entscheidung 3a):
// Ein Rückgabe-Termin setzt die Frist der offenen Schulbücher seiner Klassen; eine
// Änderung zieht nach; Klassen, die den Termin verlieren, kehren zum Stichtag zurück.
// Nicht angefasst: Nicht-Lernmittel, gesperrte Schüler, mehrjährige Ausleihen, andere Klassen.
func TestLmfTermin_RueckgabeTerminIstDieFristDerKlasse(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	ctx := context.Background()
	srv := &Server{DB: &db.Database{Pool: pool}}

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

	// 1. Anlegen: Rückgabe 28.06.2027 für 9H1.
	rec := lmfTerminAufruf(t, srv, http.MethodPost, "", `{"datum":"2027-06-28","stunde":3,"art":"rueckgabe","klassen":["9H1"],"vermerk":""}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("anlegen: %d %s", rec.Code, rec.Body.String())
	}
	var antwort LmfTerminAntwort
	if err := json.Unmarshal(rec.Body.Bytes(), &antwort); err != nil {
		t.Fatal(err)
	}
	if antwort.FristenAngepasst != 1 {
		t.Errorf("genau Annas Schulbuch folgt dem Termin, gemeldet: %d", antwort.FristenAngepasst)
	}
	erwarte("Annas Schulbuch", annaLmf, tag("2027-06-28"))
	erwarte("Annas Roman (kein Lernmittel)", annaRoman, tag("2026-10-01"))
	erwarte("Annas mehrjähriger Atlas", annaMehrjahr, tag("2028-07-31"))
	erwarte("Bens Schulbuch (gesperrt)", benLmf, stichtag)
	erwarte("Emils Schulbuch (8G1)", emilLmf, stichtag)

	// 2. Datum ändern: die Frist zieht nach.
	rec = lmfTerminAufruf(t, srv, http.MethodPut, antwort.ID, `{"datum":"2027-06-30","stunde":3,"art":"rueckgabe","klassen":["9H1"],"vermerk":""}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("ändern: %d %s", rec.Code, rec.Body.String())
	}
	erwarte("Annas Schulbuch nach Verschiebung", annaLmf, tag("2027-06-30"))

	// 3. Klasse tauschen: 9H1 verliert den Termin (zurück zum Stichtag), 8G1 bekommt ihn.
	rec = lmfTerminAufruf(t, srv, http.MethodPut, antwort.ID, `{"datum":"2027-06-30","stunde":3,"art":"rueckgabe","klassen":["8G1"],"vermerk":""}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("Klasse tauschen: %d %s", rec.Code, rec.Body.String())
	}
	erwarte("Annas Schulbuch ohne Termin", annaLmf, stichtag)
	erwarte("Emils Schulbuch mit Termin", emilLmf, tag("2027-06-30"))

	// 4. Art auf Ausgabe: Ausgabe-Zeilen setzen keine Frist — 8G1 fällt zurück.
	rec = lmfTerminAufruf(t, srv, http.MethodPut, antwort.ID, `{"datum":"2027-08-10","stunde":2,"art":"ausgabe","klassen":["8G1"],"vermerk":"neu"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("Art wechseln: %d %s", rec.Code, rec.Body.String())
	}
	erwarte("Emils Schulbuch nach Wechsel auf Ausgabe", emilLmf, stichtag)

	// 5. Löschen eines Rückgabe-Termins: Rückweg, aber nur für Fristen, die auf ihm lagen.
	rec = lmfTerminAufruf(t, srv, http.MethodPost, "", `{"datum":"2027-06-28","stunde":1,"art":"rueckgabe","klassen":["9H1"],"vermerk":""}`)
	if err := json.Unmarshal(rec.Body.Bytes(), &antwort); err != nil || rec.Code != http.StatusCreated {
		t.Fatalf("zweiter Termin: %d %s (%v)", rec.Code, rec.Body.String(), err)
	}
	erwarte("Annas Schulbuch am zweiten Termin", annaLmf, tag("2027-06-28"))
	// Von Hand gesetzte Frist bleibt beim Löschen stehen (liegt nicht auf dem Termin-Tag).
	if _, err := pool.Exec(ctx, `UPDATE ausleihen SET rueckgabe_frist = $1 WHERE id = $2`, tag("2027-05-15"), annaLmf); err != nil {
		t.Fatal(err)
	}
	rec = lmfTerminAufruf(t, srv, http.MethodDelete, antwort.ID, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("löschen: %d %s", rec.Code, rec.Body.String())
	}
	erwarte("Annas von Hand gesetzte Frist überlebt das Löschen", annaLmf, tag("2027-05-15"))

	// Gegenprobe des Rückwegs: Frist auf dem Termin-Tag → Stichtag.
	rec = lmfTerminAufruf(t, srv, http.MethodPost, "", `{"datum":"2027-06-28","stunde":1,"art":"rueckgabe","klassen":["9H1"],"vermerk":""}`)
	if err := json.Unmarshal(rec.Body.Bytes(), &antwort); err != nil {
		t.Fatal(err)
	}
	erwarte("Annas Schulbuch am dritten Termin", annaLmf, tag("2027-06-28"))
	if rec = lmfTerminAufruf(t, srv, http.MethodDelete, antwort.ID, ""); rec.Code != http.StatusOK {
		t.Fatalf("löschen: %d %s", rec.Code, rec.Body.String())
	}
	erwarte("Annas Schulbuch nach dem Löschen", annaLmf, stichtag)
	if !strings.Contains(rec.Body.String(), `"fristen_angepasst":1`) {
		t.Errorf("Löschantwort nennt die zurückgesetzten Fristen nicht: %s", rec.Body.String())
	}
}

// lmfTerminAufruf ruft die Plan-Handler wie der Router: POST ohne id, PUT/DELETE mit {id}.
func lmfTerminAufruf(t *testing.T, srv *Server, methode, id, body string) *httptest.ResponseRecorder {
	t.Helper()
	pfad := "/api/lmf-termine"
	if id != "" {
		pfad += "/" + id
	}
	req := httptest.NewRequest(methode, pfad, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if id != "" {
		req.SetPathValue("id", id)
	}
	rec := httptest.NewRecorder()
	switch methode {
	case http.MethodPost:
		srv.CreateLmfTerminHandler()(rec, req)
	case http.MethodPut:
		srv.UpdateLmfTerminHandler()(rec, req)
	default:
		srv.DeleteLmfTerminHandler()(rec, req)
	}
	return rec
}

package mahnwesen

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"bibliothek/api/tuer"
	"bibliothek/db"
	"bibliothek/internal/pgtest"
	"bibliothek/internal/testdaten"
	"bibliothek/internal/tuertest"
)

// Die Abgängerliste meint Abschlussklassen mit offenen Büchern, noch an der Schule, zum
// Einsammeln vor der Entlassung, nicht ist_abgaenger („fehlt im LUSD-Export"), einen anderen
// Begriff unter demselben Namen. Diese Tests halten die Bedeutung am echten SQL fest.

// abgaengerAntwort ruft GET /api/abgaenger — die Ansicht, aus der heraus gedruckt und
// versendet wird — und liefert die ganze Antwort samt Saisonfenster.
func abgaengerAntwort(t *testing.T, srv *Tueren) AbgaengerAntwort {
	t.Helper()
	rec := httptest.NewRecorder()
	srv.Abgaenger()(rec, httptest.NewRequest(http.MethodGet, "/api/abgaenger", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("Abgängerliste: Status %d — %s", rec.Code, rec.Body.String())
	}
	var antwort AbgaengerAntwort
	if err := json.Unmarshal(rec.Body.Bytes(), &antwort); err != nil {
		t.Fatalf("Abgängerliste unlesbar: %v", err)
	}
	return antwort
}

func TestAbgaengerliste_AbschlussklassenMitOffenenBuechern(t *testing.T) {
	pool := pgtest.Pool(t)
	testdaten.LeereBestandsdaten(t, pool)

	// Drin: Abschlussklassen mit Buch, in jeder Schreibweise, die das Vokabular zulässt.
	testdaten.LegeSchuelerMitAusleiheAn(t, pool, "S-1", "Anna", "9H1")
	testdaten.LegeSchuelerMitAusleiheAn(t, pool, "S-2", "Bea", "09h2")
	testdaten.LegeSchuelerMitAusleiheAn(t, pool, "S-3", "Cem", "10R1")
	testdaten.LegeSchuelerMitAusleiheAn(t, pool, "S-4", "Dana", "13")
	testdaten.LegeSchuelerMitAusleiheAn(t, pool, "S-5", "Emil", "10H1")
	// Draußen: kein Abschlussjahrgang, obwohl mit Buch.
	testdaten.LegeSchuelerMitAusleiheAn(t, pool, "S-6", "Finn", "10G1")
	testdaten.LegeSchuelerMitAusleiheAn(t, pool, "S-7", "Gül", "8H1")
	testdaten.LegeSchuelerMitAusleiheAn(t, pool, "S-8", "Hans", "9R1")
	// Draußen: Abschlussklasse, aber nichts mehr offen — nichts einzusammeln.
	testdaten.LegeSchuelerAn(t, pool, "S-9", "Ida", "9H1")
	// Draußen: laut LUSD schon weg (ist_abgaenger, Klasse ABG) — mit Buch, aber das ist
	// Sache des Mahnwesens, nicht dieser Liste.
	weg := testdaten.LegeAbgaengerAn(t, pool, "S-10", "Jo", "ABG")
	testdaten.LegeAusleiheMitFristAn(t, pool, weg, "Buch Jo", time.Now().AddDate(0, 0, -3))

	srv := &Tueren{Basis: &tuer.Basis{DB: &db.Verbindung{Pool: pool}, Uhr: testdaten.SaisonUhr}}
	antwort := abgaengerAntwort(t, srv)
	if !antwort.Fenster.Offen {
		t.Fatalf("Mitte Juni muss die Saison offen sein: %+v", antwort.Fenster)
	}
	inListe := map[string]bool{}
	for _, z := range antwort.Abgaenger {
		inListe[z.Vorname] = true
	}
	imKonto := kontoauszugNamen(t, srv)

	for _, name := range []string{"Anna", "Bea", "Cem", "Dana", "Emil"} {
		if !inListe[name] || !imKonto[name] {
			t.Errorf("%s (Abschlussklasse, offenes Buch) fehlt: Liste=%v Kontoauszug=%v", name, inListe[name], imKonto[name])
		}
	}
	for _, name := range []string{"Finn", "Gül", "Hans", "Ida", "Jo"} {
		if inListe[name] || imKonto[name] {
			t.Errorf("%s steht fälschlich in Liste=%v / Kontoauszug=%v", name, inListe[name], imKonto[name])
		}
	}
	if len(antwort.Abgaenger) != 5 {
		t.Errorf("erwartet 5 Zeilen, waren %d", len(antwort.Abgaenger))
	}
}

// Außerhalb der Saison ist die Liste leer und sagt warum — Druck und Versand folgen ihr.
// Gegenprobe im selben Test: dieselben Daten, Uhr auf Juni, und die Zeile ist da.
func TestAbgaengerliste_AusserhalbDerSaison(t *testing.T) {
	pool := pgtest.Pool(t)
	testdaten.LeereBestandsdaten(t, pool)
	tuertest.FangeMailAb(t) // SMTP konfiguriert, damit der Versand bis zur Saisonprüfung kommt

	testdaten.LegeSchuelerMitAusleiheAn(t, pool, "S-1", "Anna", "9H1")
	srv := &Tueren{Basis: &tuer.Basis{DB: &db.Verbindung{Pool: pool}, Uhr: testdaten.WinterUhr}}

	antwort := abgaengerAntwort(t, srv)
	if antwort.Fenster.Offen || len(antwort.Abgaenger) != 0 {
		t.Fatalf("im Oktober muss die Liste leer und das Fenster zu sein: %+v", antwort)
	}
	if antwort.Fenster.Von != "01.05." || antwort.Fenster.Bis != "31.07." {
		t.Errorf("die Oberfläche braucht beide Daten für den Hinweis: %+v", antwort.Fenster)
	}

	rec := httptest.NewRecorder()
	srv.AbgaengerKontoauszuegePDF()(rec, httptest.NewRequest(http.MethodGet, "/api/abgaenger/pdf", nil))
	if rec.Code != http.StatusNotFound || !strings.Contains(rec.Body.String(), "01.05.") {
		t.Errorf("PDF außerhalb der Saison: Status %d — %s (erwartet 404 mit den Saisondaten)", rec.Code, rec.Body.String())
	}

	req := httptest.NewRequest(http.MethodPost, "/api/abgaenger/mail", strings.NewReader(`{"klassen":["9H1"]}`))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	srv.AbgaengerKontoauszuegeVersenden()(rec, req)
	if rec.Code != http.StatusConflict {
		t.Errorf("Versand außerhalb der Saison: Status %d — %s (erwartet 409)", rec.Code, rec.Body.String())
	}

	srv.Uhr = testdaten.SaisonUhr
	if antwort = abgaengerAntwort(t, srv); len(antwort.Abgaenger) != 1 || antwort.Abgaenger[0].Vorname != "Anna" {
		t.Errorf("Gegenprobe im Juni: Anna fehlt — %+v", antwort)
	}
}

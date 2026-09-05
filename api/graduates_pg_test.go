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
	"bibliothek/pkg/schulzeit"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Die Abgängerliste in ihrer ursprünglichen Bedeutung (Register, Entscheidung 2 vom
// 05.09.2026): Abschlussklassen mit offenen Büchern, noch an der Schule, zum Einsammeln
// vor der Entlassung. Vom 25.06. bis 05.09.2026 filterte sie stattdessen ist_abgaenger
// („fehlt im LUSD-Export") — ein anderer Begriff unter demselben Namen, den jede Sitzung
// als Wahrheit übernahm. Diese Tests halten die Bedeutung am echten SQL fest.

// saisonUhr und winterUhr sind feste Uhren für die Liste. Ohne sie hinge jeder Test hier
// am Kalender des Rechners: acht Monate im Jahr rot — oder grün aus dem falschen Grund.
func saisonUhr() time.Time { return time.Date(2026, time.June, 15, 10, 0, 0, 0, schulzeit.Zone()) }
func winterUhr() time.Time { return time.Date(2026, time.October, 15, 10, 0, 0, 0, schulzeit.Zone()) }

// abgaengerAntwort ruft GET /api/abgaenger — die Ansicht, aus der heraus gedruckt und
// versendet wird — und liefert die ganze Antwort samt Saisonfenster.
func abgaengerAntwort(t *testing.T, srv *Server) AbgaengerAntwort {
	t.Helper()
	rec := httptest.NewRecorder()
	srv.GetGraduatesHandler()(rec, httptest.NewRequest(http.MethodGet, "/api/abgaenger", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("Abgängerliste: Status %d — %s", rec.Code, rec.Body.String())
	}
	var antwort AbgaengerAntwort
	if err := json.Unmarshal(rec.Body.Bytes(), &antwort); err != nil {
		t.Fatalf("Abgängerliste unlesbar: %v", err)
	}
	return antwort
}

// schuelerMitBuch legt einen aktiven Schüler mit genau einem offenen Buch an.
func schuelerMitBuch(t *testing.T, pool *pgxpool.Pool, barcode, vorname, klasse string) string {
	t.Helper()
	id := seedSchueler(t, pool, barcode, vorname, klasse)
	seedAusleihe(t, pool, id, "Buch "+vorname, time.Now().AddDate(0, 0, -3))
	return id
}

func TestAbgaengerliste_AbschlussklassenMitOffenenBuechern(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)

	// Drin: Abschlussklassen mit Buch, in jeder Schreibweise, die das Vokabular zulässt.
	schuelerMitBuch(t, pool, "S-1", "Anna", "9H1")
	schuelerMitBuch(t, pool, "S-2", "Bea", "09h2")
	schuelerMitBuch(t, pool, "S-3", "Cem", "10R1")
	schuelerMitBuch(t, pool, "S-4", "Dana", "13")
	schuelerMitBuch(t, pool, "S-5", "Emil", "10H1")
	// Draußen: kein Abschlussjahrgang, obwohl mit Buch.
	schuelerMitBuch(t, pool, "S-6", "Finn", "10G1")
	schuelerMitBuch(t, pool, "S-7", "Gül", "8H1")
	schuelerMitBuch(t, pool, "S-8", "Hans", "9R1")
	// Draußen: Abschlussklasse, aber nichts mehr offen — nichts einzusammeln.
	seedSchueler(t, pool, "S-9", "Ida", "9H1")
	// Draußen: laut LUSD schon weg (ist_abgaenger, Klasse ABG) — mit Buch, aber das ist
	// Sache des Mahnwesens, nicht dieser Liste.
	weg := seedAbgaengerRet(t, pool, "S-10", "Jo", "ABG")
	seedAusleihe(t, pool, weg, "Buch Jo", time.Now().AddDate(0, 0, -3))

	srv := &Server{DB: &db.Database{Pool: pool}, Uhr: saisonUhr}
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
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	mailAbfangen(t) // SMTP konfiguriert, damit der Versand bis zur Saisonprüfung kommt

	schuelerMitBuch(t, pool, "S-1", "Anna", "9H1")
	srv := &Server{DB: &db.Database{Pool: pool}, Uhr: winterUhr}

	antwort := abgaengerAntwort(t, srv)
	if antwort.Fenster.Offen || len(antwort.Abgaenger) != 0 {
		t.Fatalf("im Oktober muss die Liste leer und das Fenster zu sein: %+v", antwort)
	}
	if antwort.Fenster.Von != "01.05." || antwort.Fenster.Bis != "31.07." {
		t.Errorf("die Oberfläche braucht beide Daten für den Hinweis: %+v", antwort.Fenster)
	}

	rec := httptest.NewRecorder()
	srv.GetGraduatesPDFHandler()(rec, httptest.NewRequest(http.MethodGet, "/api/abgaenger/pdf", nil))
	if rec.Code != http.StatusNotFound || !strings.Contains(rec.Body.String(), "01.05.") {
		t.Errorf("PDF außerhalb der Saison: Status %d — %s (erwartet 404 mit den Saisondaten)", rec.Code, rec.Body.String())
	}

	req := httptest.NewRequest(http.MethodPost, "/api/abgaenger/mail", strings.NewReader(`{"klassen":["9H1"]}`))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	srv.SendAbgaengerKontoauszuegeHandler()(rec, req)
	if rec.Code != http.StatusConflict {
		t.Errorf("Versand außerhalb der Saison: Status %d — %s (erwartet 409)", rec.Code, rec.Body.String())
	}

	srv.Uhr = saisonUhr
	if antwort = abgaengerAntwort(t, srv); len(antwort.Abgaenger) != 1 || antwort.Abgaenger[0].Vorname != "Anna" {
		t.Errorf("Gegenprobe im Juni: Anna fehlt — %+v", antwort)
	}
}

// Paar-Gate: Die Abgängerliste und die Versetzung müssen dieselbe Menge sehen. Die Liste
// zeigt im Juni, wen man einsammelt; die Versetzung markiert im Sommer, wer geht. Weichen
// sie ab, bekommt eine Klasse Kontoauszüge und bleibt — oder geht ohne Kontoauszug.
// Beide lesen repository.AbschlussklasseSQL; dieser Test hält fest, dass das so bleibt.
func TestAbgaengerliste_UndVersetzungSehenDieselbeMenge(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	ctx := context.Background()

	for i, k := range []string{"9H1", "09h2", "10H1", "10R1", "10R2", "13", "10G1", "8H1", "9R1", "5F1", "12"} {
		schuelerMitBuch(t, pool, "P-"+string(rune('A'+i)), "Kind"+k, k)
	}

	srv := &Server{DB: &db.Database{Pool: pool}, Uhr: saisonUhr}
	ausListe := map[string]bool{}
	for _, z := range abgaengerAntwort(t, srv).Abgaenger {
		ausListe[z.Vorname] = true
	}

	versetzungAusfuehren(t, pool, false)
	rows, err := pool.Query(ctx, `SELECT vorname FROM schueler WHERE ist_abgaenger = true`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	ausVersetzung := map[string]bool{}
	for rows.Next() {
		var v string
		if err := rows.Scan(&v); err != nil {
			t.Fatal(err)
		}
		ausVersetzung[v] = true
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}

	if len(ausListe) == 0 {
		t.Fatal("die Liste ist leer — das Gate vergliche zwei Nullmengen")
	}
	for name := range ausListe {
		if !ausVersetzung[name] {
			t.Errorf("%s steht in der Abgängerliste, die Versetzung lässt ihn aber bleiben", name)
		}
	}
	for name := range ausVersetzung {
		if !ausListe[name] {
			t.Errorf("die Versetzung schickt %s von der Schule, die Abgängerliste hat ihn nie gezeigt", name)
		}
	}
}

package katalog

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"bibliothek/api/tuer"
	"bibliothek/db"
	"bibliothek/internal/pgtest"
	"bibliothek/internal/testdaten"
	"bibliothek/internal/tuertest"
	"bibliothek/pkg/schulzeit"
	"bibliothek/repository"

	"github.com/jackc/pgx/v5/pgxpool"
)

// titelMitVerliehenemExemplar legt einen Titel mit zwei Exemplaren an, eines davon an Lotte (07A)
// verliehen, dazu das Konto, das löscht.
func titelMitVerliehenemExemplar(t *testing.T) (pool *pgxpool.Pool, adminID, titelID, verliehenID string) {
	t.Helper()
	p := pgtest.Pool(t)
	testdaten.LeereBestandsdaten(t, p)
	adminID = testdaten.LegeKollegiumskontoAn(t, p, "ada@titelloesch.invalid")
	titelID = testdaten.LegeTitelAn(t, p, "Titel mit verliehenem Exemplar", 0)
	verliehenID = testdaten.LegeExemplarAn(t, p, titelID, "B-TITELLOESCH-1", true, "")
	testdaten.LegeExemplarAn(t, p, titelID, "B-TITELLOESCH-2", true, "")
	schuelerID := testdaten.LegeSchuelerAn(t, p, "S-TITELLOESCH-1", "Lotte", "07A")
	testdaten.LegeAusleiheVorZweiTagenAn(t, p, verliehenID, schuelerID)
	return p, adminID, titelID, verliehenID
}

// Die Vorschau nennt, was der Dialog sagt: ein Exemplar im Haus, eines verliehen, mit Leser und
// Klasse und der Kennung, die der Dialog zum Bestätigen zurückschickt.
func TestTitelLoeschfolgen_NenntImHausUndVerliehen(t *testing.T) {
	pool, adminID, titelID, verliehenID := titelMitVerliehenemExemplar(t)
	srv := &Tueren{Basis: &tuer.Basis{DB: &db.Verbindung{Pool: pool}}}

	rec := tuertest.RufeAlsAdmin(t, adminID, srv.TitelLoeschfolgen(),
		http.MethodGet, "/api/buecher/titel/loeschfolgen?ids="+titelID, "", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("Vorschau: Status %d: %s", rec.Code, rec.Body.String())
	}
	var folgen repository.Loeschfolgen
	if err := json.Unmarshal(rec.Body.Bytes(), &folgen); err != nil {
		t.Fatalf("Antwort: %v", err)
	}
	if folgen.Titel != 1 || folgen.ImHaus != 1 || len(folgen.Verliehen) != 1 {
		t.Fatalf("Vorschau: %+v, erwartet ein Titel, ein Exemplar im Haus, eines verliehen", folgen)
	}
	v := folgen.Verliehen[0]
	if v.ExemplarID != verliehenID || v.Barcode != "B-TITELLOESCH-1" || v.Leser != "Lotte Test" || v.Klasse != "07A" {
		t.Errorf("verliehenes Exemplar: %+v", v)
	}

	rec = tuertest.RufeAlsAdmin(t, adminID, srv.TitelLoeschfolgen(),
		http.MethodGet, "/api/buecher/titel/loeschfolgen?ids=keine-kennung", "", nil)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("Vorschau mit ungültiger Kennung: Status %d, erwartet 400", rec.Code)
	}
}

// Ohne Bestätigung bleibt ein verliehenes Exemplar beim Leser und sein Titel stehen; das Exemplar
// im Haus geht mit dem gewählten Grund ins Abgangsbuch. Mit Bestätigung geht alles, und das schon
// ausgesonderte Exemplar behält seinen Grund.
func TestTitelLoeschen_VerlieheneNurMitBestaetigung(t *testing.T) {
	pool, adminID, titelID, verliehenID := titelMitVerliehenemExemplar(t)
	p := pool
	ctx := context.Background()
	srv := &Tueren{Basis: &tuer.Basis{DB: &db.Verbindung{Pool: p}}}
	loesche := func(rumpf string) (int, repository.TitelLoeschergebnis) {
		t.Helper()
		rec := tuertest.RufeAlsAdmin(t, adminID, srv.MarkierteTitelLoeschen(), http.MethodDelete, "/api/books", rumpf, nil)
		var ergebnis repository.TitelLoeschergebnis
		if rec.Code == http.StatusOK {
			if err := json.Unmarshal(rec.Body.Bytes(), &ergebnis); err != nil {
				t.Fatalf("Antwort: %v", err)
			}
		}
		return rec.Code, ergebnis
	}

	code, ergebnis := loesche(`{"ids":["` + titelID + `"],"grund":"BESCHAEDIGUNG","verliehene":[]}`)
	if code != http.StatusOK || len(ergebnis.Geloescht) != 0 || len(ergebnis.Geblieben) != 1 {
		t.Fatalf("ohne Bestätigung: Status %d, %+v, erwartet der Titel unter geblieben", code, ergebnis)
	}
	if n := pgtest.ZaehleZeilen(t, p, `SELECT count(*) FROM ausleihen WHERE exemplar_id = $1 AND rueckgabe_am IS NULL`, verliehenID); n != 1 {
		t.Errorf("ohne Bestätigung: %d offene Ausleihen des verliehenen Exemplars, erwartet 1", n)
	}
	von, bis := schulzeit.Halbjahr(schulzeit.Jetzt())
	buch, err := repository.LadeAbgangsbuch(ctx, p, von, bis)
	if err != nil {
		t.Fatalf("Abgangsbuch: %v", err)
	}
	if len(buch.Zeilen) != 1 || buch.Zeilen[0].Barcode != "B-TITELLOESCH-2" || buch.Zeilen[0].Grund != "BESCHAEDIGUNG" {
		t.Errorf("Abgangsbuch ohne Bestätigung: %+v, erwartet B-TITELLOESCH-2 mit BESCHAEDIGUNG", buch.Zeilen)
	}

	code, ergebnis = loesche(`{"ids":["` + titelID + `"],"grund":"AUSSORTIERT","verliehene":["` + verliehenID + `"]}`)
	if code != http.StatusOK || len(ergebnis.Geloescht) != 1 || len(ergebnis.Geblieben) != 0 {
		t.Fatalf("mit Bestätigung: Status %d, %+v, erwartet der Titel unter gelöscht", code, ergebnis)
	}
	if n := pgtest.ZaehleZeilen(t, p, `SELECT count(*) FROM buecher_titel WHERE id = $1`, titelID); n != 0 {
		t.Errorf("mit Bestätigung: %d Titel stehen noch, erwartet 0", n)
	}
	buch, err = repository.LadeAbgangsbuch(ctx, p, von, bis)
	if err != nil {
		t.Fatalf("Abgangsbuch: %v", err)
	}
	gruende := map[string]string{}
	for _, z := range buch.Zeilen {
		gruende[z.Barcode] = z.Grund
	}
	if gruende["B-TITELLOESCH-1"] != "AUSSORTIERT" || gruende["B-TITELLOESCH-2"] != "BESCHAEDIGUNG" || len(gruende) != 2 {
		t.Errorf("Abgangsbuch mit Bestätigung: %v, erwartet das verliehene mit AUSSORTIERT und das andere mit seinem Grund", gruende)
	}
}

// Ohne einen der vier Gründe, mit ungültigen Kennungen oder zu vielen Titeln lehnt die Tür ab, und
// nichts ist gelöscht.
func TestTitelLoeschen_PrueftDenAuftrag(t *testing.T) {
	pool, adminID, titelID, _ := titelMitVerliehenemExemplar(t)
	p := pool
	srv := &Tueren{Basis: &tuer.Basis{DB: &db.Verbindung{Pool: p}}}
	viele := make([]string, maxTitelJeLoeschen+1)
	for i := range viele {
		viele[i] = `"` + titelID + `"`
	}
	for _, fall := range []struct{ name, rumpf, meldung string }{
		{"ohne Grund", `{"ids":["` + titelID + `"]}`, meldungGrundFehlt},
		{"unbekannter Grund", `{"ids":["` + titelID + `"],"grund":"VERSCHENKT"}`, meldungGrundFehlt},
		{"ohne Titel", `{"ids":[],"grund":"AUSSORTIERT"}`, "Keine Titel ausgewählt"},
		{"verliehene ungültig", `{"ids":["` + titelID + `"],"grund":"AUSSORTIERT","verliehene":["x"]}`, "Eine der Kennungen ist ungültig"},
		{"zu viele", `{"ids":[` + strings.Join(viele, ",") + `],"grund":"AUSSORTIERT"}`, "Höchstens 200 Titel auf einmal"},
	} {
		rec := tuertest.RufeAlsAdmin(t, adminID, srv.MarkierteTitelLoeschen(), http.MethodDelete, "/api/books", fall.rumpf, nil)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("%s: Status %d, erwartet 400", fall.name, rec.Code)
			continue
		}
		if got := tuertest.Fehlermeldung(t, rec); got != fall.meldung {
			t.Errorf("%s: Meldung %q, erwartet %q", fall.name, got, fall.meldung)
		}
	}
	if n := pgtest.ZaehleZeilen(t, p, `SELECT count(*) FROM buecher_exemplare WHERE titel_id = $1 AND NOT ist_ausgesondert`, titelID); n != 2 {
		t.Errorf("nach den Ablehnungen: %d Exemplare im Bestand, erwartet 2", n)
	}
}

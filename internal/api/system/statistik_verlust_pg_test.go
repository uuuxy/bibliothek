package system

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"bibliothek/internal/api/tuer"
	"bibliothek/internal/db"
	"bibliothek/internal/pgtest"
	"bibliothek/internal/repository"
	"bibliothek/internal/testdaten"
)

// Ein bestelltes Exemplar, das nie eintraf und als verloren ausgebucht wurde, ist kein Verlust.
// Die Statistik zählt in der Grenze des Abgangsbuchs; ein Exemplar aus dem Bestand ist die
// Gegenprobe.
func TestStatistik_NieEingetroffenesIstKeinVerlust(t *testing.T) {
	pool := pgtest.Pool(t)
	testdaten.LeereBestandsdaten(t, pool)
	ctx := context.Background()
	srv := &Tueren{Basis: &tuer.Basis{DB: &db.Verbindung{Pool: pool}}}
	titelID := testdaten.LegeTitelMitSignaturAn(t, pool, "Statistik Zulauf", "Sta 1")
	nie := testdaten.LegeExemplarImZulaufAn(t, pool, titelID, "ST-ZULAUF")
	imBestand := testdaten.LegeExemplarAn(t, pool, titelID, "ST-BESTAND", true, "")
	if _, err := pool.Exec(ctx, `UPDATE buecher_exemplare SET einkaufspreis = 20 WHERE id = ANY($1)`,
		[]string{nie, imBestand}); err != nil {
		t.Fatalf("Preis setzen: %v", err)
	}
	buecher := repository.NewKatalogAbfragen(pool)
	for _, id := range []string{nie, imBestand} {
		if err := buecher.AendereExemplarStatus(ctx, id, nil, repository.ExemplarStatusStand{IstAusgesondert: true}, "", nil, testdaten.LegeBearbeiterAn(t, pool)); err != nil {
			t.Fatalf("als verloren ausbuchen: %v", err)
		}
	}

	rec := httptest.NewRecorder()
	srv.Statistik().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/statistiken", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("Status %d: %s", rec.Code, rec.Body.String())
	}
	var antwort struct {
		Verluste struct {
			Verlorene int `json:"verlorene_exemplare"`
		} `json:"loss_stats"`
		Wiederbeschaffung float64 `json:"wiederbeschaffungswert_defekt"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &antwort); err != nil {
		t.Fatalf("Antwort lesen: %v", err)
	}
	if antwort.Verluste.Verlorene != 1 {
		t.Errorf("verlorene Exemplare: %d, erwartet 1 — ST-ZULAUF war nie im Bestand", antwort.Verluste.Verlorene)
	}
	if antwort.Wiederbeschaffung != 20 {
		t.Errorf("Wiederbeschaffungswert: %.2f, erwartet 20.00 — nur ST-BESTAND muss nachgekauft werden", antwort.Wiederbeschaffung)
	}
}

// Ein bestelltes Exemplar zählt erst zum Bestand, wenn es eingetroffen ist. Sonst stünde es im
// Gesamtbestand und unter den aktiven Exemplaren, und die Zirkulationsquote teilte durch es mit.
func TestStatistik_BestelltesZaehltNichtZumBestand(t *testing.T) {
	pool := pgtest.Pool(t)
	testdaten.LeereBestandsdaten(t, pool)
	srv := &Tueren{Basis: &tuer.Basis{DB: &db.Verbindung{Pool: pool}}}
	titelID := testdaten.LegeTitelMitSignaturAn(t, pool, "Statistik Bestand", "Sta 2")
	testdaten.LegeExemplarImZulaufAn(t, pool, titelID, "SB-ZULAUF")
	imBestand := testdaten.LegeExemplarAn(t, pool, titelID, "SB-BESTAND", true, "")
	testdaten.LegeAusleiheVorZweiTagenAn(t, pool, imBestand, testdaten.LegeSchuelerAn(t, pool, "SB-LESER", "Statistik", "07A"))

	rec := httptest.NewRecorder()
	srv.Statistik().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/statistiken", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("Status %d: %s", rec.Code, rec.Body.String())
	}
	var antwort struct {
		Verluste struct {
			Gesamt int `json:"gesamt_bestand"`
		} `json:"loss_stats"`
		Zirkulation struct {
			Verliehen int `json:"aktuell_verliehen"`
			Aktiv     int `json:"aktiver_bestand"`
		} `json:"zirkulation"`
		Quote float64 `json:"zirkulationsquote"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &antwort); err != nil {
		t.Fatalf("Antwort lesen: %v", err)
	}
	if antwort.Verluste.Gesamt != 1 || antwort.Zirkulation.Aktiv != 1 {
		t.Errorf("Gesamtbestand %d, aktive Exemplare %d, erwartet je 1 — SB-ZULAUF ist noch nicht eingetroffen",
			antwort.Verluste.Gesamt, antwort.Zirkulation.Aktiv)
	}
	if antwort.Zirkulation.Verliehen != 1 || antwort.Quote != 100 {
		t.Errorf("verliehen %d, Zirkulationsquote %.2f, erwartet 1 und 100.00",
			antwort.Zirkulation.Verliehen, antwort.Quote)
	}
}

// Ein endgültig gelöschter Verlust zählt in der Statistik weiter, wie im Abgangsbuch: Verluste,
// Wiederbeschaffungswert und Quote bleiben, nur der Gesamtbestand nennt das Exemplar nicht mehr.
// Der Filter Lernmittel/Freihand ordnet die Zeile nach der Art ihres Titels.
func TestStatistik_GeloeschterVerlustZaehltWeiter(t *testing.T) {
	pool := pgtest.Pool(t)
	testdaten.LeereBestandsdaten(t, pool)
	ctx := context.Background()
	srv := &Tueren{Basis: &tuer.Basis{DB: &db.Verbindung{Pool: pool}}}
	lernmittel := testdaten.LegeTitelMitSignaturAn(t, pool, "Statistik Lernmittel", "Sta 3")
	if _, err := pool.Exec(ctx, `UPDATE buecher_titel SET ist_lernmittel = true WHERE id = $1`, lernmittel); err != nil {
		t.Fatalf("Lernmittel setzen: %v", err)
	}
	freihand := testdaten.LegeTitelMitSignaturAn(t, pool, "Statistik Freihand", "Sta 4")
	verloren := testdaten.LegeExemplarAn(t, pool, lernmittel, "SG-VERLOREN", true, "")
	testdaten.LegeExemplarAn(t, pool, lernmittel, "SG-DA", true, "")
	verlorenFrei := testdaten.LegeExemplarAn(t, pool, freihand, "SG-FREI-VERLOREN", true, "")
	if _, err := pool.Exec(ctx, `UPDATE buecher_exemplare SET einkaufspreis = CASE WHEN id = $1 THEN 20 ELSE 10 END
		WHERE id = ANY($2)`, verloren, []string{verloren, verlorenFrei}); err != nil {
		t.Fatalf("Preis setzen: %v", err)
	}
	bearbeiter := testdaten.LegeBearbeiterAn(t, pool)
	buecher := repository.NewKatalogAbfragen(pool)
	for _, id := range []string{verloren, verlorenFrei} {
		if err := buecher.AendereExemplarStatus(ctx, id, nil, repository.ExemplarStatusStand{IstAusgesondert: true}, "", nil, bearbeiter); err != nil {
			t.Fatalf("als verloren ausbuchen: %v", err)
		}
	}

	type zahlen struct {
		Gesamt, Verlorene int
		Quote, Wert       float64
	}
	lies := func(filter string) zahlen {
		t.Helper()
		rec := httptest.NewRecorder()
		srv.Statistik().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/statistiken?type="+filter, nil))
		if rec.Code != http.StatusOK {
			t.Fatalf("Status %d: %s", rec.Code, rec.Body.String())
		}
		var antwort struct {
			Verluste struct {
				Gesamt    int     `json:"gesamt_bestand"`
				Verlorene int     `json:"verlorene_exemplare"`
				Quote     float64 `json:"verlust_quote"`
			} `json:"loss_stats"`
			Wert float64 `json:"wiederbeschaffungswert_defekt"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &antwort); err != nil {
			t.Fatalf("Antwort lesen: %v", err)
		}
		return zahlen{antwort.Verluste.Gesamt, antwort.Verluste.Verlorene, antwort.Verluste.Quote, antwort.Wert}
	}
	erwartet := map[string]zahlen{
		"":         {Gesamt: 3, Verlorene: 2, Quote: 66.67, Wert: 30},
		"lmf":      {Gesamt: 2, Verlorene: 1, Quote: 50, Wert: 20},
		"freihand": {Gesamt: 1, Verlorene: 1, Quote: 100, Wert: 10},
	}
	for filter, soll := range erwartet {
		if ist := lies(filter); ist != soll {
			t.Fatalf("vor dem Löschen, Filter %q: %+v, erwartet %+v", filter, ist, soll)
		}
	}

	geloescht, err := repository.NewInventurAbfragen(pool).LoescheVerlusteEndgueltig(ctx, []string{verloren}, bearbeiter)
	if err != nil || len(geloescht) != 1 {
		t.Fatalf("endgültig löschen: %v, %d gelöscht", err, len(geloescht))
	}

	for filter, soll := range erwartet {
		if filter != "freihand" {
			soll.Gesamt-- // Exemplare im System: das gelöschte fehlt
		}
		if ist := lies(filter); ist != soll {
			t.Errorf("nach dem Löschen, Filter %q: %+v, erwartet %+v — der gelöschte Verlust zählt weiter", filter, ist, soll)
		}
	}
}

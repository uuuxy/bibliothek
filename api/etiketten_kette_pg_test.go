package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"bibliothek/api/druck"
	"bibliothek/api/tuer"
	"bibliothek/db"
	"bibliothek/internal/pgtest"
	"bibliothek/internal/service"
	"bibliothek/internal/smtptest"
	"bibliothek/internal/testdaten"
	"bibliothek/internal/tuertest"
	"bibliothek/pkg/schulzeit"
	"bibliothek/repository"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Die Etiketten-Kette: Wareneingang → Etikettenstatus → Nachdruck-Liste.
//
// Die Glieder sind einzeln geprüft, die Übergänge nicht, und dort sitzt hier eine ganze Reihe
// stiller Ausfälle:
//
//   - Der Wareneingang schreibt an denselben Zeilen (ist_ausleihbar, zustand_notiz). Nähme
//     er etikett_gedruckt mit, verschwänden frisch gelieferte Bücher ohne Aufkleber
//     lautlos aus der Nachdruck-Liste — niemand vermisst, was nie in der Liste stand.
//   - Liste und Zähler sind zwei Abfragen. Die Bedingung teilen sie sich
//     (repository.EtikettOffenBedingung), den Rest nicht; driften sie auseinander, nennt der
//     Hinweis im Bestellwesen eine Zahl, die zu keiner Liste passt.
//   - Der Hauptlieferant klebt selbst. Seine Exemplare dürfen nach dem Wareneingang nicht
//     auf der Nachdruck-Liste stehen, sonst klebt die Bibliothek ein zweites Etikett auf
//     ein bereits beklebtes Buch.

// jsonHolen ruft eine Tür und liest die Antwort in ziel.
func jsonHolen(t *testing.T, handler http.HandlerFunc, pfad string, ziel any) {
	t.Helper()
	rec := httptest.NewRecorder()
	handler(rec, httptest.NewRequest(http.MethodGet, pfad, nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET %s: Status %d — %s", pfad, rec.Code, rec.Body.String())
	}
	if err := json.Unmarshal(rec.Body.Bytes(), ziel); err != nil {
		t.Fatalf("GET %s: Antwort unlesbar: %v", pfad, err)
	}
}

// offeneEtikettenUeberTuer liefert Liste und Zähler — die beiden Wege zur selben Frage.
//
// Bewusst über die Türen und nicht über eigenes SQL: Der Helfer in
// internal/service/haendler_beklebt_pg_test.go schreibt die Bedingung noch einmal selbst hin
// („Bewusst dieselbe Bedingung wie repository.EtikettOffenBedingung") und kann eine Abweichung
// im Programmcode deshalb gar nicht sehen.
func offeneEtikettenUeberTuer(t *testing.T, srv *Server) (liste []druck.ExemplarOhneEtikett, anzahl int) {
	t.Helper()
	jsonHolen(t, srv.druck().EtikettenOffen(), "/api/exemplare/etiketten-offen", &liste)

	var zaehler struct {
		Anzahl int `json:"anzahl"`
	}
	jsonHolen(t, srv.druck().EtikettenOffenAnzahl(), "/api/exemplare/etiketten-offen/anzahl", &zaehler)
	return liste, zaehler.Anzahl
}

func enthaeltBarcode(liste []druck.ExemplarOhneEtikett, barcode string) bool {
	for _, e := range liste {
		if e.BarcodeID == barcode {
			return true
		}
	}
	return false
}

// wareneingang bucht alle noch nicht freigegebenen Exemplare ein — wie der Knopf im
// Bestellwesen, über denselben Dienst.
func wareneingang(t *testing.T, pool *pgxpool.Pool) []service.EingebuchtesExemplar {
	t.Helper()
	ctx := context.Background()
	ids := testdaten.LadeExemplarIDsDerBestellung(t, pool)
	if len(ids) == 0 {
		t.Fatal("nichts einzubuchen — der Testaufbau hat keine Exemplare erzeugt")
	}
	items, err := service.BucheWareneingang(ctx, pool, repository.NewProtokollAbfragen(pool), service.WareneingangAngaben{
		ExemplarIDs:  ids,
		BearbeiterID: "",
		IPAdresse:    "127.0.0.1",
	})
	if err != nil {
		t.Fatalf("Wareneingang: %v", err)
	}
	return items
}

func TestEtikettenkette_NormalerLieferantBleibtAufDerListeBisGedruckt(t *testing.T) {
	pool := pgtest.Pool(t)
	testdaten.LeereBestandsdaten(t, pool)

	testdaten.SetzeOeffentlicheAdresse(t, pool, "")
	post, sitzungen := tuertest.FangeMailAb(t)

	srv := &Server{Basis: tuer.Basis{Mail: post, DB: &db.Verbindung{Pool: pool}}}
	lieferant := testdaten.LegeLieferantAn(t, pool, "Cornelsen-Kette", false)
	titel := testdaten.LegeTitelAn(t, pool, "LMF-Etikettenkette", 0)

	bestellTuer := srv.bestellung().BestellungAnlegen(service.NewBestellDienst(srv.DB, repository.NewKatalogAbfragen(pool)))
	if rec := tuertest.Bestelle(t, bestellTuer, lieferant, titel); rec.Code != http.StatusOK {
		t.Fatalf("Bestellung: Status %d — %s", rec.Code, rec.Body.String())
	}
	smtptest.WarteAufMail(t, sitzungen)

	barcodes := testdaten.LadeBarcodesDerBestellung(t, pool)
	if len(barcodes) != 2 {
		t.Fatalf("Exemplare = %v, erwartet 2", barcodes)
	}

	// Wareneingang: freigeben, ohne den Etikettenstatus anzufassen.
	items := wareneingang(t, pool)
	for _, item := range items {
		if item.EtikettGedruckt {
			t.Errorf("%s kommt aus dem Wareneingang als 'Etikett gedruckt' zurück — der Druckvorschlag "+
				"nach der Lieferung würde dieses Buch überspringen", item.BarcodeID)
		}
	}

	// Nachdruck-Liste und Zähler müssen dasselbe sagen.
	liste, anzahl := offeneEtikettenUeberTuer(t, srv)
	if anzahl != len(liste) {
		t.Errorf("Zähler nennt %d, die Liste hat %d Zeilen — zwei Abfragen, zwei Wahrheiten", anzahl, len(liste))
	}
	for _, barcode := range barcodes {
		if !enthaeltBarcode(liste, barcode) {
			t.Errorf("%s fehlt in der Nachdruck-Liste, obwohl das Etikett noch aussteht", barcode)
		}
	}

	// Druck gegenbuchen — der Weg, über den die Liste überhaupt leer werden kann.
	gedruckt := sendeJSON(t, srv.druck().EtikettenGedrucktVermerken(), "/api/exemplare/etiketten-gedruckt",
		map[string]any{"barcode_ids": barcodes})
	if gedruckt["markiert"] != float64(len(barcodes)) {
		t.Errorf("vermerkt wurden %v Etiketten, erwartet %d", gedruckt["markiert"], len(barcodes))
	}

	liste, anzahl = offeneEtikettenUeberTuer(t, srv)
	if anzahl != len(liste) {
		t.Errorf("nach dem Druck: Zähler %d, Liste %d Zeilen", anzahl, len(liste))
	}
	for _, barcode := range barcodes {
		if enthaeltBarcode(liste, barcode) {
			t.Errorf("%s steht nach dem Gegenbuchen weiter auf der Nachdruck-Liste", barcode)
		}
	}

	// Und der Weg zurück (Papierstau): Zurücksetzen bringt sie wieder.
	sendeJSON(t, srv.druck().EtikettenZuruecksetzen(), "/api/exemplare/etiketten-zuruecksetzen",
		map[string]any{"barcode_ids": barcodes})
	liste, _ = offeneEtikettenUeberTuer(t, srv)
	for _, barcode := range barcodes {
		if !enthaeltBarcode(liste, barcode) {
			t.Errorf("%s kam nach dem Zurücksetzen nicht auf die Liste zurück — der Papierstau-Fall "+
				"wäre eine Einbahnstrasse", barcode)
		}
	}
}

// Die Gegenrichtung: Beim Hauptlieferanten klebt der Händler. Seine Exemplare dürfen die
// Nachdruck-Liste zu keinem Zeitpunkt sehen — auch nicht nach dem Wareneingang, der die
// Zeilen erneut anfasst.
func TestEtikettenkette_HauptlieferantErscheintNieAufDerNachdruckListe(t *testing.T) {
	pool := pgtest.Pool(t)
	testdaten.LeereBestandsdaten(t, pool)

	testdaten.SetzeOeffentlicheAdresse(t, pool, "https://bib.example.invalid")
	post, sitzungen := tuertest.FangeMailAb(t)

	srv := &Server{Basis: tuer.Basis{Mail: post, DB: &db.Verbindung{Pool: pool}}}
	lieferant := testdaten.LegeLieferantAn(t, pool, "Naacher-Etikettenkette", true)
	titel := testdaten.LegeTitelAn(t, pool, "LMF-Beklebt", 0)

	bestellTuer := srv.bestellung().BestellungAnlegen(service.NewBestellDienst(srv.DB, repository.NewKatalogAbfragen(pool)))
	if rec := tuertest.Bestelle(t, bestellTuer, lieferant, titel); rec.Code != http.StatusOK {
		t.Fatalf("Bestellung: Status %d — %s", rec.Code, rec.Body.String())
	}
	smtptest.WarteAufMail(t, sitzungen)

	barcodes := testdaten.LadeBarcodesDerBestellung(t, pool)
	liste, anzahl := offeneEtikettenUeberTuer(t, srv)
	if anzahl != 0 || len(liste) != 0 {
		t.Fatalf("vor dem Wareneingang stehen %d Zeilen (Zähler %d) auf der Nachdruck-Liste — "+
			"der Händler klebt selbst", len(liste), anzahl)
	}

	items := wareneingang(t, pool)
	for _, item := range items {
		if !item.EtikettGedruckt {
			t.Errorf("%s meldet nach dem Wareneingang 'Etikett fehlt' — die Bibliothek bekäme einen "+
				"Druckvorschlag für ein bereits beklebtes Buch", item.BarcodeID)
		}
	}

	liste, anzahl = offeneEtikettenUeberTuer(t, srv)
	if anzahl != 0 || len(liste) != 0 {
		t.Errorf("nach dem Wareneingang stehen %d Zeilen (Zähler %d) auf der Liste: %v",
			len(liste), anzahl, barcodes)
	}
}

// sendeJSON ruft eine schreibende Tür mit einem Rumpf in JSON und liefert die Antwort.
func sendeJSON(t *testing.T, handler http.HandlerFunc, pfad string, body map[string]any) map[string]any {
	t.Helper()
	roh, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("Body bauen: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, pfad, strings.NewReader(string(roh)))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("POST %s: Status %d — %s", pfad, rec.Code, rec.Body.String())
	}
	var antwort map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &antwort); err != nil {
		t.Fatalf("POST %s: Antwort unlesbar: %v", pfad, err)
	}
	return antwort
}

// Der Zähler muss dieselben Filter verstehen wie die Liste.
//
// Die Liste ist bei 300 Zeilen gedeckelt, und die Fußzeile nennt beide Zahlen („300 von
// 30.674"); dafür muss der Zähler dieselbe Menge zählen, die die Liste zeigt. Verstünde er
// status/q nicht, stünde unter einer gefilterten Liste die Zahl des ungefilterten Bestands, eine
// Angabe, die schlimmer ist als gar keine, weil sie glaubwürdig aussieht.
func TestEtikettenkette_ZaehlerFolgtDenFilternDerListe(t *testing.T) {
	pool := pgtest.Pool(t)
	testdaten.LeereBestandsdaten(t, pool)

	srv := &Server{Basis: tuer.Basis{DB: &db.Verbindung{Pool: pool}}}
	ctx := context.Background()

	// Drei offene und zwei erledigte Exemplare unter einer eigenen Marke, dazu ein
	// Fremdkörper ohne die Marke: Ohne ihn wäre ein Zähler, der `q` schlicht ignoriert,
	// zufällig richtig.
	if _, err := pool.Exec(ctx, `
		WITH t AS (
			INSERT INTO buecher_titel (titel, autor) VALUES ('ZAEHLERMARKE Titel', 'A') RETURNING id
		), f AS (
			INSERT INTO buecher_titel (titel, autor) VALUES ('Anderer Titel', 'B') RETURNING id
		), a AS (
			INSERT INTO buecher_exemplare (titel_id, barcode_id, etikett_gedruckt, erworben_am)
			SELECT t.id, 'ZM-OFFEN-' || g, false, CURRENT_DATE FROM t, generate_series(1, 3) AS g
		), b AS (
			INSERT INTO buecher_exemplare (titel_id, barcode_id, etikett_gedruckt, erworben_am)
			SELECT t.id, 'ZM-FERTIG-' || g, true, CURRENT_DATE FROM t, generate_series(1, 2) AS g
		)
		INSERT INTO buecher_exemplare (titel_id, barcode_id, etikett_gedruckt, erworben_am)
		SELECT f.id, 'ZM-FREMD-1', false, CURRENT_DATE FROM f
	`); err != nil {
		t.Fatalf("Seed: %v", err)
	}

	faelle := []struct {
		name    string
		query   string
		erwarte int
	}{
		{"offen mit Marke", "?q=ZAEHLERMARKE", 3},
		{"erledigt mit Marke", "?q=ZAEHLERMARKE&status=erledigt", 2},
		{"alle mit Marke", "?q=ZAEHLERMARKE&status=alle", 5},
		{"offen ueber Barcode", "?q=ZM-FREMD", 1},
	}

	for _, f := range faelle {
		t.Run(f.name, func(t *testing.T) {
			var liste []druck.ExemplarOhneEtikett
			jsonHolen(t, srv.druck().EtikettenOffen(), "/api/exemplare/etiketten-offen"+f.query, &liste)

			var zaehler struct {
				Anzahl int `json:"anzahl"`
			}
			jsonHolen(t, srv.druck().EtikettenOffenAnzahl(),
				"/api/exemplare/etiketten-offen/anzahl"+f.query, &zaehler)

			if len(liste) != f.erwarte {
				t.Errorf("Liste hat %d Zeilen, erwartet %d", len(liste), f.erwarte)
			}
			if zaehler.Anzahl != f.erwarte {
				t.Errorf("Zaehler nennt %d, erwartet %d — die Fusszeile der Liste zeigte damit "+
					"eine Zahl aus einer anderen Menge als die Zeilen darueber", zaehler.Anzahl, f.erwarte)
			}
		})
	}

	// Mit Stichtag zählt der Zähler weiter nur offene Etiketten: Das ist die Zahl für das
	// Aufräumen des Altbestands. Ein `bis`, das auch Erledigte zählte, verspräche im Dialog eine
	// zu große Zahl.
	var mitStichtag struct {
		Anzahl int `json:"anzahl"`
	}
	// Der Stichtag ist ein Tag der Schule: Westlich von UTC ist das Heute des Testprozesses zu
	// manchen Stunden der Vortag, und die heute angelegten Exemplare zählten nicht mit.
	jsonHolen(t, srv.druck().EtikettenOffenAnzahl(),
		"/api/exemplare/etiketten-offen/anzahl?bis="+time.Now().In(schulzeit.Zone()).Format(tuer.DatumsformatISO), &mitStichtag)
	if mitStichtag.Anzahl != 4 {
		t.Errorf("Stichtag-Zaehler nennt %d, erwartet 4 (drei markierte + ein fremdes offenes)",
			mitStichtag.Anzahl)
	}
}

// Liste und Zähler vergleichen den Suchtext mit dem Wortlaut des Titels. Die Datenbank
// speichert Titeltexte mit einem Leerzeichen zwischen den Wörtern (Migration 160), und beide
// lesen den Suchtext in derselben Form (etikettenSuchtext): Wer den Titel tippt, wie er in
// Littera stand, findet seine Etiketten trotzdem.
func TestEtikettenkette_LeerraumInFolgeTrenntSuchtextUndTitelNicht(t *testing.T) {
	pool := pgtest.Pool(t)
	testdaten.LeereBestandsdaten(t, pool)

	srv := &Server{Basis: tuer.Basis{DB: &db.Verbindung{Pool: pool}}}
	ctx := context.Background()
	geschuetzt := string(rune(0x00A0))

	// Zwei offene Exemplare am Titel, dazu ein fremdes: Ohne das wäre eine Suche, die
	// alles trifft, zufällig richtig.
	if _, err := pool.Exec(ctx, `
		WITH t AS (
			INSERT INTO buecher_titel (titel) VALUES ($1) RETURNING id
		), f AS (
			INSERT INTO buecher_titel (titel) VALUES ('Anderer Titel') RETURNING id
		), a AS (
			INSERT INTO buecher_exemplare (titel_id, barcode_id, etikett_gedruckt, erworben_am)
			SELECT t.id, 'LR-OFFEN-' || g, false, CURRENT_DATE FROM t, generate_series(1, 2) AS g
		)
		INSERT INTO buecher_exemplare (titel_id, barcode_id, etikett_gedruckt, erworben_am)
		SELECT f.id, 'LR-FREMD-1', false, CURRENT_DATE FROM f
	`, "La  Peste"); err != nil {
		t.Fatalf("Seed: %v", err)
	}

	for _, suchtext := range []string{"La Peste", "La  Peste", "la" + geschuetzt + "pes"} {
		query := "?q=" + url.QueryEscape(suchtext)
		var liste []druck.ExemplarOhneEtikett
		jsonHolen(t, srv.druck().EtikettenOffen(), "/api/exemplare/etiketten-offen"+query, &liste)
		var zaehler struct {
			Anzahl int `json:"anzahl"`
		}
		jsonHolen(t, srv.druck().EtikettenOffenAnzahl(), "/api/exemplare/etiketten-offen/anzahl"+query, &zaehler)
		if len(liste) != 2 || zaehler.Anzahl != 2 {
			t.Errorf("Suche %+q: Liste %d Zeilen, Zähler %d — erwartet je 2", suchtext, len(liste), zaehler.Anzahl)
		}
	}
}

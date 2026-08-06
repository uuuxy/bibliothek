package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"bibliothek/db"
	"bibliothek/internal/service"
	"bibliothek/repository"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Die Etiketten-Kette: Wareneingang → Etikettenstatus → Nachdruck-Liste.
//
// Die Glieder sind einzeln geprüft, die ÜBERGÄNGE nicht — und dort sitzt hier eine ganze
// Reihe stiller Ausfälle:
//
//   - Der Wareneingang schreibt an denselben Zeilen (ist_ausleihbar, zustand_notiz). Nähme
//     er etikett_gedruckt mit, verschwänden frisch gelieferte Bücher ohne Aufkleber
//     lautlos aus der Nachdruck-Liste — niemand vermisst, was nie in der Liste stand.
//   - Liste und Zähler haben zwei getrennte Abfragen. Dass sie dieselbe Bedingung
//     benutzen, behauptet bisher nur ein Kommentar (etikettenOffenBedingung); driften sie
//     auseinander, nennt der Hinweis im Bestellwesen eine Zahl, die zu keiner Liste passt.
//   - Der Hauptlieferant klebt selbst. Seine Exemplare dürfen nach dem Wareneingang NICHT
//     auf der Nachdruck-Liste stehen, sonst klebt die Bibliothek ein zweites Etikett auf
//     ein bereits beklebtes Buch.

// jsonHolen ruft einen Handler und liest die Antwort in ziel.
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

// offeneEtikettenUeberHandler liefert Liste und Zähler — die beiden Wege zur selben Frage.
//
// Bewusst über die HTTP-Handler und nicht über eigenes SQL: Der Helfer in
// haendler_beklebt_pg_test.go schreibt die Bedingung noch einmal selbst hin ("bewusst
// dieselbe wie etikettenOffenBedingung") und kann eine Abweichung im Produktivcode
// deshalb gar nicht sehen — er würde mitwandern, nur eben nur im Test.
func offeneEtikettenUeberHandler(t *testing.T, srv *Server) (liste []ExemplarOhneEtikett, anzahl int) {
	t.Helper()
	jsonHolen(t, srv.EtikettenOffenHandler(), "/api/exemplare/etiketten-offen", &liste)

	var zaehler struct {
		Anzahl int `json:"anzahl"`
	}
	jsonHolen(t, srv.EtikettenOffenAnzahlHandler(), "/api/exemplare/etiketten-offen/anzahl", &zaehler)
	return liste, zaehler.Anzahl
}

func enthaeltBarcode(liste []ExemplarOhneEtikett, barcode string) bool {
	for _, e := range liste {
		if e.BarcodeID == barcode {
			return true
		}
	}
	return false
}

// wareneingang bucht alle noch nicht freigegebenen Exemplare ein — wie der Knopf im
// Bestellwesen, über denselben Dienst.
func wareneingang(t *testing.T, pool *pgxpool.Pool) []service.ReceivedItem {
	t.Helper()
	ctx := context.Background()
	ids := exemplarIDsDerBestellung(t, pool)
	if len(ids) == 0 {
		t.Fatal("nichts einzubuchen — der Testaufbau hat keine Exemplare erzeugt")
	}
	items, err := service.BulkReceiveOrder(ctx, pool, repository.NewAuditRepository(pool), ids, "", "127.0.0.1")
	if err != nil {
		t.Fatalf("Wareneingang: %v", err)
	}
	return items
}

func TestEtikettenkette_NormalerLieferantBleibtAufDerListeBisGedruckt(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)

	setzeOeffentlicheAdresse(t, pool, "")
	sitzungen := mailAbfangen(t)

	srv := &Server{DB: &db.Database{Pool: pool}}
	lieferant := haendler(t, pool, "Cornelsen-Kette", false)
	titel := titelMitMeldebestand(t, pool, "LMF-Etikettenkette", 0)

	if rec := bestelleUeberHandler(t, srv, lieferant, titel); rec.Code != http.StatusOK {
		t.Fatalf("Bestellung: Status %d — %s", rec.Code, rec.Body.String())
	}
	warteAufMail(t, sitzungen)

	barcodes := barcodesDerBestellung(t, pool)
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

	// Nachdruck-Liste UND Zähler müssen dasselbe sagen.
	liste, anzahl := offeneEtikettenUeberHandler(t, srv)
	if anzahl != len(liste) {
		t.Errorf("Zähler nennt %d, die Liste hat %d Zeilen — zwei Abfragen, zwei Wahrheiten", anzahl, len(liste))
	}
	for _, barcode := range barcodes {
		if !enthaeltBarcode(liste, barcode) {
			t.Errorf("%s fehlt in der Nachdruck-Liste, obwohl das Etikett noch aussteht", barcode)
		}
	}

	// Druck gegenbuchen — der Weg, über den die Liste überhaupt leer werden kann.
	gedruckt := postJSON(t, srv.EtikettenGedrucktHandler(), "/api/exemplare/etiketten-gedruckt",
		map[string]any{"barcode_ids": barcodes})
	if gedruckt["markiert"] != float64(len(barcodes)) {
		t.Errorf("vermerkt wurden %v Etiketten, erwartet %d", gedruckt["markiert"], len(barcodes))
	}

	liste, anzahl = offeneEtikettenUeberHandler(t, srv)
	if anzahl != len(liste) {
		t.Errorf("nach dem Druck: Zähler %d, Liste %d Zeilen", anzahl, len(liste))
	}
	for _, barcode := range barcodes {
		if enthaeltBarcode(liste, barcode) {
			t.Errorf("%s steht nach dem Gegenbuchen weiter auf der Nachdruck-Liste", barcode)
		}
	}

	// Und der Weg zurück (Papierstau): Zurücksetzen bringt sie wieder.
	postJSON(t, srv.EtikettenZuruecksetzenHandler(), "/api/exemplare/etiketten-zuruecksetzen",
		map[string]any{"barcode_ids": barcodes})
	liste, _ = offeneEtikettenUeberHandler(t, srv)
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
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)

	setzeOeffentlicheAdresse(t, pool, "https://bib.example.invalid")
	sitzungen := mailAbfangen(t)

	srv := &Server{DB: &db.Database{Pool: pool}}
	lieferant := haendler(t, pool, "Naacher-Etikettenkette", true)
	titel := titelMitMeldebestand(t, pool, "LMF-Beklebt", 0)

	if rec := bestelleUeberHandler(t, srv, lieferant, titel); rec.Code != http.StatusOK {
		t.Fatalf("Bestellung: Status %d — %s", rec.Code, rec.Body.String())
	}
	warteAufMail(t, sitzungen)

	barcodes := barcodesDerBestellung(t, pool)
	liste, anzahl := offeneEtikettenUeberHandler(t, srv)
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

	liste, anzahl = offeneEtikettenUeberHandler(t, srv)
	if anzahl != 0 || len(liste) != 0 {
		t.Errorf("nach dem Wareneingang stehen %d Zeilen (Zähler %d) auf der Liste: %v",
			len(liste), anzahl, barcodes)
	}
}

// postJSON ruft einen schreibenden Handler mit JSON-Body und liefert die Antwort.
func postJSON(t *testing.T, handler http.HandlerFunc, pfad string, body map[string]any) map[string]any {
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

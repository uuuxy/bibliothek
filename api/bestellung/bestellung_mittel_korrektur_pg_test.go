package bestellung

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"bibliothek/api/tuer"
	"bibliothek/auth"
	"bibliothek/db"
	"bibliothek/internal/pgtest"
	"bibliothek/internal/testdaten"
	"bibliothek/pkg/mitteltopf"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Der Rückweg für den Topf einer Bestellung, am Live-Pfad über die Tür, mit Protokoll.
//
// Drei Dinge müssen zusammen wahr sein: Die Spalte trägt den neuen Topf, das
// Protokoll der Verwaltung kennt von/nach/Grund, und ohne Grund oder mit unbekanntem Topf passiert
// nichts — weder Zeile noch Protokoll. Der Kundennummer-Beleg der Bestellung bleibt
// unangetastet: Er ist die Abschrift dessen, was der Händler bekommen hat.

// korrigiereTopfUeberTuer ruft PUT /api/bestellungen/{id}/mittel als angemeldeter
// Bearbeiter (echter Benutzer in den Claims — audit_logs.admin_id zeigt auf benutzer,
// ein erfundener Akteur ließe den Protokolleintrag still scheitern).
func korrigiereTopfUeberTuer(t *testing.T, srv *Tueren, pool *pgxpool.Pool, bestellungID, rumpf string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPut, "/api/bestellungen/"+bestellungID+"/mittel", strings.NewReader(rumpf))
	req.SetPathValue("id", bestellungID)
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(context.WithValue(req.Context(), auth.ClaimsSchluessel,
		&auth.Claims{KontoID: testdaten.LegeBearbeiterAn(t, pool), Rolle: auth.RolleAdmin}))
	rec := httptest.NewRecorder()
	srv.BestellungMittelKorrigieren()(rec, req)
	return rec
}

func topfInDB(t *testing.T, pool *pgxpool.Pool, id string) *string {
	t.Helper()
	var mittel *string
	if err := pool.QueryRow(context.Background(), `SELECT mittel FROM bestellungen_verlauf WHERE id = $1`, id).Scan(&mittel); err != nil {
		t.Fatalf("Topf lesen: %v", err)
	}
	return mittel
}

func TestBestellungTopfKorrektur_SchreibtTopfUndProtokoll(t *testing.T) {
	pool := pgtest.Pool(t)
	testdaten.LeereBestandsdaten(t, pool)
	ctx := context.Background()
	srv := &Tueren{Basis: &tuer.Basis{DB: &db.Verbindung{Pool: pool}}}
	if _, err := pool.Exec(ctx, `TRUNCATE audit_logs`); err != nil {
		t.Fatal(err)
	}

	lieferant := testdaten.LegeLieferantAn(t, pool, "Korrektur-Haendler", false)
	titel := testdaten.LegeTitelAn(t, pool, "LMF-Korrektur", 0)
	rec := bestelleMitTopf(t, srv, lieferant, titel, mitteltopf.Land)
	if rec.Code != http.StatusOK {
		t.Fatalf("Bestellung anlegen: %d %s", rec.Code, rec.Body.String())
	}
	var id, kundennummerVorher string
	if err := pool.QueryRow(ctx, `SELECT id, kundennummer FROM bestellungen_verlauf WHERE lieferant_id = $1`, lieferant).Scan(&id, &kundennummerVorher); err != nil {
		t.Fatal(err)
	}

	// 1. Korrektur Land → Schulträger mit Grund.
	rec = korrigiereTopfUeberTuer(t, srv, pool, id, `{"mittel":"schultraeger","grund":"Titel war falsch als Lernmittel gekennzeichnet"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("Korrektur: %d %s", rec.Code, rec.Body.String())
	}
	var antwort struct {
		Mittel    string `json:"mittel"`
		Von       string `json:"von"`
		Geaendert bool   `json:"geaendert"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &antwort); err != nil {
		t.Fatal(err)
	}
	if antwort.Mittel != "schultraeger" || antwort.Von != "land" || !antwort.Geaendert {
		t.Errorf("Antwort: %+v — muss neuen Topf, alten Topf und geaendert=true nennen", antwort)
	}
	if got := topfInDB(t, pool, id); got == nil || *got != "schultraeger" {
		t.Errorf("Topf in der Datenbank: %v, erwartet schultraeger", got)
	}

	// Die Kundennummer auf dem Beleg bleibt — sie ist die Abschrift dessen, was rausging.
	var kundennummerNachher string
	if err := pool.QueryRow(ctx, `SELECT kundennummer FROM bestellungen_verlauf WHERE id = $1`, id).Scan(&kundennummerNachher); err != nil {
		t.Fatal(err)
	}
	if kundennummerNachher != kundennummerVorher {
		t.Errorf("Kundennummer auf dem Beleg wurde umgeschrieben: %q → %q", kundennummerVorher, kundennummerNachher)
	}

	// 2. Das Protokoll kennt von, nach und Grund — als JSON gelesen, nicht als Text verglichen
	//    (jsonb rendert mit eigenen Leerzeichen).
	var detailsRoh []byte
	err := pool.QueryRow(ctx, `SELECT details FROM audit_logs WHERE aktion = $1`, aktionBestellungMittelKorrigiert).Scan(&detailsRoh)
	if err != nil {
		t.Fatalf("Audit-Eintrag fehlt: %v", err)
	}
	var details map[string]string
	if err := json.Unmarshal(detailsRoh, &details); err != nil {
		t.Fatalf("Audit-Details lesen: %v", err)
	}
	erwartet := map[string]string{"von": "land", "nach": "schultraeger", "grund": "Titel war falsch als Lernmittel gekennzeichnet", "bestellung_id": id}
	for schluessel, wert := range erwartet {
		if details[schluessel] != wert {
			t.Errorf("Audit-Details[%s] = %q, erwartet %q (gesamt: %s)", schluessel, details[schluessel], wert, string(detailsRoh))
		}
	}

	// 3. Dieselbe Korrektur noch einmal: kein zweiter Eintrag im Protokoll, geaendert=false.
	rec = korrigiereTopfUeberTuer(t, srv, pool, id, `{"mittel":"schultraeger","grund":"nochmal"}`)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"geaendert":false`) {
		t.Errorf("Wiederholung: %d %s — muss 200 mit geaendert=false sein", rec.Code, rec.Body.String())
	}
	var anzahl int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM audit_logs WHERE aktion = $1`, aktionBestellungMittelKorrigiert).Scan(&anzahl); err != nil {
		t.Fatal(err)
	}
	if anzahl != 1 {
		t.Errorf("%d Audit-Einträge, erwartet 1 — eine Wiederholung ohne Änderung darf nichts protokollieren", anzahl)
	}
}

func TestBestellungTopfKorrektur_OhneGrundOderMitUnbekanntemTopfPassiertNichts(t *testing.T) {
	pool := pgtest.Pool(t)
	testdaten.LeereBestandsdaten(t, pool)
	ctx := context.Background()
	srv := &Tueren{Basis: &tuer.Basis{DB: &db.Verbindung{Pool: pool}}}
	if _, err := pool.Exec(ctx, `TRUNCATE audit_logs`); err != nil {
		t.Fatal(err)
	}

	lieferant := testdaten.LegeLieferantAn(t, pool, "Korrektur-Haendler-2", false)
	titel := testdaten.LegeTitelAn(t, pool, "LMF-Korrektur-2", 0)
	if rec := bestelleMitTopf(t, srv, lieferant, titel, mitteltopf.Land); rec.Code != http.StatusOK {
		t.Fatalf("Bestellung anlegen: %d", rec.Code)
	}
	var id string
	if err := pool.QueryRow(ctx, `SELECT id FROM bestellungen_verlauf WHERE lieferant_id = $1`, lieferant).Scan(&id); err != nil {
		t.Fatal(err)
	}

	for _, rumpf := range []string{
		`{"mittel":"schultraeger","grund":"   "}`,
		`{"mittel":"schultraeger"}`,
		`{"mittel":"kreis","grund":"x"}`,
		`{"grund":"x"}`,
	} {
		if rec := korrigiereTopfUeberTuer(t, srv, pool, id, rumpf); rec.Code != http.StatusBadRequest {
			t.Errorf("%s: Status %d, erwartet 400: %s", rumpf, rec.Code, rec.Body.String())
		}
	}
	if got := topfInDB(t, pool, id); got == nil || *got != "land" {
		t.Errorf("Topf wurde trotz abgewiesener Anfrage verändert: %v", got)
	}
	var anzahl int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM audit_logs WHERE aktion = $1`, aktionBestellungMittelKorrigiert).Scan(&anzahl); err != nil {
		t.Fatal(err)
	}
	if anzahl != 0 {
		t.Errorf("%d Audit-Einträge für abgewiesene Anfragen", anzahl)
	}

	unbekannt := "00000000-0000-0000-0000-000000000000"
	if rec := korrigiereTopfUeberTuer(t, srv, pool, unbekannt, `{"mittel":"land","grund":"x"}`); rec.Code != http.StatusNotFound {
		t.Errorf("unbekannte Bestellung: Status %d, erwartet 404", rec.Code)
	}
}

// Alt-Bestellung „ohne Zuordnung" (NULL): Der Rückweg ordnet sie zu, das Protokoll nennt
// von="" — so ist im Protokoll sichtbar, dass hier erstmals zugeordnet wurde.
func TestBestellungTopfKorrektur_OrdnetAltBestellungOhneZuordnungZu(t *testing.T) {
	pool := pgtest.Pool(t)
	testdaten.LeereBestandsdaten(t, pool)
	ctx := context.Background()
	srv := &Tueren{Basis: &tuer.Basis{DB: &db.Verbindung{Pool: pool}}}
	if _, err := pool.Exec(ctx, `TRUNCATE audit_logs`); err != nil {
		t.Fatal(err)
	}

	id := testdaten.LegeBestellungOhneTopfAn(t, pool, "alt-ohne-zuordnung", nil)
	rec := korrigiereTopfUeberTuer(t, srv, pool, id, `{"mittel":"land","grund":"Rechnung liegt vor, war Lernmittel"}`)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"von":""`) {
		t.Fatalf("Alt-Bestellung zuordnen: %d %s", rec.Code, rec.Body.String())
	}
	if got := topfInDB(t, pool, id); got == nil || *got != "land" {
		t.Errorf("Topf: %v, erwartet land", got)
	}
}

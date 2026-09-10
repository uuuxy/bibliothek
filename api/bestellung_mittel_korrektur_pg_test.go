package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"bibliothek/auth"
	"bibliothek/db"
	"bibliothek/repository"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Der Rückweg für den Topf einer Bestellung — am Live-Pfad über den Handler, mit Audit.
//
// Drei Dinge müssen zusammen wahr sein: Die Spalte trägt den neuen Topf, das
// Admin-Audit-Log kennt von/nach/Grund, und ohne Grund oder mit unbekanntem Topf passiert
// NICHTS — weder Zeile noch Protokoll. Der Kundennummer-Beleg der Bestellung bleibt
// unangetastet: Er ist die Abschrift dessen, was der Händler bekommen hat.

// korrigiereTopfUeberHandler ruft PUT /api/bestellungen/{id}/mittel als angemeldeter
// Bearbeiter (echter Benutzer in den Claims — audit_logs.admin_id zeigt auf benutzer,
// ein erfundener Akteur ließe den Protokolleintrag still scheitern).
func korrigiereTopfUeberHandler(t *testing.T, srv *Server, pool *pgxpool.Pool, bestellungID, rumpf string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPut, "/api/bestellungen/"+bestellungID+"/mittel", strings.NewReader(rumpf))
	req.SetPathValue("id", bestellungID)
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(context.WithValue(req.Context(), auth.ClaimsContextKey,
		&auth.Claims{UserID: adminFuerAudit(t, pool), Rolle: auth.RoleAdmin}))
	rec := httptest.NewRecorder()
	srv.UpdateBestellungMittelHandler()(rec, req)
	return rec
}

func topfInDb(t *testing.T, pool *pgxpool.Pool, id string) *string {
	t.Helper()
	var mittel *string
	if err := pool.QueryRow(context.Background(), `SELECT mittel FROM bestellungen_verlauf WHERE id = $1`, id).Scan(&mittel); err != nil {
		t.Fatalf("Topf lesen: %v", err)
	}
	return mittel
}

func TestBestellungTopfKorrektur_SchreibtTopfUndAudit(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	ctx := context.Background()
	srv := &Server{DB: &db.Database{Pool: pool}}
	if _, err := pool.Exec(ctx, `TRUNCATE audit_logs`); err != nil {
		t.Fatal(err)
	}

	lieferant := haendler(t, pool, "Korrektur-Haendler", false)
	titel := titelMitMeldebestand(t, pool, "LMF-Korrektur", 0)
	rec := bestelleMitTopf(t, srv, lieferant, titel, repository.MittelLand)
	if rec.Code != http.StatusOK {
		t.Fatalf("Bestellung anlegen: %d %s", rec.Code, rec.Body.String())
	}
	var id, kundennummerVorher string
	if err := pool.QueryRow(ctx, `SELECT id, kundennummer FROM bestellungen_verlauf WHERE lieferant_id = $1`, lieferant).Scan(&id, &kundennummerVorher); err != nil {
		t.Fatal(err)
	}

	// 1. Korrektur Land → Schulträger mit Grund.
	rec = korrigiereTopfUeberHandler(t, srv, pool, id, `{"mittel":"schultraeger","grund":"Titel war falsch als Lernmittel gekennzeichnet"}`)
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
	if got := topfInDb(t, pool, id); got == nil || *got != "schultraeger" {
		t.Errorf("Topf in der Datenbank: %v, want schultraeger", got)
	}

	// Die Kundennummer auf dem Beleg bleibt — sie ist die Abschrift dessen, was rausging.
	var kundennummerNachher string
	if err := pool.QueryRow(ctx, `SELECT kundennummer FROM bestellungen_verlauf WHERE id = $1`, id).Scan(&kundennummerNachher); err != nil {
		t.Fatal(err)
	}
	if kundennummerNachher != kundennummerVorher {
		t.Errorf("Kundennummer auf dem Beleg wurde umgeschrieben: %q → %q", kundennummerVorher, kundennummerNachher)
	}

	// 2. Das Audit kennt von, nach und Grund — als JSON gelesen, nicht als Text verglichen
	//    (jsonb rendert mit eigenen Leerzeichen).
	var detailsRoh []byte
	err := pool.QueryRow(ctx, `SELECT details FROM audit_logs WHERE aktion = $1`, auditBestellungMittelKorrigiert).Scan(&detailsRoh)
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
			t.Errorf("Audit-Details[%s] = %q, want %q (gesamt: %s)", schluessel, details[schluessel], wert, string(detailsRoh))
		}
	}

	// 3. Dieselbe Korrektur noch einmal: kein zweiter Audit-Eintrag, geaendert=false.
	rec = korrigiereTopfUeberHandler(t, srv, pool, id, `{"mittel":"schultraeger","grund":"nochmal"}`)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"geaendert":false`) {
		t.Errorf("Wiederholung: %d %s — muss 200 mit geaendert=false sein", rec.Code, rec.Body.String())
	}
	var anzahl int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM audit_logs WHERE aktion = $1`, auditBestellungMittelKorrigiert).Scan(&anzahl); err != nil {
		t.Fatal(err)
	}
	if anzahl != 1 {
		t.Errorf("%d Audit-Einträge, want 1 — eine Wiederholung ohne Änderung darf nichts protokollieren", anzahl)
	}
}

func TestBestellungTopfKorrektur_OhneGrundOderMitUnbekanntemTopfPassiertNichts(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	ctx := context.Background()
	srv := &Server{DB: &db.Database{Pool: pool}}
	if _, err := pool.Exec(ctx, `TRUNCATE audit_logs`); err != nil {
		t.Fatal(err)
	}

	lieferant := haendler(t, pool, "Korrektur-Haendler-2", false)
	titel := titelMitMeldebestand(t, pool, "LMF-Korrektur-2", 0)
	if rec := bestelleMitTopf(t, srv, lieferant, titel, repository.MittelLand); rec.Code != http.StatusOK {
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
		if rec := korrigiereTopfUeberHandler(t, srv, pool, id, rumpf); rec.Code != http.StatusBadRequest {
			t.Errorf("%s: Status %d, want 400: %s", rumpf, rec.Code, rec.Body.String())
		}
	}
	if got := topfInDb(t, pool, id); got == nil || *got != "land" {
		t.Errorf("Topf wurde trotz abgewiesener Anfrage verändert: %v", got)
	}
	var anzahl int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM audit_logs WHERE aktion = $1`, auditBestellungMittelKorrigiert).Scan(&anzahl); err != nil {
		t.Fatal(err)
	}
	if anzahl != 0 {
		t.Errorf("%d Audit-Einträge für abgewiesene Anfragen", anzahl)
	}

	unbekannt := "00000000-0000-0000-0000-000000000000"
	if rec := korrigiereTopfUeberHandler(t, srv, pool, unbekannt, `{"mittel":"land","grund":"x"}`); rec.Code != http.StatusNotFound {
		t.Errorf("unbekannte Bestellung: Status %d, want 404", rec.Code)
	}
}

// Alt-Bestellung „ohne Zuordnung" (NULL): Der Rückweg ordnet sie zu, das Audit nennt
// von="" — so ist im Protokoll sichtbar, dass hier erstmals zugeordnet wurde.
func TestBestellungTopfKorrektur_OrdnetAltBestellungOhneZuordnungZu(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	ctx := context.Background()
	srv := &Server{DB: &db.Database{Pool: pool}}
	if _, err := pool.Exec(ctx, `TRUNCATE audit_logs`); err != nil {
		t.Fatal(err)
	}

	id := altBestellung(t, pool, "alt-ohne-zuordnung", nil)
	rec := korrigiereTopfUeberHandler(t, srv, pool, id, `{"mittel":"land","grund":"Rechnung liegt vor, war Lernmittel"}`)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"von":""`) {
		t.Fatalf("Alt-Bestellung zuordnen: %d %s", rec.Code, rec.Body.String())
	}
	if got := topfInDb(t, pool, id); got == nil || *got != "land" {
		t.Errorf("Topf: %v, want land", got)
	}
}

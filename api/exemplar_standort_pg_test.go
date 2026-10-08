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
	"bibliothek/inventur"
	"bibliothek/repository"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Der Standort in der Buchakte (docs/OFFEN.md 5.53): Die Exemplarliste nennt ihn,
// PUT /api/exemplare/standort setzt ihn für markierte Exemplare — alle oder keins, mit altem
// und neuem Wert im Protokoll —, und die Titel-Verwaltung zählt je Titel dieselben Exemplare
// wie die Karten der Akte.

// standortSetzen ruft PUT /api/exemplare/standort als Admin.
func standortSetzen(t *testing.T, srv *Server, pool *pgxpool.Pool, ids []string, standort string) *httptest.ResponseRecorder {
	t.Helper()
	rumpf, err := json.Marshal(ExemplarStandortRequest{ExemplarIDs: ids, Standort: &standort})
	if err != nil {
		t.Fatal(err)
	}
	return standortRumpf(t, srv, pool, string(rumpf))
}

// standortRumpf schickt einen Körper wörtlich an PUT /api/exemplare/standort.
func standortRumpf(t *testing.T, srv *Server, pool *pgxpool.Pool, rumpf string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPut, "/api/exemplare/standort", strings.NewReader(rumpf))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(context.WithValue(req.Context(), auth.ClaimsContextKey,
		&auth.Claims{UserID: adminFuerAudit(t, pool), Rolle: auth.RoleAdmin}))
	rec := httptest.NewRecorder()
	srv.ExemplarStandortHandler()(rec, req)
	return rec
}

// standortVorschlaege liest GET /api/exemplare/standorte.
func standortVorschlaege(t *testing.T, srv *Server) []repository.StandortZahl {
	t.Helper()
	rec := httptest.NewRecorder()
	srv.ExemplarStandorteHandler()(rec, httptest.NewRequest(http.MethodGet, "/api/exemplare/standorte", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("Vorschläge: Status %d: %s", rec.Code, rec.Body.String())
	}
	var liste []repository.StandortZahl
	if err := json.Unmarshal(rec.Body.Bytes(), &liste); err != nil {
		t.Fatal(err)
	}
	return liste
}

// standortInDerDatenbank liest die Spalte; „(NULL)" heißt: kein besonderer Standort.
func standortInDerDatenbank(t *testing.T, pool *pgxpool.Pool, exemplarID string) string {
	t.Helper()
	var wert string
	if err := pool.QueryRow(context.Background(),
		`SELECT coalesce(standort, '(NULL)') FROM buecher_exemplare WHERE id = $1`, exemplarID).Scan(&wert); err != nil {
		t.Fatalf("Standort lesen: %v", err)
	}
	return wert
}

func pruefeStandortKarte(t *testing.T, karten map[string]map[string]any, barcode, standort string) {
	t.Helper()
	k := karten[barcode]
	if k == nil {
		t.Fatalf("%s fehlt in der Exemplarliste", barcode)
	}
	if k["standort"] != standort {
		t.Errorf("%s: standort %v, erwartet %q", barcode, k["standort"], standort)
	}
}

func TestExemplarStandort_AnzeigenUndAendern(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	ctx := context.Background()
	srv := &Server{DB: &db.Database{Pool: pool}}

	schulbuch := titelMitSignatur(t, pool, "Natura 2", "LMF Bio 7", 0)
	ohne := exemplar(t, pool, schulbuch, "ST-OHNE", true, "")
	schrank := exemplar(t, pool, schulbuch, "ST-SCHRANK", true, "")
	if _, err := pool.Exec(ctx, `UPDATE buecher_exemplare SET standort = 'Lehrerschrank' WHERE id = $1`, schrank); err != nil {
		t.Fatalf("Standort aus der Übernahme: %v", err)
	}

	// 1. Anzeigen: Die Karte nennt den Standort, ohne Standort einen leeren Wert.
	karten := exemplarKarten(t, srv, pool, schulbuch)
	pruefeStandortKarte(t, karten, "ST-OHNE", "")
	pruefeStandortKarte(t, karten, "ST-SCHRANK", "Lehrerschrank")

	// 2. Ändern: zwei markiert, Leerraum am Rand fällt weg.
	protokoll := func() int {
		return zaehleZeilen(t, pool, `SELECT count(*) FROM audit_log WHERE tabelle = 'buecher_exemplare'
			AND kontext = 'Standort geändert'`)
	}
	rec := standortSetzen(t, srv, pool, []string{ohne, strings.ToUpper(schrank)}, "  Bibliothek, Regal 3B ")
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"geaendert":2`) {
		t.Fatalf("Ändern: Status %d: %s — erwartet 200 und 2 geändert", rec.Code, rec.Body.String())
	}
	karten = exemplarKarten(t, srv, pool, schulbuch)
	pruefeStandortKarte(t, karten, "ST-OHNE", "Bibliothek, Regal 3B")
	pruefeStandortKarte(t, karten, "ST-SCHRANK", "Bibliothek, Regal 3B")
	if n := protokoll(); n != 2 {
		t.Errorf("Protokoll: %d Einträge, erwartet 2", n)
	}
	var alt, neu string
	if err := pool.QueryRow(ctx, `SELECT details->>'standort_alt', details->>'standort_neu'
		FROM audit_log WHERE datensatz_id = $1 AND kontext = 'Standort geändert'`, schrank).Scan(&alt, &neu); err != nil {
		t.Fatalf("Protokolleintrag lesen: %v", err)
	}
	if alt != "Lehrerschrank" || neu != "Bibliothek, Regal 3B" {
		t.Errorf("Protokoll: alt %q, neu %q", alt, neu)
	}

	// 3. Dasselbe noch einmal ändert nichts und schreibt nichts.
	if rec := standortSetzen(t, srv, pool, []string{ohne, schrank}, "Bibliothek, Regal 3B"); !strings.Contains(rec.Body.String(), `"geaendert":0`) {
		t.Errorf("Wiederholung: %s — erwartet 0 geändert", rec.Body.String())
	}
	if n := protokoll(); n != 2 {
		t.Errorf("Wiederholung schrieb ins Protokoll: %d Einträge", n)
	}

	// 4. Entfernen: Die Spalte ist danach NULL, das Exemplar steht wieder nach der Signatur.
	if rec := standortSetzen(t, srv, pool, []string{ohne}, ""); rec.Code != http.StatusOK ||
		!strings.Contains(rec.Body.String(), `"geaendert":1`) {
		t.Fatalf("Entfernen: Status %d: %s", rec.Code, rec.Body.String())
	}
	if wert := standortInDerDatenbank(t, pool, ohne); wert != "(NULL)" {
		t.Errorf("nach dem Entfernen steht %q in der Spalte, erwartet NULL", wert)
	}
	pruefeStandortKarte(t, exemplarKarten(t, srv, pool, schulbuch), "ST-OHNE", "")

	// 5. Die Länge zählt in Zeichen wie in der Datenbank: 255 Umlaute passen, 256 nicht.
	if rec := standortSetzen(t, srv, pool, []string{ohne}, strings.Repeat("ä", repository.ExemplarStandortMaxZeichen)); rec.Code != http.StatusOK {
		t.Errorf("255 Zeichen: Status %d: %s — erwartet 200", rec.Code, rec.Body.String())
	}
	if rec := standortSetzen(t, srv, pool, []string{ohne}, ""); rec.Code != http.StatusOK {
		t.Fatalf("Aufräumen nach der Längenprobe: Status %d", rec.Code)
	}

	// 6. Abweisen, ohne etwas zu ändern — auch am Exemplar, das es gibt.
	unbekannt := "00000000-0000-4000-8000-000000000000"
	faelle := []struct {
		name     string
		ids      []string
		standort string
		status   int
	}{
		{"zu lang", []string{schrank}, strings.Repeat("ä", repository.ExemplarStandortMaxZeichen+1), http.StatusBadRequest},
		{"keine Kennung", []string{"ST-SCHRANK"}, "Keller", http.StatusBadRequest},
		{"leere Auswahl", nil, "Keller", http.StatusBadRequest},
		{"eine gibt es nicht", []string{schrank, unbekannt}, "Keller", http.StatusNotFound},
	}
	for _, f := range faelle {
		if rec := standortSetzen(t, srv, pool, f.ids, f.standort); rec.Code != f.status {
			t.Errorf("%s: Status %d, erwartet %d: %s", f.name, rec.Code, f.status, rec.Body.String())
		}
	}
	// Ein Körper ohne das Feld ist kein Auftrag zum Entfernen.
	if rec := standortRumpf(t, srv, pool, `{"exemplar_ids": ["`+schrank+`"]}`); rec.Code != http.StatusBadRequest {
		t.Errorf("ohne das Feld standort: Status %d, erwartet 400: %s", rec.Code, rec.Body.String())
	}
	if wert := standortInDerDatenbank(t, pool, schrank); wert != "Bibliothek, Regal 3B" {
		t.Errorf("eine abgewiesene Änderung hat geschrieben: %q", wert)
	}

	// 7. Die Datenbank nimmt keinen leeren Standort an (chk_exemplar_standort); die Tür
	// schreibt für „entfernen" NULL.
	if _, err := pool.Exec(ctx, `UPDATE buecher_exemplare SET standort = '  ' WHERE id = $1`, ohne); err == nil {
		t.Error("ein Standort aus Leerraum wurde angenommen — chk_exemplar_standort fehlt")
	}
}

// Die Spalte „Standort" der Titel-Verwaltung und die Vorschläge des Dialogs zählen dieselben
// Exemplare wie die Karten der Akte: die im Bestand. Ein ausgesondertes Exemplar trägt seinen
// Standort weiter an der Karte und zählt nirgends mit.
func TestExemplarStandort_ListeUndVorschlaegeZaehlenDenBestand(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	ctx := context.Background()
	srv := &Server{DB: &db.Database{Pool: pool}}

	schulbuch := titelMitSignatur(t, pool, "Natura 2", "LMF Bio 7", 0)
	roman := titelMitSignatur(t, pool, "Tschick", "Ju Her", 0)
	ohneStandort := titelMitSignatur(t, pool, "Momo", "Ju End", 0)
	exemplar(t, pool, ohneStandort, "SL-MOMO", true, "")
	setze := func(titelID, barcode, standort string) string {
		id := exemplar(t, pool, titelID, barcode, true, "")
		if rec := standortSetzen(t, srv, pool, []string{id}, standort); rec.Code != http.StatusOK {
			t.Fatalf("%s: Status %d: %s", barcode, rec.Code, rec.Body.String())
		}
		return id
	}
	setze(schulbuch, "SL-1", "Bibliothek, Regal 3B")
	setze(schulbuch, "SL-2", "Bibliothek, Regal 3B")
	setze(schulbuch, "SL-3", "Lehrerschrank")
	exemplar(t, pool, schulbuch, "SL-4", true, "")
	weg := setze(schulbuch, "SL-5", "Keller")
	if _, err := pool.Exec(ctx, `UPDATE buecher_exemplare
		SET ist_ausgesondert = true, ist_ausleihbar = false, aussonderung_grund = 'AUSSORTIERT' WHERE id = $1`, weg); err != nil {
		t.Fatalf("Aussondern: %v", err)
	}
	setze(roman, "SL-6", "Lehrerschrank")
	setze(roman, "SL-7", "Lehrerschrank")

	// Die Titel-Verwaltung: je Titel der häufigste Standort zuerst, ohne das ausgesonderte.
	liste, err := inventur.NewBookRepository(pool).ListBooks(ctx, "", "", false)
	if err != nil {
		t.Fatalf("Katalogliste: %v", err)
	}
	jeTitel := map[string][]repository.StandortZahl{}
	for _, b := range liste {
		jeTitel[b.ID] = b.Standorte
	}
	gleich := func(a, b []repository.StandortZahl) bool {
		if len(a) != len(b) {
			return false
		}
		for i := range a {
			if a[i] != b[i] {
				return false
			}
		}
		return true
	}
	if soll := []repository.StandortZahl{{Standort: "Bibliothek, Regal 3B", Anzahl: 2}, {Standort: "Lehrerschrank", Anzahl: 1}}; !gleich(jeTitel[schulbuch], soll) {
		t.Errorf("Schulbuch: %v, erwartet %v", jeTitel[schulbuch], soll)
	}
	if soll := []repository.StandortZahl{{Standort: "Lehrerschrank", Anzahl: 2}}; !gleich(jeTitel[roman], soll) {
		t.Errorf("Roman: %v, erwartet %v", jeTitel[roman], soll)
	}
	if len(jeTitel[ohneStandort]) != 0 {
		t.Errorf("Titel ohne Standort: %v, erwartet nichts", jeTitel[ohneStandort])
	}

	// Die Karten der Akte, gezählt wie der Kopf der Akte zählt: Exemplare im Bestand mit Standort.
	ausKarten := map[string]int{}
	karten := exemplarKarten(t, srv, pool, schulbuch)
	for _, k := range karten {
		if standort, ok := k["standort"].(string); ok && standort != "" && k["im_bestand"] == true {
			ausKarten[standort]++
		}
	}
	if len(ausKarten) != len(jeTitel[schulbuch]) {
		t.Errorf("Karten zählen %v, die Liste %v", ausKarten, jeTitel[schulbuch])
	}
	for _, z := range jeTitel[schulbuch] {
		if ausKarten[z.Standort] != z.Anzahl {
			t.Errorf("%q: Karten %d, Liste %d", z.Standort, ausKarten[z.Standort], z.Anzahl)
		}
	}
	pruefeStandortKarte(t, karten, "SL-5", "Keller")

	// Die Vorschläge: über alle Titel, der häufigste zuerst, „Keller" steht nicht mehr im Bestand.
	vorschlaege := standortVorschlaege(t, srv)
	if soll := []repository.StandortZahl{{Standort: "Lehrerschrank", Anzahl: 3}, {Standort: "Bibliothek, Regal 3B", Anzahl: 2}}; !gleich(vorschlaege, soll) {
		t.Errorf("Vorschläge: %v, erwartet %v", vorschlaege, soll)
	}
}

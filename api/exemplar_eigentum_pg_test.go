package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"bibliothek/auth"
	"bibliothek/db"
	"bibliothek/pkg/mitteltopf"
	"bibliothek/repository"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Das Eigentum in der Buchakte (docs/OFFEN.md 4.24, Stufe 3): Die Exemplarliste nennt es mit
// Herkunft, und PUT /api/exemplare/eigentum setzt es für markierte Exemplare — mit Grund und
// Protokoll, alle oder keins.

// eigentumSetzen ruft PUT /api/exemplare/eigentum als Admin.
func eigentumSetzen(t *testing.T, srv *Server, pool *pgxpool.Pool, ids []string, eigentum, grund string) *httptest.ResponseRecorder {
	t.Helper()
	rumpf, err := json.Marshal(ExemplarEigentumRequest{ExemplarIDs: ids, Eigentum: eigentum, Grund: grund})
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPut, "/api/exemplare/eigentum", strings.NewReader(string(rumpf)))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(context.WithValue(req.Context(), auth.ClaimsContextKey,
		&auth.Claims{UserID: adminFuerAudit(t, pool), Rolle: auth.RoleAdmin}))
	rec := httptest.NewRecorder()
	srv.ExemplarEigentumHandler()(rec, req)
	return rec
}

// exemplarKarten liest GET /api/buecher/titel/{id}/exemplare, je Barcode eine Zeile.
func exemplarKarten(t *testing.T, srv *Server, pool *pgxpool.Pool, titelID string) map[string]map[string]any {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/api/buecher/titel/"+titelID+"/exemplare", nil)
	req.SetPathValue("id", titelID)
	rec := httptest.NewRecorder()
	srv.GetTitleCopiesHandler(repository.NewBescheidRepository(pool))(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("Exemplarliste: Status %d: %s", rec.Code, rec.Body.String())
	}
	var liste []map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &liste); err != nil {
		t.Fatal(err)
	}
	je := map[string]map[string]any{}
	for _, z := range liste {
		je[fmt.Sprint(z["barcode_id"])] = z
	}
	return je
}

func pruefeKarte(t *testing.T, karten map[string]map[string]any, barcode, eigentum, herkunft, vermerk string) {
	t.Helper()
	k := karten[barcode]
	if k == nil {
		t.Fatalf("%s fehlt in der Exemplarliste", barcode)
	}
	if k["eigentum"] != eigentum || k["eigentum_herkunft"] != herkunft || k["littera_eigentumsvermerk"] != vermerk {
		t.Errorf("%s: eigentum %v, herkunft %v, vermerk %v — erwartet %q, %q, %q", barcode,
			k["eigentum"], k["eigentum_herkunft"], k["littera_eigentumsvermerk"], eigentum, herkunft, vermerk)
	}
}

func TestExemplarEigentum_AnzeigenUndAendern(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	ctx := context.Background()
	srv := &Server{DB: &db.Database{Pool: pool}}

	lektuere := titelMitSignatur(t, pool, "Nathan der Weise", "Ga Les", 0)
	einfach := exemplar(t, pool, lektuere, "EE-VORGABE", true, "")
	ausLittera := exemplar(t, pool, lektuere, "EE-LITTERA", true, "")
	if _, err := pool.Exec(ctx, `
		UPDATE buecher_exemplare SET eigentum = 'land', eigentum_quelle = 'littera',
		       erweiterte_eigenschaften = '{"littera_eigentumsvermerk": "Land Hessen"}'
		WHERE id = $1`, ausLittera); err != nil {
		t.Fatalf("Littera-Stand: %v", err)
	}
	foerderverein := exemplar(t, pool, lektuere, "EE-FOERDERVEREIN", true, "")
	if _, err := pool.Exec(ctx, `UPDATE buecher_exemplare
		SET erweiterte_eigenschaften = '{"littera_eigentumsvermerk": "Förderverein"}' WHERE id = $1`, foerderverein); err != nil {
		t.Fatalf("Vermerk ohne Zuordnung: %v", err)
	}
	bestellt := exemplar(t, pool, lektuere, "EE-BESTELLT", true, "")
	if _, err := pool.Exec(ctx, `UPDATE buecher_exemplare SET bestellung_id = $2 WHERE id = $1`,
		bestellt, topfBestellung(t, mitteltopf.Land)); err != nil {
		t.Fatalf("Bestellung: %v", err)
	}

	// 1. Anzeigen: Eigentum und Herkunft nach der einen Regel.
	karten := exemplarKarten(t, srv, pool, lektuere)
	pruefeKarte(t, karten, "EE-VORGABE", mitteltopf.Schultraeger, "vorgabe", "")
	pruefeKarte(t, karten, "EE-LITTERA", mitteltopf.Land, "littera", "Land Hessen")
	pruefeKarte(t, karten, "EE-FOERDERVEREIN", mitteltopf.Schultraeger, "vorgabe", "Förderverein")
	pruefeKarte(t, karten, "EE-BESTELLT", mitteltopf.Land, "bestellung", "")

	// 2. Ändern: zwei markiert, beide werden „von Hand" — auch das aus Littera, das schon Land war.
	protokoll := func() int {
		return zaehleZeilen(t, pool, `SELECT count(*) FROM audit_log WHERE tabelle = 'buecher_exemplare'
			AND kontext = 'Eigentum von Hand geändert'`)
	}
	rec := eigentumSetzen(t, srv, pool, []string{einfach, strings.ToUpper(ausLittera)}, mitteltopf.Land, " Klassensatz aus LMF-Mitteln ")
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"geaendert":2`) {
		t.Fatalf("Ändern: Status %d: %s — erwartet 200 und 2 geändert", rec.Code, rec.Body.String())
	}
	karten = exemplarKarten(t, srv, pool, lektuere)
	pruefeKarte(t, karten, "EE-VORGABE", mitteltopf.Land, "hand", "")
	pruefeKarte(t, karten, "EE-LITTERA", mitteltopf.Land, "hand", "Land Hessen")
	if n := protokoll(); n != 2 {
		t.Errorf("Protokoll: %d Einträge, erwartet 2", n)
	}
	var alt, neu, grund string
	if err := pool.QueryRow(ctx, `SELECT details->>'eigentum_alt', details->>'eigentum_neu', details->>'grund'
		FROM audit_log WHERE datensatz_id = $1 AND kontext = 'Eigentum von Hand geändert'`, einfach).Scan(&alt, &neu, &grund); err != nil {
		t.Fatalf("Protokolleintrag lesen: %v", err)
	}
	if alt != "" || neu != mitteltopf.Land || grund != "Klassensatz aus LMF-Mitteln" {
		t.Errorf("Protokoll: alt %q, neu %q, Grund %q", alt, neu, grund)
	}

	// 3. Dasselbe noch einmal ändert nichts und schreibt nichts.
	if rec := eigentumSetzen(t, srv, pool, []string{einfach}, mitteltopf.Land, "doppelt"); !strings.Contains(rec.Body.String(), `"geaendert":0`) {
		t.Errorf("Wiederholung: %s — erwartet 0 geändert", rec.Body.String())
	}
	if n := protokoll(); n != 2 {
		t.Errorf("Wiederholung schrieb ins Protokoll: %d Einträge", n)
	}

	// 4. Zurück auf die Vorgabe: Die Angabe am Exemplar fällt, die Faustregel gilt wieder.
	if rec := eigentumSetzen(t, srv, pool, []string{einfach}, "", "versehentlich gesetzt"); rec.Code != http.StatusOK {
		t.Fatalf("Vorgabe: Status %d: %s", rec.Code, rec.Body.String())
	}
	pruefeKarte(t, exemplarKarten(t, srv, pool, lektuere), "EE-VORGABE", mitteltopf.Schultraeger, "vorgabe", "")

	// 5. Abweisen, ohne etwas zu ändern.
	unbekannt := "00000000-0000-4000-8000-000000000000"
	faelle := []struct {
		name     string
		ids      []string
		eigentum string
		grund    string
		status   int
	}{
		{"ohne Grund", []string{foerderverein}, mitteltopf.Land, "  ", http.StatusBadRequest},
		{"unbekanntes Eigentum", []string{foerderverein}, "stadt", "Grund", http.StatusBadRequest},
		{"keine Kennung", []string{"EE-FOERDERVEREIN"}, mitteltopf.Land, "Grund", http.StatusBadRequest},
		{"leere Auswahl", nil, mitteltopf.Land, "Grund", http.StatusBadRequest},
		{"eine gibt es nicht", []string{foerderverein, unbekannt}, mitteltopf.Land, "Grund", http.StatusNotFound},
	}
	for _, f := range faelle {
		if rec := eigentumSetzen(t, srv, pool, f.ids, f.eigentum, f.grund); rec.Code != f.status {
			t.Errorf("%s: Status %d, erwartet %d: %s", f.name, rec.Code, f.status, rec.Body.String())
		}
	}
	pruefeKarte(t, exemplarKarten(t, srv, pool, lektuere), "EE-FOERDERVEREIN", mitteltopf.Schultraeger, "vorgabe", "Förderverein")

	// 6. Die Datenbank verlangt zum Eigentum die Quelle (chk_exemplar_eigentum_mit_quelle).
	if _, err := pool.Exec(ctx, `UPDATE buecher_exemplare SET eigentum = 'land' WHERE id = $1`, foerderverein); err == nil {
		t.Error("Eigentum ohne Quelle wurde angenommen — chk_exemplar_eigentum_mit_quelle fehlt")
	}
}

// zaehleZeilen liest eine Zählung.
func zaehleZeilen(t *testing.T, pool *pgxpool.Pool, sql string, args ...any) int {
	t.Helper()
	var n int
	if err := pool.QueryRow(context.Background(), sql, args...).Scan(&n); err != nil {
		t.Fatalf("Zählung: %v", err)
	}
	return n
}

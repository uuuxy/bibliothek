package api

// PG-Integrationstests für die Tresen-Auskunft (audit_tresen_auskunft.go).
//
// Echtes Postgres statt pgxmock, weil die geprüfte Logik in SQL lebt (JSONB-Suche
// in audit_log.details, LEFT JOINs auf gelöschte Personen) — und weil die Ereignisse
// von den ECHTEN Schreibern stammen sollen (LogAusleihe, EndgueltigLoescheVerlust-
// Exemplare, TilgeSchuelerSpuren), nicht von nachgestellten INSERTs: Nur so prüft
// der Test den Live-Pfad und nicht die eigene Annahme über dessen Format.

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"bibliothek/auth"
	"bibliothek/db"
	"bibliothek/repository"

	"github.com/jackc/pgx/v5/pgxpool"
)

// tresenTestWelt legt Bearbeiter, Schüler, Titel und Exemplar an und räumt alles
// wieder ab — inklusive der Audit-Spuren, die die echten Schreiber hinterlassen.
type tresenTestWelt struct {
	adminID, schuelerID, titelID, exemplarID, barcode string
}

func baueTresenWelt(t *testing.T, pool *pgxpool.Pool) tresenTestWelt {
	t.Helper()
	ctx := context.Background()
	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	w := tresenTestWelt{barcode: "TRS-" + suffix}

	if err := pool.QueryRow(ctx, `
		INSERT INTO benutzer (barcode_id, vorname, nachname, email, rolle, aktiv)
		VALUES ($1, 'Theo', 'Theke', $2, 'admin', true) RETURNING id
	`, "TRSB-"+suffix, "tresen-"+suffix+"@example.org").Scan(&w.adminID); err != nil {
		t.Fatalf("Bearbeiter anlegen: %v", err)
	}
	if err := pool.QueryRow(ctx, `
		INSERT INTO schueler (barcode_id, vorname, nachname, klasse, abgaenger_jahr)
		VALUES ($1, 'Selma', 'Suchspur', '05A', 2030) RETURNING id
	`, "TRSS-"+suffix).Scan(&w.schuelerID); err != nil {
		t.Fatalf("Schüler anlegen: %v", err)
	}
	if err := pool.QueryRow(ctx, `
		INSERT INTO buecher_titel (titel, autor, medientyp)
		VALUES ('Tresen-Testband', 'Prüfer', 'Buch') RETURNING id
	`).Scan(&w.titelID); err != nil {
		t.Fatalf("Titel anlegen: %v", err)
	}
	if err := pool.QueryRow(ctx, `
		INSERT INTO buecher_exemplare (titel_id, barcode_id, ist_ausleihbar)
		VALUES ($1, $2, true) RETURNING id
	`, w.titelID, w.barcode).Scan(&w.exemplarID); err != nil {
		t.Fatalf("Exemplar anlegen: %v", err)
	}

	t.Cleanup(func() {
		aufraeumen(t, pool, `DELETE FROM audit_log WHERE datensatz_id IN ($1, $2)`, w.exemplarID, w.schuelerID)
		aufraeumen(t, pool, `DELETE FROM audit_logs WHERE details->>'barcode' = $1`, w.barcode)
		aufraeumen(t, pool, `DELETE FROM buecher_exemplare WHERE id = $1`, w.exemplarID)
		aufraeumen(t, pool, `DELETE FROM buecher_titel WHERE id = $1`, w.titelID)
		aufraeumen(t, pool, `DELETE FROM schueler WHERE id = $1`, w.schuelerID)
		aufraeumen(t, pool, `DELETE FROM benutzer WHERE id = $1`, w.adminID)
	})
	return w
}

// rufeTresenAuskunft führt den Handler mit Sitzung im Kontext aus (so, wie die
// Middleware sie nach RequirePermission hinterlegt) und liest die Antwort.
func rufeTresenAuskunft(t *testing.T, srv *Server, adminID, barcode string) TresenAuskunft {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/api/audit/tresen-auskunft?barcode="+barcode, nil)
	req = req.WithContext(context.WithValue(req.Context(), auth.ClaimsContextKey,
		&auth.Claims{UserID: adminID, Rolle: auth.RoleAdmin}))
	rec := httptest.NewRecorder()
	srv.TresenAuskunftHandler(repository.NewAuditRepository(srv.DB.Pool)).ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("Status %d, Körper: %s", rec.Code, rec.Body.String())
	}
	var auskunft TresenAuskunft
	if err := json.Unmarshal(rec.Body.Bytes(), &auskunft); err != nil {
		t.Fatalf("Antwort unlesbar: %v", err)
	}
	return auskunft
}

// Der namensgebende Fall: Exemplar ausgeliehen, zurückgegeben, als Verlust gebucht
// und ENDGÜLTIG gelöscht — die Zeile in buecher_exemplare existiert nicht mehr.
// Die Auskunft muss den Weg über den Audit-Snapshot finden und den Entleiher nennen.
func TestTresenAuskunftFindetGeloeschtesExemplar(t *testing.T) {
	pool := pgTestPool(t)
	ctx := context.Background()
	srv := &Server{DB: &db.Database{Pool: pool}}
	w := baueTresenWelt(t, pool)

	auditRepo := repository.NewAuditRepository(pool)
	if err := auditRepo.LogAusleihe(ctx, w.exemplarID, w.schuelerID, "", w.adminID); err != nil {
		t.Fatalf("LogAusleihe: %v", err)
	}
	if err := auditRepo.LogRueckgabe(ctx, w.exemplarID, w.schuelerID, "", w.adminID); err != nil {
		t.Fatalf("LogRueckgabe: %v", err)
	}

	// Verlust buchen und über den echten Löschpfad endgültig entfernen — der
	// schreibt den Barcode-Snapshot, an dem die Auskunft hängt.
	if _, err := pool.Exec(ctx, `
		UPDATE buecher_exemplare
		SET ist_ausgesondert = true, ist_ausleihbar = false, aussonderung_grund = 'VERLUST'
		WHERE id = $1`, w.exemplarID); err != nil {
		t.Fatalf("Verlust buchen: %v", err)
	}
	geloescht, err := repository.NewInventoryRepository(pool).
		EndgueltigLoescheVerlustExemplare(ctx, []string{w.exemplarID}, w.adminID)
	if err != nil || len(geloescht) != 1 {
		t.Fatalf("endgültig löschen: %v (gelöscht: %v)", err, geloescht)
	}

	auskunft := rufeTresenAuskunft(t, srv, w.adminID, w.barcode)

	if len(auskunft.Exemplare) != 1 {
		t.Fatalf("erwartet 1 Exemplar-Treffer, waren %d: %+v", len(auskunft.Exemplare), auskunft.Exemplare)
	}
	if e := auskunft.Exemplare[0]; e.Status != "geloescht" || e.Titel != "Tresen-Testband" {
		t.Errorf("Exemplar-Treffer falsch: %+v", e)
	}
	if len(auskunft.Ereignisse) != 2 {
		t.Fatalf("erwartet 2 Ereignisse (Ausleihe+Rückgabe), waren %d", len(auskunft.Ereignisse))
	}
	for _, e := range auskunft.Ereignisse {
		if e.Entleiher != "Selma Suchspur" || e.Klasse != "05A" || e.PersonenbezugGetilgt {
			t.Errorf("Ereignis %s: Entleiher %q / Klasse %q / getilgt %v — erwartet Selma Suchspur, 05A, false",
				e.Aktion, e.Entleiher, e.Klasse, e.PersonenbezugGetilgt)
		}
		if e.Bearbeiter != "Theo Theke" {
			t.Errorf("Bearbeiter %q, erwartet Theo Theke", e.Bearbeiter)
		}
	}

	// Die Zusage „jeder Abruf protokolliert sich selbst": genau ein Eintrag mit
	// Barcode und Trefferzahlen im Admin-Audit-Log.
	var protokolle int
	if err := pool.QueryRow(ctx, `
		SELECT count(*) FROM audit_logs
		WHERE aktion = 'TRESEN_AUSKUNFT' AND admin_id = $1 AND details->>'barcode' = $2
	`, w.adminID, w.barcode).Scan(&protokolle); err != nil {
		t.Fatalf("Protokoll zählen: %v", err)
	}
	if protokolle != 1 {
		t.Errorf("erwartet genau 1 TRESEN_AUSKUNFT-Protokolleintrag, waren %d", protokolle)
	}
}

// Nach der DSGVO-Tilgung darf auch dieser Leseweg nichts mehr zuordnen — der
// Vorgang bleibt sichtbar (das Exemplar war unterwegs), aber ausdrücklich als
// „Personenbezug getilgt", nicht als leere Zelle.
func TestTresenAuskunftZeigtGetilgtenBezug(t *testing.T) {
	pool := pgTestPool(t)
	ctx := context.Background()
	srv := &Server{DB: &db.Database{Pool: pool}}
	w := baueTresenWelt(t, pool)

	auditRepo := repository.NewAuditRepository(pool)
	if err := auditRepo.LogAusleihe(ctx, w.exemplarID, w.schuelerID, "", w.adminID); err != nil {
		t.Fatalf("LogAusleihe: %v", err)
	}
	// Die echte Tilgung (dieselbe Liste wie Purge und Cron), nicht ein nachgebautes UPDATE.
	if err := repository.TilgeSchuelerSpuren(ctx, pool, w.schuelerID, "Testtilgung"); err != nil {
		t.Fatalf("TilgeSchuelerSpuren: %v", err)
	}

	auskunft := rufeTresenAuskunft(t, srv, w.adminID, w.barcode)

	if len(auskunft.Exemplare) != 1 || auskunft.Exemplare[0].Status != "im_bestand" {
		t.Fatalf("erwartet 1 Treffer im Bestand, war: %+v", auskunft.Exemplare)
	}
	if len(auskunft.Ereignisse) != 1 {
		t.Fatalf("erwartet 1 Ereignis, waren %d", len(auskunft.Ereignisse))
	}
	if e := auskunft.Ereignisse[0]; !e.PersonenbezugGetilgt || e.Entleiher != "" {
		t.Errorf("nach Tilgung erwartet getilgt=true ohne Namen, war: %+v", e)
	}
}

// Ein nie vergebener Barcode ist eine leere, aber gültige Antwort (200) — und auch
// der ergebnislose Abruf hinterlässt seine Protokollspur.
func TestTresenAuskunftOhneTreffer(t *testing.T) {
	pool := pgTestPool(t)
	ctx := context.Background()
	srv := &Server{DB: &db.Database{Pool: pool}}
	w := baueTresenWelt(t, pool)

	auskunft := rufeTresenAuskunft(t, srv, w.adminID, "NIE-VERGEBEN-4711")
	if len(auskunft.Exemplare) != 0 || len(auskunft.Ereignisse) != 0 {
		t.Errorf("erwartet leere Auskunft, war: %+v", auskunft)
	}

	var protokolle int
	if err := pool.QueryRow(ctx, `
		SELECT count(*) FROM audit_logs
		WHERE aktion = 'TRESEN_AUSKUNFT' AND admin_id = $1 AND details->>'barcode' = 'NIE-VERGEBEN-4711'
	`, w.adminID).Scan(&protokolle); err != nil {
		t.Fatalf("Protokoll zählen: %v", err)
	}
	if protokolle != 1 {
		t.Errorf("auch der Abruf ohne Treffer muss protokolliert sein (waren %d)", protokolle)
	}
	t.Cleanup(func() {
		aufraeumen(t, pool, `DELETE FROM audit_logs WHERE details->>'barcode' = 'NIE-VERGEBEN-4711'`)
	})
}

// Ohne Barcode gibt es keine Suche — 400, bevor irgendetwas die Datenbank berührt
// (deshalb kommt dieser Fall ohne Pool aus).
func TestTresenAuskunftOhneBarcode(t *testing.T) {
	srv := &Server{}
	req := httptest.NewRequest(http.MethodGet, "/api/audit/tresen-auskunft", nil)
	rec := httptest.NewRecorder()
	srv.TresenAuskunftHandler(nil).ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("erwartet 400 ohne barcode, war %d", rec.Code)
	}
}

package system

// PG-Integrationstests für die Theken-Auskunft (theken_auskunft.go).
//
// Echtes Postgres statt pgxmock, weil die geprüfte Logik in SQL lebt (JSONB-Suche
// in audit_log.details, LEFT JOINs auf gelöschte Personen) — und weil die Ereignisse
// von den echten Schreibern stammen sollen (ProtokolliereAusleihe, LoescheVerlusteEndgueltig,
// TilgeSchuelerSpuren), nicht von nachgestellten INSERTs: Nur so prüft
// der Test den Live-Pfad und nicht die eigene Annahme über dessen Format.

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"

	"bibliothek/api/tuer"
	"bibliothek/auth"
	"bibliothek/db"
	"bibliothek/internal/pgtest"
	"bibliothek/repository"

	"github.com/jackc/pgx/v5/pgxpool"
)

// thekenTestWelt legt Bearbeiter, Schüler, Titel und Exemplar an und räumt alles
// wieder ab — inklusive der Audit-Spuren, die die echten Schreiber hinterlassen.
type thekenTestWelt struct {
	adminID, schuelerID, titelID, exemplarID, barcode string
}

func baueThekenWelt(t *testing.T, pool *pgxpool.Pool) thekenTestWelt {
	t.Helper()
	ctx := context.Background()
	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	w := thekenTestWelt{barcode: "TRS-" + suffix}

	if err := pool.QueryRow(ctx, `
		INSERT INTO benutzer (vorname, nachname, email, rolle, aktiv)
		VALUES ('Theo', 'Theke', $1, 'admin', true) RETURNING id
	`, "tresen-"+suffix+"@example.org").Scan(&w.adminID); err != nil {
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
		VALUES ('Theken-Testband', 'Prüfer', 'Buch') RETURNING id
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
		pgtest.RaeumeAuf(t, pool, `DELETE FROM audit_log WHERE datensatz_id IN ($1, $2)`, w.exemplarID, w.schuelerID)
		pgtest.RaeumeAuf(t, pool, `DELETE FROM audit_logs WHERE details->>'barcode' = $1`, w.barcode)
		pgtest.RaeumeAuf(t, pool, `DELETE FROM buecher_exemplare WHERE id = $1`, w.exemplarID)
		pgtest.RaeumeAuf(t, pool, `DELETE FROM buecher_titel WHERE id = $1`, w.titelID)
		pgtest.RaeumeAuf(t, pool, `DELETE FROM schueler WHERE id = $1`, w.schuelerID)
		pgtest.RaeumeAuf(t, pool, `DELETE FROM benutzer WHERE id = $1`, w.adminID)
	})
	return w
}

// rufeThekenAuskunft führt die Tür mit Sitzung im Kontext aus (so, wie die
// Middleware sie nach verlangeRecht hinterlegt) und liest die Antwort.
func rufeThekenAuskunft(t *testing.T, srv *Tueren, adminID, barcode string) thekenAuskunft {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/api/audit/tresen-auskunft?barcode="+barcode, nil)
	req = req.WithContext(context.WithValue(req.Context(), auth.ClaimsSchluessel,
		&auth.Claims{KontoID: adminID, Rolle: auth.RolleAdmin}))
	rec := httptest.NewRecorder()
	srv.ThekenAuskunft(repository.NewProtokollAbfragen(srv.DB.Pool)).ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("Status %d, Körper: %s", rec.Code, rec.Body.String())
	}
	var auskunft thekenAuskunft
	if err := json.Unmarshal(rec.Body.Bytes(), &auskunft); err != nil {
		t.Fatalf("Antwort unlesbar: %v", err)
	}
	return auskunft
}

// Der namensgebende Fall: Exemplar ausgeliehen, zurückgegeben, als Verlust gebucht
// und endgültig gelöscht — die Zeile in buecher_exemplare existiert nicht mehr.
// Die Auskunft muss den Weg über die Angaben im Protokoll finden und den Entleiher nennen.
func TestThekenAuskunftFindetGeloeschtesExemplar(t *testing.T) {
	pool := pgtest.Pool(t)
	ctx := context.Background()
	srv := &Tueren{Basis: &tuer.Basis{DB: &db.Verbindung{Pool: pool}}}
	w := baueThekenWelt(t, pool)

	auditRepo := repository.NewProtokollAbfragen(pool)
	pgtest.InTx(t, pool, func(tx pgx.Tx) error {
		if err := auditRepo.ProtokolliereAusleihe(ctx, tx, w.exemplarID, w.schuelerID, "", w.adminID); err != nil {
			return err
		}
		return auditRepo.ProtokolliereRueckgabe(ctx, tx, w.exemplarID, w.schuelerID, "", w.adminID)
	})

	// Verlust buchen und über den echten Löschpfad endgültig entfernen; der schreibt den
	// Barcode ins Protokoll, und daran hängt die Auskunft.
	if _, err := pool.Exec(ctx, `
		UPDATE buecher_exemplare
		SET ist_ausgesondert = true, ist_ausleihbar = false, aussonderung_grund = 'VERLUST'
		WHERE id = $1`, w.exemplarID); err != nil {
		t.Fatalf("Verlust buchen: %v", err)
	}
	geloescht, err := repository.NewInventurAbfragen(pool).
		LoescheVerlusteEndgueltig(ctx, []string{w.exemplarID}, w.adminID)
	if err != nil || len(geloescht) != 1 {
		t.Fatalf("endgültig löschen: %v (gelöscht: %v)", err, geloescht)
	}

	auskunft := rufeThekenAuskunft(t, srv, w.adminID, w.barcode)

	if len(auskunft.Exemplare) != 1 {
		t.Fatalf("len(auskunft.Exemplare) = %d, erwartet 1 Exemplar-Treffer: %+v", len(auskunft.Exemplare), auskunft.Exemplare)
	}
	if e := auskunft.Exemplare[0]; e.Status != "geloescht" || e.Titel != "Theken-Testband" {
		t.Errorf("Exemplar-Treffer falsch: %+v", e)
	}
	if len(auskunft.Ereignisse) != 2 {
		t.Fatalf("len(auskunft.Ereignisse) = %d, erwartet 2 Ereignisse (Ausleihe+Rückgabe)", len(auskunft.Ereignisse))
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
		t.Errorf("protokolle = %d, erwartet genau 1 TRESEN_AUSKUNFT-Protokolleintrag", protokolle)
	}
}

// Nach der DSGVO-Tilgung darf auch dieser Leseweg nichts mehr zuordnen — der
// Vorgang bleibt sichtbar (das Exemplar war unterwegs), aber ausdrücklich als
// „Personenbezug getilgt", nicht als leere Zelle.
func TestThekenAuskunftZeigtGetilgtenBezug(t *testing.T) {
	pool := pgtest.Pool(t)
	ctx := context.Background()
	srv := &Tueren{Basis: &tuer.Basis{DB: &db.Verbindung{Pool: pool}}}
	w := baueThekenWelt(t, pool)

	auditRepo := repository.NewProtokollAbfragen(pool)
	pgtest.InTx(t, pool, func(tx pgx.Tx) error {
		return auditRepo.ProtokolliereAusleihe(ctx, tx, w.exemplarID, w.schuelerID, "", w.adminID)
	})
	// Die echte Tilgung (dieselbe Liste wie Purge und Cron), nicht ein nachgebautes UPDATE.
	if err := repository.TilgeSchuelerSpuren(ctx, pool, w.schuelerID, "Testtilgung"); err != nil {
		t.Fatalf("TilgeSchuelerSpuren: %v", err)
	}

	auskunft := rufeThekenAuskunft(t, srv, w.adminID, w.barcode)

	if len(auskunft.Exemplare) != 1 || auskunft.Exemplare[0].Status != "im_bestand" {
		t.Fatalf("auskunft.Exemplare = %+v, erwartet 1 Treffer im Bestand", auskunft.Exemplare)
	}
	if len(auskunft.Ereignisse) != 1 {
		t.Fatalf("len(auskunft.Ereignisse) = %d, erwartet 1 Ereignis", len(auskunft.Ereignisse))
	}
	if e := auskunft.Ereignisse[0]; !e.PersonenbezugGetilgt || e.Entleiher != "" {
		t.Errorf("nach Tilgung erwartet getilgt=true ohne Namen, war: %+v", e)
	}
}

// Ein nie vergebener Barcode ist eine leere, aber gültige Antwort (200) — und auch
// der ergebnislose Abruf hinterlässt seine Protokollspur.
func TestThekenAuskunftOhneTreffer(t *testing.T) {
	pool := pgtest.Pool(t)
	ctx := context.Background()
	srv := &Tueren{Basis: &tuer.Basis{DB: &db.Verbindung{Pool: pool}}}
	w := baueThekenWelt(t, pool)

	auskunft := rufeThekenAuskunft(t, srv, w.adminID, "NIE-VERGEBEN-4711")
	if len(auskunft.Exemplare) != 0 || len(auskunft.Ereignisse) != 0 {
		t.Errorf("rufeThekenAuskunft(…) = %+v, erwartet leere Auskunft", auskunft)
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
		pgtest.RaeumeAuf(t, pool, `DELETE FROM audit_logs WHERE details->>'barcode' = 'NIE-VERGEBEN-4711'`)
	})
}

// Ohne Barcode gibt es keine Suche — 400, bevor irgendetwas die Datenbank berührt
// (deshalb kommt dieser Fall ohne Pool aus).
func TestThekenAuskunftOhneBarcode(t *testing.T) {
	srv := &Tueren{Basis: &tuer.Basis{}}
	req := httptest.NewRequest(http.MethodGet, "/api/audit/tresen-auskunft", nil)
	rec := httptest.NewRecorder()
	srv.ThekenAuskunft(nil).ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("rec.Code = %d, erwartet 400 ohne barcode", rec.Code)
	}
}

// Ein Kollege leiht wie jeder Leser (Migration 125): ProtokolliereAusleihe bekommt seine
// Leser-Id als schuelerID, benutzerID bleibt leer (alle acht Aufrufer). Löste die Auskunft den
// Namen über die Sicht `schueler` auf, fände sie für einen Kollegen nichts und meldete
// „Personenbezug getilgt", obwohl Person und Bezug da sind.
func TestThekenAuskunftNenntDenKollegen(t *testing.T) {
	pool := pgtest.Pool(t)
	ctx := context.Background()
	srv := &Tueren{Basis: &tuer.Basis{DB: &db.Verbindung{Pool: pool}}}
	w := baueThekenWelt(t, pool)

	var kollegeID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO leser (barcode_id, vorname, nachname, art)
		VALUES ($1, 'Kim', 'Kollegin', 'liv') RETURNING id
	`, "TRSK-"+w.barcode).Scan(&kollegeID); err != nil {
		t.Fatalf("Kollege anlegen: %v", err)
	}
	t.Cleanup(func() {
		pgtest.RaeumeAuf(t, pool, `DELETE FROM audit_log WHERE datensatz_id = $1`, kollegeID)
		pgtest.RaeumeAuf(t, pool, `DELETE FROM leser WHERE id = $1`, kollegeID)
	})

	auditRepo := repository.NewProtokollAbfragen(pool)
	pgtest.InTx(t, pool, func(tx pgx.Tx) error {
		return auditRepo.ProtokolliereAusleihe(ctx, tx, w.exemplarID, kollegeID, "", w.adminID)
	})

	auskunft := rufeThekenAuskunft(t, srv, w.adminID, w.barcode)
	if len(auskunft.Ereignisse) != 1 {
		t.Fatalf("len(auskunft.Ereignisse) = %d, erwartet 1 Ereignis", len(auskunft.Ereignisse))
	}
	e := auskunft.Ereignisse[0]
	if e.PersonenbezugGetilgt || e.Entleiher != "Kim Kollegin" || e.Klasse != "LiV" {
		t.Errorf("Kollege als Entleiher: erwartet Kim Kollegin / LiV ohne Tilgung, war: %+v", e)
	}
}

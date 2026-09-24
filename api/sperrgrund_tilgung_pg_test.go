package api

import (
	"context"
	"net/http"
	"testing"

	"bibliothek/db"
)

// Der Grund einer Sperre fällt mit der Anonymisierung auch im Admin-Protokoll.
//
// anonymisiereAbgaenger ersetzt block_reason, weil ein alter Grund andere Personen nennen
// kann. Seit dem 24.09.2026 schreibt die Sperr-Tür denselben Freitext ins Admin-Protokoll
// (LESER_GESPERRT, LESER_ENTSPERRT: grund), und bis zum 24.09.2026 stand er im Übergehen an
// der Theke (OVERRIDE_BLOCK: reason). Die Spuren-Tilgung räumte dort nur LUSD-ID und
// Ausweisnummer; der Grund blieb bis zur Audit-Aufbewahrung (24 Monate) neben der Kennung
// des Lesers stehen (Rasterdurchgang 24.09.2026).
//
// Rot gesehen am Rückbau: die Schlüssel grund und reason aus der Anweisung
// „audit_logs (LUSD-ID, Barcodes, Sperrgrund)" in repository.spurTilgungen entfernt — drei
// Einträge der Tür und der alte OVERRIDE_BLOCK behalten den Grund.
func TestAnonymisierung_TilgtDenSperrgrundImProtokoll(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	ctx := context.Background()
	akteur := adminFuerAudit(t, pool)
	const grund = "SPERRGRUND-PROBE Mutter von Max Muster zahlt nicht"

	var id string
	if err := pool.QueryRow(ctx,
		`INSERT INTO schueler (barcode_id, vorname, nachname, klasse, abgaenger_jahr)
		 VALUES ('SGT-1', 'Max', 'Muster', '10R', 2026) RETURNING id`).Scan(&id); err != nil {
		t.Fatalf("Schüler anlegen: %v", err)
	}
	for _, body := range []string{
		`{"is_locked":true,"reason":"` + grund + `"}`,
		`{"is_locked":false}`,
		`{"is_locked":true,"reason":"` + grund + `"}`,
	} {
		if rec := sperrTuerAufruf(t, pool, akteur, id, body); rec.Code != http.StatusOK {
			t.Fatalf("Sperr-Tür %s: %d %s", body, rec.Code, rec.Body.String())
		}
	}
	// So schrieb die Theke das Übergehen einer Sperre bis zum 24.09.2026.
	if _, err := pool.Exec(ctx, `INSERT INTO audit_logs (aktion, details) VALUES ('OVERRIDE_BLOCK',
		jsonb_build_object('schueler_id', $1::text, 'reason', 'Ausleihsperre manuell ignoriert (Manuelle Sperre: ' || $2 || ')'))`,
		id, grund); err != nil {
		t.Fatalf("alten OVERRIDE_BLOCK anlegen: %v", err)
	}

	mitGrund := func() int {
		t.Helper()
		var n int
		if err := pool.QueryRow(ctx,
			`SELECT count(*) FROM audit_logs WHERE details::text LIKE '%' || $1 || '%'`, grund).Scan(&n); err != nil {
			t.Fatal(err)
		}
		return n
	}
	// Positivkontrolle: Ohne die vier Einträge wäre die Prüfung unten grün, ohne etwas zu prüfen.
	if n := mitGrund(); n != 4 {
		t.Fatalf("vor der Anonymisierung %d Einträge mit dem Grund, erwartet 4 (drei der Tür, ein alter)", n)
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer db.SafeRollback(ctx, tx)
	if err := anonymisiereAbgaenger(ctx, tx, id); err != nil {
		t.Fatalf("anonymisieren: %v", err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}

	if n := mitGrund(); n != 0 {
		t.Errorf("der Sperrgrund überlebt die Anonymisierung in %d Einträgen des Admin-Protokolls", n)
	}
	// Die Einträge selbst bleiben: Wer wann gesperrt, aufgehoben oder übergangen hat, ist
	// Rechenschaft, und schueler_id ist nach der Anonymisierung ein Pseudonym.
	var bleiben int
	if err := pool.QueryRow(ctx,
		`SELECT count(*) FROM audit_logs WHERE details->>'schueler_id' = $1
		   AND aktion IN ('LESER_GESPERRT', 'LESER_ENTSPERRT', 'OVERRIDE_BLOCK')`, id).Scan(&bleiben); err != nil {
		t.Fatal(err)
	}
	if bleiben != 4 {
		t.Errorf("%d Einträge mit der Kennung des Lesers nach der Anonymisierung, erwartet 4", bleiben)
	}
}

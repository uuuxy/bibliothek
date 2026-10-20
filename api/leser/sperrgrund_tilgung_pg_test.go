package leser

import (
	"context"
	"net/http"
	"testing"

	"bibliothek/db"
	"bibliothek/internal/pgtest"
	"bibliothek/internal/testdaten"
	"bibliothek/repository"
)

// Der Grund einer Sperre fällt mit der Anonymisierung auch im Admin-Protokoll.
//
// repository.AnonymisiereAbgaenger ersetzt block_reason, weil ein alter Grund andere
// Personen nennen kann. Die Sperr-Tür schreibt denselben Freitext ins Protokoll der
// Verwaltung (LESER_GESPERRT, LESER_ENTSPERRT: grund), und das frühere Übergehen an der Theke
// (OVERRIDE_BLOCK: reason) trug ihn ebenso. Räumte die Spuren-Tilgung dort nur LUSD-ID und
// Ausweisnummer, bliebe der Grund bis zum Ende der Aufbewahrung (24 Monate) neben der Kennung
// des Lesers stehen.
//
// Rot gesehen am Rückbau: die Schlüssel grund und reason aus der Anweisung
// „audit_logs (LUSD-ID, Barcodes, Sperrgrund)" in repository.spurTilgungen entfernt — drei
// Einträge der Tür und der alte OVERRIDE_BLOCK behalten den Grund.
func TestAnonymisierung_TilgtDenSperrgrundImProtokoll(t *testing.T) {
	pool := pgtest.Pool(t)
	testdaten.LeereBestandsdaten(t, pool)
	ctx := context.Background()
	akteur := testdaten.LegeBearbeiterAn(t, pool)
	const grund = "SPERRGRUND-PROBE Mutter von Max Muster zahlt nicht"

	id := testdaten.LegeSchuelerMitNamenAn(t, pool, "SGT-1", "Max", "Muster", "10R", 2026)
	for _, body := range []string{
		`{"is_locked":true,"reason":"` + grund + `"}`,
		`{"is_locked":false}`,
		`{"is_locked":true,"reason":"` + grund + `"}`,
	} {
		if rec := sperrTuerAufruf(t, pool, akteur, id, body); rec.Code != http.StatusOK {
			t.Fatalf("Sperr-Tür %s: %d %s", body, rec.Code, rec.Body.String())
		}
	}
	// So schrieb die Theke das Übergehen einer Sperre früher.
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
	defer db.Verwerfe(ctx, tx)
	if err := repository.AnonymisiereAbgaenger(ctx, tx, id); err != nil {
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

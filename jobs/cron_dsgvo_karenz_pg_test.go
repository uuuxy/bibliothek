package jobs

import (
	"context"
	"os"
	"testing"

	"bibliothek/repository"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Die Karenz-Uhr am nächtlichen Job (Rasterdurchgang 02.09.2026, Frage 7): Kein Test
// fuhr RunGDPRAnonymizeOldData über eine Zeile mit altem abgaenger_seit und frischem
// aktualisiert_am — das alte Prädikat (nur aktualisiert_am) und ein fest verdrahtetes
// 360 wären unter allen Tests grün geblieben. Hier: die Uhr ist abgaenger_seit, nicht
// aktualisiert_am, und die Frist kommt aus der Einstellung abgaenger_karenz_tage.
func TestGDPRAnonymize_KarenzUhrUndEinstellung(t *testing.T) {
	adminDSN := os.Getenv(drillEnvVar)
	if adminDSN == "" {
		t.Skipf("%s nicht gesetzt — Test übersprungen", drillEnvVar)
	}
	_, dsn := legeProbeDatenbankAn(t, adminDSN, "karenz")
	befuelleQuelle(t, dsn)
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("Pool: %v", err)
	}
	t.Cleanup(pool.Close)

	lege := func(barcode string, seit, aktualisiert string) string {
		t.Helper()
		var id string
		if err := pool.QueryRow(ctx, `
			INSERT INTO schueler (barcode_id, vorname, nachname, klasse, abgaenger_jahr, ist_abgaenger, ist_gesperrt, block_reason, abgaenger_seit, aktualisiert_am)
			VALUES ($1, 'Karenz', $1, 'ABG', 2026, true, true, $4, NOW() - $2::interval, NOW() - $3::interval)
			RETURNING id`, barcode, seit, aktualisiert, repository.AbgaengerSperrgrundKarenz).Scan(&id); err != nil {
			t.Fatal(err)
		}
		return id
	}
	setzeKarenz := func(tage string) {
		t.Helper()
		if _, err := pool.Exec(ctx, `INSERT INTO system_einstellungen (schluessel, wert) VALUES ($1, $2)
			ON CONFLICT (schluessel) DO UPDATE SET wert = EXCLUDED.wert`, repository.AbgaengerKarenzSchluessel, tage); err != nil {
			t.Fatal(err)
		}
	}
	anonymisiert := func(id string) bool {
		t.Helper()
		var ja bool
		if err := pool.QueryRow(ctx, `SELECT anonymized_at IS NOT NULL FROM schueler WHERE id = $1`, id).Scan(&ja); err != nil {
			t.Fatal(err)
		}
		return ja
	}

	// Uhr: A ist seit 100 Tagen Abgänger, aber gestern angefasst; B ist seit heute
	// Abgänger, aber seit 400 Tagen unberührt. Bei Karenz 90 fällt A, B bleibt.
	setzeKarenz("90")
	a := lege("KZ-A", "100 days", "1 day")
	b := lege("KZ-B", "0 days", "400 days")
	s := NewScheduler(pool, repository.NewAuditRepository(pool))
	s.RunGDPRAnonymizeOldData()
	if !anonymisiert(a) {
		t.Error("A (abgaenger_seit vor 100 Tagen) muss bei Karenz 90 anonymisiert sein — rechnet der Job mit aktualisiert_am?")
	}
	if anonymisiert(b) {
		t.Error("B (abgaenger_seit heute) darf nicht anonymisiert sein — rechnet der Job mit aktualisiert_am?")
	}

	// Einstellung: dieselbe Zeile wie A überlebt eine Karenz von 200 Tagen.
	setzeKarenz("200")
	c := lege("KZ-C", "100 days", "1 day")
	s.RunGDPRAnonymizeOldData()
	if anonymisiert(c) {
		t.Error("C darf bei Karenz 200 nicht anonymisiert sein — liest der Job die Einstellung?")
	}
}

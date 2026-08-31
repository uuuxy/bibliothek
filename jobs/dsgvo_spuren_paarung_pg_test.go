package jobs

import (
	"context"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Die PAARUNG aus Purge-Tilgung und Cron-Tilgung, am echten Postgres.
//
// Fund (31.08.2026, Auswertung des Komplett-Durchgangs): repository.TilgeSchuelerSpuren
// (Purge/LUSD-Abgang) und bereinigeAnonymisierteSchuelerSpuren (nächtlicher Cron über
// anonymized_at) beantworteten dieselbe Frage — „welche Spuren eines Schülers stehen in
// den Neben-Tabellen?" — mit zwei getrennt gepflegten Statement-Listen. Der Cron hatte
// drei der vier Statements; es fehlte genau die Lesehistorie (audit_log mit
// tabelle='ausleihen'). Folge: Nach der Cron-Anonymisierung stand der KLARNAME des
// Schülers (details->>'entleiher') weiter in jeder CHECKOUT-Zeile — „anonymisiert" laut
// schueler-Zeile, re-identifizierbar über das Protokoll, bis irgendwann die (abschaltbare!)
// Lesehistorie-Befristung ihn zufällig einholte. Klassiker Doppelte Wahrheitsquelle;
// seit diesem Fund teilen beide Pfade die Statement-Liste (repository.SpurTilgungen).
func TestSpurenTilgung_CronDecktDieselbenSpurenWieDerPurge(t *testing.T) {
	adminDSN := os.Getenv(drillEnvVar)
	if adminDSN == "" {
		t.Skipf("%s nicht gesetzt — Test übersprungen", drillEnvVar)
	}
	_, dsn := legeProbeDatenbankAn(t, adminDSN, "spuren")
	befuelleQuelle(t, dsn)

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("Pool: %v", err)
	}
	t.Cleanup(pool.Close)

	// Ein anonymisierter Schüler (den der Cron aufgreifen muss) und ein aktiver
	// (dessen Spuren stehen bleiben müssen — Über-Tilgungs-Gegenprobe).
	var anonID, aktivID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO schueler (vorname, nachname, klasse, barcode_id, abgaenger_jahr, anonymized_at)
		VALUES (left(md5(random()::text), 8), 'Anonym', '', 'ANON-SPUR-1', 2020, NOW())
		RETURNING id`).Scan(&anonID); err != nil {
		t.Fatalf("anonymisierten Schüler anlegen: %v", err)
	}
	if err := pool.QueryRow(ctx, `
		INSERT INTO schueler (vorname, nachname, klasse, barcode_id, abgaenger_jahr)
		VALUES ('Aktiv', 'Bleibt', '05A', 'S-SPUR-AKTIV', 2030)
		RETURNING id`).Scan(&aktivID); err != nil {
		t.Fatalf("aktiven Schüler anlegen: %v", err)
	}

	// Je eine Lesehistorie-Zeile, wie sie die Ausleihe schreibt (repository/audit_books.go:
	// tabelle='ausleihen', Schüler nur in details), plus je eine Vormerkung und je eine
	// schueler-Historienzeile — die Spuren, die der Purge-Pfad nachweislich tilgt.
	for _, s := range []struct{ id, name string }{
		{anonID, "Klarname Getilgt"},
		{aktivID, "Klarname Bleibt"},
	} {
		if _, err := pool.Exec(ctx, `
			INSERT INTO audit_log (tabelle, aktion, datensatz_id, akteur, details)
			VALUES ('ausleihen', 'CHECKOUT', gen_random_uuid(), 'USER',
			        jsonb_build_object('schueler_id', $1::text, 'entleiher', $2::text))`,
			s.id, s.name); err != nil {
			t.Fatalf("Lesehistorie-Zeile: %v", err)
		}
	}

	s := NewScheduler(pool, nil)
	s.bereinigeAnonymisierteSchuelerSpuren(ctx)

	// Die Frage der Art.-15-Auskunft und der Purge-Tilgung, wortgleich: Für den
	// anonymisierten Schüler darf sie nichts mehr finden.
	var verbleibend int
	if err := pool.QueryRow(ctx, `
		SELECT count(*) FROM audit_log
		WHERE tabelle = 'ausleihen' AND details->>'schueler_id' = $1`, anonID).Scan(&verbleibend); err != nil {
		t.Fatalf("Lesehistorie zählen: %v", err)
	}
	if verbleibend != 0 {
		t.Errorf("%d Lesehistorie-Zeilen überleben die Cron-Anonymisierung — der Cron tilgt "+
			"andere Spuren als der Purge (Doppelte Wahrheitsquelle)", verbleibend)
	}

	// Der Klarname darf NIRGENDS in den Protokoll-Details überleben — wertbasiert
	// gesucht, damit auch ein künftiger neuer Schlüssel auffällt.
	var klarname int
	if err := pool.QueryRow(ctx,
		`SELECT count(*) FROM audit_log WHERE details::text LIKE '%Klarname Getilgt%'`).Scan(&klarname); err != nil {
		t.Fatalf("Klarnamen suchen: %v", err)
	}
	if klarname != 0 {
		t.Errorf("der Klarname des anonymisierten Schülers steht noch in %d audit_log-Details", klarname)
	}

	// Gegenprobe am Detektor: Die Zeile des aktiven Schülers ist unangetastet.
	var aktivDa int
	if err := pool.QueryRow(ctx, `
		SELECT count(*) FROM audit_log
		WHERE tabelle = 'ausleihen' AND details->>'schueler_id' = $1
		  AND details->>'entleiher' = 'Klarname Bleibt'`, aktivID).Scan(&aktivDa); err != nil {
		t.Fatalf("aktive Zeile zählen: %v", err)
	}
	if aktivDa != 1 {
		t.Errorf("Zeile des aktiven Schülers: %d (erwartet 1) — die Tilgung greift zu weit", aktivDa)
	}
}

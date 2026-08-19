package jobs

import (
	"context"
	"os"
	"strings"
	"testing"

	"bibliothek/repository"

	"github.com/jackc/pgx/v5/pgxpool"
)

// TestGDPRAnonymizeOldData_LeertAllePII deckt einen kritischen, lange unentdeckten Fehler
// ab (PII-Lebenszyklus-Sweep 19.08.2026): RunGDPRAnonymizeOldData referenzierte eine nicht
// existente Spalte (foto_url) und scheiterte deshalb ZUR LAUFZEIT — die Anonymisierung lief
// NIE, der Fehler wurde nur geloggt. Die bestehenden gdpr_test.go nutzen pgxmock und konnten
// das prinzipiell nicht sehen; erst dieser Lauf gegen echtes Postgres deckt es auf.
//
// Der Test befüllt einen alten, soft-gelöschten Schüler mit unverwechselbaren Sentinel-Werten
// in JEDER PII-Spalte und prüft nach der Anonymisierung, dass (a) sie wirklich lief
// (anonymized_at gesetzt) und (b) KEIN Sentinel mehr in der Zeile steht. Eine neu
// hinzugefügte PII-Spalte, die der Anonymisierer vergisst, fällt hier auf, sobald der Seed
// sie mit einem Sentinel füllt.
func TestGDPRAnonymizeOldData_LeertAllePII(t *testing.T) {
	adminDSN := os.Getenv(drillEnvVar)
	if adminDSN == "" {
		t.Skipf("%s nicht gesetzt — Test übersprungen", drillEnvVar)
	}

	_, dsn := legeProbeDatenbankAn(t, adminDSN, "gdpranon")
	befuelleQuelle(t, dsn)

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("Pool: %v", err)
	}
	t.Cleanup(pool.Close)

	// Sentinel-Werte: unverwechselbar, damit ein Überleben eindeutig auffällt.
	const (
		vn     = "SENTINELVORNAME"
		nn     = "SENTINELNACHNAME"
		strdat = "SENTINELSTRASSE"
		ortdat = "SENTINELORT"
		email  = "sentinel@eltern.invalid"
		lusd   = "SENTINEL-LUSD-42"
		reason = "SENTINEL Sperrgrund vertraulich"
		barc   = "SENTINEL-BARCODE"
	)
	var id string
	if err := pool.QueryRow(ctx, `
		INSERT INTO schueler
		    (barcode_id, vorname, nachname, klasse, geburtsdatum, abgaenger_jahr, lusd_id,
		     strasse, hausnummer, plz, ort, eltern_email,
		     ist_gesperrt, block_reason, deleted_at)
		VALUES ($1, $2, $3, '7b', '2011-05-05', 2030, $4,
		        $5, '12a', '61250', $6, $7,
		        true, $8, now() - interval '200 days')
		RETURNING id`,
		barc, vn, nn, lusd, strdat, ortdat, email, reason).Scan(&id); err != nil {
		t.Fatalf("Schüler anlegen: %v", err)
	}

	s := NewScheduler(pool, repository.NewAuditRepository(pool))
	s.RunGDPRAnonymizeOldData()

	// (a) Die Anonymisierung MUSS gelaufen sein. Vor dem Fix scheiterte die Query still
	// (foto_url/anonymized_at existierten nicht) → anonymized_at bliebe NULL.
	var anonymisiert bool
	if err := pool.QueryRow(ctx,
		`SELECT anonymized_at IS NOT NULL FROM schueler WHERE id = $1`, id).Scan(&anonymisiert); err != nil {
		t.Fatalf("anonymized_at lesen: %v", err)
	}
	if !anonymisiert {
		t.Fatal("RunGDPRAnonymizeOldData hat den fälligen Schüler NICHT anonymisiert (Query still gescheitert?)")
	}

	// (b) KEIN Sentinel darf überleben — die ganze Zeile als Text zusammenziehen und prüfen.
	var zeile string
	if err := pool.QueryRow(ctx, `
		SELECT concat_ws('|',
			coalesce(vorname,''), coalesce(nachname,''), coalesce(klasse,''),
			coalesce(barcode_id,''), coalesce(geburtsdatum::text,''), coalesce(lusd_id,''),
			coalesce(strasse,''), coalesce(hausnummer,''), coalesce(plz,''), coalesce(ort,''),
			coalesce(eltern_email,''), coalesce(block_reason,''))
		FROM schueler WHERE id = $1`, id).Scan(&zeile); err != nil {
		t.Fatalf("Zeile lesen: %v", err)
	}
	for _, sentinel := range []string{vn, nn, strdat, ortdat, email, lusd, reason, barc} {
		if strings.Contains(zeile, sentinel) {
			t.Errorf("PII überlebt die Anonymisierung: %q steht noch in der Zeile: %s", sentinel, zeile)
		}
	}
	// Auch das Geburtsdatum darf nicht mehr dastehen (direkt identifizierend).
	if strings.Contains(zeile, "2011-05-05") {
		t.Errorf("Geburtsdatum überlebt die Anonymisierung: %s", zeile)
	}
}

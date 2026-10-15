package jobs

import (
	"context"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Die Paarung aus der Tilgung beim endgültigen Löschen und der Tilgung in der Nacht, an
// Postgres.
//
// repository.TilgeSchuelerSpuren (endgültige Löschung, Abgang aus der LUSD) und
// bereinigeAnonymisierteSchuelerSpuren (Lauf der Nacht über anonymized_at) beantworten dieselbe
// Frage: Welche Spuren eines Schülers stehen in den übrigen Tabellen? Pflegte jeder Weg seine
// eigene Liste von Anweisungen, fehlte einer leicht eine, etwa die Lesehistorie (audit_log mit
// tabelle='ausleihen'): Dann stünde nach der Anonymisierung der Klarname des Schülers
// (details->>'entleiher') weiter in jeder Zeile CHECKOUT, „anonymisiert" laut Zeile in leser und
// über das Protokoll wieder erkennbar, bis irgendwann die abschaltbare Befristung der
// Lesehistorie ihn zufällig einholte. Deshalb teilen beide Wege die Liste
// (repository.SpurTilgungen).
func TestSpurenTilgung_CronDecktDieselbenSpurenWieDerPurge(t *testing.T) {
	adminDSN := os.Getenv(datenbankVariable)
	if adminDSN == "" {
		t.Skipf("%s nicht gesetzt — Test übersprungen", datenbankVariable)
	}
	_, dsn := legeProbeDatenbankAn(t, adminDSN, "spuren")
	befuelleQuelle(t, dsn)

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("Pool: %v", err)
	}
	t.Cleanup(pool.Close)

	// Ein anonymisierter Schüler, den der Lauf aufgreifen muss, und ein aktiver, dessen Spuren
	// stehen bleiben müssen (Gegenprobe gegen zu viel Tilgung).
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

	// Je eine Zeile der Lesehistorie, wie sie die Ausleihe schreibt (repository/protokoll.go:
	// tabelle='ausleihen', der Schüler nur in details), dazu je eine Vormerkung und je eine Zeile
	// der Historie in schueler: die Spuren, die das endgültige Löschen nachweislich tilgt.
	for _, z := range []struct{ id, name string }{
		{anonID, "Klarname Getilgt"},
		{aktivID, "Klarname Bleibt"},
	} {
		if _, err := pool.Exec(ctx, `
			INSERT INTO audit_log (tabelle, aktion, datensatz_id, akteur, details)
			VALUES ('ausleihen', 'CHECKOUT', gen_random_uuid(), 'USER',
			        jsonb_build_object('schueler_id', $1::text, 'entleiher', $2::text))`,
			z.id, z.name); err != nil {
			t.Fatalf("Lesehistorie-Zeile: %v", err)
		}
	}

	z := NewZeitplan(pool, nil)
	z.bereinigeAnonymisierteSchuelerSpuren(ctx)

	// Die Frage der Auskunft nach Art. 15 und der Tilgung beim endgültigen Löschen, wortgleich: Für
	// den anonymisierten Schüler darf sie nichts mehr finden.
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

	// Der Klarname darf nirgends in den Details des Protokolls überleben, gesucht am Wert, damit
	// auch ein neuer Schlüssel auffällt.
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

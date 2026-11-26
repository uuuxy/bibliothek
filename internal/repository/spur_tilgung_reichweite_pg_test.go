package repository_test

import (
	"context"
	"fmt"
	"testing"

	"bibliothek/internal/pgtest"
	"bibliothek/internal/repository"
	"bibliothek/internal/testdaten"
)

// Die Tilgung trifft genau die Spuren des Lesers: den Empfänger in seinem Bescheid, seine
// Datensatz-Historie auch ohne details, den Stornogrund eines Falls ohne Beschreibung und sein
// Feld in einer Nachbuch-Meldung. Die Spuren eines anderen Lesers bleiben stehen. Jede Prüfung
// gehört zu einer Bedingung der Anweisungen in spur_tilgung.go.
func TestTilgeSchuelerSpuren_TrifftGenauDenLeser(t *testing.T) {
	pool := pgtest.Pool(t)
	testdaten.LeereBestandsdaten(t, pool)
	ctx := context.Background()

	sid := testdaten.LegeSchuelerAn(t, pool, "S-REICH-1", "Mia", "5a")
	fremd := testdaten.LegeSchuelerAn(t, pool, "S-REICH-2", "Ben", "5a")
	titelID := testdaten.LegeTitelMitCoverUndAlterAn(t, pool, "Reichweite-Titel", "Jug Rei", false, 0)
	exID := testdaten.LegeExemplarAn(t, pool, titelID, "REICH-1", true, "")

	// Die Bescheide stehen über ihre Referenznummer fest, nicht über schueler_id: Die gilt nach
	// der Tilgung nicht mehr als Merkmal.
	for i, s := range []string{sid, fremd} {
		if _, err := pool.Exec(ctx, `
			INSERT INTO schadensersatz_bescheide
			       (schueler_id, mittel, kassenjahr, laufende_nr, referenznummer, frist_bis, empfaenger_snapshot)
			VALUES ($1, 'land', 2026, $2, $3, CURRENT_DATE + 14, '{"name": "Familie Test"}')`,
			s, 900+i, fmt.Sprintf("REICH-%d", i)); err != nil {
			t.Fatalf("Bescheid anlegen: %v", err)
		}
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO audit_log (tabelle, aktion, datensatz_id, details) VALUES
		('schueler', 'UPDATE', $1, NULL),
		('schueler', 'UPDATE', $2, '{"vorname": "Ben"}')`, sid, fremd); err != nil {
		t.Fatalf("Datensatz-Historie anlegen: %v", err)
	}
	var fall string
	if err := pool.QueryRow(ctx, `
		INSERT INTO schadensfaelle (exemplar_id, schueler_id, beschreibung, betrag, ist_bezahlt, storniert_am, stornierungsgrund)
		VALUES ($1, $2, '', 10.00, true, NOW(), 'Familie zieht nach Kassel') RETURNING id`,
		exID, sid).Scan(&fall); err != nil {
		t.Fatalf("stornierten Fall anlegen: %v", err)
	}

	var meldung string
	if err := pool.QueryRow(ctx, `
		INSERT INTO nachbuch_meldungen (idempotency_key, barcode, ergebnis, ausleiher_schueler_id, vorbesitzer_schueler_id, gescannt_am)
		VALUES (gen_random_uuid(), 'REICH-1', 'umgebucht', $1, $2, NOW()) RETURNING id`, sid, fremd).Scan(&meldung); err != nil {
		t.Fatalf("Nachbuch-Meldung anlegen: %v", err)
	}

	if err := repository.TilgeSchuelerSpuren(ctx, pool, sid, "Test"); err != nil {
		t.Fatalf("TilgeSchuelerSpuren: %v", err)
	}

	var eigenerLeer, fremderLeer bool
	if err := pool.QueryRow(ctx, `
		SELECT bool_or(empfaenger_snapshot = '{}'::jsonb) FILTER (WHERE referenznummer = 'REICH-0'),
		       bool_or(empfaenger_snapshot = '{}'::jsonb) FILTER (WHERE referenznummer = 'REICH-1')
		FROM schadensersatz_bescheide`).Scan(&eigenerLeer, &fremderLeer); err != nil {
		t.Fatalf("Bescheide lesen: %v", err)
	}
	if !eigenerLeer {
		t.Error("der Bescheid nennt den Empfänger nach der Tilgung noch")
	}
	if fremderLeer {
		t.Error("der Bescheid eines anderen Lesers hat seinen Empfänger verloren")
	}

	var eigeneMarke, fremdeMarke bool
	if err := pool.QueryRow(ctx, `
		SELECT coalesce(bool_or(details ? 'anonymisiert') FILTER (WHERE datensatz_id = $1), false),
		       coalesce(bool_or(details ? 'anonymisiert') FILTER (WHERE datensatz_id = $2), false)
		FROM audit_log WHERE tabelle = 'schueler'`, sid, fremd).Scan(&eigeneMarke, &fremdeMarke); err != nil {
		t.Fatalf("Datensatz-Historie lesen: %v", err)
	}
	if !eigeneMarke {
		t.Error("die Datensatz-Historie ohne details ist nicht als anonymisiert markiert")
	}
	if fremdeMarke {
		t.Error("die Datensatz-Historie eines anderen Lesers wurde anonymisiert")
	}

	var grund *string
	if err := pool.QueryRow(ctx, `SELECT stornierungsgrund FROM schadensfaelle WHERE id = $1`, fall).Scan(&grund); err != nil {
		t.Fatalf("Fall lesen: %v", err)
	}
	if grund != nil {
		t.Errorf("der Stornogrund eines Falls ohne Beschreibung steht noch da: %q", *grund)
	}

	var ausleiher, vorbesitzer *string
	if err := pool.QueryRow(ctx, `SELECT ausleiher_schueler_id::text, vorbesitzer_schueler_id::text
		FROM nachbuch_meldungen WHERE id = $1`, meldung).Scan(&ausleiher, &vorbesitzer); err != nil {
		t.Fatalf("Nachbuch-Meldung lesen: %v", err)
	}
	if ausleiher != nil {
		t.Error("die Nachbuch-Meldung nennt den Leser noch als Ausleiher")
	}
	if vorbesitzer == nil || *vorbesitzer != fremd {
		t.Errorf("der Vorbesitzer, ein anderer Leser, ist nicht mehr eingetragen: %v", vorbesitzer)
	}
}

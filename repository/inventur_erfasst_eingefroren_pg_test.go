package repository

import (
	"context"
	"testing"

	"bibliothek/internal/pgtest"
)

// inventur_erfassungen.exemplar_id steht auf ON DELETE CASCADE. Live gezählt senkte ein später
// gelöschtes Buch (Verlust endgültig, Titel gelöscht) rückwirkend das Ergebnis eines
// abgeschlossenen Laufs, während verloren_gemeldet in derselben Zeile feststand: Aus „312
// erfasst, 4 verloren" würde über die Monate „298 erfasst, 4 verloren". Ein Lauf ist die
// Abschrift einer Zählung am Regal; was gezählt wurde, bleibt gezählt.
func TestInventurErgebnisAendertSichNichtNachtraeglich(t *testing.T) {
	pool := pgtest.Pool(t)
	ctx := context.Background()
	repo := NewInventurAbfragen(pool)

	eins := func(was, sql string, args ...any) string {
		t.Helper()
		var id string
		if err := pool.QueryRow(ctx, sql, args...).Scan(&id); err != nil {
			t.Fatalf("%s: %v", was, err)
		}
		return id
	}

	var titelID, bearbeiter string
	t.Cleanup(func() {
		ctx := context.Background()
		if titelID != "" {
			if _, err := pool.Exec(ctx, `DELETE FROM buecher_titel WHERE id = $1`, titelID); err != nil {
				t.Errorf("Aufräumen Titel: %v", err)
			}
		}
		if bearbeiter != "" {
			if _, err := pool.Exec(ctx, `DELETE FROM benutzer WHERE id = $1`, bearbeiter); err != nil {
				t.Errorf("Aufräumen Bearbeiter: %v", err)
			}
		}
	})
	bearbeiter = eins("Bearbeiter anlegen",
		`INSERT INTO benutzer (vorname, nachname, email, rolle, aktiv)
		 VALUES ('Inventur', 'Abschluss', 'inventur-eingefroren@schule.invalid', 'mitarbeiter', true) RETURNING id`)

	titelID = eins("Titel anlegen",
		`INSERT INTO buecher_titel (titel, autor, signatur) VALUES ('Inventur-Titel', 'Test', 'INV 1') RETURNING id`)
	ex1 := eins("Exemplar 1", `INSERT INTO buecher_exemplare (titel_id, barcode_id) VALUES ($1, 'INV-E1') RETURNING id`, titelID)
	ex2 := eins("Exemplar 2", `INSERT INTO buecher_exemplare (titel_id, barcode_id) VALUES ($1, 'INV-E2') RETURNING id`, titelID)

	praefix := "INV"
	sess, err := repo.LegeLaufAn(ctx, "signature",
		InventurBereich{Signatur: &praefix}, "INV", "")
	if err != nil {
		t.Fatalf("Session anlegen: %v", err)
	}
	for _, ex := range []string{ex1, ex2} {
		if _, err := pool.Exec(ctx,
			`INSERT INTO inventur_erfassungen (session_id, exemplar_id) VALUES ($1, $2)`, sess.ID, ex); err != nil {
			t.Fatalf("Erfassung anlegen: %v", err)
		}
	}

	if _, err := repo.SchliesseLaufAb(ctx, sess.ID, InventurBereich{Signatur: &praefix}, bearbeiter); err != nil {
		t.Fatalf("Session abschließen: %v", err)
	}

	vorher := erfassteLautBericht(t, pool, repo, sess.ID)
	if vorher != 2 {
		t.Fatalf("nach dem Abschluss erwartet 2 erfasste Exemplare, waren %d", vorher)
	}

	// Ein Exemplar fällt später aus dem Bestand; der CASCADE nimmt seine Erfassung mit.
	if _, err := pool.Exec(ctx, `DELETE FROM buecher_exemplare WHERE id = $1`, ex2); err != nil {
		t.Fatalf("Exemplar löschen: %v", err)
	}

	nachher := erfassteLautBericht(t, pool, repo, sess.ID)
	if nachher != vorher {
		t.Errorf("der abgeschlossene Durchgang meldet jetzt %d statt %d erfasste Exemplare — "+
			"ein Bericht über eine körperliche Zählung darf sich nicht ändern, weil später "+
			"ein Buch aus dem Bestand fällt", nachher, vorher)
	}
}

// erfassteLautBericht liest die Zahl so, wie die Oberfläche sie zeigt: über die Liste der
// abgeschlossenen Läufe.
func erfassteLautBericht(t *testing.T, pool any, repo *InventurAbfragen, sessionID string) int {
	t.Helper()
	_ = pool
	sessions, err := repo.LadeAbgeschlosseneLaeufe(context.Background(), 50)
	if err != nil {
		t.Fatalf("abgeschlossene Sessions lesen: %v", err)
	}
	for _, s := range sessions {
		if s.ID == sessionID {
			return s.Erfasst
		}
	}
	t.Fatalf("Session %s steht nicht in der Liste der abgeschlossenen Durchgänge", sessionID)
	return 0
}

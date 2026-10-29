package repository

import (
	"context"
	"errors"
	"testing"

	"bibliothek/db"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Eine gelöschte Inventur nimmt ihre Erfassungen und ihren Fehlbestandsbericht mit, nicht aber,
// was sie an den Büchern gebucht hat: Die als Verlust gebuchten Exemplare bleiben ausgesondert,
// sonst stünden sie wieder als vorhanden im Katalog, ohne dass jemand sie gefunden hat. Wer
// gelöscht hat, steht im Protokoll.
func TestLoescheAbgeschlossenenLauf(t *testing.T) {
	pool := pgTestPool(t)
	resetInventurDaten(t, pool)
	ctx := context.Background()
	repo := NewInventurAbfragen(pool)
	bearbeiter := legeBearbeiterAn(t, pool)

	signatur := "Physik"
	exemplare := legeSignaturMitExemplarenAn(t, pool, signatur, 3)
	lauf, err := repo.LegeLaufAn(ctx, "signature", InventurBereich{Signatur: &signatur}, signatur, "")
	if err != nil {
		t.Fatalf("Lauf anlegen: %v", err)
	}
	if _, err := repo.VerbucheScan(ctx, lauf.ID, lauf.Bereich(), exemplare[0]); err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if _, err := repo.SchliesseLaufAb(ctx, lauf.ID, lauf.Bereich(), bearbeiter); err != nil {
		t.Fatalf("Abschluss: %v", err)
	}
	// Die Gegenprobe vorher: Ohne Erfassung und Bericht bewiese das Löschen nichts.
	if n := zaehle(t, pool, `SELECT count(*) FROM inventur_verluste WHERE session_id = $1`, lauf.ID); n != 2 {
		t.Fatalf("%d Zeilen im Fehlbestand vor dem Löschen, erwartet 2", n)
	}
	if n := zaehle(t, pool, `SELECT count(*) FROM inventur_erfassungen WHERE session_id = $1`, lauf.ID); n != 1 {
		t.Fatalf("%d Erfassungen vor dem Löschen, erwartet 1", n)
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("Transaktion: %v", err)
	}
	defer db.Verwerfe(ctx, tx)
	if err := NewInventurAbfragen(tx).LoescheAbgeschlossenenLauf(ctx, lauf.ID, bearbeiter); err != nil {
		t.Fatalf("Löschen: %v", err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatalf("Commit: %v", err)
	}

	for frage, sql := range map[string]string{
		"Lauf":        `SELECT count(*) FROM inventur_sessions WHERE id = $1`,
		"Erfassungen": `SELECT count(*) FROM inventur_erfassungen WHERE session_id = $1`,
		"Fehlbestand": `SELECT count(*) FROM inventur_verluste WHERE session_id = $1`,
	} {
		if n := zaehle(t, pool, sql, lauf.ID); n != 0 {
			t.Errorf("%s: %d Zeilen nach dem Löschen, erwartet keine", frage, n)
		}
	}
	if n := zaehle(t, pool, `SELECT count(*) FROM buecher_exemplare
		WHERE id = ANY($1) AND ist_ausgesondert AND aussonderung_grund = 'VERLUST'`, exemplare[1:]); n != 2 {
		t.Errorf("%d der zwei Verluste noch als Verlust gebucht, erwartet beide", n)
	}
	if n := zaehle(t, pool, `SELECT count(*) FROM audit_log
		WHERE tabelle = 'inventur_sessions' AND aktion = 'DELETE' AND datensatz_id = $1::uuid
		  AND bearbeiter_id = $2::uuid AND details->>'bezeichnung' = 'Physik' AND (details->>'verluste')::int = 2`,
		lauf.ID, bearbeiter); n != 1 {
		t.Errorf("%d Einträge im Protokoll über das Löschen, erwartet einen mit Bezeichnung und Verlusten", n)
	}
}

// Ein offener Lauf wird verworfen, nicht gelöscht; eine unbekannte Kennung löscht nichts.
func TestLoescheAbgeschlossenenLauf_NurAbgeschlossene(t *testing.T) {
	pool := pgTestPool(t)
	resetInventurDaten(t, pool)
	ctx := context.Background()
	repo := NewInventurAbfragen(pool)
	bearbeiter := legeBearbeiterAn(t, pool)

	signatur := "Chemie"
	legeSignaturMitExemplarenAn(t, pool, signatur, 1)
	offen, err := repo.LegeLaufAn(ctx, "signature", InventurBereich{Signatur: &signatur}, signatur, "")
	if err != nil {
		t.Fatalf("Lauf anlegen: %v", err)
	}

	for name, id := range map[string]string{"offener Lauf": offen.ID, "unbekannte Kennung": "7c9e6679-7425-40de-944b-e07fc1f90ae7"} {
		err := repo.LoescheAbgeschlossenenLauf(ctx, id, bearbeiter)
		if !errors.Is(err, ErrInventurlaufNichtGefunden) {
			t.Errorf("%s: Fehler %v, erwartet ErrInventurlaufNichtGefunden", name, err)
		}
	}
	if n := zaehle(t, pool, `SELECT count(*) FROM inventur_sessions WHERE id = $1 AND abgeschlossen_am IS NULL`, offen.ID); n != 1 {
		t.Errorf("der offene Lauf ist weg oder beendet (%d Zeilen), erwartet unverändert", n)
	}
}

func zaehle(t *testing.T, pool *pgxpool.Pool, sql string, args ...any) int {
	t.Helper()
	var n int
	if err := pool.QueryRow(context.Background(), sql, args...).Scan(&n); err != nil {
		t.Fatalf("Zählen: %v", err)
	}
	return n
}

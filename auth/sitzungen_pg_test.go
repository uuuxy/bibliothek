package auth

import (
	"context"
	"testing"
	"time"
)

// Zwei Zusagen der Sitzungszeile, die nur an der Tabelle zu sehen sind: Eine schon gesperrte
// Anmeldung behält den Zeitpunkt ihrer Sperre, und das Verlängern schreibt das neue Ende.
func TestSitzungen_SperrzeitpunktBleibtUndEndeWirdGeschrieben(t *testing.T) {
	pool := pgPoolFuerSelbstanmeldung(t)
	ctx := context.Background()
	benutzerID, _ := sperreKonto(t, pool)
	s := NewSitzungen(pool, []byte(sperreTestGeheimnis))
	t.Cleanup(s.Stop)

	sitzungID, err := s.Beginne(ctx, benutzerID, "geheim", time.Now().Add(time.Hour))
	if err != nil {
		t.Fatalf("Beginne: %v", err)
	}

	// Gesperrt seit zehn Minuten; ein zweites Sperren, etwa aus einem zweiten Fenster, darf
	// den Zeitpunkt nicht auf jetzt setzen.
	if gesperrt, err := s.Sperre(ctx, sitzungID); err != nil || !gesperrt {
		t.Fatalf("Sperre: gesperrt=%v, err=%v", gesperrt, err)
	}
	if _, err := pool.Exec(ctx, `UPDATE sitzungen SET gesperrt_seit = NOW() - INTERVAL '10 minutes' WHERE id = $1`,
		sitzungID); err != nil {
		t.Fatalf("Sperre zurückdatieren: %v", err)
	}
	if gesperrt, err := s.Sperre(ctx, sitzungID); err != nil || !gesperrt {
		t.Fatalf("zweite Sperre: gesperrt=%v, err=%v", gesperrt, err)
	}
	var behalten bool
	if err := pool.QueryRow(ctx, `SELECT gesperrt_seit < NOW() - INTERVAL '5 minutes' FROM sitzungen WHERE id = $1`,
		sitzungID).Scan(&behalten); err != nil {
		t.Fatalf("Sperrzeitpunkt lesen: %v", err)
	}
	if !behalten {
		t.Error("das zweite Sperren hat den Zeitpunkt der Sperre überschrieben")
	}

	// Verlängern schreibt das neue Ende in die Zeile.
	neuesEnde := time.Now().Add(7 * time.Hour).Truncate(time.Second)
	if verlaengert, err := s.Verlaengere(ctx, sitzungID, neuesEnde); err != nil || !verlaengert {
		t.Fatalf("Verlaengere: verlaengert=%v, err=%v", verlaengert, err)
	}
	var ende time.Time
	if err := pool.QueryRow(ctx, `SELECT laeuft_ab FROM sitzungen WHERE id = $1`, sitzungID).Scan(&ende); err != nil {
		t.Fatalf("Ende lesen: %v", err)
	}
	if !ende.Equal(neuesEnde) {
		t.Errorf("Ende der Zeile = %s, erwartet %s", ende, neuesEnde)
	}
}

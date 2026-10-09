package service

import (
	"context"
	"testing"

	"bibliothek/internal/pgtest"
	"bibliothek/pkg/bestelllink"
)

// Die Frist eines neuen Bestätigungs-Links am echten Postgres: Die Einstellung gilt, und wo sie
// fehlt, unter einem Tag liegt oder sich nicht lesen lässt, gilt die Vorgabe.
func TestBestelllinkTage(t *testing.T) {
	pool := pgtest.Pool(t)
	ctx := context.Background()
	setze := func(wert string) {
		t.Helper()
		if _, err := pool.Exec(ctx, `
			INSERT INTO system_einstellungen (schluessel, wert) VALUES ('bestelllink_gueltigkeit_tage', $1)
			ON CONFLICT (schluessel) DO UPDATE SET wert = EXCLUDED.wert`, wert); err != nil {
			t.Fatalf("Einstellung schreiben: %v", err)
		}
	}

	entferne := func() {
		if _, err := pool.Exec(ctx,
			`DELETE FROM system_einstellungen WHERE schluessel = 'bestelllink_gueltigkeit_tage'`); err != nil {
			t.Errorf("Einstellung entfernen: %v", err)
		}
	}
	entferne()
	t.Cleanup(entferne)

	if ist := BestelllinkTage(ctx, pool); ist != bestelllink.VorgabeTage {
		t.Errorf("ohne Einstellung %d Tage, erwartet die Vorgabe %d", ist, bestelllink.VorgabeTage)
	}

	setze("45")
	if ist := BestelllinkTage(ctx, pool); ist != 45 {
		t.Errorf("eingestellt sind 45 Tage, geliefert %d", ist)
	}

	// Lässt sich die Einstellung nicht lesen, gilt die Vorgabe und nicht der Wert in der Tabelle.
	abgebrochen, abbruch := context.WithCancel(ctx)
	abbruch()
	if ist := BestelllinkTage(abgebrochen, pool); ist != bestelllink.VorgabeTage {
		t.Errorf("bei einem Lesefehler %d Tage, erwartet die Vorgabe %d", ist, bestelllink.VorgabeTage)
	}

	// Die Tür nimmt 1 bis 365 an; ein Wert darunter kommt nur an ihr vorbei in die Tabelle.
	setze("0")
	if ist := BestelllinkTage(ctx, pool); ist != bestelllink.VorgabeTage {
		t.Errorf("eingestellt sind 0 Tage, geliefert %d statt der Vorgabe %d", ist, bestelllink.VorgabeTage)
	}
}

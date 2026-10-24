package jobs

import (
	"context"
	"testing"
	"time"

	"bibliothek/internal/pgtest"
	"bibliothek/pkg/logger"
	"bibliothek/repository"
)

// Der Weg über die echte Tabelle: Zwei Läufe nacheinander, wie vor und nach einem Update; der
// zweite zählt die Zeilen beider. Die Einstellungen lesen den Schlüssel mit und übergehen ihn.
func TestFehlerstandAblage_UeberstehtDenNeustartAnDerDatenbank(t *testing.T) {
	pool := pgtest.Pool(t)
	ctx := context.Background()
	loesche := func() {
		pgtest.RaeumeAuf(t, pool, `DELETE FROM system_einstellungen WHERE schluessel = $1`, fehlerstandSchluessel)
	}
	loesche()
	t.Cleanup(loesche)

	vorher := standMitFehler(t, "Backup: S3 upload failed")
	vorher.Beende()
	SchreibeFehlerstand(ctx, pool, vorher)
	nachher := standMitFehler(t, "Backup: S3 upload failed")
	SchreibeFehlerstand(ctx, pool, nachher)

	if g := nachher.Gruppen(time.Now()); len(g) != 1 || g[0].Anzahl != 2 {
		t.Errorf("Gruppen nach dem Neustart = %+v, erwartet eine Gruppe mit beiden Zeilen", g)
	}
	abgelegt, err := repository.NewBetriebszustandAbfragen(pool).LadeEinstellungswert(ctx, fehlerstandSchluessel)
	if err != nil {
		t.Fatalf("LadeEinstellungswert: %v", err)
	}
	dritter := logger.NewFehlerstand()
	abgebrochen, err := dritter.Uebernimm(abgelegt, time.Now())
	if err != nil {
		t.Fatalf("Uebernimm: %v", err)
	}
	if !abgebrochen {
		t.Error("der zweite Lauf fuhr nicht herunter, sein Stand gilt aber als beendet")
	}
	if g := dritter.Gruppen(time.Now()); len(g) != 1 || g[0].Anzahl != 2 {
		t.Errorf("abgelegter Stand = %+v, erwartet eine Gruppe mit beiden Zeilen", g)
	}
	if _, err := repository.NewEinstellungsAbfragen(pool).Lade(ctx); err != nil {
		t.Errorf("Einstellungen laden neben dem Fehlerstand: %v", err)
	}
}

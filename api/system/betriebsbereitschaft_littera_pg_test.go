package system

import (
	"context"
	"strings"
	"testing"
	"time"

	"bibliothek/api/tuer"
	"bibliothek/db"
	"bibliothek/internal/bereitschaft"
	"bibliothek/internal/littera"
	"bibliothek/internal/pgtest"
	"bibliothek/repository"
)

// Der Live-Pfad des Ergebnisses der Littera-Übernahme: littera.SchreibeErgebnis → sammleLage →
// Pruefe. Ohne die Verdrahtung in sammleLage stünde der Bereich immer auf „keine Übernahme", und
// der Unit-Test mit fertiger Lage bliebe trotzdem grün. Ebenso erreicht der Schalter
// alarm_warnungen die Lage.
func TestLitteraUebernahmeUndSchalterErreichenDieSelbstpruefung(t *testing.T) {
	pool := pgtest.Pool(t)
	ctx := context.Background()
	srv := &Tueren{Basis: &tuer.Basis{DB: &db.Verbindung{Pool: pool}}}
	settingsRepo := repository.NewEinstellungsAbfragen(pool)
	mailRepo := repository.NewMailEinstellungsAbfragen(pool)
	zustandRepo := repository.NewBetriebszustandAbfragen(pool)
	t.Cleanup(func() {
		if _, err := pool.Exec(context.Background(),
			`DELETE FROM system_einstellungen WHERE schluessel IN ($1, 'alarm_warnungen')`, littera.ErgebnisSchluessel); err != nil {
			t.Logf("Aufräumen: %v", err)
		}
	})

	lage := func() bereitschaft.Lage {
		t.Helper()
		return srv.sammleLage(ctx, settingsRepo, mailRepo, zustandRepo)
	}
	befund := func() bereitschaft.Befund {
		t.Helper()
		for _, b := range bereitschaft.Pruefe(lage()) {
			if b.Bereich == "Littera-Übernahme" {
				return b
			}
		}
		t.Fatal("Die Selbstprüfung enthält keinen Bereich 'Littera-Übernahme'")
		return bereitschaft.Befund{}
	}

	if b := befund(); b.Stufe != bereitschaft.StufeOK || !strings.Contains(b.Befund, "Keine Littera-Übernahme") {
		t.Fatalf("ohne Ergebnis: %+v, erwartet ok und „keine Übernahme“", b)
	}
	e := littera.Ergebnis{Zeitpunkt: time.Now(), Fehler: 2, Warnungen: 7, Protokoll: "littera_import.log"}
	if err := littera.SchreibeErgebnis(ctx, pool, e); err != nil {
		t.Fatalf("Ergebnis ablegen: %v", err)
	}
	if b := befund(); b.Stufe != bereitschaft.StufeKritisch || !strings.Contains(b.Befund, "unvollständig (2 Fehler, 7 Warnungen") {
		t.Errorf("unvollständige Übernahme: %+v, erwartet kritisch mit den Zahlen", b)
	}

	if lage().AlarmWarnungen {
		t.Fatal("Vorgabe des Schalters: an, erwartet aus")
	}
	an := true
	if err := settingsRepo.Schreibe(ctx, &repository.EinstellungenPatch{AlarmWarnungen: &an}); err != nil {
		t.Fatalf("Schalter setzen: %v", err)
	}
	if !lage().AlarmWarnungen {
		t.Error("Schalter an, die Lage sieht ihn nicht")
	}
}

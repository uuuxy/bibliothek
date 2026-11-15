package system

import (
	"context"
	"strings"
	"testing"

	"bibliothek/api/tuer"
	"bibliothek/db"
	"bibliothek/internal/bereitschaft"
	"bibliothek/internal/pgtest"
	"bibliothek/repository"
)

// Der Live-Pfad des Schalters „Jede Nacht selbst sichern": Einstellung → sammleLage → Pruefe.
// Ohne die Verdrahtung in sammleLage meldete die Seite bei abgeschalteter Sicherung jeden Tag
// einen fehlenden Backup-Lauf, und der Unit-Test mit fertiger Lage bliebe grün.
func TestSicherungsschalterErreichtDieSelbstpruefung(t *testing.T) {
	pool := pgtest.Pool(t)
	ctx := context.Background()
	srv := &Tueren{Basis: &tuer.Basis{DB: &db.Verbindung{Pool: pool}}}
	settingsRepo := repository.NewEinstellungsAbfragen(pool)
	mailRepo := repository.NewMailEinstellungsAbfragen(pool)
	zustandRepo := repository.NewBetriebszustandAbfragen(pool)
	t.Cleanup(func() {
		pgtest.RaeumeAuf(t, pool, `DELETE FROM system_einstellungen WHERE schluessel = 'sicherung_an'`)
	})
	t.Setenv("APP_ENV", "production") // im Echtbetrieb, sonst meldet der Bereich nur „kein Backup-Betrieb"
	befund := func() bereitschaft.Befund {
		t.Helper()
		for _, b := range bereitschaft.Pruefe(srv.sammleLage(ctx, settingsRepo, mailRepo, zustandRepo)) {
			if b.Bereich == "Nächtliches Backup" {
				return b
			}
		}
		t.Fatal("Die Selbstprüfung enthält keinen Bereich „Nächtliches Backup“")
		return bereitschaft.Befund{}
	}

	if b := befund(); b.Stufe != bereitschaft.StufeOK || !strings.Contains(b.Befund, "abgeschaltet") {
		t.Errorf("Vorgabe: Stufe %q, Befund %q — erwartet ok mit „abgeschaltet“", b.Stufe, b.Befund)
	}

	an := true
	if err := settingsRepo.Schreibe(ctx, &repository.EinstellungenPatch{SicherungAn: &an}); err != nil {
		t.Fatalf("Sicherung einschalten: %v", err)
	}
	if b := befund(); strings.Contains(b.Befund, "abgeschaltet") {
		t.Errorf("eingeschaltet: Befund %q nennt die Sicherung weiter abgeschaltet", b.Befund)
	}
}

package api

import (
	"context"
	"strings"
	"testing"

	"bibliothek/db"
	"bibliothek/repository"
)

// Der LIVE-PFAD des Wächters: nicht die Regel für sich, sondern der Weg, den die
// System-Seite und die tägliche Alarm-Mail wirklich nehmen — sammleLage → Pruefe.
//
// Warum extra: Dieses Projekt hatte schon einen isoliert grünen Test für eine Funktion,
// die auf dem echten Weg gar nicht erreicht wurde. Ein Wächter, der nur im Unit-Test
// ausschlägt, ist kein Wächter. Beide Verbraucher hängen an sammleLage — schlägt es hier
// aus, geht auch die Mail raus.
func TestLoeschRueckstandErreichtDieSelbstpruefung(t *testing.T) {
	pool := pgTestPool(t)
	ctx := context.Background()
	srv := &Server{DB: &db.Database{Pool: pool}}

	settingsRepo := repository.NewSystemSettingsRepository(pool)
	mailRepo := repository.NewMailSettingsRepository(pool)
	zustandRepo := repository.NewBetriebszustandRepository(pool)

	dsgvoBefund := func() Befund {
		t.Helper()
		for _, b := range Pruefe(srv.sammleLage(ctx, settingsRepo, mailRepo, zustandRepo)) {
			if b.Bereich == "DSGVO-Löschroutinen" {
				return b
			}
		}
		t.Fatal("Die Selbstprüfung enthält keinen Bereich 'DSGVO-Löschroutinen'")
		return Befund{}
	}

	// Erhoben heißt erhoben: Ohne Zutun darf der Bereich nicht auf „nicht erhoben"
	// stehen — sonst liefe der Wächter auf jeder echten Datenbank ins Leere und
	// niemand merkte es, weil eine Warnung wie ein Randfall aussieht.
	if b := dsgvoBefund(); strings.Contains(b.Befund, "konnte nicht erhoben werden") {
		t.Fatalf("Wächter kommt am echten Schema nicht durch: %s", b.Befund)
	}

	// Jetzt ein überfälliger Datensatz — deutlich jenseits von Frist plus Kulanz.
	t.Cleanup(func() {
		aufraeumen(t, pool, `DELETE FROM lehrer_anliegen WHERE titel_text = $1`, "RUECKSTAND-LIVE")
	})
	if _, err := pool.Exec(ctx, `
		INSERT INTO lehrer_anliegen (art, titel_text, kommentar, erstellt_am, erledigt_am)
		VALUES ('wunsch', 'RUECKSTAND-LIVE', 'x',
		        NOW() - make_interval(days => $1 + 60), NOW() - make_interval(days => $1 + 30))`,
		repository.StandardAnliegenTage); err != nil {
		t.Fatalf("überfälliges Anliegen anlegen: %v", err)
	}

	b := dsgvoBefund()
	if b.Stufe != StufeKritisch {
		t.Fatalf("überfälliger Datensatz erreicht die Selbstprüfung nicht: %+v", b)
	}
	if !strings.Contains(b.Befund, "Erledigte Anliegen") {
		t.Errorf("Befund nennt die betroffene Routine nicht: %s", b.Befund)
	}
}

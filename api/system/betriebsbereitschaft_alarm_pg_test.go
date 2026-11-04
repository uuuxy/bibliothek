package system

import (
	"context"
	"errors"
	"log/slog"
	"slices"
	"testing"
	"time"

	"bibliothek/api/tuer"
	"bibliothek/db"
	"bibliothek/internal/pgtest"
	"bibliothek/internal/testdaten"
	"bibliothek/mailservice"
	"bibliothek/pkg/logger"
	"bibliothek/repository"
)

// alarmPost zählt die zugestellten Alarm-Mails; mit scheitert lehnt sie jede ab.
type alarmPost struct {
	zugestellt int
	scheitert  bool
}

func (p *alarmPost) Sende(context.Context, mailservice.Nachricht) error {
	if p.scheitert {
		return errors.New("mailserver antwortet nicht")
	}
	p.zugestellt++
	return nil
}

func (*alarmPost) IstKonfiguriert(context.Context) bool { return true }

// Der Wächter läuft drei Minuten nach jedem Start (main.go). Über die echte Datenbank: Dieselbe
// Mail geht nur einmal hinaus, ein neuer Punkt mit dem nächsten Lauf, nach einem Tag eine
// Erinnerung. Ein gescheiterter Versand zählt nicht als gemeldet, sonst verschluckte ein Aussetzer
// des Mailservers den Alarm für einen Tag.
func TestBereitschaftsAlarm_WiederholtUnveraendertesErstNachEinemTag(t *testing.T) {
	pool := pgtest.Pool(t)
	t.Setenv("APP_ENV", "production")
	// Ohne Ziel außer Haus ist die Auslagerung kritisch: Der Alarm hat immer etwas zu melden.
	for _, name := range []string{"S3_ENDPOINT", "S3_ACCESS_KEY", "S3_SECRET_KEY", "S3_BUCKET"} {
		t.Setenv(name, "")
	}
	loescheEintrag := func() {
		if _, err := pool.Exec(context.Background(),
			`DELETE FROM system_einstellungen WHERE schluessel = $1`, alarmGemeldetSchluessel); err != nil {
			t.Errorf("Eintrag %s löschen: %v", alarmGemeldetSchluessel, err)
		}
	}
	loescheEintrag()
	t.Cleanup(loescheEintrag)
	testdaten.LegeAdminKontoAn(t, pool, "alarm-wiederholung@schule.example", "Alarm", "Probe")

	post := &alarmPost{}
	stand := logger.NewFehlerstand()
	srv := &Tueren{Basis: &tuer.Basis{DB: &db.Verbindung{Pool: pool}, Mail: post}, Fehlerstand: stand}
	zustand := repository.NewBetriebszustandAbfragen(pool)
	lauf := func(wann string, wantMail bool) {
		t.Helper()
		vorher := post.zugestellt
		srv.BereitschaftsAlarm(t.Context())
		if mail := post.zugestellt > vorher; mail != wantMail {
			t.Fatalf("%s: Mail zugestellt = %v, erwartet %v", wann, mail, wantMail)
		}
	}

	post.scheitert = true
	lauf("der Mailserver antwortet nicht", false)
	post.scheitert = false
	lauf("nächster Lauf nach dem Aussetzer", true)
	lauf("Neustart ohne Neues", false)

	slog.New(stand.SchalteVor(slog.DiscardHandler)).Error("Probe-Meldung für den Alarm", "fehler", "Probe")
	lauf("neue Fehlermeldung im Log", true)
	punkte := ladeGemeldetenAlarm(t.Context(), zustand).Punkte
	for _, p := range []string{"Befund: Auslagerung der Backups", "Fehler: Probe-Meldung für den Alarm"} {
		if !slices.Contains(punkte, p) {
			t.Fatalf("festgehaltene Punkte %q ohne %q", punkte, p)
		}
	}
	lauf("Neustart mit derselben Fehlermeldung", false)

	zuletzt := ladeGemeldetenAlarm(t.Context(), zustand)
	zuletzt.Zeit = zuletzt.Zeit.Add(-24 * time.Hour)
	schreibeGemeldetenAlarm(t.Context(), zustand, zuletzt)
	lauf("letzte Mail einen Tag her", true)
}

package laufvermerk

import (
	"errors"
	"log/slog"
	"os"
	"os/exec"
	"runtime/debug"
	"slices"
	"strings"
	"testing"
	"time"

	"bibliothek/pkg/logger"
)

const (
	meldungAbsturz = "Voriger Lauf des Servers ist abgestürzt"
	meldungHart    = "Voriger Lauf des Servers wurde hart beendet"
	meldungRechner = "Voriger Lauf des Servers endete mit einem Ausfall des Rechners"
)

// starte ruft beginne wie ein Start des Servers und liefert, was der Fehlerstand dabei zählte:
// dieselben Gruppen, die die Seite „Fehler im Log“ zeigt.
func starte(t *testing.T, verzeichnis, kennung string) []logger.Fehlergruppe {
	t.Helper()
	stand := logger.NewFehlerstand()
	vorher := slog.Default()
	slog.SetDefault(slog.New(stand.SchalteVor(slog.DiscardHandler)))
	defer slog.SetDefault(vorher)
	t.Cleanup(func() {
		if err := debug.SetCrashOutput(nil, debug.CrashOptions{}); err != nil {
			t.Errorf("SetCrashOutput(nil): %v", err)
		}
	})
	beginne(verzeichnis, kennung)
	return stand.Gruppen(time.Now())
}

// Was der vorige Lauf hinterließ, bestimmt die Meldung beim nächsten Start: nichts nach einem
// geordneten Ende, sonst genau eine Zeile.
func TestBeginne_MeldetWieDerVorigeLaufEndete(t *testing.T) {
	faelle := []struct {
		name    string
		vorher  func(verzeichnis string)
		kennung string
		will    []string
	}{
		{"erster Start", func(string) {}, "k1", nil},
		{"geordnet beendet", func(v string) { beginne(v, "k1"); Beende(v) }, "k1", nil},
		{"hart beendet", func(v string) { beginne(v, "k1") }, "k1", []string{meldungHart}},
		{"Rechner ausgefallen", func(v string) { beginne(v, "k1") }, "k2", []string{meldungRechner}},
	}
	for _, f := range faelle {
		t.Run(f.name, func(t *testing.T) {
			verzeichnis := t.TempDir()
			f.vorher(verzeichnis)
			var meldungen []string
			for _, g := range starte(t, verzeichnis, f.kennung) {
				meldungen = append(meldungen, g.Meldung)
			}
			if !slices.Equal(meldungen, f.will) {
				t.Errorf("Meldungen = %q, erwartet %q", meldungen, f.will)
			}
		})
	}
}

func TestStelleFest(t *testing.T) {
	faelle := []struct {
		name                string
		vermerkt            bool
		vorige, jetzt, text string
		will                ende
	}{
		{"erster Start oder geordnet beendet", false, "", "a", "", geordnet},
		{"Vermerk liegt, Rechner lief durch", true, "a", "a", "", hartBeendet},
		{"Vermerk liegt, Rechner fuhr neu hoch", true, "a", "b", "", rechnerAusgefallen},
		{"Vermerk liegt, Startkennung unbekannt", true, "", "b", "", hartBeendet},
		{"Absturzmeldung geht vor", true, "a", "b", "panic: x", abgestuerzt},
		{"Absturz beim Herunterfahren", false, "", "a", "panic: x", abgestuerzt},
	}
	for _, f := range faelle {
		if ist := stelleFest(f.vermerkt, f.vorige, f.jetzt, f.text); ist != f.will {
			t.Errorf("%s: stelleFest = %d, erwartet %d", f.name, ist, f.will)
		}
	}
}

// Die Stelle ist der erste Eintrag des Stapels im eigenen Code, auch wenn Einträge der Laufzeit
// davorstehen; „created by“ nennt nur, wer die Goroutine startete.
func TestFasseAbsturzZusammen(t *testing.T) {
	faelle := []struct{ text, will string }{
		{
			"panic: assignment to entry in nil map\n\ngoroutine 42 [running]:\n" +
				"bibliothek/api.(*Server).leiheAus(0xc000123456)\n\t/app/api/ausleihe.go:123 +0x1d\n" +
				"created by bibliothek/api.NewServer in goroutine 1\n\t/app/api/server.go:40 +0x88",
			"api/ausleihe.go:123: panic: assignment to entry in nil map",
		},
		{
			"fatal error: concurrent map writes\n\ngoroutine 7 [running]:\n" +
				"internal/runtime/maps.fatal({0x1234, 0x15})\n\t/usr/local/go/src/runtime/panic.go:1058 +0x18\n" +
				"bibliothek/jobs.(*Zeitplan).zaehle(...)\n\t/app/jobs/zeitplan.go:88 +0x3c",
			"jobs/zeitplan.go:88: fatal error: concurrent map writes",
		},
		{"panic: ohne Stapel", "panic: ohne Stapel"},
	}
	for _, f := range faelle {
		if ist := fasseAbsturzZusammen(f.text); ist != f.will {
			t.Errorf("fasseAbsturzZusammen = %q, erwartet %q", ist, f.will)
		}
	}
}

// absturzProbe trägt dem Kindprozess das Verzeichnis zu; ist sie gesetzt, stürzt er ab.
const absturzProbe = "LAUFVERMERK_ABSTURZPROBE"

// Ein echter Absturz in einem zweiten Prozess, in einer Goroutine ohne recover wie im Betrieb: Go
// schreibt ihn in die Datei, und der nächste Start meldet Stelle und erste Zeile.
func TestBeginne_MeldetEinenEchtenAbsturz(t *testing.T) {
	if verzeichnis := os.Getenv(absturzProbe); verzeichnis != "" {
		beginne(verzeichnis, "k1")
		go func() { panic("Probe-Absturz") }()
		select {}
	}
	verzeichnis := t.TempDir()
	kind := exec.Command(os.Args[0], "-test.run=^TestBeginne_MeldetEinenEchtenAbsturz$")
	kind.Env = append(os.Environ(), absturzProbe+"="+verzeichnis)
	ausgabe, err := kind.CombinedOutput()
	var abbruch *exec.ExitError
	if !errors.As(err, &abbruch) {
		t.Fatalf("Kindprozess stürzte nicht ab (%v):\n%s", err, ausgabe)
	}
	gruppen := starte(t, verzeichnis, "k1")
	if len(gruppen) != 1 || gruppen[0].Meldung != meldungAbsturz ||
		!strings.HasPrefix(gruppen[0].Ursache, "laufvermerk/laufvermerk_test.go:") ||
		!strings.HasSuffix(gruppen[0].Ursache, ": panic: Probe-Absturz") {
		t.Errorf("Gruppen = %+v, erwartet „%s“ mit Stelle und erster Zeile der Meldung", gruppen, meldungAbsturz)
	}
}

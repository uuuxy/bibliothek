// Package laufvermerk meldet beim Start, wenn der vorige Lauf nicht geordnet herunterfuhr, und
// nennt, wie er endete.
//
// Der Vermerk ist eine Datei neben dem Programm, wie bei Chrome (clean_exit_beacon.h), keine Zeile
// in der Datenbank: Hält Docker selbst an, endet die Datenbank im selben Augenblick wie der Server,
// und ein Vermerk dort käme oft nicht mehr an.
package laufvermerk

import (
	"errors"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"runtime/debug"
	"strings"

	"bibliothek/pkg/closeutil"
)

const (
	// laeuftDatei liegt da, solange der Server läuft, und trägt die Startkennung des Rechners.
	laeuftDatei = "laeuft"
	// absturzDatei nimmt die Absturzmeldung von Go auf.
	absturzDatei = "absturz.txt"
	// startkennungDatei vergibt Linux bei jedem Hochfahren neu; ein Container sieht die des Rechners.
	startkennungDatei = "/proc/sys/kernel/random/boot_id"
)

// ende sagt, wie der vorige Lauf endete.
type ende int

const (
	geordnet ende = iota
	abgestuerzt
	hartBeendet
	rechnerAusgefallen
)

// Beginne meldet als ERROR, wie der vorige Lauf endete, wenn er nicht geordnet herunterfuhr, und
// vermerkt diesen Lauf als laufend. Einen Absturz schreibt Go ab hier zusätzlich in eine Datei,
// denn ein Absturz lässt keine Zeit, etwas zu vermerken.
func Beginne(verzeichnis string) {
	beginne(verzeichnis, startkennung())
}

func beginne(verzeichnis, kennung string) {
	if err := os.MkdirAll(verzeichnis, 0o700); err != nil {
		slog.Warn("Laufvermerk: Verzeichnis nicht anlegbar, ein Abbruch bleibt unerkannt", "fehler", err)
		return
	}
	// os.OpenRoot bindet jeden Zugriff an das Verzeichnis, wie bei den Covern (api/katalog/cover.go).
	ablage, err := os.OpenRoot(verzeichnis)
	if err != nil {
		slog.Warn("Laufvermerk: Verzeichnis nicht offen, ein Abbruch bleibt unerkannt", "fehler", err)
		return
	}
	defer closeutil.Schliesse(ablage, "laufvermerk")
	vorigeKennung, vermerkt := leseDatei(ablage, laeuftDatei)
	absturzText, _ := leseDatei(ablage, absturzDatei)
	melde(stelleFest(vermerkt, vorigeKennung, kennung, absturzText), absturzText)

	leiteAbsturzUm(ablage)
	// Ohne Sync: Der Kernel schreibt die Datei binnen einer halben Minute, unerkannt bliebe nur ein
	// Stromausfall gleich nach dem Start.
	if err := ablage.WriteFile(laeuftDatei, []byte(kennung+"\n"), 0o600); err != nil {
		slog.Warn("Laufvermerk: nicht geschrieben, ein Abbruch bleibt unerkannt", "fehler", err)
	}
}

// Beende vermerkt, dass der Server geordnet herunterfährt.
func Beende(verzeichnis string) {
	err := os.Remove(filepath.Join(verzeichnis, laeuftDatei))
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		slog.Warn("Laufvermerk: nicht entfernt, der nächste Start meldet einen Abbruch", "fehler", err)
	}
}

// stelleFest bestimmt, wie der vorige Lauf endete: Eine Absturzmeldung von Go geht vor, ein
// liegengebliebener Vermerk heißt hart beendet, mit neuer Startkennung Ausfall des Rechners. Vor
// einem geordneten Neustart des Rechners fährt auch der Server geordnet herunter.
func stelleFest(vermerkt bool, vorigeKennung, kennung, absturzText string) ende {
	switch {
	case absturzText != "":
		return abgestuerzt
	case !vermerkt:
		return geordnet
	case vorigeKennung != "" && kennung != "" && vorigeKennung != kennung:
		return rechnerAusgefallen
	default:
		return hartBeendet
	}
}

// melde schreibt das Ende als ERROR. Die Meldungen stehen fest, damit der Fehlerstand sie zählt
// (docs/STIL.md, Abschnitt 11).
func melde(e ende, absturzText string) {
	switch e {
	case abgestuerzt:
		slog.Error("Voriger Lauf des Servers ist abgestürzt", "fehler", fasseAbsturzZusammen(absturzText))
	case hartBeendet:
		slog.Error("Voriger Lauf des Servers wurde hart beendet",
			"fehler", "Speichermangel oder Abbruch von außen, der Rechner lief weiter")
	case rechnerAusgefallen:
		slog.Error("Voriger Lauf des Servers endete mit einem Ausfall des Rechners",
			"fehler", "Stromausfall, Absturz oder harter Neustart des Rechners")
	case geordnet:
	}
}

// fasseAbsturzZusammen nennt die Stelle im eigenen Code und die erste Zeile der Meldung, etwa
// „jobs/fehlerstand.go:61: panic: assignment to entry in nil map“. Die Stelle steht vorn, weil der
// Fehlerstand die Ursache kürzt.
func fasseAbsturzZusammen(text string) string {
	zeilen := strings.Split(text, "\n")
	meldung := strings.TrimSpace(zeilen[0])
	for i := 1; i+1 < len(zeilen); i++ {
		if !strings.HasPrefix(zeilen[i], "bibliothek/") && !strings.HasPrefix(zeilen[i], "main.") {
			continue
		}
		stelle, _, _ := strings.Cut(strings.TrimSpace(zeilen[i+1]), " +0x")
		return filepath.Base(filepath.Dir(stelle)) + "/" + filepath.Base(stelle) + ": " + meldung
	}
	return meldung
}

// leiteAbsturzUm lässt Go einen Absturz zusätzlich in die Datei schreiben, die der nächste Start
// liest. Leeren darf es sie, weil Beginne ihren Inhalt vorher gemeldet hat.
func leiteAbsturzUm(ablage *os.Root) {
	datei, err := ablage.OpenFile(absturzDatei, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		slog.Warn("Laufvermerk: Absturzdatei nicht anlegbar, ein Absturz steht nur im Log", "fehler", err)
		return
	}
	// SetCrashOutput verdoppelt den Dateideskriptor; die Datei darf danach geschlossen werden.
	defer closeutil.Schliesse(datei, "absturzdatei")
	if err := debug.SetCrashOutput(datei, debug.CrashOptions{}); err != nil {
		slog.Warn("Laufvermerk: Absturzdatei nicht eingerichtet, ein Absturz steht nur im Log", "fehler", err)
	}
}

// leseDatei liefert den Inhalt ohne Leerraum am Rand; vorhanden ist falsch, wenn die Datei fehlt
// oder nicht lesbar ist.
func leseDatei(ablage *os.Root, name string) (inhalt string, vorhanden bool) {
	roh, err := ablage.ReadFile(name)
	if err != nil {
		if !errors.Is(err, fs.ErrNotExist) {
			slog.Warn("Laufvermerk: Datei nicht lesbar", "datei", name, "fehler", err)
		}
		return "", false
	}
	return strings.TrimSpace(string(roh)), true
}

// startkennung ist die Kennung des laufenden Hochfahrens; leer, wo es sie nicht gibt (macOS).
func startkennung() string {
	roh, err := os.ReadFile(startkennungDatei)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(roh))
}

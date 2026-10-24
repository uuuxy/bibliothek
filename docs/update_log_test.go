package docs

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// Ersetzt update.sh den Backend-Container, löscht Docker dessen Log mit. Das Skript legt es vorher
// unter logs/ ab, nur für den Besitzer lesbar, und löscht dort Logs, die älter als 14 Tage sind.
//
// Blindheit: docker ist der Stellvertreter aus fahreUpdate. Ob Docker den Container wirklich
// ersetzt und `docker logs` das ganze Log liefert, sieht der Test nicht.

// schreibeLog legt in ordner die Datei name an, zuletzt geändert vor alter.
func schreibeLog(t *testing.T, ordner, name string, alter time.Duration) {
	t.Helper()
	if err := os.MkdirAll(ordner, 0o700); err != nil {
		t.Fatal(err)
	}
	pfad := filepath.Join(ordner, name)
	if err := os.WriteFile(pfad, []byte("alte Zeile\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	zeit := time.Now().Add(-alter)
	if err := os.Chtimes(pfad, zeit, zeit); err != nil {
		t.Fatal(err)
	}
}

func TestUpdate_HebtDasLogDesErsetztenContainersAuf(t *testing.T) {
	alt, jung := "backend_20260901_000000.log", "backend_20261010_000000.log"
	var logs string
	lauf := fahreUpdate(t, "", true, false, func(server string) []string {
		logs = filepath.Join(server, "logs")
		schreibeLog(t, logs, alt, 20*24*time.Hour)
		schreibeLog(t, logs, jung, 10*24*time.Hour)
		return nil
	})
	if lauf.err != nil {
		t.Fatalf("update.sh: %v\n%s", lauf.err, lauf.ausgabe)
	}

	eintraege, err := os.ReadDir(logs)
	if err != nil {
		t.Fatal(err)
	}
	da := map[string]bool{}
	var neu []string
	for _, eintrag := range eintraege {
		da[eintrag.Name()] = true
		if eintrag.Name() != alt && eintrag.Name() != jung {
			neu = append(neu, eintrag.Name())
		}
	}
	if da[alt] || !da[jung] {
		t.Errorf("update.sh: danach %s da = %v und %s da = %v, erwartet false und true", alt, da[alt], jung, da[jung])
	}
	if len(neu) != 1 || !strings.HasPrefix(neu[0], "backend_") || !strings.HasSuffix(neu[0], ".log") {
		t.Fatalf("update.sh legte in logs/ %v an, erwartet eine Datei backend_<Zeit>.log", neu)
	}
	pfad := filepath.Join(logs, neu[0])
	inhalt, err := os.ReadFile(pfad)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(inhalt), "Zeile aus dem alten Container") {
		t.Errorf("update.sh: %s enthält %q, erwartet das Log des alten Containers", neu[0], inhalt)
	}
	info, err := os.Stat(pfad)
	if err != nil {
		t.Fatal(err)
	}
	if rechte := info.Mode().Perm(); rechte&0o077 != 0 {
		t.Errorf("update.sh: %s hat die Rechte %v, erwartet nur für den Besitzer", neu[0], rechte)
	}
	gelesen := strings.Index(lauf.docker, "logs bibliothek-backend")
	gestartet := strings.Index(lauf.docker, "docker-compose.yml up -d")
	if gelesen < 0 || gelesen > gestartet {
		t.Errorf("update.sh liest das Log an Stelle %d und startet an Stelle %d, erwartet das Log vor dem Start:\n%s",
			gelesen, gestartet, lauf.docker)
	}
}

// Bleibt der Container, steht sein Log weiter in Docker; eine Kopie hielte dieselben Zeilen über
// die 14 Tage hinaus.
func TestUpdate_OhneNeuenContainerBleibtKeinLog(t *testing.T) {
	var logs string
	lauf := fahreUpdate(t, "", true, false, func(server string) []string {
		logs = filepath.Join(server, "logs")
		return []string{"GLEICHER_CONTAINER=1"}
	})
	if lauf.err != nil {
		t.Fatalf("update.sh: %v\n%s", lauf.err, lauf.ausgabe)
	}
	eintraege, err := os.ReadDir(logs)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		t.Fatal(err)
	}
	if len(eintraege) > 0 {
		t.Errorf("update.sh ließ in logs/ %d Datei/en zurück, erwartet keine", len(eintraege))
	}
}

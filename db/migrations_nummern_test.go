package db

import (
	"os"
	"strings"
	"testing"
)

// Befund F7 der unabhängigen Prüfung (bewertung/datenbank-pruefbericht.md):
// Vier Migrationsnummern existieren doppelt (003, 008, 021, 022 — je zwei
// verschiedene Dateien). Kaputt geht dadurch nichts, weil der Läufer Skripte
// am VOLLEN Dateinamen erfasst — aber die Nummerierung verspricht eine
// Ordnung, die sie nicht hält. Entschieden (18.08.2026): Die vier Altfälle
// bleiben eingefroren — ein nachträgliches Umnummerieren wäre gefährlich,
// weil migrations-Einträge am Dateinamen hängen. NEUE Doppelnummern sperrt
// dieses Gate.
func TestKeineNeuenDoppeltenMigrationsnummern(t *testing.T) {
	eingefroren := map[string]bool{"003": true, "008": true, "021": true, "022": true}

	eintraege, err := os.ReadDir("../migrations")
	if err != nil {
		t.Fatalf("migrations lesen: %v", err)
	}
	gesehen := map[string][]string{}
	for _, e := range eintraege {
		name := e.Name()
		if !strings.HasSuffix(name, ".sql") {
			continue
		}
		nummer, _, ok := strings.Cut(name, "_")
		if !ok || nummer == "" {
			t.Errorf("Migrationsdatei ohne Nummern-Präfix: %q", name)
			continue
		}
		gesehen[nummer] = append(gesehen[nummer], name)
	}
	for nummer, dateien := range gesehen {
		if len(dateien) > 1 && !eingefroren[nummer] {
			t.Errorf("Migrationsnummer %s ist doppelt vergeben: %v — nächste freie Nummer nehmen", nummer, dateien)
		}
	}
	// Gegenrichtung: Verschwinden die Altfälle (Aufräumaktion), muss die
	// Ausnahmeliste mit schrumpfen, sonst deckt sie irgendwann neue Fehler.
	for nummer := range eingefroren {
		if len(gesehen[nummer]) < 2 {
			t.Errorf("Ausnahme %s ist nicht mehr doppelt — aus der eingefroren-Liste entfernen", nummer)
		}
	}
}

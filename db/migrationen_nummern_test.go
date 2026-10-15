package db

import (
	"os"
	"strings"
	"testing"
)

// Vier Nummern von Migrationen gibt es doppelt (003, 008, 021, 022, je zwei Dateien). Kaputt geht
// dadurch nichts, weil der Lauf eine Datei am vollen Namen erkennt, aber die Nummern versprechen
// eine Ordnung, die sie nicht halten. Die vier bleiben, wie sie sind: Umnummerieren wäre
// gefährlich, weil die Einträge in schema_migrations am Dateinamen hängen. Eine neue doppelte
// Nummer weist dieser Test ab (bewertung/datenbank-pruefbericht.md, Befund F7).
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
	// Gegenrichtung: Verschwindet eine der vier Dateien, schrumpft die Ausnahmeliste mit, sonst
	// deckte sie irgendwann einen neuen Fehler.
	for nummer := range eingefroren {
		if len(gesehen[nummer]) < 2 {
			t.Errorf("Ausnahme %s ist nicht mehr doppelt — aus der eingefroren-Liste entfernen", nummer)
		}
	}
}

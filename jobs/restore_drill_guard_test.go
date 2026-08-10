package jobs

import (
	"os"
	"os/exec"
	"testing"
)

// TestRestoreProbeLaeuftInCI schließt dieselbe Lücke wie db/pgtest_guard_test.go, nur für
// die zweite Voraussetzung dieser Probe.
//
// Der Guard drüben prüft, ob TEST_DATABASE_URL gesetzt ist. Die Restore-Probe braucht aber
// zusätzlich pg_dump und psql im PATH — fehlen sie, überspringt sie sich selbst, und zwar
// bei gesetzter Variable und mit grünem Haken daneben. Von allen Tests dieses Projekts ist
// ausgerechnet der Notfallplan derjenige, dessen stiller Ausfall am teuersten ist: Man
// erfährt davon am schlechtesten denkbaren Tag.
//
// Lokal ist der Skip in Ordnung — nicht jeder Arbeitsplatz hat den postgresql-client. In
// CI ist er ein Fehler: Dort installiert der Workflow ihn ausdrücklich (Schritt "Install
// PostgreSQL client"), und wenn dieser Schritt eines Tages ausfällt oder wegfällt, soll
// das auffallen, statt still eine Prüfung weniger zu bedeuten.
func TestRestoreProbeLaeuftInCI(t *testing.T) {
	if os.Getenv("CI") == "" {
		t.Skip("nur in CI relevant")
	}
	if os.Getenv(drillEnvVar) == "" {
		t.Fatalf("%s ist in CI nicht gesetzt — die Restore-Probe würde sich überspringen "+
			"(siehe Service-Container in .github/workflows/ci.yml)", drillEnvVar)
	}
	for _, werkzeug := range []string{"pg_dump", "psql"} {
		if _, err := exec.LookPath(werkzeug); err != nil {
			t.Fatalf("%s fehlt in CI — die Restore-Probe würde sich überspringen und der "+
				"Notfallplan bliebe ungeprüft (siehe Schritt \"Install PostgreSQL client\" "+
				"in .github/workflows/ci.yml): %v", werkzeug, err)
		}
	}
}

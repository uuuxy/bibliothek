package db

import (
	"os"
	"testing"

	"bibliothek/internal/pgtest"
)

// Der gefährlichste Ausfall dieser Tests ist der stille: Fehlt TEST_DATABASE_URL, überspringen
// sie sich selbst, und CI wäre grün, ohne eine Zusage der Datenbank geprüft zu haben. Am
// Arbeitsplatz ohne CI ist das gewollt, in CI ein Fehler.
func TestDBTestsLaufenInCI(t *testing.T) {
	if os.Getenv("CI") == "" {
		t.Skip("nur in CI relevant")
	}
	if os.Getenv(pgtest.Variable) == "" {
		t.Fatalf("%s ist in CI nicht gesetzt — die DB-Invarianten würden ungeprüft "+
			"durchgewinkt (siehe Service-Container in .github/workflows/ci.yml)", pgtest.Variable)
	}
}

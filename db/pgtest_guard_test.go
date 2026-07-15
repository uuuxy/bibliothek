package db

import (
	"os"
	"testing"
)

// TestDBTestsLaufenInCI verhindert den gefährlichsten Ausfall dieser Testebene: den
// stillen. Fehlt TEST_DATABASE_URL, überspringen sich alle Tests hier selbst — CI wäre
// grün, ohne eine einzige DB-Invariante geprüft zu haben. Lokal (ohne CI) ist das
// gewollt, in CI ist es ein Fehler.
func TestDBTestsLaufenInCI(t *testing.T) {
	if os.Getenv("CI") == "" {
		t.Skip("nur in CI relevant")
	}
	if os.Getenv(testDBEnvVar) == "" {
		t.Fatalf("%s ist in CI nicht gesetzt — die DB-Invarianten würden ungeprüft "+
			"durchgewinkt (siehe Service-Container in .github/workflows/ci.yml)", testDBEnvVar)
	}
}

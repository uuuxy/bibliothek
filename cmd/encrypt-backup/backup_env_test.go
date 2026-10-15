package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Prüft, wie scripts/backup.sh die .env liest.
//
// Lüde das Skript die Datei mit `export $(grep -v '^#' .env | xargs)`, stünde jedes Geheimnis
// der Datei in der Umgebung des Skripts und aller Kindprozesse (docker, gzip, find), obwohl es
// nur POSTGRES_USER und POSTGRES_DB braucht; den Schlüssel der Sicherung liest der Container
// (backup_krypto.sh), nicht das Skript. Dazu zerlegte xargs Werte mit Leerzeichen, und ein `*`
// im Wert würde als Dateimuster aufgelöst.
//
// Geprüft wird am ausgeführten Skript: Eine Attrappe von `docker` schreibt ihre Argumente und
// ihre Umgebung mit. Sie meldet keinen laufenden Container des Backends, also läuft der Zweig
// für den Notfall (Dump im Klartext); für die Frage ist der Zweig gleich, beide rufen pg_dump
// über docker.
func TestBackupSkript_LiestAusDerEnvNurDieZweiWerte(t *testing.T) {
	verzeichnis := t.TempDir()
	skripte := filepath.Join(verzeichnis, "scripts")
	require.NoError(t, os.MkdirAll(skripte, 0o755))
	for _, name := range []string{"backup.sh", "backup_krypto.sh"} {
		inhalt, err := os.ReadFile(filepath.Join("..", "..", "scripts", name))
		require.NoError(t, err)
		require.NoError(t, os.WriteFile(filepath.Join(skripte, name), inhalt, 0o755))
	}
	env := "# Kommentar\n" +
		"POSTGRES_USER=gateuser\n" +
		"POSTGRES_DB=\"gatedb\"\n" +
		"JWT_SECRET=jwt-geheimnis-darf-nicht-mitlaufen\n" +
		"SMTP_PASSWORD=mit leerzeichen und * stern\n"
	require.NoError(t, os.WriteFile(filepath.Join(verzeichnis, ".env"), []byte(env), 0o600))

	binVerzeichnis := filepath.Join(verzeichnis, "bin")
	require.NoError(t, os.MkdirAll(binVerzeichnis, 0o755))
	protokoll := filepath.Join(verzeichnis, "docker-aufrufe")
	umgebung := filepath.Join(verzeichnis, "docker-umgebung")
	attrappe := `#!/bin/bash
echo "$*" >> "` + protokoll + `"
env >> "` + umgebung + `"
case "$1" in
  ps) exit 0 ;;
  exec) echo "-- pg_dump" ;;
esac
exit 0
`
	require.NoError(t, os.WriteFile(filepath.Join(binVerzeichnis, "docker"), []byte(attrappe), 0o755))

	befehl := exec.Command("bash", filepath.Join(skripte, "backup.sh"))
	befehl.Dir = verzeichnis
	befehl.Env = []string{"PATH=" + binVerzeichnis + string(os.PathListSeparator) + os.Getenv("PATH"), "HOME=" + verzeichnis}
	aus, err := befehl.CombinedOutput()
	require.NoError(t, err, "backup.sh: %s", aus)

	aufrufe, err := os.ReadFile(protokoll)
	require.NoError(t, err)
	assert.Contains(t, string(aufrufe), "pg_dump -U gateuser -d gatedb",
		"Benutzer und Datenbank kommen aus der .env, umschließende Anführungszeichen fallen weg")

	umgebungDocker, err := os.ReadFile(umgebung)
	require.NoError(t, err)
	assert.NotContains(t, string(umgebungDocker), "JWT_SECRET",
		"ein Geheimnis aus der .env steht in der Umgebung der Kindprozesse")
	assert.NotContains(t, string(umgebungDocker), "SMTP_PASSWORD")
}

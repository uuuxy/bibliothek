package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Gate für den .env-Teil von scripts/backup.sh (docs/OFFEN.md 6.2, 23.09.2026).
//
// Bis dahin lud das Skript die Datei mit `export $(grep -v '^#' .env | xargs)`: Jedes
// Geheimnis der Datei stand danach in der Umgebung des Skripts und aller Kindprozesse
// (docker, gzip, find), obwohl es nur POSTGRES_USER und POSTGRES_DB braucht — den
// Backup-Schlüssel liest der Container (backup_krypto.sh), nicht das Skript. Dazu zerlegte
// xargs Werte mit Leerzeichen, und ein `*` im Wert wurde als Dateimuster aufgelöst.
//
// Geprüft wird am ausgeführten Skript: Eine `docker`-Attrappe schreibt ihre Argumente und
// ihre Umgebung mit. Sie meldet keinen laufenden Backend-Container, also läuft der
// Notnagel-Zweig (Klartext-Dump); für die Frage ist der Zweig gleich, beide rufen pg_dump
// über docker.
func TestBackupSkript_LiestAusDerEnvNurDieZweiWerte(t *testing.T) {
	dir := t.TempDir()
	skripte := filepath.Join(dir, "scripts")
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
	require.NoError(t, os.WriteFile(filepath.Join(dir, ".env"), []byte(env), 0o600))

	binDir := filepath.Join(dir, "bin")
	require.NoError(t, os.MkdirAll(binDir, 0o755))
	protokoll := filepath.Join(dir, "docker-aufrufe")
	umgebung := filepath.Join(dir, "docker-umgebung")
	attrappe := `#!/bin/bash
echo "$*" >> "` + protokoll + `"
env >> "` + umgebung + `"
case "$1" in
  ps) exit 0 ;;
  exec) echo "-- pg_dump" ;;
esac
exit 0
`
	require.NoError(t, os.WriteFile(filepath.Join(binDir, "docker"), []byte(attrappe), 0o755))

	cmd := exec.Command("bash", filepath.Join(skripte, "backup.sh"))
	cmd.Dir = dir
	cmd.Env = []string{"PATH=" + binDir + string(os.PathListSeparator) + os.Getenv("PATH"), "HOME=" + dir}
	aus, err := cmd.CombinedOutput()
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

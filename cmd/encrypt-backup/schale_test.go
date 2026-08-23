package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Gate für scripts/backup_krypto.sh — den Helfer, der dieses Werkzeug im Betrieb ruft.
//
// Der gefährliche Teil steckt nicht im Go-Code, sondern in der Reihenfolge der Shell:
// verschlüsseln, PRÜFEN, erst dann den Klartext löschen. Kehrt die sich um oder fällt
// die Prüfung weg, löscht ein Deploy die einzige lesbare Sicherung und stellt eine
// unbrauchbare Datei an ihre Stelle — bemerkt würde das erst beim Wiederherstellen.
//
// Ein echtes Docker läuft dafür nicht. `docker` ist hier ein Skript im PATH, das die
// drei Aufrufe des Helfers beantwortet und die Verschlüsselung an das echte, frisch
// gebaute Binary weiterreicht.

const gatePassphrase = "gate-passphrase-mind-32-zeichen-xxxxxxx"

// baueWerkzeuge baut BEIDE echten Werkzeuge — verschlüsseln und wiederherstellen. Die
// Attrappe reicht an sie durch, damit im Gate derselbe Rundweg läuft wie im Betrieb.
func baueWerkzeuge(t *testing.T, dir string) (verschluesseln, wiederherstellen string) {
	t.Helper()
	for pfad, ziel := range map[string]string{
		".":                 filepath.Join(dir, "encrypt-backup"),
		"../restore-backup": filepath.Join(dir, "restore-backup"),
	} {
		cmd := exec.Command("go", "build", "-o", ziel, pfad)
		cmd.Env = append(os.Environ(), "GOWORK=off")
		aus, err := cmd.CombinedOutput()
		require.NoError(t, err, "Bau von %s: %s", pfad, aus)
	}
	return filepath.Join(dir, "encrypt-backup"), filepath.Join(dir, "restore-backup")
}

// dockerAttrappe legt ein ausführbares `docker` in dir/bin ab. modus steuert, was beim
// Verschlüsselungs-Aufruf herauskommt: "echt", "fehler", "muell" oder "abgeschnitten".
// Der Prüf-Aufruf (`sh -c … restore-backup`) läuft in allen Modi über das echte
// Wiederherstellungs-Werkzeug — genau das soll ja die Aussage tragen.
func dockerAttrappe(t *testing.T, dir, modus string) string {
	t.Helper()
	verschluesselWerkzeug, restoreWerkzeug := baueWerkzeuge(t, dir)

	binDir := filepath.Join(dir, "bin")
	require.NoError(t, os.MkdirAll(binDir, 0o755))

	var verschluesseln string
	switch modus {
	case "echt":
		verschluesseln = "exec env BACKUP_ENCRYPTION_KEY=" + gatePassphrase + " " + verschluesselWerkzeug
	case "fehler":
		verschluesseln = "echo 'docker: container weg' >&2; exit 1"
	case "muell":
		// Exit-Code 0, aber der Inhalt ist kein Backup.
		verschluesseln = "cat > /dev/null; echo 'kein backup'; exit 0"
	case "abgeschnitten":
		// Die volle Platte: gültiger Anfang samt BKDF-Kennung, abgeschnittener Rest.
		// Die Formprüfung sieht hier nichts — nur der Rückweg fällt darauf herein.
		verschluesseln = "env BACKUP_ENCRYPTION_KEY=" + gatePassphrase + " " + verschluesselWerkzeug + " | head -c 60"
	default:
		t.Fatalf("unbekannter Modus %q", modus)
	}

	// Die Attrappe unterscheidet die beiden Aufrufe des Helfers an ihrem Kommando:
	// `./encrypt-backup` (verschlüsseln) und `sh -c …` (Rückweg prüfen).
	skript := `#!/bin/bash
case "$1" in
  ps) echo bibliothek-backend ;;
  exec)
    shift
    if [ "$1" = "-i" ]; then
      shift 2   # -i und den Containernamen weg
      case "$1" in
        ./encrypt-backup)
          ` + verschluesseln + `
          ;;
        sh)
          tmp=$(mktemp)
          cat > "$tmp"
          env BACKUP_ENCRYPTION_KEY=` + gatePassphrase + ` ` + restoreWerkzeug + ` "$tmp" > /dev/null
          rc=$?
          rm -f "$tmp"
          exit $rc
          ;;
      esac
    fi
    exit 0
    ;;
esac
`
	pfad := filepath.Join(binDir, "docker")
	require.NoError(t, os.WriteFile(pfad, []byte(skript), 0o755))
	return binDir
}

// ruftHelfer sourct scripts/backup_krypto.sh und ruft verschluessele_datei auf.
func ruftHelfer(t *testing.T, binDir, quelle string) error {
	t.Helper()
	helfer, err := filepath.Abs(filepath.Join("..", "..", "scripts", "backup_krypto.sh"))
	require.NoError(t, err)

	cmd := exec.Command("bash", "-c", `source "$1"; verschluessele_datei "$2"`, "bash", helfer, quelle)
	cmd.Env = append(os.Environ(), "PATH="+binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	aus, err := cmd.CombinedOutput()
	if err != nil && len(aus) > 0 {
		t.Logf("Helfer-Ausgabe: %s", aus)
	}
	return err
}

func klartextDatei(t *testing.T, dir string) string {
	t.Helper()
	pfad := filepath.Join(dir, "backup_20260823_120000.sql.gz")
	// Inhalt muss ein gültiger gzip-Strom sein — das Werkzeug lehnt alles andere ab.
	require.NoError(t, os.WriteFile(pfad, gzipData(t, []byte("-- pg_dump\nCREATE TABLE schueler (id uuid);\n")), 0o600))
	return pfad
}

func TestSchale_ErfolgLoeschtKlartext(t *testing.T) {
	dir := t.TempDir()
	binDir := dockerAttrappe(t, dir, "echt")
	quelle := klartextDatei(t, dir)

	require.NoError(t, ruftHelfer(t, binDir, quelle))

	assert.NoFileExists(t, quelle, "der Klartext-Dump muss nach der Verschlüsselung weg sein")
	inhalt, err := os.ReadFile(quelle + ".enc")
	require.NoError(t, err)
	assert.True(t, strings.HasPrefix(string(inhalt), "BKDF"), "Ergebnis trägt die Format-Kennung nicht")
	assert.NotContains(t, string(inhalt), "schueler")
}

// TestSchale_FehlerBehaeltKlartext: Wenn die Verschlüsselung scheitert, ist der
// Klartext das EINZIGE, was noch da ist. Er darf unter keinen Umständen gelöscht werden.
func TestSchale_FehlerBehaeltKlartext(t *testing.T) {
	dir := t.TempDir()
	binDir := dockerAttrappe(t, dir, "fehler")
	quelle := klartextDatei(t, dir)

	assert.Error(t, ruftHelfer(t, binDir, quelle), "ein gescheiterter Lauf muss als Fehler zurückkommen")
	assert.FileExists(t, quelle, "bei Fehler muss der Klartext liegen bleiben")
	assert.NoFileExists(t, quelle+".enc", "eine angefangene .enc-Datei darf nicht liegen bleiben")
}

// TestSchale_MuellAusgabeBehaeltKlartext ist die Gegenprobe am fertigen Artefakt: Der
// Aufruf meldet Erfolg, aber die Datei ist kein Backup. Ohne die Prüfung in
// pruefe_enc_datei wäre hier der Klartext gelöscht und nichts Brauchbares an seiner
// Stelle.
func TestSchale_MuellAusgabeBehaeltKlartext(t *testing.T) {
	dir := t.TempDir()
	binDir := dockerAttrappe(t, dir, "muell")
	quelle := klartextDatei(t, dir)

	assert.Error(t, ruftHelfer(t, binDir, quelle))
	assert.FileExists(t, quelle, "ohne gültige .enc-Datei muss der Klartext bleiben")
	assert.NoFileExists(t, quelle+".enc")
}

// TestSchale_AbgeschnitteneDateiBehaeltKlartext ist der Fall, den die Formprüfung NICHT
// sieht: Die Platte läuft während des Schreibens voll, die Datei trägt ihre
// BKDF-Kennung und hat plausible Größe — nur entschlüsseln lässt sie sich nicht mehr.
// Ohne pruefe_enc_rundweg wäre hier der Klartext gelöscht und an seiner Stelle ein
// Backup, das erst im Ernstfall als leer auffällt.
func TestSchale_AbgeschnitteneDateiBehaeltKlartext(t *testing.T) {
	dir := t.TempDir()
	binDir := dockerAttrappe(t, dir, "abgeschnitten")
	quelle := klartextDatei(t, dir)

	assert.Error(t, ruftHelfer(t, binDir, quelle))
	assert.FileExists(t, quelle, "eine nicht entschlüsselbare .enc-Datei ist kein Grund, den Klartext zu löschen")
	assert.NoFileExists(t, quelle+".enc")
}

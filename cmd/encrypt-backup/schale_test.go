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

// Prüft scripts/backup_krypto.sh, den Helfer, der dieses Werkzeug im Betrieb ruft.
//
// Der gefährliche Teil steckt nicht im Go-Code, sondern in der Reihenfolge der Shell:
// verschlüsseln, prüfen, erst dann den Klartext löschen. Kehrt sie sich um oder fällt die Prüfung
// weg, löscht ein Deploy die einzige lesbare Sicherung und stellt eine unbrauchbare Datei an ihre
// Stelle; bemerkt würde das erst beim Wiederherstellen.
//
// Ein echtes Docker läuft dafür nicht. `docker` ist hier ein Skript im PATH, das die drei
// Aufrufe des Helfers beantwortet und die Verschlüsselung an das echte, frisch gebaute Programm
// weiterreicht.

const gatePassphrase = "gate-passphrase-mind-32-zeichen-xxxxxxx"

// baueWerkzeuge baut beide echten Werkzeuge, verschlüsseln und wiederherstellen. Die Attrappe
// reicht an sie durch, damit im Test derselbe Rundweg läuft wie im Betrieb.
func baueWerkzeuge(t *testing.T, verzeichnis string) (verschluesseln, wiederherstellen string) {
	t.Helper()
	for pfad, ziel := range map[string]string{
		".":                 filepath.Join(verzeichnis, "encrypt-backup"),
		"../restore-backup": filepath.Join(verzeichnis, "restore-backup"),
	} {
		befehl := exec.Command("go", "build", "-o", ziel, pfad)
		befehl.Env = append(os.Environ(), "GOWORK=off")
		aus, err := befehl.CombinedOutput()
		require.NoError(t, err, "Bau von %s: %s", pfad, aus)
	}
	return filepath.Join(verzeichnis, "encrypt-backup"), filepath.Join(verzeichnis, "restore-backup")
}

// dockerAttrappe legt ein ausführbares `docker` in verzeichnis/bin ab. modus steuert, was beim
// Aufruf der Verschlüsselung herauskommt: "echt", "fehler", "muell" oder "abgeschnitten". Der
// Aufruf der Prüfung (`sh -c … restore-backup`) läuft in allen Modi über das echte Werkzeug der
// Wiederherstellung, denn genau das soll die Aussage tragen.
func dockerAttrappe(t *testing.T, verzeichnis, modus string) string {
	t.Helper()
	verschluesselWerkzeug, wiederherstellWerkzeug := baueWerkzeuge(t, verzeichnis)

	binVerzeichnis := filepath.Join(verzeichnis, "bin")
	require.NoError(t, os.MkdirAll(binVerzeichnis, 0o755))

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
		// Die volle Platte: gültiger Anfang samt Kennung BKDF, abgeschnittener Rest. Die Prüfung der
		// Form sieht hier nichts, nur der Rückweg fällt darauf herein.
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
          env BACKUP_ENCRYPTION_KEY=` + gatePassphrase + ` ` + wiederherstellWerkzeug + ` "$tmp" > /dev/null
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
	pfad := filepath.Join(binVerzeichnis, "docker")
	require.NoError(t, os.WriteFile(pfad, []byte(skript), 0o755))
	return binVerzeichnis
}

// ruftHelfer lädt scripts/backup_krypto.sh mit source und ruft verschluessele_datei auf.
func ruftHelfer(t *testing.T, binVerzeichnis, quelle string) error {
	t.Helper()
	helfer, err := filepath.Abs(filepath.Join("..", "..", "scripts", "backup_krypto.sh"))
	require.NoError(t, err)

	befehl := exec.Command("bash", "-c", `source "$1"; verschluessele_datei "$2"`, "bash", helfer, quelle)
	befehl.Env = append(os.Environ(), "PATH="+binVerzeichnis+string(os.PathListSeparator)+os.Getenv("PATH"))
	aus, err := befehl.CombinedOutput()
	if err != nil && len(aus) > 0 {
		t.Logf("Helfer-Ausgabe: %s", aus)
	}
	return err
}

func klartextDatei(t *testing.T, verzeichnis string) string {
	t.Helper()
	pfad := filepath.Join(verzeichnis, "backup_20260823_120000.sql.gz")
	// Der Inhalt muss ein gültiger gzip-Strom sein; das Werkzeug lehnt alles andere ab.
	require.NoError(t, os.WriteFile(pfad, gzipDaten(t, []byte("-- pg_dump\nCREATE TABLE schueler (id uuid);\n")), 0o600))
	return pfad
}

func TestSchale_ErfolgLoeschtKlartext(t *testing.T) {
	verzeichnis := t.TempDir()
	binVerzeichnis := dockerAttrappe(t, verzeichnis, "echt")
	quelle := klartextDatei(t, verzeichnis)

	require.NoError(t, ruftHelfer(t, binVerzeichnis, quelle))

	assert.NoFileExists(t, quelle, "der Klartext-Dump muss nach der Verschlüsselung weg sein")
	inhalt, err := os.ReadFile(quelle + ".enc")
	require.NoError(t, err)
	assert.True(t, strings.HasPrefix(string(inhalt), "BKDF"), "Ergebnis trägt die Format-Kennung nicht")
	assert.NotContains(t, string(inhalt), "schueler")
}

// Scheitert die Verschlüsselung, ist der Klartext das Einzige, was noch da ist. Er darf unter
// keinen Umständen gelöscht werden.
func TestSchale_FehlerBehaeltKlartext(t *testing.T) {
	verzeichnis := t.TempDir()
	binVerzeichnis := dockerAttrappe(t, verzeichnis, "fehler")
	quelle := klartextDatei(t, verzeichnis)

	assert.Error(t, ruftHelfer(t, binVerzeichnis, quelle), "ein gescheiterter Lauf muss als Fehler zurückkommen")
	assert.FileExists(t, quelle, "bei Fehler muss der Klartext liegen bleiben")
	assert.NoFileExists(t, quelle+".enc", "eine angefangene .enc-Datei darf nicht liegen bleiben")
}

// Die Gegenprobe am fertigen Ergebnis: Der Aufruf meldet Erfolg, aber die Datei ist keine
// Sicherung. Ohne die Prüfung in pruefe_enc_datei wäre hier der Klartext gelöscht und nichts
// Brauchbares an seiner Stelle.
func TestSchale_MuellAusgabeBehaeltKlartext(t *testing.T) {
	verzeichnis := t.TempDir()
	binVerzeichnis := dockerAttrappe(t, verzeichnis, "muell")
	quelle := klartextDatei(t, verzeichnis)

	assert.Error(t, ruftHelfer(t, binVerzeichnis, quelle))
	assert.FileExists(t, quelle, "ohne gültige .enc-Datei muss der Klartext bleiben")
	assert.NoFileExists(t, quelle+".enc")
}

// Der Fall, den die Prüfung der Form nicht sieht: Die Platte läuft beim Schreiben voll, die
// Datei trägt ihre Kennung BKDF und hat plausible Größe, nur entschlüsseln lässt sie sich nicht
// mehr. Ohne pruefe_enc_rundweg wäre hier der Klartext gelöscht und an seiner Stelle eine
// Sicherung, die erst im Ernstfall als leer auffällt.
func TestSchale_AbgeschnitteneDateiBehaeltKlartext(t *testing.T) {
	verzeichnis := t.TempDir()
	binVerzeichnis := dockerAttrappe(t, verzeichnis, "abgeschnitten")
	quelle := klartextDatei(t, verzeichnis)

	assert.Error(t, ruftHelfer(t, binVerzeichnis, quelle))
	assert.FileExists(t, quelle, "eine nicht entschlüsselbare .enc-Datei ist kein Grund, den Klartext zu löschen")
	assert.NoFileExists(t, quelle+".enc")
}

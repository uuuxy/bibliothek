package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
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
		if err != nil {
			t.Fatalf("Bau von %s: %s: %v", pfad, aus, err)
		}
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
	if err := os.MkdirAll(binVerzeichnis, 0o755); err != nil {
		t.Fatalf("os.MkdirAll: %v", err)
	}

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
	if err := os.WriteFile(pfad, []byte(skript), 0o755); err != nil {
		t.Fatalf("os.WriteFile: %v", err)
	}
	return binVerzeichnis
}

// ruftHelfer lädt scripts/backup_krypto.sh mit source und ruft verschluessele_datei auf.
func ruftHelfer(t *testing.T, binVerzeichnis, quelle string) error {
	t.Helper()
	helfer, err := filepath.Abs(filepath.Join("..", "..", "scripts", "backup_krypto.sh"))
	if err != nil {
		t.Fatalf("filepath.Abs: %v", err)
	}

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
	if err := os.WriteFile(pfad, gzipDaten(t, []byte("-- pg_dump\nCREATE TABLE schueler (id uuid);\n")), 0o600); err != nil {
		t.Fatalf("os.WriteFile: %v", err)
	}
	return pfad
}

func TestSchale_ErfolgLoeschtKlartext(t *testing.T) {
	verzeichnis := t.TempDir()
	binVerzeichnis := dockerAttrappe(t, verzeichnis, "echt")
	quelle := klartextDatei(t, verzeichnis)

	if err := ruftHelfer(t, binVerzeichnis, quelle); err != nil {
		t.Fatalf("ruftHelfer: %v", err)
	}

	if _, err := os.Stat(quelle); err == nil {
		t.Errorf("Datei %s besteht noch: der Klartext-Dump muss nach der Verschlüsselung weg sein", quelle)
	}
	inhalt, err := os.ReadFile(quelle + ".enc")
	if err != nil {
		t.Fatalf("os.ReadFile: %v", err)
	}
	if !strings.HasPrefix(string(inhalt), "BKDF") {
		t.Error("Ergebnis trägt die Format-Kennung nicht")
	}
	if strings.Contains(string(inhalt), "schueler") {
		t.Errorf("string(inhalt) = %q, erwartet ohne %q", string(inhalt), "schueler")
	}
}

// Scheitert die Verschlüsselung, ist der Klartext das Einzige, was noch da ist. Er darf unter
// keinen Umständen gelöscht werden.
func TestSchale_FehlerBehaeltKlartext(t *testing.T) {
	verzeichnis := t.TempDir()
	binVerzeichnis := dockerAttrappe(t, verzeichnis, "fehler")
	quelle := klartextDatei(t, verzeichnis)

	if ruftHelfer(t, binVerzeichnis, quelle) == nil {
		t.Error("ruftHelfer: kein Fehler, erwartet einen: ein gescheiterter Lauf muss als Fehler zurückkommen")
	}
	if _, err := os.Stat(quelle); err != nil {
		t.Errorf("Datei %s fehlt: %v: bei Fehler muss der Klartext liegen bleiben", quelle, err)
	}
	if _, err := os.Stat(quelle + ".enc"); err == nil {
		t.Errorf("Datei %s besteht noch: eine angefangene .enc-Datei darf nicht liegen bleiben", quelle+".enc")
	}
}

// Die Gegenprobe am fertigen Ergebnis: Der Aufruf meldet Erfolg, aber die Datei ist keine
// Sicherung. Ohne die Prüfung in pruefe_enc_datei wäre hier der Klartext gelöscht und nichts
// Brauchbares an seiner Stelle.
func TestSchale_MuellAusgabeBehaeltKlartext(t *testing.T) {
	verzeichnis := t.TempDir()
	binVerzeichnis := dockerAttrappe(t, verzeichnis, "muell")
	quelle := klartextDatei(t, verzeichnis)

	if ruftHelfer(t, binVerzeichnis, quelle) == nil {
		t.Error("ruftHelfer: kein Fehler, erwartet einen")
	}
	if _, err := os.Stat(quelle); err != nil {
		t.Errorf("Datei %s fehlt: %v: ohne gültige .enc-Datei muss der Klartext bleiben", quelle, err)
	}
	if _, err := os.Stat(quelle + ".enc"); err == nil {
		t.Errorf("Datei %s besteht noch", quelle+".enc")
	}
}

// Der Fall, den die Prüfung der Form nicht sieht: Die Platte läuft beim Schreiben voll, die
// Datei trägt ihre Kennung BKDF und hat plausible Größe, nur entschlüsseln lässt sie sich nicht
// mehr. Ohne pruefe_enc_rundweg wäre hier der Klartext gelöscht und an seiner Stelle eine
// Sicherung, die erst im Ernstfall als leer auffällt.
func TestSchale_AbgeschnitteneDateiBehaeltKlartext(t *testing.T) {
	verzeichnis := t.TempDir()
	binVerzeichnis := dockerAttrappe(t, verzeichnis, "abgeschnitten")
	quelle := klartextDatei(t, verzeichnis)

	if ruftHelfer(t, binVerzeichnis, quelle) == nil {
		t.Error("ruftHelfer: kein Fehler, erwartet einen")
	}
	if _, err := os.Stat(quelle); err != nil {
		t.Errorf("Datei %s fehlt: %v: eine nicht entschlüsselbare .enc-Datei ist kein Grund, den Klartext zu löschen", quelle, err)
	}
	if _, err := os.Stat(quelle + ".enc"); err == nil {
		t.Errorf("Datei %s besteht noch", quelle+".enc")
	}
}

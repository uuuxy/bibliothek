package jobs

import (
	"errors"
	"io/fs"
	"os"
	"slices"
	"strings"
	"time"

	"bibliothek/pkg/closeutil"
)

// Die Sicherungen im Backup-Verzeichnis: EINE Liste, zwei Verbraucher — die Rotation
// (rotateBackups) und der Status-Wächter (api/backup_status.go, newestBackupTime).
//
// Gelesen wird über os.OpenRoot, wie überall im Projekt bei Verzeichnissen mit Dateinamen
// von außen (inventur/uploads_pfad.go): Die Wurzel bindet jeden Zugriff OS-seitig an das
// Verzeichnis; ein Symlink darin wird als Link gesehen und als Link gelöscht, nie sein Ziel.
//
// Einordnung (Jules-Sentinel #601, 09.09.2026, als „HIGH: Path Traversal" gemeldet): Einen
// Eingang für Angreifer gibt es hier nicht — das Verzeichnis kommt aus BACKUP_DIR (Betreiber),
// die Namen schreibt der Job selbst, kein Endpunkt nimmt einen Dateinamen entgegen. Die
// Umstellung ist Hausform und Einhegung, kein Sicherheitsfix; neu sind vor allem die Tests,
// die beide Funktionen bis dahin nicht hatten.
const (
	backupPraefix = "backup_"
	backupEndung  = ".sql.gz.enc"
)

// BackupDatei ist eine Sicherung im Backup-Verzeichnis.
type BackupDatei struct {
	Name        string
	GeaendertAm time.Time
}

// BackupDateien listet die backup_*.sql.gz.enc des Verzeichnisses, nach Namen sortiert —
// die Namen tragen den Zeitstempel, die Reihenfolge ist also die zeitliche. Fehlt das
// Verzeichnis, ist die Liste leer: „noch kein Backup" ist kein Fehler.
func BackupDateien(dir string) ([]BackupDatei, error) {
	root, err := os.OpenRoot(dir)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	defer closeutil.LogClose(root, "backup dir")
	return listeBackups(root)
}

// listeBackups liest die Sicherungen einer bereits geöffneten Wurzel. Unterverzeichnisse
// und Fremddateien bleiben draußen; die Änderungszeit kommt per Lstat (e.Info), ein
// Symlink wird also nicht verfolgt.
func listeBackups(root *os.Root) ([]BackupDatei, error) {
	verzeichnis, err := root.Open(".")
	if err != nil {
		return nil, err
	}
	defer closeutil.LogClose(verzeichnis, "backup dir")
	eintraege, err := verzeichnis.ReadDir(-1)
	if err != nil {
		return nil, err
	}
	var dateien []BackupDatei
	for _, e := range eintraege {
		name := e.Name()
		if e.IsDir() || !strings.HasPrefix(name, backupPraefix) || !strings.HasSuffix(name, backupEndung) {
			continue
		}
		info, infoErr := e.Info()
		if infoErr != nil {
			continue
		}
		dateien = append(dateien, BackupDatei{Name: name, GeaendertAm: info.ModTime()})
	}
	// ReadDir liefert in Verzeichnisreihenfolge — anders als das frühere filepath.Glob, das
	// sortiert zurückgab. Ohne die Sortierung löschte die Rotation Beliebiges.
	slices.SortFunc(dateien, func(a, b BackupDatei) int { return strings.Compare(a.Name, b.Name) })
	return dateien, nil
}

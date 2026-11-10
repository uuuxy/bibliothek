package jobs

import (
	"errors"
	"io/fs"
	"os"
	"slices"
	"strings"
	"time"

	"bibliothek/internal/backupkrypto"
	"bibliothek/pkg/closeutil"
)

// Die Sicherungen im Backup-Verzeichnis: eine Liste, drei Verbraucher, die Aufbewahrung
// (rotiereBackups), die Restore-Probe (juengsteBackupDatei) und der Wächter des Stands
// (api/system/backup_stand.go).
//
// Gelesen wird über os.OpenRoot, wie überall im Projekt bei Verzeichnissen mit Dateinamen von
// außen (pkg/coverablage/coverablage.go): Die Wurzel bindet jeden Zugriff im Betriebssystem an das
// Verzeichnis; ein Symlink darin wird als Link gesehen und als Link gelöscht, nie sein Ziel.
//
// Einen Eingang für Angreifer gibt es hier nicht: Das Verzeichnis kommt aus BACKUP_DIR, also vom
// Betreiber, die Namen schreibt der Lauf selbst, und keine Tür nimmt einen Dateinamen entgegen.
// os.OpenRoot ist hier Hausform und Einhegung.
const (
	backupPraefix = "backup_"
	backupEndung  = ".sql.gz.enc"
)

// BackupDatei ist eine Sicherung im Backup-Verzeichnis.
type BackupDatei struct {
	Name        string
	GeaendertAm time.Time
}

// SicherungsZeit ist der Zeitpunkt aus dem Namen, nicht die Änderungszeit, die ein Kopieren
// oder Zurückspielen verschiebt. Passt der Name nicht zum Stempel, gilt die Änderungszeit.
// Aufbewahrung, Restore-Probe und der Stand der Sicherungen lesen die Zeit nur hier.
func (d BackupDatei) SicherungsZeit() time.Time {
	stempel := strings.TrimSuffix(strings.TrimPrefix(d.Name, backupPraefix), backupEndung)
	if t, err := time.Parse(sicherungsStempel, stempel); err == nil {
		return t
	}
	return d.GeaendertAm
}

// HatKopf sagt, ob die Sicherung im Verzeichnis mit dem Kopf ihres Formats beginnt. Eine leere
// oder fremde Datei unter dem Namen einer Sicherung lässt sich nicht zurückspielen; der Stand der
// Sicherungen zählt sie nicht.
func (d BackupDatei) HatKopf(verzeichnis string) bool {
	wurzel, err := os.OpenRoot(verzeichnis)
	if err != nil {
		return false
	}
	defer closeutil.Schliesse(wurzel, "backup dir")
	datei, err := wurzel.Open(d.Name)
	if err != nil {
		return false
	}
	defer closeutil.Schliesse(datei, "backup file")
	return backupkrypto.HatKopf(datei)
}

// LeseBackupDateien liest die backup_*.sql.gz.enc des Verzeichnisses, nach Namen sortiert; die
// Namen tragen den Zeitstempel, die Reihenfolge ist also die zeitliche. Fehlt das Verzeichnis,
// ist die Liste leer: „noch keine Sicherung" ist kein Fehler.
func LeseBackupDateien(verzeichnis string) ([]BackupDatei, error) {
	wurzel, err := os.OpenRoot(verzeichnis)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	defer closeutil.Schliesse(wurzel, "backup dir")
	return leseBackups(wurzel)
}

// leseBackups liest die Sicherungen einer schon geöffneten Wurzel. Unterverzeichnisse und fremde
// Dateien bleiben draußen; die Änderungszeit kommt über Lstat (e.Info), ein Symlink wird also
// nicht verfolgt.
func leseBackups(wurzel *os.Root) ([]BackupDatei, error) {
	verzeichnis, err := wurzel.Open(".")
	if err != nil {
		return nil, err
	}
	defer closeutil.Schliesse(verzeichnis, "backup dir")
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
		info, infoFehler := e.Info()
		if infoFehler != nil {
			continue
		}
		dateien = append(dateien, BackupDatei{Name: name, GeaendertAm: info.ModTime()})
	}
	// ReadDir liefert in der Reihenfolge des Verzeichnisses, nicht sortiert. Ohne die Sortierung
	// löschte die Aufbewahrung Beliebiges.
	slices.SortFunc(dateien, func(a, b BackupDatei) int { return strings.Compare(a.Name, b.Name) })
	return dateien, nil
}

package inventur

import (
	"fmt"
	"io"
	"log"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

// escapePgPass escapes backslashes and colons as required by PostgreSQL .pgpass format.
func escapePgPass(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	return strings.ReplaceAll(s, ":", "\\:")
}

// dumpDatabase führt pg_dump aus und speichert die Ausgabe als db_dump.sql.
func (bm *BackupManager) dumpDatabase(backupPath string) error {
	u, err := url.Parse(bm.databaseURL)
	if err != nil {
		return fmt.Errorf("ungültige DATABASE_URL: %w", err)
	}

	host := u.Hostname()
	port := u.Port()
	if port == "" {
		port = "5432"
	}
	dbName := u.Path[1:]
	user := u.User.Username()
	password, _ := u.User.Password()

	passFile, err := os.CreateTemp("", "pgpass-*")
	if err != nil {
		return fmt.Errorf("konnte pgpass-Datei nicht erstellen: %w", err)
	}
	defer func() { _ = os.Remove(passFile.Name()) }()

	pgPassLine := fmt.Sprintf("%s:%s:%s:%s:%s\n",
		escapePgPass(host),
		escapePgPass(port),
		escapePgPass(dbName),
		escapePgPass(user),
		escapePgPass(password),
	)
	if _, err := passFile.WriteString(pgPassLine); err != nil {
		_ = passFile.Close()
		return fmt.Errorf("konnte in pgpass-Datei nicht schreiben: %w", err)
	}
	_ = passFile.Close()

	dumpFile := filepath.Join(backupPath, "db_dump.sql")
	// #nosec G304 - dumpFile is safely constructed inside the backup directory
	outFile, err := os.Create(dumpFile)
	if err != nil {
		return fmt.Errorf("konnte Dump-Datei nicht erstellen: %w", err)
	}
	defer func() { _ = outFile.Close() }()

	cmd := exec.Command("pg_dump",
		"-h", host,
		"-p", port,
		"-U", user,
		"-d", dbName,
		"--clean",
		"--if-exists",
	)
	cmd.Env = append(os.Environ(), "PGPASSFILE="+passFile.Name())
	cmd.Stdout = outFile
	cmd.Stderr = nil

	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("stderr pipe fehler: %w", err)
	}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("pg_dump konnte nicht gestartet werden: %w", err)
	}

	stderr, _ := io.ReadAll(stderrPipe)
	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("pg_dump fehlgeschlagen: %w (stderr: %s)", err, string(stderr))
	}

	info, err := os.Stat(dumpFile)
	if err != nil || info.Size() == 0 {
		return fmt.Errorf("dump-Datei ist leer oder nicht vorhanden")
	}

	log.Printf("Backup: Datenbank-Dump erstellt (%d Bytes)", info.Size())
	return nil
}

// copyUploads kopiert den uploads-Ordner ins Backup.
func (bm *BackupManager) copyUploads(backupPath string) error {
	srcDir := "uploads"
	if _, err := os.Stat(srcDir); os.IsNotExist(err) {
		log.Println("Backup: Kein uploads-Ordner vorhanden, überspringe")
		return nil
	}

	destDir := filepath.Join(backupPath, "uploads")
	return copyDir(srcDir, destDir)
}

// rotateBackups löscht die ältesten Backups, wenn mehr als maxBackups vorhanden sind.
func (bm *BackupManager) rotateBackups() {
	entries, err := os.ReadDir(bm.backupDir)
	if err != nil {
		log.Printf("Backup WARNUNG: Konnte Backup-Verzeichnis nicht lesen: %v", err)
		return
	}

	var dirs []os.DirEntry
	for _, e := range entries {
		if e.IsDir() {
			dirs = append(dirs, e)
		}
	}
	if len(dirs) <= bm.maxBackups {
		return
	}

	sort.Slice(dirs, func(i, j int) bool {
		return dirs[i].Name() < dirs[j].Name()
	})

	toDelete := dirs[:len(dirs)-bm.maxBackups]
	for _, d := range toDelete {
		path := filepath.Join(bm.backupDir, d.Name())
		// #nosec G304 - path is derived from safe directory entries
		if err := os.RemoveAll(path); err != nil {
			log.Printf("Backup WARNUNG: Konnte altes Backup nicht löschen: %s: %v", path, err)
		} else {
			log.Printf("Backup: Altes Backup gelöscht: %s", d.Name())
		}
	}
}

func copyDir(src, dst string) error {
	if err := os.MkdirAll(dst, 0750); err != nil {
		return err
	}

	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())

		if entry.IsDir() {
			if err := copyDir(srcPath, dstPath); err != nil {
				return err
			}
			continue
		}

		if err := copyFile(srcPath, dstPath); err != nil {
			return err
		}
	}
	return nil
}

func copyFile(src, dst string) error {
	// #nosec G304 - src is derived from safe directory entries
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer func() { _ = in.Close() }()

	// #nosec G304 - dst is safely constructed
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer func() { _ = out.Close() }()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Sync()
}

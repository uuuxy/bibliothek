package system

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"bibliothek/api/tuer"
)

// starkerTestSchluessel erfüllt jobs.MinBackupSchluesselLaenge. Die Tests standen
// vorher auf "test-key" (8 Zeichen) — seit der Wächter kurze Passphrasen meldet, wäre
// das in jedem Fall "warning" gewesen und hätte die Alters-Schwellen verdeckt.
const starkerTestSchluessel = "ein-hinreichend-langer-testschluessel-32+"

func rufeBackupStand(t *testing.T) *httptest.ResponseRecorder {
	t.Helper()
	s := &Tueren{Basis: &tuer.Basis{}}
	req := httptest.NewRequest(http.MethodGet, "/api/admin/system/backup-status", nil)
	rec := httptest.NewRecorder()
	s.BackupStand()(rec, req)
	return rec
}

// schreibeBackupDatei legt eine Backup-Datei mit definiertem Alter an.
func schreibeBackupDatei(t *testing.T, dir string, age time.Duration) {
	t.Helper()
	f := filepath.Join(dir, "backup_2026-07-09.sql.gz.enc")
	if err := os.WriteFile(f, []byte("x"), 0600); err != nil {
		t.Fatalf("write: %v", err)
	}
	mtime := time.Now().Add(-age)
	if err := os.Chtimes(f, mtime, mtime); err != nil {
		t.Fatalf("chtimes: %v", err)
	}
}

func TestBackupStand_OhneSchluesselKritisch(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("BACKUP_DIR", dir)
	t.Setenv("BACKUP_ENCRYPTION_KEY", "")
	schreibeBackupDatei(t, dir, time.Hour) // frisches Backup ändert nichts: ohne Key läuft der Job still leer

	body := rufeBackupStand(t).Body.String()
	if !strings.Contains(body, `"status":"critical"`) || !strings.Contains(body, `"encryption_key_set":false`) {
		t.Errorf("fehlender Key muss critical sein: %s", body)
	}
}

func TestBackupStand_LeeresVerzeichnisKritischOhneAbsturz(t *testing.T) {
	t.Setenv("BACKUP_DIR", t.TempDir()) // leer
	t.Setenv("BACKUP_ENCRYPTION_KEY", starkerTestSchluessel)

	rec := rufeBackupStand(t)
	if rec.Code != http.StatusOK {
		t.Fatalf("leeres Verzeichnis darf nicht crashen: %d", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, `"status":"critical"`) || !strings.Contains(body, `"last_backup_at":null`) {
		t.Errorf("kein Backup = critical mit null-Timestamp: %s", body)
	}
}

func TestBackupStand_FehlendesVerzeichnisKritischOhneAbsturz(t *testing.T) {
	t.Setenv("BACKUP_DIR", filepath.Join(t.TempDir(), "gibt-es-nicht"))
	t.Setenv("BACKUP_ENCRYPTION_KEY", starkerTestSchluessel)

	rec := rufeBackupStand(t)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"status":"critical"`) {
		t.Errorf("fehlendes Verzeichnis = sauberer critical: %d %s", rec.Code, rec.Body.String())
	}
}

// Der AES-Schlüssel wird per SHA-256 aus der Passphrase abgeleitet — schnell, also
// offline gut durchprobierbar. Eine kurze Passphrase ist damit das schwache Glied.
// Das Format bleibt unverändert (sonst wären alte Backups unlesbar); stattdessen
// meldet der Wächter den Zustand ins Admin-Dashboard.
func TestBackupStand_KurzerSchluesselIstWarnung(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("BACKUP_DIR", dir)
	t.Setenv("BACKUP_ENCRYPTION_KEY", "kurz123")
	schreibeBackupDatei(t, dir, time.Hour) // frisch — ohne die Schlüsselprüfung wäre das "ok"

	body := rufeBackupStand(t).Body.String()
	if !strings.Contains(body, `"status":"warning"`) {
		t.Errorf("kurze Passphrase muss warnen, Body: %s", body)
	}
	if !strings.Contains(body, `"encryption_key_weak":true`) {
		t.Errorf("encryption_key_weak fehlt oder ist false, Body: %s", body)
	}
}

// Gegenprobe: Ein fehlendes Backup bleibt critical und wird von der Schlüsselwarnung
// nicht auf "warning" heruntergestuft.
func TestBackupStand_KurzerSchluesselStuftCriticalNichtHerab(t *testing.T) {
	t.Setenv("BACKUP_DIR", t.TempDir()) // leer
	t.Setenv("BACKUP_ENCRYPTION_KEY", "kurz123")

	body := rufeBackupStand(t).Body.String()
	if !strings.Contains(body, `"status":"critical"`) {
		t.Errorf("kein Backup bleibt critical, Body: %s", body)
	}
}

func TestBackupStand_Schwellen(t *testing.T) {
	cases := []struct {
		age  time.Duration
		want string
	}{
		{2 * time.Hour, `"status":"ok"`},
		{30 * time.Hour, `"status":"warning"`},  // ein verpasster Tageslauf
		{50 * time.Hour, `"status":"critical"`}, // Puffer aufgebraucht
	}
	for _, c := range cases {
		dir := t.TempDir()
		t.Setenv("BACKUP_DIR", dir)
		t.Setenv("BACKUP_ENCRYPTION_KEY", starkerTestSchluessel)
		schreibeBackupDatei(t, dir, c.age)

		body := rufeBackupStand(t).Body.String()
		if !strings.Contains(body, c.want) {
			t.Errorf("Alter %v: erwartet %s, Body: %s", c.age, c.want, body)
		}
		if strings.Contains(body, `"last_backup_at":null`) {
			t.Errorf("Alter %v: Timestamp fehlt: %s", c.age, body)
		}
	}
}

// newestBackupTime nimmt die jüngste Sicherung nach Änderungszeit — nicht die
// lexikographisch letzte (die Liste ist nach Namen sortiert, die Zeit steht am Inode).
func TestJuengstesBackup_NimmtDieJuengste(t *testing.T) {
	dir := t.TempDir()
	for name, alter := range map[string]time.Duration{
		"backup_2026-09-01.sql.gz.enc": 5 * time.Hour,
		"backup_2026-09-02.sql.gz.enc": 1 * time.Hour,
		"notiz.txt":                    0,
	} {
		f := filepath.Join(dir, name)
		if err := os.WriteFile(f, []byte("x"), 0600); err != nil {
			t.Fatal(err)
		}
		mtime := time.Now().Add(-alter)
		if err := os.Chtimes(f, mtime, mtime); err != nil {
			t.Fatal(err)
		}
	}
	got := juengstesBackup(dir)
	if got == nil || time.Since(*got) > 90*time.Minute || time.Since(*got) < 30*time.Minute {
		t.Errorf("jüngste Sicherung ist ~1 h alt, bekam %v", got)
	}
	if juengstesBackup(filepath.Join(dir, "nein")) != nil {
		t.Errorf("fehlendes Verzeichnis = nil")
	}
}

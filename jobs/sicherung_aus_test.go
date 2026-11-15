package jobs

import (
	"bytes"
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"bibliothek/db"
	"bibliothek/repository"

	"github.com/pashagolub/pgxmock/v5"
)

// Ist die eigene Sicherung abgeschaltet, entsteht keine neue, und die alten Stände laufen nach der
// Aufbewahrungsdauer aus: Die Datenschutz-Auskunft nennt etwa drei Monate, und ohne neue Sicherung
// löschte die Aufbewahrung nichts mehr.

// zeitplanMitSicherungAus liefert einen Zeitplan, dessen Einstellungen den Schalter nennen.
func zeitplanMitSicherungAus(t *testing.T, aus bool) *Zeitplan {
	t.Helper()
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock: %v", err)
	}
	t.Cleanup(mock.Close)
	wert := "true"
	if aus {
		wert = "false"
	}
	mock.ExpectQuery(`SELECT schluessel, wert FROM system_einstellungen`).
		WillReturnRows(pgxmock.NewRows([]string{"schluessel", "wert"}).AddRow("sicherung_an", &wert))
	return &Zeitplan{pool: mock}
}

// schalteSicherungEin stellt die eigene Sicherung für die Dauer des Tests an. Die Vorgabe ist
// aus, und ohne den Schalter überspringt sich die Restore-Probe.
func schalteSicherungEin(t *testing.T, pool db.Pool) {
	t.Helper()
	an := true
	if err := repository.NewEinstellungsAbfragen(pool).Schreibe(context.Background(), &repository.EinstellungenPatch{SicherungAn: &an}); err != nil {
		t.Fatalf("Sicherung einschalten: %v", err)
	}
	t.Cleanup(func() {
		if _, err := pool.Exec(context.Background(), `DELETE FROM system_einstellungen WHERE schluessel = 'sicherung_an'`); err != nil {
			t.Errorf("Schalter der Sicherung zurücksetzen: %v", err)
		}
	})
}

// legeSicherungAn legt im Verzeichnis eine Sicherung mit dem Zeitstempel zeit an.
func legeSicherungAn(t *testing.T, verzeichnis string, zeit time.Time) string {
	t.Helper()
	name := backupPraefix + zeit.UTC().Format(sicherungsStempel) + backupEndung
	if err := os.WriteFile(filepath.Join(verzeichnis, name), []byte("x"), 0o600); err != nil {
		t.Fatalf("Sicherung anlegen: %v", err)
	}
	return name
}

func TestAbgelaufen_NimmtNurWasAelterIstAlsDieAufbewahrung(t *testing.T) {
	jetzt := time.Date(2026, time.November, 15, 3, 0, 0, 0, time.UTC)
	datei := func(alter time.Duration) BackupDatei {
		return BackupDatei{Name: backupPraefix + jetzt.Add(-alter).Format(sicherungsStempel) + backupEndung}
	}
	dateien := []BackupDatei{datei(Aufbewahrungsdauer + time.Hour), datei(Aufbewahrungsdauer - time.Hour), datei(24 * time.Hour)}

	weg := abgelaufen(dateien, jetzt)

	if len(weg) != 1 || weg[0].Name != dateien[0].Name {
		t.Errorf("abgelaufen = %v, erwartet nur %s", weg, dateien[0].Name)
	}
}

func TestNachtsicherung_AbgeschaltetLaesstNurAlteStaendeAuslaufen(t *testing.T) {
	verzeichnis := t.TempDir()
	t.Setenv("BACKUP_DIR", verzeichnis)
	t.Setenv("S3_ENDPOINT", "")
	jetzt := time.Now()
	alt := legeSicherungAn(t, verzeichnis, jetzt.Add(-Aufbewahrungsdauer-48*time.Hour))
	jung := legeSicherungAn(t, verzeichnis, jetzt.Add(-48*time.Hour))

	zeitplanMitSicherungAus(t, true).fuehreNachtsicherungAus(context.Background(), &BackupLauf{})

	eintraege, err := os.ReadDir(verzeichnis)
	if err != nil {
		t.Fatalf("Verzeichnis lesen: %v", err)
	}
	var namen []string
	for _, e := range eintraege {
		namen = append(namen, e.Name())
	}
	if len(namen) != 1 || namen[0] != jung {
		t.Errorf("im Verzeichnis: %v, erwartet nur %s (%s ist abgelaufen, eine neue entsteht nicht)", namen, jung, alt)
	}
}

// Gegenprobe: Mit eingeschalteter Sicherung läuft der gewohnte Lauf. Ohne Schlüssel bricht er ab,
// bevor er etwas löscht; die alte Sicherung bleibt. So zeigt der Fall darüber wirklich den Weg der
// abgeschalteten Sicherung.
func TestNachtsicherung_EingeschaltetLoeschtNichtNachAlter(t *testing.T) {
	verzeichnis := t.TempDir()
	t.Setenv("BACKUP_DIR", verzeichnis)
	t.Setenv("BACKUP_ENCRYPTION_KEY", "")
	alt := legeSicherungAn(t, verzeichnis, time.Now().Add(-Aufbewahrungsdauer-48*time.Hour))

	zeitplanMitSicherungAus(t, false).fuehreNachtsicherungAus(context.Background(), &BackupLauf{})

	if _, err := os.Stat(filepath.Join(verzeichnis, alt)); err != nil {
		t.Errorf("die alte Sicherung ist weg, obwohl die Sicherung eingeschaltet ist: %v", err)
	}
}

// Ohne eigene Sicherung gibt es nichts zu proben: Die Probe läuft nicht, und ihr letztes Ergebnis
// bleibt. Sonst stünde nach einer Woche „fehlgeschlagen" in der Betriebsbereitschaft.
func TestRestoreProbe_AbgeschaltetUebersprungen(t *testing.T) {
	var protokoll bytes.Buffer
	vorher := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&protokoll, nil)))
	t.Cleanup(func() { slog.SetDefault(vorher) })
	t.Setenv("BACKUP_ENCRYPTION_KEY", "ein-hinreichend-langer-testschluessel-32+")
	t.Setenv("DATABASE_URL", "postgres://niemand@127.0.0.1:1/keine")
	t.Setenv("BACKUP_DIR", t.TempDir())

	zeitplanMitSicherungAus(t, true).fuehreRestoreProbeAus(context.Background())

	if !strings.Contains(protokoll.String(), "übersprungen") || strings.Contains(protokoll.String(), "FEHLGESCHLAGEN") {
		t.Errorf("die Probe lief trotz abgeschalteter Sicherung:\n%s", protokoll.String())
	}
}

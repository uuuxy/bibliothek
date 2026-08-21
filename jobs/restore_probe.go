package jobs

// restore_probe.go — Wöchentliche Wiederherstellungs-Probe der ECHTEN Backup-Dateien.
//
// Die CI-Drill (backup_drill_pg_test.go) beweist bei jedem Push, dass der MECHANISMUS
// funktioniert — mit einem in CI frisch erzeugten Backup. Was sie nicht beweisen kann:
// dass die Dateien, die der nächtliche Job auf der Betriebs-Platte hinterlässt, sich
// wirklich wiederherstellen lassen. Genau dort lauern die Ernstfall-Überraschungen:
// gewechselter Schlüssel, gekappte Datei, Eigenheiten des echten Datenbestands.
// „Wir haben Backups" ist ohne diese Probe eine Konvention — der schlechteste Moment,
// das zu erfahren, ist der Ernstfall (Schema-Erweiterung 21.08.2026).
//
// Ablauf: jüngste backup_*.sql.gz.enc → entschlüsseln → entpacken → psql in eine
// Wegwerf-Datenbank → Tabellen zählen → Ergebnis nach system_einstellungen. Die
// Betriebsbereitschafts-Seite liest das Ergebnis als eigenen Befund; eine
// fehlgeschlagene oder stehengebliebene Probe erreicht so die tägliche Alarm-Mail.

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"
)

// RestoreProbeErgebnis ist das gespeicherte Resultat des letzten Probelaufs.
type RestoreProbeErgebnis struct {
	Zeitpunkt   time.Time `json:"zeitpunkt"`
	Erfolg      bool      `json:"erfolg"`
	BackupDatei string    `json:"backup_datei,omitempty"`
	Tabellen    int       `json:"tabellen,omitempty"`
	DauerMS     int64     `json:"dauer_ms"`
	Fehler      string    `json:"fehler,omitempty"`
}

const (
	// RestoreProbeSchluessel ist der system_einstellungen-Schlüssel des Ergebnisses.
	RestoreProbeSchluessel = "restore_probe_ergebnis"
	restoreProbeDBName     = "bibliothek_restore_probe_wegwerf"
	// restoreProbeMinTabellen: Ein eingespielter Dump mit weniger Tabellen ist kein
	// Bibliotheks-Schema — „psql lief durch" allein wäre auch bei einer leeren Datei wahr.
	restoreProbeMinTabellen = 20
)

// RunRestoreProbe führt die Probe aus und speichert das Ergebnis. Fehlt der
// Backup-Schlüssel oder die DATABASE_URL, unterbleibt sie still — diesen Zustand
// meldet die Betriebsbereitschaft bereits über den Backup-Befund als kritisch.
func (s *Scheduler) RunRestoreProbe() {
	encKey, dsn, backupDir, ok := resolveBackupEnv()
	if !ok {
		return
	}
	start := time.Now()
	ergebnis := s.fuehreRestoreProbeAus(encKey, dsn, backupDir)
	ergebnis.Zeitpunkt = time.Now().UTC()
	ergebnis.DauerMS = time.Since(start).Milliseconds()
	s.speichereRestoreProbe(ergebnis)
	if ergebnis.Erfolg {
		log.Printf("Restore-Probe: ERFOLG — %s in %d ms wiederhergestellt (%d Tabellen)",
			ergebnis.BackupDatei, ergebnis.DauerMS, ergebnis.Tabellen)
	} else {
		log.Printf("Restore-Probe: FEHLGESCHLAGEN — %s", ergebnis.Fehler)
	}
}

func (s *Scheduler) fuehreRestoreProbeAus(encKey, dsn, backupDir string) RestoreProbeErgebnis {
	datei, err := juengsteBackupDatei(backupDir)
	if err != nil {
		return RestoreProbeErgebnis{Fehler: err.Error()}
	}
	e := RestoreProbeErgebnis{BackupDatei: filepath.Base(datei)}

	roh, err := os.ReadFile(datei) // #nosec G304 - Pfad stammt aus dem Glob über BACKUP_DIR
	if err != nil {
		e.Fehler = fmt.Sprintf("Backup-Datei nicht lesbar: %v", err)
		return e
	}
	sqlText, err := entschluesseleBackup(encKey, roh)
	if err != nil {
		e.Fehler = fmt.Sprintf("%s: %v", e.BackupDatei, err)
		return e
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
	defer cancel()

	// Wegwerf-Datenbank auf demselben Server; WITH (FORCE) räumt Reste eines
	// abgebrochenen Vorlaufs ab. Der App-Pool darf das: CREATE DATABASE läuft
	// ausserhalb von Transaktionen, und der Betriebs-Nutzer ist der DB-Eigentümer.
	if _, err := s.db.Exec(ctx, `DROP DATABASE IF EXISTS `+restoreProbeDBName+` WITH (FORCE)`); err != nil {
		e.Fehler = fmt.Sprintf("Wegwerf-Datenbank aufräumen: %v", err)
		return e
	}
	if _, err := s.db.Exec(ctx, `CREATE DATABASE `+restoreProbeDBName); err != nil {
		e.Fehler = fmt.Sprintf("Wegwerf-Datenbank anlegen: %v", err)
		return e
	}
	defer func() {
		hintergrund, abbruch := context.WithTimeout(context.Background(), time.Minute)
		defer abbruch()
		if _, err := s.db.Exec(hintergrund, `DROP DATABASE IF EXISTS `+restoreProbeDBName+` WITH (FORCE)`); err != nil {
			log.Printf("Restore-Probe: Wegwerf-Datenbank blieb stehen (%v) — der nächste Lauf räumt sie ab", err)
		}
	}()

	if err := spieleDumpEin(ctx, dsn, sqlText); err != nil {
		e.Fehler = err.Error()
		return e
	}

	tabellen, err := zaehleProbeTabellen(ctx, dsn)
	if err != nil {
		e.Fehler = fmt.Sprintf("Gegenprobe fehlgeschlagen: %v", err)
		return e
	}
	e.Tabellen = tabellen
	if tabellen < restoreProbeMinTabellen {
		e.Fehler = fmt.Sprintf("nur %d Tabellen wiederhergestellt (erwartet ≥ %d) — der Dump ist unvollständig", tabellen, restoreProbeMinTabellen)
		return e
	}
	e.Erfolg = true
	return e
}

// speichereRestoreProbe legt das Ergebnis als JSON in system_einstellungen ab —
// dieselbe Quelle, aus der die Betriebsbereitschafts-Seite alle Zustände liest.
func (s *Scheduler) speichereRestoreProbe(e RestoreProbeErgebnis) {
	wert, err := json.Marshal(e)
	if err != nil {
		log.Printf("Restore-Probe: Ergebnis nicht serialisierbar: %v", err)
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if _, err := s.db.Exec(ctx, `
		INSERT INTO system_einstellungen (schluessel, wert) VALUES ($1, $2)
		ON CONFLICT (schluessel) DO UPDATE SET wert = EXCLUDED.wert, aktualisiert_am = CURRENT_TIMESTAMP`,
		RestoreProbeSchluessel, string(wert)); err != nil {
		log.Printf("Restore-Probe: Ergebnis konnte nicht gespeichert werden: %v", err)
	}
}

// kuerzeFehlertext begrenzt psql-Stderr auf eine handhabbare Länge fürs Ergebnis.
func kuerzeFehlertext(s string) string {
	const max = 500
	if len(s) > max {
		return s[:max] + "…"
	}
	return s
}

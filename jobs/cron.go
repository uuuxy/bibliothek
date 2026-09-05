package jobs

import (
	"context"
	"log"
	"os"
	"time"

	"bibliothek/db"
	"bibliothek/internal/service"
	"bibliothek/repository"

	"github.com/robfig/cron/v3"
)

// Scheduler verwaltet Hintergrund-Automatisierungsaufgaben.
type Scheduler struct {
	db        db.PgxPoolIface
	auditRepo repository.AuditRepository
	cron      *cron.Cron
}

// NewScheduler erstellt und gibt eine neue Scheduler-Instanz zurück.
//
// Der Zeitplan ist fest auf UTC genagelt (WithLocation), NICHT auf time.Local. Sonst
// hinge er an der Container-Zeitzone: Setzt jemand später TZ=Europe/Berlin (die
// .env.example empfiehlt es für lesbare Logs), liefe der Backup-Job "30 2 * * *" in der
// Nacht der Sommerzeit-Umstellung ins Leere — 02:30 Ortszeit existiert dann nicht, und
// die Sicherung fiele still aus (der gefährlichste Zustand, den dieses System kennt).
// Mit UTC bleiben alle Cron-Zeiten DST-immun; die Kommentare unten rechnen bewusst in UTC.
func NewScheduler(db db.PgxPoolIface, auditRepo repository.AuditRepository) *Scheduler {
	return &Scheduler{
		db:        db,
		auditRepo: auditRepo,
		cron:      cron.New(cron.WithLocation(time.UTC)),
	}
}

// Start registriert alle Cronjobs für DSGVO, Backup und Vorhaltefristen.
func (s *Scheduler) Start() {
	// Tägliche DSGVO-Anonymisierung und Abgänger-Löschung um Mitternacht
	if _, err := s.cron.AddFunc("0 0 * * *", s.RunNaechtlicheDSGVO); err != nil {
		log.Printf("Scheduler: Failed to register GDPR jobs: %v", err)
		return
	}

	// Nächtliche Audit-Aufbewahrung um 03:00 UTC — nach dem Backup (02:30), damit die
	// Sicherung des Tages die Einträge noch VOR ihrer Löschung enthält.
	if _, err := s.cron.AddFunc("0 3 * * *", s.RunAuditAufbewahrung); err != nil {
		log.Printf("Scheduler: Failed to register audit retention job: %v", err)
	}

	// Tägliches verschlüsseltes Datenbank-Backup um 02:30 UTC (Zeitraum mit wenig Traffic)
	backup := &BackupJob{}

	// Beim Start sagen, nicht erst um 02:30 in einem Logeintrag, den niemand liest.
	//
	// Ohne BACKUP_ENCRYPTION_KEY überspringt sich der Job (jobs/backup.go) — still. Das
	// ist der gefährlichste Zustand, den dieses System kennt: Es läuft alles normal, und
	// erst wenn jemand ein Backup BRAUCHT, stellt sich heraus, dass es nie eines gab.
	// Der Admin-Wächter (api/backup_status.go) meldet den Fall zwar als "critical",
	// dafür muss aber jemand ins Dashboard sehen. Diese Zeile steht im Startprotokoll.
	if os.Getenv("BACKUP_ENCRYPTION_KEY") == "" {
		log.Printf("ACHTUNG: BACKUP_ENCRYPTION_KEY ist nicht gesetzt — es werden KEINE " +
			"Datenbank-Backups erstellt. Der nächtliche Job überspringt sich. Variable in " +
			"der .env setzen und den Stack neu starten.")
	} else if SchluesselIstSchwach(os.Getenv("BACKUP_ENCRYPTION_KEY")) {
		log.Printf("ACHTUNG: BACKUP_ENCRYPTION_KEY ist kürzer als %d Zeichen. Auch scrypt "+
			"rettet eine kurze Passphrase nicht — sie ist an einer entwendeten "+
			"Backup-Datei offline durchprobierbar.", MinBackupSchluesselLaenge)
	}

	if _, err := s.cron.AddFunc("30 2 * * *", func() {
		log.Println("Scheduler Backup: starting scheduled daily database backup...")
		backup.RunDatabaseBackup()
	}); err != nil {
		log.Printf("Scheduler: Failed to register backup job: %v", err)
	}

	// Stündliche Bereinigung abgelaufener Idempotenz-Schlüssel, damit die Tabelle nicht
	// unbegrenzt wächst. 24h Retention reicht weit über die Lebensdauer eines Scan-Retrys hinaus.
	if _, err := s.cron.AddFunc("17 * * * *", func() {
		s.RunIdempotencyCleanup()
	}); err != nil {
		log.Printf("Scheduler: Failed to register idempotency cleanup job: %v", err)
	}

	// Alle 6 Stunden fehlende/fehlgeschlagene Buchcover nachladen (neu importierte Titel
	// und transiente FAILED-Fälle). Der Re-Entrancy-Guard verhindert Überlappung mit dem
	// Start-Lauf. Lokales WebP wird dabei erzeugt.
	if _, err := s.cron.AddFunc("0 */6 * * *", func() {
		service.NewCoverService(s.db).SyncMissingCoversAsync()
	}); err != nil {
		log.Printf("Scheduler: Failed to register cover resync job: %v", err)
	}

	// Stündlich abgelaufene „abholbereit"-Reservierungen abräumen (No-Show verliert den
	// Platz) und das Exemplar dem nächsten Wartenden zuteilen. Ohne diesen Lauf blieb eine
	// nicht abgeholte Reservierung für immer 'abholbereit' — der Schüler fiel still aus der
	// Warteschlange (Betreiber-Entscheidung 19.08.2026).
	if _, err := s.cron.AddFunc("23 * * * *", s.RunVormerkungVerfall); err != nil {
		log.Printf("Scheduler: Failed to register vormerkung expiry job: %v", err)
	}

	// Wöchentliche Restore-Probe, Sonntag 03:30 UTC — nach dem Backup (02:30) und der
	// Audit-Aufbewahrung (03:00), damit sie die jüngste Datei prüft und keinem Job in
	// die Quere kommt. Beweist, dass die ECHTEN Dateien auf der Platte wiederherstellbar
	// sind — die CI-Drill beweist nur den Mechanismus (jobs/restore_probe.go).
	if _, err := s.cron.AddFunc("30 3 * * 0", s.RunRestoreProbe); err != nil {
		log.Printf("Scheduler: Failed to register restore probe job: %v", err)
	}
	// Nach einem FEHLGESCHLAGENEN Lauf probt der Start erneut — sonst stünde die
	// Betriebsbereitschaft nach der Behebung bis zu sieben Tage auf „kritisch" und die
	// Alarm-Mail käme jeden Tag (Prüfung 22.08.2026). Nur nach Fehlschlag: Ein gesunder
	// Stack soll beim Neustart nicht jedes Mal eine Datenbank anlegen und einspielen.
	go s.probeBeimStartNachFehlschlag()

	s.cron.Start()
	log.Println("Scheduler: GDPR, backup, retention, and idempotency cleanup jobs successfully started.")
}

// Stop hält den Cron-Runner des Schedulers an.
func (s *Scheduler) Stop() {
	s.cron.Stop()
}

// RunIdempotencyCleanup entfernt abgelaufene Idempotenz-Schlüssel (> 24h), damit die Tabelle
// nicht unbegrenzt wächst. Die Lebensdauer eines Scanner-Retrys liegt im Sekunden-/Minutenbereich,
// daher ist 24h Retention großzügig.
func (s *Scheduler) RunIdempotencyCleanup() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	tag, err := s.db.Exec(ctx, "DELETE FROM idempotency_keys WHERE created_at < NOW() - INTERVAL '24 hours'")
	if err != nil {
		log.Printf("Scheduler Idempotency Cleanup: Fehler beim Löschen abgelaufener Schlüssel: %v", err)
		return
	}
	if n := tag.RowsAffected(); n > 0 {
		log.Printf("Scheduler Idempotency Cleanup: %d abgelaufene Idempotenz-Schlüssel entfernt.", n)
	}
}

// RunVormerkungVerfall räumt abgelaufene „abholbereit"-Reservierungen ab und teilt das
// Exemplar dem nächsten Wartenden zu (siehe VerfalleAbgelaufeneVormerkungen).
func (s *Scheduler) RunVormerkungVerfall() {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	verfallen, neuBereit, err := repository.NewVormerkungRepository(s.db).VerfalleAbgelaufeneVormerkungen(ctx)
	if err != nil {
		log.Printf("Scheduler Vormerkung-Verfall: Fehler: %v", err)
		return
	}
	if verfallen > 0 {
		log.Printf("Scheduler Vormerkung-Verfall: %d abgelaufene Abhol-Reservierung(en) verfallen, %d Exemplar(e) an den nächsten Wartenden zugeteilt.", verfallen, neuBereit)
	}
}

// RunNaechtlicheDSGVO ist die nächtliche Folge der DSGVO-Routinen — als Methode, damit ein
// Test dieselbe Folge in derselben Reihenfolge fahren kann wie der Cron.
func (s *Scheduler) RunNaechtlicheDSGVO() {
	s.RunGDPRAnonymizeLoans()
	// Erst anonymisieren, dann löschen: Die Löschung trifft nur anonymisierte Zeilen
	// (repository.PredikatAbgaengerLoeschung). In dieser Reihenfolge fallen beide in
	// dieselbe Nacht, und der Wächter der Betriebsbereitschaft sieht nie eine
	// anonymisierte Hülle einen Tag lang als Rückstand. Bis 05.09.2026 lief die Löschung
	// zuerst — und schnitt die Karenz am Stichtag ab.
	s.RunGDPRAnonymizeOldData()
	s.RunGDPRDeleteAbgaenger()
	// Lesehistorie befristen (cron_dsgvo_lesehistorie.go): trennt abgeschlossene
	// Ausleihen nach Frist vom Schüler. Fristen stehen in den Einstellungen.
	s.RunLesehistorieBefristung()
	// Erledigte Anliegen befristen (cron_dsgvo_anliegen.go): Wünsche und Meldungen
	// aus dem Kollegiums-Portal tragen Freitext, Klasse und den Namen der Lehrkraft.
	s.RunAnliegenBefristung()
}

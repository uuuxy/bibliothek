// Package jobs fährt die zeitgesteuerten Läufe des Servers: die Löschläufe nach den
// eingestellten Fristen, die Sicherung der Datenbank mit ihrer Restore-Probe, den Verfall von
// Vormerkungen und den Abgleich der Cover. Der Zeitplan rechnet in UTC.
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

// Zeitplan meldet die Läufe beim Cron an und führt sie aus.
type Zeitplan struct {
	pool              db.Pool
	protokollAbfragen repository.ProtokollAbfragen
	cron              *cron.Cron
}

// NewZeitplan baut den Zeitplan.
//
// Er rechnet fest in UTC (WithLocation), nicht in time.Local. Sonst hinge er an der Zeitzone
// des Containers: Setzt jemand TZ=Europe/Berlin (die .env.example empfiehlt es für lesbare
// Logs), liefe die Sicherung "30 2 * * *" in der Nacht der Umstellung auf Sommerzeit ins Leere,
// denn 02:30 Ortszeit gibt es dann nicht, und die Sicherung fiele unbemerkt aus. In UTC gibt es
// keine Umstellung; die Kommentare unten rechnen deshalb in UTC.
func NewZeitplan(pool db.Pool, protokollAbfragen repository.ProtokollAbfragen) *Zeitplan {
	return &Zeitplan{
		pool:              pool,
		protokollAbfragen: protokollAbfragen,
		cron:              cron.New(cron.WithLocation(time.UTC)),
	}
}

// Die drei Zeiten der Nacht hängen aneinander: erst die Sicherung, dann die Aufbewahrung des
// Protokolls, dann am Sonntag die Restore-Probe. Die Reihenfolge prüft zeitplan_test.go.
const (
	zeitSicherung             = "30 2 * * *"
	zeitProtokollAufbewahrung = "0 3 * * *"
	zeitRestoreProbe          = "30 3 * * 0"
)

// Starte meldet alle Läufe beim Cron an und startet ihn.
func (z *Zeitplan) Starte() {
	// Täglich um Mitternacht: Anonymisierung und Löschung nach DSGVO.
	if _, err := z.cron.AddFunc("0 0 * * *", z.FuehreNaechtlicheDSGVOAus); err != nil {
		log.Printf("Scheduler: Failed to register GDPR jobs: %v", err)
		return
	}

	// Täglich um 03:00 UTC die Aufbewahrung des Protokolls, nach der Sicherung (02:30), damit
	// die Sicherung des Tages die Einträge noch vor ihrer Löschung enthält.
	if _, err := z.cron.AddFunc(zeitProtokollAufbewahrung, z.FuehreProtokollAufbewahrungAus); err != nil {
		log.Printf("Scheduler: Failed to register audit retention job: %v", err)
	}

	// Täglich um 02:30 UTC die verschlüsselte Sicherung der Datenbank, in einer ruhigen Zeit.
	backup := &BackupLauf{}

	// Beim Start sagen, nicht erst um 02:30 in einer Logzeile, die niemand liest.
	//
	// Ohne BACKUP_ENCRYPTION_KEY überspringt sich die Sicherung (jobs/backup.go) ohne Meldung.
	// Das ist der gefährlichste Zustand des Programms: Alles läuft normal, und erst wenn jemand
	// eine Sicherung braucht, stellt sich heraus, dass es nie eine gab. Der Wächter der
	// Verwaltung (api/system/backup_stand.go) meldet den Fall als "critical", dafür muss aber
	// jemand hinsehen. Diese Zeile steht im Log des Starts.
	if os.Getenv("BACKUP_ENCRYPTION_KEY") == "" {
		log.Printf("ACHTUNG: BACKUP_ENCRYPTION_KEY ist nicht gesetzt — es werden KEINE " +
			"Datenbank-Backups erstellt. Der nächtliche Job überspringt sich. Variable in " +
			"der .env setzen und den Stack neu starten.")
	} else if IstSchluesselSchwach(os.Getenv("BACKUP_ENCRYPTION_KEY")) {
		log.Printf("ACHTUNG: BACKUP_ENCRYPTION_KEY ist kürzer als %d Zeichen. Auch scrypt "+
			"rettet eine kurze Passphrase nicht — sie ist an einer entwendeten "+
			"Backup-Datei offline durchprobierbar.", MinBackupSchluesselLaenge)
	}

	if _, err := z.cron.AddFunc(zeitSicherung, func() {
		log.Println("Scheduler Backup: starting scheduled daily database backup...")
		backup.FuehreAus()
	}); err != nil {
		log.Printf("Scheduler: Failed to register backup job: %v", err)
	}

	// Stündlich die abgelaufenen Idempotenz-Schlüssel löschen, damit die Tabelle nicht ohne
	// Grenze wächst. 24 Stunden reichen weit über die Wiederholung eines Scans hinaus.
	if _, err := z.cron.AddFunc("17 * * * *", func() {
		z.FuehreIdempotenzBereinigungAus()
	}); err != nil {
		log.Printf("Scheduler: Failed to register idempotency cleanup job: %v", err)
	}

	// Alle sechs Stunden fehlende und gescheiterte Cover nachladen (neu importierte Titel und
	// vorübergehende Fehler, Status FAILED); dabei entsteht das WebP im Haus. Die Sperre gegen
	// gleichzeitige Läufe verhindert die Überschneidung mit dem Lauf beim Start.
	if _, err := z.cron.AddFunc("0 */6 * * *", func() {
		service.NewCoverDienst(z.pool).GleicheFehlendeCoverAb()
	}); err != nil {
		log.Printf("Scheduler: Failed to register cover resync job: %v", err)
	}

	// Stündlich abgelaufene Vormerkungen im Zustand „abholbereit" verfallen lassen (wer nicht
	// abholt, verliert den Platz) und das Exemplar dem nächsten Wartenden zuteilen. Ohne den
	// Lauf bliebe eine nicht abgeholte Vormerkung für immer 'abholbereit', und die Wartenden
	// dahinter kämen nie an die Reihe.
	if _, err := z.cron.AddFunc("23 * * * *", z.FuehreVormerkungsVerfallAus); err != nil {
		log.Printf("Scheduler: Failed to register vormerkung expiry job: %v", err)
	}

	// Wöchentlich die Restore-Probe, Sonntag 03:30 UTC, nach der Sicherung (02:30) und der
	// Aufbewahrung des Protokolls (03:00), damit sie die jüngste Datei prüft und keinem Lauf in
	// die Quere kommt. Sie belegt, dass die echten Dateien auf der Platte sich wiederherstellen
	// lassen; die Übung in der CI belegt nur das Verfahren (jobs/restore_probe.go).
	if _, err := z.cron.AddFunc(zeitRestoreProbe, z.FuehreRestoreProbeAus); err != nil {
		log.Printf("Scheduler: Failed to register restore probe job: %v", err)
	}
	// Nach einem gescheiterten Lauf probt der Start erneut, sonst stünde die
	// Betriebsbereitschaft nach der Behebung bis zu sieben Tage auf „kritisch", und die
	// Alarm-Mail käme jeden Tag. Nur nach einem Fehlschlag: Ein gesunder Stack soll nicht bei
	// jedem Neustart eine Datenbank anlegen und einspielen.
	go z.probeBeimStartNachFehlschlag()

	z.cron.Start()
	log.Println("Scheduler: GDPR, backup, retention, and idempotency cleanup jobs successfully started.")
}

// Stoppe hält den Cron an.
func (z *Zeitplan) Stoppe() {
	z.cron.Stop()
}

// FuehreIdempotenzBereinigungAus löscht die Idempotenz-Schlüssel, die älter als 24 Stunden
// sind, damit die Tabelle nicht ohne Grenze wächst. Ein Scanner wiederholt eine Anfrage nach
// Sekunden oder Minuten; 24 Stunden sind großzügig.
func (z *Zeitplan) FuehreIdempotenzBereinigungAus() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	n, err := repository.LoescheAlteIdempotenzSchluessel(ctx, z.pool)
	if err != nil {
		log.Printf("Scheduler Idempotency Cleanup: Fehler beim Löschen abgelaufener Schlüssel: %v", err)
		return
	}
	if n > 0 {
		log.Printf("Scheduler Idempotency Cleanup: %d abgelaufene Idempotenz-Schlüssel entfernt.", n)
	}
}

// FuehreVormerkungsVerfallAus lässt abgelaufene Vormerkungen im Zustand „abholbereit"
// verfallen und teilt das Exemplar dem nächsten Wartenden zu (LasseAbgelaufeneVerfallen).
func (z *Zeitplan) FuehreVormerkungsVerfallAus() {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	verfallen, neuBereit, err := repository.NewVormerkungsAbfragen(z.pool).LasseAbgelaufeneVerfallen(ctx)
	if err != nil {
		log.Printf("Scheduler Vormerkung-Verfall: Fehler: %v", err)
		return
	}
	if verfallen > 0 {
		log.Printf("Scheduler Vormerkung-Verfall: %d abgelaufene Abhol-Reservierung(en) verfallen, %d Exemplar(e) an den nächsten Wartenden zugeteilt.", verfallen, neuBereit)
	}
}

// FuehreNaechtlicheDSGVOAus führt die Läufe der Nacht nach DSGVO nacheinander aus. Sie ist
// eine Methode, damit ein Test dieselbe Folge in derselben Reihenfolge fahren kann wie der
// Cron.
func (z *Zeitplan) FuehreNaechtlicheDSGVOAus() {
	z.FuehreBearbeiterAnonymisierungAus()
	// Erst anonymisieren, dann löschen: Die Löschung trifft nur anonymisierte Zeilen
	// (repository.praedikatAbgaengerLoeschung). In dieser Reihenfolge fallen beide in dieselbe
	// Nacht, die Karenz wird am Stichtag nicht abgeschnitten, und der Wächter der
	// Betriebsbereitschaft sieht nie eine anonymisierte Hülle einen Tag lang als Rückstand.
	z.FuehreSchuelerAnonymisierungAus()
	z.FuehreAbgaengerLoeschungAus()
	// Gelöschte Kollegen nach 180 Tagen im Papierkorb endgültig löschen
	// (cron_dsgvo_papierkorb.go).
	z.FuehrePapierkorbLoeschungAus()
	// Lesehistorie befristen (cron_dsgvo_lesehistorie.go): trennt abgeschlossene Ausleihen nach
	// der Frist vom Leser. Die Fristen stehen in den Einstellungen.
	z.FuehreLesehistorieBefristungAus()
	// Erledigte Anliegen befristen (cron_dsgvo_anliegen.go): Wünsche und Meldungen aus dem
	// Portal des Kollegiums tragen Freitext, Klasse und den Namen der Lehrkraft.
	z.FuehreAnliegenBefristungAus()
	// Erledigte Reservierungen von Klassensätzen befristen (cron_dsgvo_anliegen.go), mit
	// derselben Frist wie die Anliegen.
	z.FuehreKlassensatzBefristungAus()
	// Quittierte Nachbuch-Meldungen befristen (cron_dsgvo_nachbuch.go): Frist der
	// Lesehistorie, höchstens 30 Tage, denn sie nennen Ausleiher und Vorbesitzer mit Namen.
	z.FuehreNachbuchMeldungenBefristungAus()
}

package jobs

// Wöchentliche Restore-Probe der echten Dateien der Sicherung.
//
// Die Übung in der CI (backup_drill_pg_test.go) belegt bei jedem Push, dass das Verfahren wirkt,
// mit einer in der CI frisch erzeugten Sicherung. Was sie nicht belegen kann: dass sich die
// Dateien, die der nächtliche Lauf auf der Platte des Betriebs hinterlässt, wirklich
// wiederherstellen lassen. Genau dort liegen die Überraschungen des Ernstfalls: gewechselter
// Schlüssel, abgeschnittene Datei, Eigenheiten des echten Bestands. „Wir haben Sicherungen" ist
// ohne die Probe eine Behauptung, und der schlechteste Moment, das zu erfahren, ist der Ernstfall.
//
// Ablauf: jüngste backup_*.sql.gz.enc → entschlüsseln → entpacken → mit psql in eine
// Wegwerf-Datenbank → Tabellen zählen → Ergebnis nach system_einstellungen. Die Seite der
// Betriebsbereitschaft liest das Ergebnis als eigenen Befund; eine gescheiterte oder
// stehengebliebene Probe erreicht so die tägliche Alarm-Mail.

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"bibliothek/repository"
)

// RestoreProbeErgebnis ist das gespeicherte Ergebnis des letzten Laufs der Probe.
type RestoreProbeErgebnis struct {
	Zeitpunkt   time.Time `json:"zeitpunkt"`
	Erfolg      bool      `json:"erfolg"`
	BackupDatei string    `json:"backup_datei,omitempty"`
	Tabellen    int       `json:"tabellen,omitempty"`
	DauerMS     int64     `json:"dauer_ms"`
	Fehler      string    `json:"fehler,omitempty"`
}

const (
	// RestoreProbeSchluessel ist der Schlüssel des Ergebnisses in system_einstellungen.
	RestoreProbeSchluessel = "restore_probe_ergebnis"
	probeDatenbank         = "bibliothek_restore_probe_wegwerf"
	// probeMindestTabellen: Ein eingespielter Dump mit weniger Tabellen ist kein Schema der
	// Bibliothek; „psql lief durch" wäre auch bei einer leeren Datei wahr.
	probeMindestTabellen = 20
)

// FuehreRestoreProbeAus führt die Probe aus und speichert das Ergebnis. Fehlt der Schlüssel der
// Sicherung oder die DATABASE_URL, unterbleibt sie ohne Meldung; diesen Zustand meldet die
// Betriebsbereitschaft schon über den Befund der Sicherung als kritisch.
func (z *Zeitplan) FuehreRestoreProbeAus() {
	schluessel, dsn, verzeichnis, ok := leseBackupUmgebung()
	if !ok {
		return
	}
	start := time.Now()
	ergebnis := z.probeJuengsteSicherung(schluessel, dsn, verzeichnis)
	ergebnis.Zeitpunkt = time.Now().UTC()
	ergebnis.DauerMS = time.Since(start).Milliseconds()
	z.speichereProbeErgebnis(ergebnis)
	if ergebnis.Erfolg {
		log.Printf("Restore-Probe: ERFOLG — %s in %d ms wiederhergestellt (%d Tabellen)",
			ergebnis.BackupDatei, ergebnis.DauerMS, ergebnis.Tabellen)
	} else {
		log.Printf("Restore-Probe: FEHLGESCHLAGEN — %s", ergebnis.Fehler)
	}
}

// probeJuengsteSicherung spielt die jüngste Sicherung in eine Wegwerf-Datenbank ein und zählt
// ihre Tabellen.
func (z *Zeitplan) probeJuengsteSicherung(schluessel, dsn, verzeichnis string) RestoreProbeErgebnis {
	datei, err := juengsteBackupDatei(verzeichnis)
	if err != nil {
		return RestoreProbeErgebnis{Fehler: err.Error()}
	}
	e := RestoreProbeErgebnis{BackupDatei: filepath.Base(datei)}

	roh, err := os.ReadFile(datei) // #nosec G304 - Pfad stammt aus dem Glob über BACKUP_DIR
	if err != nil {
		e.Fehler = fmt.Sprintf("Backup-Datei nicht lesbar: %v", err)
		return e
	}
	sqlText, err := entschluesseleBackup(schluessel, roh)
	if err != nil {
		e.Fehler = fmt.Sprintf("%s: %v", e.BackupDatei, err)
		return e
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
	defer cancel()

	// Wegwerf-Datenbank auf demselben Server; WITH (FORCE) räumt Reste eines abgebrochenen
	// Laufs ab. Der Pool des Programms darf das: CREATE DATABASE läuft außerhalb von
	// Transaktionen, und der Benutzer des Betriebs ist Eigentümer der Datenbank.
	if _, err := z.pool.Exec(ctx, `DROP DATABASE IF EXISTS `+probeDatenbank+` WITH (FORCE)`); err != nil {
		e.Fehler = fmt.Sprintf("Wegwerf-Datenbank aufräumen: %v", err)
		return e
	}
	if _, err := z.pool.Exec(ctx, `CREATE DATABASE `+probeDatenbank); err != nil {
		e.Fehler = fmt.Sprintf("Wegwerf-Datenbank anlegen: %v", err)
		return e
	}
	defer func() {
		hintergrund, abbruch := context.WithTimeout(context.Background(), time.Minute)
		defer abbruch()
		if _, err := z.pool.Exec(hintergrund, `DROP DATABASE IF EXISTS `+probeDatenbank+` WITH (FORCE)`); err != nil {
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
	if tabellen < probeMindestTabellen {
		e.Fehler = fmt.Sprintf("nur %d Tabellen wiederhergestellt (erwartet ≥ %d) — der Dump ist unvollständig", tabellen, probeMindestTabellen)
		return e
	}
	e.Erfolg = true
	return e
}

// speichereProbeErgebnis legt das Ergebnis als JSON in system_einstellungen ab, in derselben
// Quelle, aus der die Seite der Betriebsbereitschaft alle Zustände liest.
func (z *Zeitplan) speichereProbeErgebnis(e RestoreProbeErgebnis) {
	wert, err := json.Marshal(e)
	if err != nil {
		log.Printf("Restore-Probe: Ergebnis nicht serialisierbar: %v", err)
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	zustand := repository.NewBetriebszustandAbfragen(z.pool)
	if err := zustand.SchreibeEinstellungswert(ctx, RestoreProbeSchluessel, string(wert)); err != nil {
		log.Printf("Restore-Probe: Ergebnis konnte nicht gespeichert werden: %v", err)
	}
}

// probeBeimStartNachFehlschlag wiederholt die Restore-Probe beim Start, wenn der letzte
// gespeicherte Lauf gescheitert ist (cron.go).
func (z *Zeitplan) probeBeimStartNachFehlschlag() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	wert, err := repository.NewBetriebszustandAbfragen(z.pool).LadeEinstellungswert(ctx, RestoreProbeSchluessel)
	if err != nil || wert == "" {
		return // noch nie geprobt oder nicht lesbar: Der Lauf am Sonntag übernimmt
	}
	var letzte RestoreProbeErgebnis
	if err := json.Unmarshal([]byte(wert), &letzte); err != nil || letzte.Erfolg {
		return
	}
	log.Printf("Restore-Probe: letzter Lauf war fehlgeschlagen (%s) — probe beim Start erneut", letzte.Fehler)
	z.FuehreRestoreProbeAus()
}

// kuerzeFehlertext macht die Ausgabe von psql auf Stderr tauglich für das Ergebnis: Zeilen mit
// Daten (CONTEXT: COPY schueler, line 12, column vorname: "…", DETAIL:, HINT:) fallen weg, denn
// der Text landet in der Datenbank, auf der Seite der Betriebsbereitschaft und in der Alarm-Mail
// an frei eingestellte Empfänger. Danach ist er auf 500 Zeichen begrenzt.
func kuerzeFehlertext(s string) string {
	const grenze = 500
	var behalten []string
	for _, zeile := range strings.Split(s, "\n") {
		z := strings.TrimSpace(zeile)
		if z == "" {
			continue
		}
		if strings.HasPrefix(z, "CONTEXT:") || strings.HasPrefix(z, "DETAIL:") ||
			strings.HasPrefix(z, "HINT:") || strings.HasPrefix(z, "COPY ") {
			continue
		}
		behalten = append(behalten, z)
	}
	text := strings.Join(behalten, " | ")
	if len(text) > grenze {
		return text[:grenze] + "…"
	}
	return text
}

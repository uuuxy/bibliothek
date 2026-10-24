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
	"log/slog"
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

// fuehreRestoreProbeAus führt die Probe aus und speichert das Ergebnis. Fehlt der Schlüssel der
// Sicherung oder die DATABASE_URL, unterbleibt sie ohne Meldung; diesen Zustand meldet die
// Betriebsbereitschaft schon über den Befund der Sicherung als kritisch.
func (z *Zeitplan) fuehreRestoreProbeAus(ctx context.Context) {
	schluessel, dsn, verzeichnis, ok := leseBackupUmgebung()
	if !ok {
		return
	}
	start := time.Now()
	ergebnis := z.probeJuengsteSicherung(ctx, schluessel, dsn, verzeichnis)
	// Ein Abbruch beim Herunterfahren sagt nichts über die Sicherung. Gespeichert als Fehlschlag
	// stünde die Betriebsbereitschaft zu Unrecht auf kritisch; es bleibt das letzte Ergebnis.
	if ctx.Err() != nil {
		slog.Warn("Restore-Probe: abgebrochen, der Server fährt herunter", "fehler", ctx.Err())
		return
	}
	ergebnis.Zeitpunkt = time.Now().UTC()
	ergebnis.DauerMS = time.Since(start).Milliseconds()
	z.speichereProbeErgebnis(ctx, ergebnis)
	if ergebnis.Erfolg {
		slog.Info("Restore-Probe: ERFOLG", "datei", ergebnis.BackupDatei, "dauer_ms", ergebnis.DauerMS, "tabellen", ergebnis.Tabellen)
	} else {
		slog.Error("Restore-Probe: FEHLGESCHLAGEN", "fehler", ergebnis.Fehler)
	}
}

// probeJuengsteSicherung spielt die jüngste Sicherung in eine Wegwerf-Datenbank ein und zählt
// ihre Tabellen.
func (z *Zeitplan) probeJuengsteSicherung(ctx context.Context, schluessel, dsn, verzeichnis string) RestoreProbeErgebnis {
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

	ctx, cancel := context.WithTimeout(ctx, 15*time.Minute)
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
	// Das Abräumen läuft auch nach einem Abbruch: Sonst bliebe die Wegwerf-Datenbank bis zum
	// nächsten Lauf stehen.
	defer func() {
		hintergrund, abbruch := context.WithTimeout(context.WithoutCancel(ctx), time.Minute)
		defer abbruch()
		if _, err := z.pool.Exec(hintergrund, `DROP DATABASE IF EXISTS `+probeDatenbank+` WITH (FORCE)`); err != nil {
			slog.Warn("Restore-Probe: Wegwerf-Datenbank blieb stehen — der nächste Lauf räumt sie ab", "fehler", err)
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
func (z *Zeitplan) speichereProbeErgebnis(ctx context.Context, e RestoreProbeErgebnis) {
	wert, err := json.Marshal(e)
	if err != nil {
		slog.Error("Restore-Probe: Ergebnis nicht serialisierbar", "fehler", err)
		return
	}
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	zustand := repository.NewBetriebszustandAbfragen(z.pool)
	if err := zustand.SchreibeEinstellungswert(ctx, RestoreProbeSchluessel, string(wert)); err != nil {
		slog.Error("Restore-Probe: Ergebnis konnte nicht gespeichert werden", "fehler", err)
	}
}

// probeBeimStartNachFehlschlag wiederholt die Restore-Probe beim Start, wenn der letzte
// gespeicherte Lauf gescheitert ist (cron.go).
func (z *Zeitplan) probeBeimStartNachFehlschlag(ctx context.Context) {
	lesen, abbrechen := context.WithTimeout(ctx, 10*time.Second)
	defer abbrechen()
	wert, err := repository.NewBetriebszustandAbfragen(z.pool).LadeEinstellungswert(lesen, RestoreProbeSchluessel)
	if err != nil || wert == "" {
		return // noch nie geprobt oder nicht lesbar: Der Lauf am Sonntag übernimmt
	}
	var letzte RestoreProbeErgebnis
	if err := json.Unmarshal([]byte(wert), &letzte); err != nil || letzte.Erfolg {
		return
	}
	slog.Warn("Restore-Probe: letzter Lauf war fehlgeschlagen — probe beim Start erneut", "fehler", letzte.Fehler)
	z.fuehreRestoreProbeAus(ctx)
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

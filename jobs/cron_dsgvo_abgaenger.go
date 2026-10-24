package jobs

import (
	"context"
	"log/slog"
	"time"

	"bibliothek/pkg/logger"
	"bibliothek/repository"
)

// Die endgültige Löschung von Abgängern, der einzige Weg, auf dem das Programm Daten von
// Schülern unwiederbringlich entfernt. Sie steht deshalb für sich und nicht zwischen den beiden
// Läufen der Anonymisierung, die nur Felder überschreiben.

// ── DSGVO: Löschung der Abgänger (ab 30. Januar des Folgejahres, nur anonymisierte Zeilen) ──

// FuehreAbgaengerLoeschungAus löscht ehemalige Schüler (ist_abgaenger = true) endgültig nach
// DSGVO, wenn sie
//   - die Schule in einem vergangenen Jahr verlassen haben (abgaenger_jahr < aktuelles Jahr),
//   - keine Bücher mehr ausgeliehen haben,
//   - keine unbezahlten Forderungen für Schäden haben und
//   - seit Beginn des Kalenderjahres mindestens 30 Tage vergangen sind (Näherung für „30 Tage
//     nach Ende des Schuljahres"). Gelöscht wird nur, was die Karenz durchlaufen hat, also schon
//     anonymisiert ist; die Löschung läuft deshalb nach der Anonymisierung
//     (fuehreNaechtlicheDSGVOAus).
//
// Jede Löschung steht einzeln im audit_log (Akteur: SYSTEM).
func (z *Zeitplan) FuehreAbgaengerLoeschungAus(ctx context.Context) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()

	// Die Karenz von 30 Tagen als Stichjahr. Die Rechnung steht in repository/, weil die
	// Selbstprüfung mit demselben Jahr zählen muss: Zwei Rechnungen wären zwei Fristen, von denen
	// eine unbemerkt danebenläge.
	jetzt := time.Now()
	stichjahr := repository.AbgaengerStichjahr(jetzt)

	// Die Zeilen sind gelesen und geschlossen, die Verbindung ist zurück im Pool, bevor die
	// Löschphase beginnt.
	abgaenger, err := repository.LadeLoeschreifeAbgaenger(ctx, z.pool, jetzt)
	if err != nil {
		slog.Error("Scheduler GDPR Delete: Failed to fetch eligible students", "fehler", err)
		return
	}

	if len(abgaenger) == 0 {
		slog.Info("Scheduler GDPR Delete: no eligible students for deletion", "stichjahr", stichjahr)
		return
	}

	slog.Info("Scheduler GDPR Delete: students eligible for DSGVO deletion", "anzahl", len(abgaenger), "stichjahr", stichjahr)

	geloescht := 0
	var fehlschlaege []string

	for _, a := range abgaenger {
		// LoescheAbgaengerEndgueltig statt LegeLeserInPapierkorb: Der Papierkorb behielte die
		// Personendaten. LoescheAbgaengerEndgueltig entfernt sie wirklich (Ausleihen anonymisiert,
		// Datensatz gelöscht); der Grund der Löschung steht über den festen Kontext der Methode im
		// Protokoll.
		if err := z.leserLoeschung.LoescheAbgaengerEndgueltig(ctx, a.ID, ""); err != nil {
			slog.Error("Scheduler GDPR Delete: failed to purge student", "leser", logger.OhneUmbruch(a.ID), "fehler", err)
			fehlschlaege = append(fehlschlaege, a.ID)
			continue
		}

		slog.Info("Scheduler GDPR Delete: deleted student", "leser", logger.OhneUmbruch(a.ID), "klasse", logger.OhneUmbruch(a.Klasse), "abgangsjahr", a.AbgaengerJahr)
		geloescht++
	}

	// Die Zusammenfassung des Laufs ins Protokoll.
	if err := z.protokollAbfragen.ProtokolliereSystemAktion(ctx, "schueler", "BATCH_DELETE",
		"DSGVO-Abgänger-Batch-Löschung",
		map[string]any{
			"geloescht":      geloescht,
			"fehlschlaege":   len(fehlschlaege),
			"cutoff_jahr":    stichjahr,
			"ausgefuehrt_am": time.Now().UTC().Format(time.RFC3339),
		},
	); err != nil {
		slog.Error("audit: BATCH_DELETE konnte nicht protokolliert werden", "fehler", err)
	}

	if len(fehlschlaege) > 0 {
		slog.Error("Scheduler GDPR Delete: completed with failures", "anzahl", len(fehlschlaege), "leser", fehlschlaege)
	} else {
		slog.Info("Scheduler GDPR Delete: successfully deleted students", "anzahl", geloescht)
	}
}

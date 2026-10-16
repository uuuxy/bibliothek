package jobs

import (
	"context"
	"log"
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
//     (FuehreNaechtlicheDSGVOAus).
//
// Jede Löschung steht einzeln im audit_log (Akteur: SYSTEM).
func (z *Zeitplan) FuehreAbgaengerLoeschungAus() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
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
		log.Printf("Scheduler GDPR Delete: Failed to fetch eligible students: %v", err)
		return
	}

	if len(abgaenger) == 0 {
		log.Printf("Scheduler GDPR Delete: no eligible students for deletion (cutoff year: %d)", stichjahr)
		return
	}

	log.Printf("Scheduler GDPR Delete: %d student(s) eligible for DSGVO deletion (Abgangsjahr < %d)",
		len(abgaenger), stichjahr)

	geloescht := 0
	var fehlschlaege []string

	for _, a := range abgaenger {
		// LoescheAbgaengerEndgueltig statt LegeLeserInPapierkorb: Der Papierkorb behielte die
		// Personendaten. LoescheAbgaengerEndgueltig entfernt sie wirklich (Ausleihen anonymisiert,
		// Datensatz gelöscht); der Grund der Löschung steht über den festen Kontext der Methode im
		// Protokoll.
		if err := z.leserLoeschung.LoescheAbgaengerEndgueltig(ctx, a.ID, ""); err != nil {
			log.Printf("Scheduler GDPR Delete: failed to purge student ID %s: %v",
				logger.OhneUmbruch(a.ID), err)
			fehlschlaege = append(fehlschlaege, a.ID)
			continue
		}

		log.Printf("Scheduler GDPR Delete: deleted student ID %s (Klasse %s, Abgang %d)",
			logger.OhneUmbruch(a.ID), logger.OhneUmbruch(a.Klasse), a.AbgaengerJahr)
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
		log.Printf("audit: BATCH_DELETE konnte nicht protokolliert werden: %v", err)
	}

	if len(fehlschlaege) > 0 {
		log.Printf("Scheduler GDPR Delete: completed with %d failure(s): %v", len(fehlschlaege), fehlschlaege)
	} else {
		log.Printf("Scheduler GDPR Delete: successfully deleted %d student(s)", geloescht)
	}
}
